"use client";

// components/review/document-viewer/official-document-inner.tsx
// Naskah surat yang sudah terbit (SENT), ditampilkan DevExpress RichEdit dalam mode baca:
// - Memuat naskah terisi dari backend (GET /document/filled): nomor, tanggal, penerima,
//   penandatangan, dan QR sudah terisi.
// - Bila PDF resmi belum ada, naskah yang sedang tampil diekspor DevExpress (exportToPdf) lalu
//   disimpan backend (PUT /document/final). PDF dan tampilan memakai mesin tata letak dan file font
//   yang sama, jadi isinya sama persis.
// - Unduh/Cetak memakai PDF resmi yang tersimpan.
// - Setiap penerima langsung melihat salinannya sendiri ("Kepada" = jabatannya); backend yang
//   menentukan salinan, status-nya dibaca dari GET /document/status.

import { useCallback, useEffect, useRef, useState } from "react";
import {
  CheckCircle2,
  Download,
  FileCheck2,
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
import { WordZoomControl } from "@/components/ui/word-zoom-control";
import {
  createWordEditorOptions,
  loadRichEdit,
} from "@/components/surat/letter-editor/word-editor-config";

const LICENSE_KEY = process.env.NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY ?? "";
const DEFAULT_HEIGHT = "880px";
const EXPORT_TIMEOUT_MS = 60_000;

export interface OfficialDocumentInnerProps {
  letter: LetterDetail;
}

// Keadaan naskah di layar dan PDF resmi
type DocState = "loading" | "ready" | "error";
type PdfState = "checking" | "preparing" | "ready" | "error";

interface PendingExport {
  resolve: (blob: Blob) => void;
  reject: (err: Error) => void;
}

export default function OfficialDocumentInner({ letter }: OfficialDocumentInnerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<RichEdit | null>(null);
  const pendingExportRef = useRef<PendingExport | null>(null);

  const [docState, setDocState] = useState<DocState>("loading");
  const [pdfState, setPdfState] = useState<PdfState>("checking");
  const [message, setMessage] = useState<string | null>(null);
  const [zoomLevel, setZoomLevel] = useState(1);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);
  // Jabatan pemilik salinan yang dilihat (penerima langsung); null = salinan induk
  const [copyFor, setCopyFor] = useState<string | null>(null);

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
          // DevExpress memanggil dengan null bila ekspor batal (misalnya file font tidak termuat)
          modifyPdfDocument: (doc: unknown) => {
            if (!doc && pendingExportRef.current) {
              pendingExportRef.current.reject(
                new Error("PDF resmi gagal dibuat: file font naskah tidak bisa dimuat.")
              );
              pendingExportRef.current = null;
            }
          },
        });
      }),
    [pdfFileName]
  );

  /** Memastikan PDF resmi salinan yang dilihat user tersimpan: bila belum, ekspor dari tampilan lalu unggah. */
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

  // Buat editor (mode baca), muat naskah terisi, lalu pastikan PDF resmi ada
  useEffect(() => {
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
        const [mod, filled] = await Promise.all([
          loadRichEdit(), // modul + file font resmi
          fetchFilledLetterDocument(letter.id),
        ]);
        if (disposed || !host.isConnected) return;
        if (LICENSE_KEY && mod.license) mod.license(LICENSE_KEY);

        const options = createWordEditorOptions({
          mod,
          readOnly: true,
          showRibbon: false,
          initialZoom: 1,
          height: DEFAULT_HEIGHT,
        });
        // Tangkap hasil ekspor (jangan diunduh otomatis oleh browser)
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

        editor.openDocument(filled.blob, filled.fileName, mod.DocumentFormat.OpenXml, (success) => {
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
            setMessage(err instanceof Error ? err.message : "PDF resmi gagal disiapkan.");
          });
        });
      } catch (err) {
        if (disposed) return;
        setDocState("error");
        setMessage(err instanceof Error ? err.message : "Naskah resmi gagal dimuat.");
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
  }, [letter.id, reloadKey, ensureOfficialPdf]);

  // Ukuran editor disesuaikan ulang saat masuk/keluar layar penuh
  useEffect(() => {
    const t = setTimeout(() => {
      const editor = editorRef.current;
      if (editor && !editor.isDisposed) editor.adjust();
    }, 150);
    return () => clearTimeout(t);
  }, [isFullscreen]);

  const applyZoom = useCallback((level: number) => {
    const clamped = Math.min(2, Math.max(0.5, Math.round(level * 100) / 100));
    setZoomLevel(clamped);
    const editor = editorRef.current;
    if (editor && !editor.isDisposed) {
      editor.zoomLevel = clamped;
      editor.adjust();
    }
  }, []);

  function retryPdf() {
    const editor = editorRef.current;
    if (!editor) return;
    setMessage(null);
    ensureOfficialPdf(editor).catch((err: unknown) => {
      setPdfState("error");
      setMessage(err instanceof Error ? err.message : "PDF resmi gagal disiapkan.");
    });
  }

  const pdfReady = pdfState === "ready";
  const bodyHeight = isFullscreen ? "calc(100vh - 80px)" : DEFAULT_HEIGHT;
  const wrapperClass = isFullscreen
    ? "fixed inset-0 z-[45] isolate bg-surface flex flex-col p-4 shadow-2xl"
    : "relative isolate z-0 flex flex-col border border-border rounded-xl bg-surface overflow-hidden shadow-sm";

  return (
    <div className={wrapperClass}>
      <div className="flex items-center justify-between gap-2 px-3.5 py-2 bg-panel border-b border-border flex-wrap shrink-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs font-semibold text-text">Naskah Surat</span>
          <span className="text-[11px] font-medium text-text-muted bg-surface border border-border px-2 py-0.5 rounded-md shadow-2xs">
            Mode Baca
          </span>
          <span className="inline-flex items-center gap-1 text-[11px] font-medium text-status-success-text bg-status-success-bg px-2 py-0.5 rounded-md">
            <FileCheck2 size={11} /> Naskah Resmi
          </span>
          {copyFor && (
            <span className="text-[11px] text-text-muted" title="Salinan surat yang ditujukan kepada jabatan ini">
              Salinan untuk {copyFor}
            </span>
          )}
          {(pdfState === "checking" || pdfState === "preparing") && docState === "ready" && (
            <span className="inline-flex items-center gap-1 text-[11px] text-text-muted">
              <Loader2 size={11} className="animate-spin" />
              {pdfState === "preparing" ? "Menyiapkan PDF resmi..." : "Memeriksa PDF resmi..."}
            </span>
          )}
          {pdfReady && (
            <span className="inline-flex items-center gap-1 text-[11px] text-status-success-text">
              <CheckCircle2 size={11} /> PDF resmi siap
            </span>
          )}
        </div>

        <div className="flex items-center gap-2 ml-auto flex-wrap">
          <div className="inline-flex items-center rounded-lg border border-border bg-surface p-0.5 shadow-2xs">
            <a
              href={pdfReady ? officialPdfUrl(letter.id) : undefined}
              aria-disabled={!pdfReady}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium text-text hover:bg-panel transition-colors aria-disabled:opacity-50 aria-disabled:pointer-events-none"
            >
              <Download size={13} className="text-primary" />
              <span>Unduh PDF Resmi</span>
            </a>
            <span className="h-4 w-px bg-border" />
            <a
              href={pdfReady ? officialPdfUrl(letter.id, true) : undefined}
              target="_blank"
              rel="noopener noreferrer"
              aria-disabled={!pdfReady}
              title="Buka PDF resmi di tab baru untuk dicetak"
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium text-text hover:bg-panel transition-colors aria-disabled:opacity-50 aria-disabled:pointer-events-none"
            >
              <Printer size={13} className="text-primary" />
              <span>Cetak / PDF</span>
            </a>
          </div>

          <WordZoomControl zoomLevel={zoomLevel} onZoomChange={applyZoom} showPresets={false} />

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

      {pdfState === "error" && docState === "ready" && (
        <div className="flex items-center gap-2 px-3.5 py-2 bg-status-danger-bg text-status-danger-text text-xs border-b border-border">
          <Info size={13} className="shrink-0" />
          <span className="flex-1">{message}</span>
          <button
            type="button"
            onClick={retryPdf}
            className="flex items-center gap-1 px-2 py-0.5 rounded-md border border-current font-medium"
          >
            <RefreshCw size={11} /> Coba lagi
          </button>
        </div>
      )}

      <div className="relative w-full bg-white" style={{ height: bodyHeight, minHeight: bodyHeight }}>
        {/* Kop/footer tidak diredupkan (dxreBlackout): mode baca harus sama dengan PDF resmi */}
        <div ref={containerRef} className="w-full h-full [&_.dxreBlackout]:!opacity-100" />
        {docState === "loading" && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 text-text-muted bg-panel/30">
            <Loader2 size={24} className="animate-spin text-primary" />
            <p className="text-sm font-semibold text-text">Memuat naskah resmi...</p>
          </div>
        )}
        {docState === "error" && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 p-8 text-center bg-surface">
            <div className="w-12 h-12 rounded-full bg-status-danger-bg flex items-center justify-center">
              <Info size={20} className="text-status-danger-text" />
            </div>
            <p className="text-sm font-semibold text-text">Naskah resmi gagal dimuat</p>
            <p className="text-xs text-text-muted max-w-md">{message}</p>
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
    </div>
  );
}
