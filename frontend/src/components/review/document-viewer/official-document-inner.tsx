"use client";

// components/review/document-viewer/official-document-inner.tsx
// Penampil naskah resmi surat terbit (SENT), surat eksternal menunggu tanda tangan basah (APPROVED),
// dan surat masuk eksternal (RECEIVED):
// - Surat Internal (SENT): ditampilkan DevExpress RichEdit dalam mode baca + ekspor PDF ber-watermark.
// - Surat Eksternal Keluar (OUTGOING):
//   - Saat APPROVED: bila PDF cetak belum ada, diekspor dari naskah terisi (.docx) ke PDF ber-watermark
//     di latar belakang lalu langsung ditampilkan sebagai tampilan PDF siap cetak untuk ditandatangani basah.
//   - Saat SENT: menampilkan hasil scan PDF bertanda tangan basah yang telah diunggah Approver.
// - Surat Masuk Eksternal (INCOMING / RECEIVED): langsung menampilkan pratinjau dokumen scan PDF yang diunggah.

import { useCallback, useEffect, useRef, useState } from "react";
import {
  CheckCircle2,
  Download,
  ExternalLink,
  FileCheck2,
  FileText,
  Info,
  Loader2,
  Maximize2,
  Minimize2,
  Printer,
  RefreshCw,
} from "lucide-react";

import "devextreme/dist/css/dx.light.css";
import "devexpress-richedit/dist/dx.richedit.css";

import type { RichEdit } from "devexpress-richedit";
import type { LetterDetail } from "@/lib/api/letter-detail";
import {
  fetchFilledLetterDocument,
  fetchOfficialStatus,
  officialPdfUrl,
  uploadOfficialPdf,
} from "@/lib/api/letter-document";
import { ensurePdfExportLibs } from "@/lib/pdf/devexpress-pdf";
import {
  WordZoomControl,
  useWordZoomInteractivity,
} from "@/components/ui/word-zoom-control";
import {
  createWordEditorOptions,
  loadRichEdit,
} from "@/components/surat/letter-editor/word-editor-config";
import { cn } from "@/lib/utils";

const LICENSE_KEY = process.env.NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY ?? "";
const DEFAULT_HEIGHT = "880px";
const EXPORT_TIMEOUT_MS = 60_000;

export interface OfficialDocumentInnerProps {
  letter: LetterDetail;
  onPdfReadyChange?: (ready: boolean, copyFor: string | null) => void;
}

type DocState = "loading" | "ready" | "error";
type PdfState = "checking" | "preparing" | "ready" | "error";

interface PendingExport {
  resolve: (blob: Blob) => void;
  reject: (err: Error) => void;
}

export default function OfficialDocumentInner({
  letter,
  onPdfReadyChange,
}: OfficialDocumentInnerProps) {
  const wrapperRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<RichEdit | null>(null);
  const pendingExportRef = useRef<PendingExport | null>(null);

  const isExternalIncoming =
    letter.direction === "INCOMING" ||
    letter.status === "RECEIVED" ||
    letter.letterType?.code === "SURAT_MASUK_EKSTERNAL";
  const isExternalOutgoing =
    letter.direction === "OUTGOING" ||
    letter.letterType?.code === "SURAT_EKSTERNAL";
  const isPdfViewMode = isExternalIncoming || isExternalOutgoing;
  // Scan PDF sudah langsung tersedia di MinIO untuk INCOMING (RECEIVED) dan OUTGOING yang sudah SENT
  const isPreUploadedScan =
    isExternalIncoming || (isExternalOutgoing && letter.status === "SENT");

  const [docState, setDocState] = useState<DocState>(
    isPreUploadedScan ? "ready" : "loading"
  );
  const [pdfState, setPdfState] = useState<PdfState>(
    isPreUploadedScan ? "ready" : "checking"
  );
  const [message, setMessage] = useState<string | null>(null);
  const [zoomLevel, setZoomLevel] = useState(1);
  const zoomLevelRef = useRef(1);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);
  const [copyFor, setCopyFor] = useState<string | null>(null);

  const effectiveDocState: DocState = isPreUploadedScan ? "ready" : docState;
  const effectivePdfState: PdfState = isPreUploadedScan ? "ready" : pdfState;
  const effectiveMessage = isPreUploadedScan ? null : message;

  const applyZoom = useCallback((level: number) => {
    const clamped = Math.min(2, Math.max(0.5, Math.round(level * 100) / 100));
    setZoomLevel(clamped);
    zoomLevelRef.current = clamped;
    const editor = editorRef.current;
    if (editor && !editor.isDisposed) {
      try {
        editor.zoomLevel = clamped;
        editor.adjust();
      } catch {
        // Abaikan bila instance sedang ditutup
      }
    }
  }, []);

  const { feedbackPercent, handleDevExpressKeyDown } = useWordZoomInteractivity({
    wrapperRef,
    containerRef,
    zoomLevel,
    onZoomChange: applyZoom,
    isReady: !isPdfViewMode && effectiveDocState === "ready",
  });

  const keyDownRef = useRef(handleDevExpressKeyDown);
  useEffect(() => {
    keyDownRef.current = handleDevExpressKeyDown;
  }, [handleDevExpressKeyDown]);

  useEffect(() => {
    onPdfReadyChange?.(effectivePdfState === "ready", copyFor);
  }, [effectivePdfState, copyFor, onPdfReadyChange]);

  const pdfFileName = `${(letter.letterNumber ?? "Naskah_Resmi").replace(/[\\/]/g, "-")}.pdf`;

  /** Mengekspor naskah yang sedang tampil menjadi PDF (blob), tanpa mengunduhnya ke pengguna. */
  const exportCurrentDocument = useCallback(
    (editor: RichEdit) =>
      new Promise<Blob>((resolve, reject) => {
        const timer = setTimeout(() => {
          pendingExportRef.current = null;
          reject(new Error("Pembuatan PDF resmi melebihi batas waktu."));
        }, EXPORT_TIMEOUT_MS);
        pendingExportRef.current = {
          resolve: (blob) => {
            clearTimeout(timer);
            resolve(blob);
          },
          reject: (err) => {
            clearTimeout(timer);
            reject(err);
          },
        };
        editor.exportToPdf(pdfFileName, {
          modifyPdfDocument: (doc: unknown) => {
            if (!doc && pendingExportRef.current) {
              pendingExportRef.current.reject(
                new Error(
                  "PDF resmi gagal dibuat: file font naskah tidak bisa dimuat."
                )
              );
              pendingExportRef.current = null;
            }
          },
        });
      }),
    [pdfFileName]
  );

  /** Memastikan PDF resmi tersimpan: bila belum, ekspor dari tampilan lalu unggah. */
  const ensureOfficialPdf = useCallback(
    async (editor: RichEdit) => {
      setPdfState("checking");
      const status = await fetchOfficialStatus(letter.id);
      setCopyFor(status.copyFor);
      if (status.pdfReady) {
        setPdfState("ready");
        return;
      }
      setPdfState("preparing");
      await ensurePdfExportLibs();
      const blob = await exportCurrentDocument(editor);
      await uploadOfficialPdf(letter.id, blob, pdfFileName);
      setPdfState("ready");
    },
    [exportCurrentDocument, letter.id, pdfFileName]
  );

  // Alur pemuatan dokumen berdasarkan jenis & status surat
  useEffect(() => {
    // 1. Untuk Surat Masuk Eksternal (RECEIVED) atau Surat Eksternal Keluar yang sudah diunggah scan TTD (SENT):
    //    File PDF sudah ada langsung di MinIO — tidak perlu DevExpress sama sekali.
    if (isPreUploadedScan) {
      return;
    }

    const outer = containerRef.current;
    if (!outer) return;
    let disposed = false;

    const host = document.createElement("div");
    host.style.width = "100%";
    host.style.height = "100%";
    outer.innerHTML = "";
    outer.appendChild(host);

    (async () => {
      try {
        // 2. Untuk Surat Eksternal Keluar berstatus APPROVED:
        //    Cek dulu apakah PDF siap cetak sudah pernah dibuat di server.
        //    Jika sudah ada, langsung tampilkan PDF tanpa memuat DevExpress!
        if (isExternalOutgoing && letter.status === "APPROVED") {
          setDocState("loading");
          setPdfState("checking");
          const st = await fetchOfficialStatus(letter.id);
          if (disposed) return;
          setCopyFor(st.copyFor);
          if (st.pdfReady) {
            setDocState("ready");
            setPdfState("ready");
            return;
          }
        }

        const [mod, filled] = await Promise.all([
          loadRichEdit(),
          fetchFilledLetterDocument(letter.id),
        ]);
        if (disposed || !host.isConnected) return;
        if (LICENSE_KEY && mod.license) mod.license(LICENSE_KEY);

        const options = createWordEditorOptions({
          mod,
          readOnly: true,
          showRibbon: false,
          initialZoom: zoomLevelRef.current,
          height: DEFAULT_HEIGHT,
          onKeyDown: (s, args) => keyDownRef.current(s, args),
        });
        options.events.pdfExporting = (_s, e) => {
          const pending = pendingExportRef.current;
          if (!pending) return;
          e.handled = true;
          pendingExportRef.current = null;
          pending.resolve(e.blob);
        };

        const editor = mod.create(host, options);
        if (disposed) {
          editor.dispose();
          return;
        }
        editorRef.current = editor;

        editor.openDocument(
          filled.blob,
          filled.fileName,
          mod.DocumentFormat.OpenXml,
          (success) => {
            if (disposed) return;
            if (!success) {
              setDocState("error");
              setMessage("Naskah resmi tidak bisa dibuka editor.");
              return;
            }
            setDocState("ready");
            ensureOfficialPdf(editor).catch((err: unknown) => {
              if (disposed) return;
              setPdfState("error");
              setMessage(
                err instanceof Error ? err.message : "PDF resmi gagal disiapkan."
              );
            });
          }
        );
      } catch (err) {
        if (disposed) return;
        setDocState("error");
        setMessage(
          err instanceof Error ? err.message : "Naskah resmi gagal dimuat."
        );
      }
    })();

    return () => {
      disposed = true;
      pendingExportRef.current = null;
      const editor = editorRef.current;
      editorRef.current = null;
      try {
        if (editor && !editor.isDisposed) editor.dispose();
      } catch {
        // Abaikan error internal DevExpress saat elemen sudah terlepas
      }
      outer.innerHTML = "";
    };
  }, [
    letter.id,
    letter.status,
    letter.currentVersion,
    isPreUploadedScan,
    isExternalOutgoing,
    reloadKey,
    ensureOfficialPdf,
  ]);

  // Ukuran editor disesuaikan ulang saat masuk/keluar layar penuh atau lebar panel berubah
  useEffect(() => {
    if (isPdfViewMode) return;
    const outer = containerRef.current;
    const t = setTimeout(() => {
      const editor = editorRef.current;
      if (editor && !editor.isDisposed) {
        try {
          editor.zoomLevel = zoomLevelRef.current;
          editor.adjust();
        } catch {
          // Abaikan
        }
      }
    }, 150);

    if (!outer) return () => clearTimeout(t);

    const ro = new ResizeObserver(() => {
      const editor = editorRef.current;
      if (editor && !editor.isDisposed) {
        try {
          editor.adjust();
        } catch {
          // Abaikan
        }
      }
    });
    ro.observe(outer);

    return () => {
      clearTimeout(t);
      ro.disconnect();
    };
  }, [isFullscreen, docState, isPdfViewMode]);

  function retryPdf() {
    const editor = editorRef.current;
    if (!editor) {
      setReloadKey((k) => k + 1);
      return;
    }
    setMessage(null);
    ensureOfficialPdf(editor).catch((err: unknown) => {
      setPdfState("error");
      setMessage(
        err instanceof Error ? err.message : "PDF resmi gagal disiapkan."
      );
    });
  }

  const pdfReady = effectivePdfState === "ready";
  const bodyHeight = isFullscreen ? "calc(100vh - 80px)" : DEFAULT_HEIGHT;
  const wrapperClass = isFullscreen
    ? "@container fixed inset-0 z-[45] isolate bg-surface flex flex-col p-4 shadow-2xl"
    : "@container relative isolate z-0 flex flex-col border border-border rounded-xl bg-surface overflow-hidden shadow-sm min-w-0 max-w-full";

  const inlinePdfHref = `${officialPdfUrl(letter.id, true)}&v=${letter.currentVersion ?? 1}`;

  return (
    <div ref={wrapperRef} className={wrapperClass}>
      <div className="flex items-center justify-between gap-2 px-3 py-2 bg-panel border-b border-border flex-wrap @4xl:flex-nowrap shrink-0">
        <div className="flex items-center gap-1.5 flex-wrap @4xl:flex-nowrap min-w-0">
          <span className="text-xs font-semibold text-text shrink-0">
            {isPdfViewMode ? "Dokumen PDF Surat" : "Naskah Surat"}
          </span>
          <span className="hidden @xl:inline-block text-[11px] font-medium text-text-muted bg-surface border border-border px-2 py-0.5 rounded-md shadow-2xs shrink-0">
            {isPdfViewMode ? "Tampilan PDF" : "Mode Baca"}
          </span>

          {isExternalIncoming ? (
            <span className="inline-flex items-center gap-1 text-[11px] font-medium text-status-info-text bg-status-info-bg px-2 py-0.5 rounded-md shrink-0">
              <FileText size={11} />
              <span className="hidden @md:inline">Scan Surat Masuk Eksternal</span>
              <span className="@md:hidden">Surat Masuk</span>
            </span>
          ) : isExternalOutgoing && letter.status === "APPROVED" ? (
            <span className="inline-flex items-center gap-1 text-[11px] font-medium text-status-warn-text bg-status-warn-bg px-2 py-0.5 rounded-md shrink-0">
              <Printer size={11} />
              <span className="hidden @md:inline">PDF Siap Cetak (Menunggu TTD Basah)</span>
              <span className="@md:hidden">Menunggu TTD Basah</span>
            </span>
          ) : isExternalOutgoing && letter.status === "SENT" ? (
            <span className="inline-flex items-center gap-1 text-[11px] font-medium text-status-success-text bg-status-success-bg px-2 py-0.5 rounded-md shrink-0">
              <FileCheck2 size={11} />
              <span className="hidden @md:inline">Scan Resmi Bertanda Tangan Basah</span>
              <span className="@md:hidden">Scan TTD Basah</span>
            </span>
          ) : (
            <span className="inline-flex items-center gap-1 text-[11px] font-medium text-status-success-text bg-status-success-bg px-2 py-0.5 rounded-md shrink-0">
              <FileCheck2 size={11} /> Naskah Resmi
            </span>
          )}

          {copyFor && (
            <span
              className="text-[11px] text-text-muted truncate max-w-36 @2xl:max-w-52"
              title={`Salinan untuk ${copyFor}`}
            >
              Salinan untuk {copyFor}
            </span>
          )}
          {(effectivePdfState === "checking" || effectivePdfState === "preparing") && (
            <span className="inline-flex items-center gap-1 text-[11px] text-text-muted shrink-0">
              <Loader2 size={11} className="animate-spin" />
              <span className="hidden @lg:inline">
                {effectivePdfState === "preparing"
                  ? "Menyiapkan PDF siap cetak..."
                  : "Memeriksa PDF resmi..."}
              </span>
            </span>
          )}
          {pdfReady && !isPdfViewMode && (
            <span className="hidden @lg:inline-flex items-center gap-1 text-[11px] text-status-success-text shrink-0">
              <CheckCircle2 size={11} /> PDF siap
            </span>
          )}
        </div>

        <div className="flex items-center gap-1.5 ml-auto shrink-0">
          {!isPdfViewMode && (
            <div className="inline-flex items-center rounded-lg border border-border bg-surface p-0.5 shadow-2xs shrink-0">
              <a
                href={pdfReady ? officialPdfUrl(letter.id) : undefined}
                aria-disabled={!pdfReady}
                title="Unduh PDF Resmi"
                className="flex items-center gap-1 px-2 py-1 rounded-md text-xs font-medium text-text hover:bg-panel transition-colors aria-disabled:opacity-50 aria-disabled:pointer-events-none whitespace-nowrap"
              >
                <Download size={13} className="text-primary shrink-0" />
                <span className="hidden @xl:inline">Unduh PDF Resmi</span>
                <span className="@xl:hidden">Unduh PDF</span>
              </a>
              <span className="h-4 w-px bg-border" />
              <a
                href={pdfReady ? officialPdfUrl(letter.id, true) : undefined}
                target="_blank"
                rel="noopener noreferrer"
                aria-disabled={!pdfReady}
                title="Buka PDF di tab baru"
                className="flex items-center gap-1 px-2 py-1 rounded-md text-xs font-medium text-text hover:bg-panel transition-colors aria-disabled:opacity-50 aria-disabled:pointer-events-none whitespace-nowrap"
              >
                <ExternalLink size={13} className="text-primary shrink-0" />
                <span>Buka PDF</span>
              </a>
            </div>
          )}

          {!isPdfViewMode && (
            <WordZoomControl
              zoomLevel={zoomLevel}
              onZoomChange={applyZoom}
              showPresets={false}
            />
          )}

          <button
            type="button"
            onClick={() => setIsFullscreen((v) => !v)}
            className="p-1.5 rounded-lg border border-border bg-surface hover:bg-panel text-text-muted hover:text-text transition-colors shadow-2xs"
            title={isFullscreen ? "Keluar layar penuh" : "Layar penuh"}
          >
            {isFullscreen ? <Minimize2 size={14} /> : <Maximize2 size={14} />}
          </button>
        </div>
      </div>

      {effectivePdfState === "error" && (
        <div className="flex items-center gap-2 px-3.5 py-2 bg-status-danger-bg text-status-danger-text text-xs border-b border-border">
          <Info size={13} className="shrink-0" />
          <span className="flex-1">{effectiveMessage}</span>
          <button
            type="button"
            onClick={retryPdf}
            className="flex items-center gap-1 px-2 py-0.5 rounded-md border border-current font-medium"
          >
            <RefreshCw size={11} /> Coba lagi
          </button>
        </div>
      )}

      <div
        className="relative w-full bg-white"
        style={{ height: bodyHeight, minHeight: bodyHeight }}
      >
        {/* Jika mode PDF (Surat Masuk Eksternal atau Surat Eksternal Keluar) dan PDF sudah siap, tampilkan iframe PDF */}
        {isPdfViewMode && pdfReady && (
          <iframe
            key={inlinePdfHref}
            src={inlinePdfHref}
            title="Pratinjau Dokumen PDF Surat"
            className="w-full h-full border-0 bg-white relative z-10"
          />
        )}

        {/* Kontainer DevExpress RichEdit:
            - Untuk surat internal (SENT): tampil normal
            - Untuk surat eksternal (APPROVED) yang sedang mengekspor PDF pertama kali: disembunyikan di belakang */}
        {!isPreUploadedScan && (
          <div
            ref={containerRef}
            className={cn(
              "w-full h-full [&_.dxreBlackout]:!opacity-100",
              isPdfViewMode && "opacity-0 pointer-events-none absolute inset-0"
            )}
          />
        )}

        {(effectiveDocState === "loading" ||
          (isPdfViewMode && !pdfReady && effectivePdfState !== "error")) && (
          <div className="absolute inset-0 z-20 flex flex-col items-center justify-center gap-3 text-text-muted bg-panel/30">
            <Loader2 size={24} className="animate-spin text-primary" />
            <p className="text-sm font-semibold text-text">
              {isPdfViewMode
                ? "Menyiapkan dokumen PDF siap cetak..."
                : "Memuat naskah resmi..."}
            </p>
          </div>
        )}
        {effectiveDocState === "error" && (
          <div className="absolute inset-0 z-20 flex flex-col items-center justify-center gap-3 p-8 text-center bg-surface">
            <div className="w-12 h-12 rounded-full bg-status-danger-bg flex items-center justify-center">
              <Info size={20} className="text-status-danger-text" />
            </div>
            <p className="text-sm font-semibold text-text">
              Dokumen gagal dimuat
            </p>
            <p className="text-xs text-text-muted max-w-md">{effectiveMessage}</p>
            <button
              type="button"
              onClick={() => {
                setDocState("loading");
                setMessage(null);
                setReloadKey((k) => k + 1);
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-border bg-surface hover:bg-panel text-xs font-medium text-text transition-colors"
            >
              <RefreshCw size={13} /> Coba lagi
            </button>
          </div>
        )}
      </div>

      {!isPdfViewMode && feedbackPercent !== null && (
        <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-50 pointer-events-none bg-neutral-900/90 text-white text-xs px-3.5 py-1.5 rounded-full font-medium shadow-lg flex items-center gap-1.5 animate-in fade-in duration-150">
          <span>Zoom: {feedbackPercent}%</span>
        </div>
      )}
    </div>
  );
}
