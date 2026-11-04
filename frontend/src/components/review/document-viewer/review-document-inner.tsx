"use client";

// components/review/document-viewer/review-document-inner.tsx
// Naskah surat di halaman detail / review / approval (/surat/[id]).
// - Bila belum disetujui approver & currentVersion > 0: memuat .docx asli dari GET /api/v1/letters/{id}/document
// - Reviewer yang sedang memegang surat (isActiveReviewer): boleh mengedit & menyimpan versi baru (PUT /document)
// - Approver dan pembaca lainnya: mode baca (readOnly: true, tanpa ribbon)
// - Bila belum pernah ada versi diunggah: fallback ke template .docx sesuai jenis surat

import { useEffect, useRef, useState, useCallback } from "react";
import {
  Download,
  Printer,
  Info,
  Maximize2,
  Minimize2,
  Save,
  CheckCircle2,
  History,
} from "lucide-react";

import "devextreme/dist/css/dx.light.css";
import "devexpress-richedit/dist/dx.richedit.css";

import type { RichEdit, DocumentFormat } from "devexpress-richedit";
import type { LetterDetail } from "@/lib/api/letter-detail";
import {
  fetchLetterDocumentBlob,
  uploadLetterDocument,
} from "@/lib/api/letter-document";
import {
  resolveTemplateDocxBlob,
  applyTemplateContextToEditor,
  type LetterTemplateContext,
} from "@/lib/templates/letter-templates";
import {
  WordZoomControl,
  useWordZoomInteractivity,
} from "@/components/ui/word-zoom-control";
import { useSelectOnFocus } from "@/components/surat/letter-editor/use-select-on-focus";
import {
  createWordEditorOptions,
  type RichEditModule,
} from "@/components/surat/letter-editor/word-editor-config";

const LICENSE_KEY = process.env.NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY ?? "";

export interface ReviewDocumentInnerProps {
  letter: LetterDetail;
  onVersionSaved?: (newVersionNumber: number) => void;
}

/** Data surat -> konteks template naskah (fallback bila surat belum punya versi .docx di server). */
function buildContext(letter: LetterDetail): LetterTemplateContext {
  const approverStep = letter.steps.find((s) => s.role === "APPROVER");
  return {
    title: letter.title ?? "",
    sifat: letter.confidentiality?.code,
    kecepatan: letter.urgency?.code,
    classificationCode: letter.classification?.code,
    classificationName: letter.classification?.name,
    recipients: letter.recipients.to.map((r) =>
      r.type === "UNIT"
        ? { name: r.unit.name, position: r.resolvedHead?.positionName }
        : { name: r.alias.label }
    ),
    approver: approverStep
      ? { name: approverStep.user.name, position: approverStep.user.positionName }
      : undefined,
    drafter: {
      name: letter.drafter.name,
      position: letter.drafter.positionName,
    },
    date: letter.letterDate
      ? new Intl.DateTimeFormat("id-ID", {
          day: "numeric",
          month: "long",
          year: "numeric",
        }).format(new Date(letter.letterDate))
      : undefined,
    attachmentsCount: letter.attachments.length,
    letterNumberPlaceholder: letter.letterNumber ?? undefined,
  };
}

export default function ReviewDocumentInner({
  letter,
  onVersionSaved,
}: ReviewDocumentInnerProps) {
  const wrapperRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<RichEdit | null>(null);
  const moduleRef = useRef<RichEditModule | null>(null);
  const docFmtRef = useRef<typeof DocumentFormat | null>(null);

  // Sesuai aturan backend (dokumen.go): Reviewer yang sedang memegang surat (IN_REVIEW / RETURNED_TO_REVIEWER)
  // boleh mengedit naskah .docx; Approver hanya membaca.
  const activeStep = letter.steps.find((s) => s.state === "ACTIVE");
  const isActiveReviewer =
    (letter.status === "IN_REVIEW" ||
      letter.status === "RETURNED_TO_REVIEWER") &&
    activeStep?.role === "REVIEWER" &&
    (letter.allowedActions ?? []).some(
      (a) => a.toUpperCase() === "APPROVE" || a.toUpperCase() === "RETURN"
    );

  const defaultHeight = isActiveReviewer ? "880px" : "760px";

  const [isReady, setIsReady] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isSavingVersion, setIsSavingVersion] = useState(false);
  const [hasEdited, setHasEdited] = useState(false);
  const [toastMsg, setToastMsg] = useState<string | null>(null);
  const [savedVersionOverride, setSavedVersionOverride] = useState<
    number | null
  >(null);
  const currentVersion = Math.max(
    letter.currentVersion ?? 0,
    savedVersionOverride ?? 0
  );

  const isLoadingDocRef = useRef(false);

  const [zoomLevel, setZoomLevel] = useState<number>(
    isActiveReviewer ? 1 : 0.75
  );
  const zoomLevelRef = useRef<number>(zoomLevel);

  useEffect(() => {
    zoomLevelRef.current = zoomLevel;
  }, [zoomLevel]);

  const applyZoom = useCallback((level: number) => {
    const clamped = Math.min(2.0, Math.max(0.5, Math.round(level * 100) / 100));
    setZoomLevel(clamped);
    zoomLevelRef.current = clamped;
    if (editorRef.current) {
      editorRef.current.zoomLevel = clamped;
      editorRef.current.adjust();
    }
  }, []);

  const { feedbackPercent, handleDevExpressKeyDown } = useWordZoomInteractivity({
    wrapperRef,
    containerRef,
    zoomLevel,
    onZoomChange: applyZoom,
    isReady,
  });

  const keyDownRef = useRef(handleDevExpressKeyDown);
  useEffect(() => {
    keyDownRef.current = handleDevExpressKeyDown;
  }, [handleDevExpressKeyDown]);

  const handleSaveReviewerVersion = useCallback(() => {
    const editor = editorRef.current;
    const docFmt = docFmtRef.current;
    if (!editor || !docFmt || !isActiveReviewer || isSavingVersion) return;

    setIsSavingVersion(true);
    editor.exportToFile(async (file: File) => {
      try {
        const res = await uploadLetterDocument(
          letter.id,
          file,
          `Revisi_${letter.id.slice(0, 8)}.docx`
        );
        setSavedVersionOverride(res.version.versionNumber);
        setHasEdited(false);
        setToastMsg(`Disimpan sebagai Versi ${res.version.versionNumber}`);
        setTimeout(() => setToastMsg(null), 3000);
        onVersionSaved?.(res.version.versionNumber);
      } catch (err) {
        alert(
          err instanceof Error ? err.message : "Gagal menyimpan revisi naskah."
        );
      } finally {
        setIsSavingVersion(false);
      }
    }, docFmt.OpenXml);
  }, [isActiveReviewer, isSavingVersion, letter.id, onVersionSaved]);

  useSelectOnFocus(
    containerRef,
    isReady && isActiveReviewer,
    isActiveReviewer ? handleSaveReviewerVersion : undefined
  );

  // Inisialisasi DevExpress RichEdit
  useEffect(() => {
    if (!containerRef.current) return;
    let disposed = false;

    (async () => {
      try {
        const mod = (await import("devexpress-richedit")) as RichEditModule;
        if (disposed || !containerRef.current) return;
        moduleRef.current = mod;
        docFmtRef.current = mod.DocumentFormat;
        if (LICENSE_KEY && mod.license) mod.license(LICENSE_KEY);

        const options = createWordEditorOptions({
          mod,
          readOnly: !isActiveReviewer,
          showRibbon: isActiveReviewer,
          initialZoom: zoomLevelRef.current,
          height: defaultHeight,
          onDocumentChanged: () => {
            if (!isLoadingDocRef.current && isActiveReviewer) {
              setHasEdited(true);
            }
          },
          onKeyDown: (s, args) => keyDownRef.current(s, args),
        });

        editorRef.current = mod.create(containerRef.current, options);
        setIsReady(true);
      } catch (err) {
        console.error("[ReviewDocumentViewer] Inisialisasi gagal:", err);
        setErrorMsg(
          err instanceof Error
            ? err.message
            : "Gagal memuat DevExpress RichEdit."
        );
      }
    })();

    return () => {
      disposed = true;
      editorRef.current?.dispose?.();
      editorRef.current = null;
      setIsReady(false);
    };
  }, [isActiveReviewer, defaultHeight]);

  // Muat naskah: utamakan .docx dari server (bila currentVersion > 0 dan status != SENT);
  // fallback ke generator template .docx.
  useEffect(() => {
    const editor = editorRef.current;
    const mod = moduleRef.current;
    if (!isReady || !editor || !mod) return;

    let cancelled = false;
    isLoadingDocRef.current = true;

    const openBlob = (blob: Blob, name: string, syncContext = false) => {
      if (cancelled || !editorRef.current) return;
      editorRef.current.openDocument(
        blob,
        name,
        mod.DocumentFormat.OpenXml,
        (success) => {
          if (success && editorRef.current) {
            if (syncContext) {
              applyTemplateContextToEditor(
                editorRef.current,
                buildContext(letter)
              );
            }
            setHasEdited(false);
            editorRef.current.zoomLevel = zoomLevelRef.current;
            editorRef.current.adjust();
          }
          setTimeout(() => {
            isLoadingDocRef.current = false;
          }, 500);
        }
      );
    };

    const loadFallbackTemplate = () => {
      resolveTemplateDocxBlob(letter.template?.code ?? undefined)
        .then(({ blob, template: tpl }) => {
          openBlob(blob, `${tpl.name}.docx`, true);
        })
        .catch(() => {
          isLoadingDocRef.current = false;
        });
    };

    if ((letter.currentVersion ?? 0) > 0 && letter.status !== "SENT") {
      fetchLetterDocumentBlob(letter.id)
        .then((res) => {
          if (cancelled) return;
          if (res) openBlob(res.blob, res.fileName);
          else loadFallbackTemplate();
        })
        .catch(() => {
          if (!cancelled) loadFallbackTemplate();
        });
    } else {
      loadFallbackTemplate();
    }

    return () => {
      cancelled = true;
    };
  }, [isReady, letter]);

  // Sesuaikan ukuran saat fullscreen berubah
  useEffect(() => {
    const targetHeight = isFullscreen ? "calc(100vh - 80px)" : defaultHeight;
    if (containerRef.current) {
      containerRef.current.style.height = targetHeight;
      containerRef.current.style.minHeight = targetHeight;
    }
    const t = setTimeout(() => {
      if (editorRef.current) {
        editorRef.current.zoomLevel = zoomLevelRef.current;
        editorRef.current.adjust();
      }
    }, 150);
    return () => clearTimeout(t);
  }, [isFullscreen, defaultHeight]);

  function handleDownload() {
    if (letter.status === "SENT" && currentVersion > 0) {
      window.open(
        `/api/v1/letters/${encodeURIComponent(letter.id)}/document`,
        "_blank"
      );
      return;
    }
    const editor = editorRef.current;
    const mod = moduleRef.current;
    if (!editor || !mod) return;
    editor.exportToFile((file: File) => {
      const url = URL.createObjectURL(file);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${letter.letterNumber ?? "Naskah_Dinas"}.docx`.replace(
        /[\\/]/g,
        "-"
      );
      a.click();
      URL.revokeObjectURL(url);
    }, mod.DocumentFormat.OpenXml);
  }

  function handlePrintPdf() {
    const editor = editorRef.current;
    const mod = moduleRef.current;
    if (!editor) return;
    editor.printDocument(mod?.PrintMode?.Html ?? 1);
  }

  if (errorMsg) {
    return (
      <div className="flex flex-col items-center justify-center p-8 bg-panel/50 border border-border rounded-xl text-center">
        <div className="w-12 h-12 rounded-full bg-status-danger-bg flex items-center justify-center mb-3">
          <Info size={20} className="text-status-danger-text" />
        </div>
        <p className="text-sm font-semibold text-text">
          Pratinjau naskah gagal dimuat
        </p>
        <p className="text-xs text-text-muted mt-1 max-w-md">{errorMsg}</p>
      </div>
    );
  }

  const wrapperClass = isFullscreen
    ? "fixed inset-0 z-[45] isolate bg-surface flex flex-col p-4 shadow-2xl"
    : "relative isolate z-0 flex flex-col border border-border rounded-xl bg-surface overflow-hidden shadow-sm";

  const activeHeight = isFullscreen ? "calc(100vh - 80px)" : defaultHeight;

  return (
    <div ref={wrapperRef} className={wrapperClass}>
      <div className="flex items-center justify-between gap-2 px-3.5 py-2 bg-panel border-b border-border flex-wrap shrink-0">
        {/* Kiri: Label Naskah, Mode, & Versi */}
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs font-semibold text-text">Naskah Surat</span>
          <span className="text-[11px] font-medium text-text-muted bg-surface border border-border px-2 py-0.5 rounded-md shadow-2xs">
            {isActiveReviewer ? "Mode Edit Reviewer" : "Mode Baca"}
          </span>
          {currentVersion > 0 && (
            <span className="inline-flex items-center gap-1 text-[11px] text-primary bg-primary/10 px-2 py-0.5 rounded-md font-medium">
              <History size={11} /> Versi {currentVersion}
            </span>
          )}
          {toastMsg && (
            <span className="inline-flex items-center gap-1 text-xs font-medium text-status-success-text bg-status-success-bg border border-status-success-text/20 px-2.5 py-0.5 rounded-md">
              <CheckCircle2 size={12} /> {toastMsg}
            </span>
          )}
        </div>

        {/* Kanan: Aksi Simpan Reviewer, Segmented Unduh/Cetak, Zoom, Fullscreen */}
        <div className="flex items-center gap-2 ml-auto flex-wrap">
          {isActiveReviewer && (
            <button
              type="button"
              onClick={handleSaveReviewerVersion}
              disabled={!isReady || isSavingVersion}
              className="flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold bg-primary text-white hover:bg-primary-dark transition-colors shadow-2xs disabled:opacity-50"
              title="Simpan koreksi naskah sebagai versi .docx baru di server (Ctrl+S)"
            >
              <Save size={13} />
              {isSavingVersion
                ? "Menyimpan..."
                : hasEdited
                  ? "Simpan Koreksi (.docx)"
                  : "Simpan Versi Baru"}
            </button>
          )}

          <div className="inline-flex items-center rounded-lg border border-border bg-surface p-0.5 shadow-2xs">
            <button
              type="button"
              onClick={handleDownload}
              disabled={!isReady}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium text-text hover:bg-panel transition-colors disabled:opacity-50"
            >
              <Download size={13} className="text-primary" />
              <span>
                {letter.status === "SENT" && currentVersion > 0
                  ? "Unduh PDF Resmi"
                  : "Unduh .docx"}
              </span>
            </button>
            <span className="h-4 w-px bg-border" />
            <button
              type="button"
              onClick={handlePrintPdf}
              disabled={!isReady}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium text-text hover:bg-panel transition-colors disabled:opacity-50"
            >
              <Printer size={13} className="text-primary" />
              <span>Cetak / PDF</span>
            </button>
          </div>

          <WordZoomControl
            zoomLevel={zoomLevel}
            onZoomChange={applyZoom}
            showPresets={false}
          />

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

      <div
        ref={containerRef}
        style={{
          height: activeHeight,
          minHeight: activeHeight,
          width: "100%",
        }}
        className="w-full bg-white relative"
      />

      {feedbackPercent !== null && (
        <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-50 pointer-events-none bg-neutral-900/90 text-white text-xs px-3.5 py-1.5 rounded-full font-medium shadow-lg flex items-center gap-1.5 animate-in fade-in duration-150">
          <span>Zoom: {feedbackPercent}%</span>
        </div>
      )}
    </div>
  );
}
