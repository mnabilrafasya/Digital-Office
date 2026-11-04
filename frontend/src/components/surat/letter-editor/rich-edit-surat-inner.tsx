"use client";

// components/surat/letter-editor/rich-edit-surat-inner.tsx
// Editor naskah dinas berbasis DevExpress RichEdit (Section 3 Buat Surat).
// Terhubung dengan:
// - Template .docx dari backend & MinIO (`GET /api/v1/templates` & `/api/v1/templates/{id}/file`)
// - Server dokumen MinIO: GET & PUT /api/v1/letters/{id}/document (.docx per versi)

import { useRef, useEffect, useCallback, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { Info } from "lucide-react";

import "devextreme/dist/css/dx.light.css";
import "devexpress-richedit/dist/dx.richedit.css";

import type { RichEdit, DocumentFormat } from "devexpress-richedit";
import type { BuatSuratFormValues } from "@/lib/schemas";
import {
  resolveTemplateDocxBlob,
  applyTemplateContextToEditor,
  getTemplateById,
  type LetterTemplateContext,
} from "@/lib/templates/letter-templates";
import { fetchLetterDocumentBlob } from "@/lib/api/letter-document";
import { useWordZoomInteractivity } from "@/components/ui/word-zoom-control";

import { useSelectOnFocus } from "./use-select-on-focus";
import {
  createWordEditorOptions,
  type RichEditModule,
} from "./word-editor-config";
import { LetterEditorToolbar } from "./letter-editor-toolbar";

const LICENSE_KEY = process.env.NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY ?? "";
const DEFAULT_EDITOR_HEIGHT = "980px";
const FULLSCREEN_EDITOR_HEIGHT = "calc(100vh - 110px)";

export interface LetterEditorHandle {
  /** Mengekspor naskah yang sedang dibuka di editor menjadi File .docx */
  exportDocxFile: () => Promise<{ file: File; fileName: string } | null>;
}

export interface RichEditSuratInnerProps {
  letterId?: string | null;
  serverVersionNumber?: number | null;
  editorHandleRef?: React.MutableRefObject<LetterEditorHandle | null>;
  onDocumentChange?: () => void;
  onQuickSaveDraft?: () => void;
}

export default function RichEditSuratInner({
  letterId,
  serverVersionNumber,
  editorHandleRef,
  onDocumentChange,
  onQuickSaveDraft,
}: RichEditSuratInnerProps) {
  const wrapperRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const instanceRef = useRef<RichEdit | null>(null);
  const moduleRef = useRef<RichEditModule | null>(null);
  const docFmtRef = useRef<typeof DocumentFormat | null>(null);

  const isLoadingDocRef = useRef(false);
  const userEditedRef = useRef(false);
  const loadedServerDocKeyRef = useRef<string | null>(null);

  const [isReady, setIsReady] = useState(false);
  const [fileName, setFileName] = useState<string>("Naskah_Dinas_Draf.docx");
  const [isSaving, setIsSaving] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [appliedTemplateId, setAppliedTemplateId] = useState<string>(
    "tmpl-dinas-standar"
  );
  const [statusNotification, setStatusNotification] = useState<string | null>(
    null
  );
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);

  const [zoomLevel, setZoomLevel] = useState<number>(1);
  const zoomLevelRef = useRef<number>(1);

  useEffect(() => {
    zoomLevelRef.current = zoomLevel;
  }, [zoomLevel]);

  const showToast = useCallback((msg: string) => {
    setStatusNotification(msg);
    setTimeout(() => setStatusNotification(null), 3000);
  }, []);

  const applyZoom = useCallback((level: number) => {
    const clamped = Math.min(2.0, Math.max(0.5, Math.round(level * 100) / 100));
    setZoomLevel(clamped);
    zoomLevelRef.current = clamped;
    if (instanceRef.current) {
      instanceRef.current.zoomLevel = clamped;
      instanceRef.current.adjust();
    }
  }, []);

  useSelectOnFocus(containerRef, isReady, onQuickSaveDraft);

  const { feedbackPercent, handleDevExpressKeyDown } = useWordZoomInteractivity({
    wrapperRef,
    containerRef,
    zoomLevel,
    onZoomChange: applyZoom,
    isReady,
  });

  // Ekspos fungsi ekspor .docx ke induk halaman Buat Surat (untuk diunggah ke MinIO saat Simpan Draft / Submit)
  const fileNameRef = useRef(fileName);
  useEffect(() => {
    fileNameRef.current = fileName;
  }, [fileName]);

  useEffect(() => {
    if (!editorHandleRef) return;
    editorHandleRef.current = {
      exportDocxFile: () =>
        new Promise((resolve) => {
          const editor = instanceRef.current;
          const docFmt = docFmtRef.current;
          if (!editor || !docFmt) {
            resolve(null);
            return;
          }
          try {
            editor.exportToFile((file: File) => {
              setHasUnsavedChanges(false);
              resolve({
                file,
                fileName: fileNameRef.current || "Naskah_Dinas.docx",
              });
            }, docFmt.OpenXml);
          } catch {
            resolve(null);
          }
        }),
    };
    return () => {
      editorHandleRef.current = null;
    };
  }, [editorHandleRef, isReady]);

  // Ambil state form dari Buat Surat
  const { control, setValue } = useFormContext<BuatSuratFormValues>();
  const formTitle = useWatch({ control, name: "title" });
  const formSifat = useWatch({ control, name: "sifat" });
  const formKecepatan = useWatch({ control, name: "kecepatan" });
  const formTemplateId = useWatch({ control, name: "templateId" });
  const formToUnits = useWatch({ control, name: "toUnits" });
  const formApprovers = useWatch({ control, name: "approvers" });

  const getTemplateContext = useCallback((): LetterTemplateContext => {
    return {
      title: formTitle,
      sifat: formSifat,
      kecepatan: formKecepatan,
      recipients: (formToUnits ?? []).map((u) => ({ name: u.name })),
      approver: formApprovers?.[0]
        ? {
            name: formApprovers[0].employeeName,
            position: formApprovers[0].jabatan,
          }
        : undefined,
    };
  }, [formTitle, formSifat, formKecepatan, formToUnits, formApprovers]);

  const getTemplateContextRef = useRef(getTemplateContext);
  useEffect(() => {
    getTemplateContextRef.current = getTemplateContext;
  }, [getTemplateContext]);

  const keyDownHandlerRef = useRef(handleDevExpressKeyDown);
  useEffect(() => {
    keyDownHandlerRef.current = handleDevExpressKeyDown;
  }, [handleDevExpressKeyDown]);

  const confirmOverwrite = useCallback(() => {
    if (!userEditedRef.current) return true;
    return window.confirm(
      "Isi naskah yang sudah Anda ketik akan diganti dengan template. Lanjutkan?"
    );
  }, []);

  // Memuat template .docx ke dalam DevExpress RichEdit
  const loadTemplateIntoEditor = useCallback(
    (templateId: string, notify = true, syncFormFields = false) => {
      const editor = instanceRef.current;
      const docFmt = docFmtRef.current;
      if (!editor || !docFmt) return;

      isLoadingDocRef.current = true;
      userEditedRef.current = false;
      const releaseLoading = () =>
        setTimeout(() => {
          isLoadingDocRef.current = false;
        }, 500);

      resolveTemplateDocxBlob(templateId)
        .then(({ blob, template: tpl }) => {
          editor.openDocument(
            blob,
            `${tpl.name}.docx`,
            docFmt.OpenXml,
            (success) => {
              if (success) {
                if (syncFormFields) {
                  applyTemplateContextToEditor(
                    editor,
                    getTemplateContextRef.current()
                  );
                }
                releaseLoading();
                setAppliedTemplateId(tpl.id);
                setValue("templateId", tpl.id);
                setFileName(`${tpl.name}.docx`);
                setHasUnsavedChanges(true);
                if (notify) {
                  showToast(
                    syncFormFields
                      ? `Data form disinkronkan ke "${tpl.name}"`
                      : `Template "${tpl.name}" berhasil dimuat`
                  );
                }
                editor.zoomLevel = zoomLevelRef.current;
                editor.adjust();
                editor.focus();
                onDocumentChange?.();
              } else {
                releaseLoading();
              }
            }
          );
        })
        .catch((err) => {
          isLoadingDocRef.current = false;
          console.error("[LetterEditor] Gagal memuat template .docx:", err);
        });
    },
    [onDocumentChange, setValue, showToast]
  );

  // Inisialisasi DevExpress RichEdit sekali saat mount
  useEffect(() => {
    if (!containerRef.current) return;
    let disposed = false;

    (async () => {
      try {
        const mod = (await import("devexpress-richedit")) as RichEditModule;
        if (disposed || !containerRef.current) return;
        moduleRef.current = mod;
        docFmtRef.current = mod.DocumentFormat;

        if (LICENSE_KEY && mod.license) {
          mod.license(LICENSE_KEY);
        }

        const options = createWordEditorOptions({
          mod,
          readOnly: false,
          showRibbon: true,
          initialZoom: 1,
          height: DEFAULT_EDITOR_HEIGHT,
          onDocumentChanged: () => {
            if (!isLoadingDocRef.current) {
              userEditedRef.current = true;
              setHasUnsavedChanges(true);
            }
          },
          onKeyDown: (s, args) => keyDownHandlerRef.current(s, args),
        });

        const editor = mod.create(containerRef.current, options);
        instanceRef.current = editor;
        setIsReady(true);
      } catch (err) {
        console.error("[LetterEditor] Inisialisasi gagal:", err);
        setErrorMsg(
          err instanceof Error
            ? err.message
            : "Gagal memuat DevExpress RichEdit."
        );
      }
    })();

    return () => {
      disposed = true;
      instanceRef.current?.dispose?.();
      instanceRef.current = null;
      setIsReady(false);
    };
  }, []);

  // Muat naskah awal: jika draf sudah punya versi .docx di server, ambil dari GET /letters/{id}/document;
  // jika belum, muat template .docx dari /api/v1/templates/{id}/file (fallback public/templates/).
  useEffect(() => {
    const editor = instanceRef.current;
    const docFmt = docFmtRef.current;
    if (!isReady || !editor || !docFmt) return;

    const serverKey =
      letterId && serverVersionNumber && serverVersionNumber > 0
        ? `${letterId}:v${serverVersionNumber}`
        : null;

    if (serverKey && loadedServerDocKeyRef.current !== serverKey) {
      loadedServerDocKeyRef.current = serverKey;
      isLoadingDocRef.current = true;
      fetchLetterDocumentBlob(letterId!)
        .then((res) => {
          if (!res || !instanceRef.current) {
            loadTemplateIntoEditor(
              formTemplateId || "tmpl-dinas-standar",
              false,
              false
            );
            return;
          }
          instanceRef.current.openDocument(
            res.blob,
            res.fileName,
            docFmt.OpenXml,
            (success) => {
              setTimeout(() => {
                isLoadingDocRef.current = false;
              }, 500);
              if (success && instanceRef.current) {
                setFileName(res.fileName);
                setHasUnsavedChanges(false);
                userEditedRef.current = true;
                instanceRef.current.zoomLevel = zoomLevelRef.current;
                instanceRef.current.adjust();
              }
            }
          );
        })
        .catch(() => {
          isLoadingDocRef.current = false;
          loadTemplateIntoEditor(
            formTemplateId || "tmpl-dinas-standar",
            false,
            false
          );
        });
      return;
    }

    if (!loadedServerDocKeyRef.current) {
      loadedServerDocKeyRef.current = "initial-template";
      loadTemplateIntoEditor(
        formTemplateId || "tmpl-dinas-standar",
        false,
        false
      );
    }
  }, [
    isReady,
    letterId,
    serverVersionNumber,
    formTemplateId,
    loadTemplateIntoEditor,
  ]);

  // Jika user memilih template di Section 1
  useEffect(() => {
    if (!isReady || !formTemplateId || formTemplateId === appliedTemplateId) {
      return;
    }
    const resolvedFormTpl = getTemplateById(formTemplateId);
    const resolvedAppliedTpl = getTemplateById(appliedTemplateId);
    if (resolvedFormTpl.id === resolvedAppliedTpl.id) {
      return;
    }
    if (!confirmOverwrite()) {
      setValue("templateId", appliedTemplateId);
      return;
    }
    loadTemplateIntoEditor(formTemplateId, false, false);
  }, [
    formTemplateId,
    isReady,
    appliedTemplateId,
    loadTemplateIntoEditor,
    confirmOverwrite,
    setValue,
  ]);

  // Adjust editor & perbarui tinggi eksplisit saat toggle fullscreen
  useEffect(() => {
    const targetHeight = isFullscreen
      ? FULLSCREEN_EDITOR_HEIGHT
      : DEFAULT_EDITOR_HEIGHT;
    if (containerRef.current) {
      containerRef.current.style.height = targetHeight;
      containerRef.current.style.minHeight = targetHeight;
    }
    if (!instanceRef.current) return;
    const t = setTimeout(() => {
      if (instanceRef.current) {
        instanceRef.current.zoomLevel = zoomLevelRef.current;
        instanceRef.current.adjust();
      }
    }, 150);
    return () => clearTimeout(t);
  }, [isFullscreen]);

  const handleOpenDocx = useCallback(() => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".docx,.rtf,.txt";
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      const editor = instanceRef.current;
      const docFmt = docFmtRef.current;
      if (!file || !editor || !docFmt) return;

      const buffer = await file.arrayBuffer();
      const ext = file.name.split(".").pop()?.toLowerCase();
      const fmt =
        ext === "rtf"
          ? docFmt.Rtf
          : ext === "txt"
            ? docFmt.PlainText
            : docFmt.OpenXml;

      isLoadingDocRef.current = true;
      editor.openDocument(buffer, file.name, fmt, (success) => {
        setTimeout(() => {
          isLoadingDocRef.current = false;
        }, 500);
        if (success) {
          userEditedRef.current = true;
          setHasUnsavedChanges(true);
          setFileName(file.name.replace(/\.(rtf|txt)$/i, ".docx"));
          showToast(`File "${file.name}" berhasil dibuka`);
          editor.zoomLevel = zoomLevelRef.current;
          editor.adjust();
          editor.focus();
          onDocumentChange?.();
        }
      });
    };
    input.click();
  }, [onDocumentChange, showToast]);

  const handleDownloadDocx = useCallback(() => {
    const editor = instanceRef.current;
    const docFmt = docFmtRef.current;
    if (!editor || !docFmt) return;
    setIsSaving(true);
    editor.exportToFile((file: File) => {
      const url = URL.createObjectURL(file);
      const a = document.createElement("a");
      a.href = url;
      a.download = fileName || "Naskah_Dinas.docx";
      a.click();
      URL.revokeObjectURL(url);
      setIsSaving(false);
      showToast(`Dokumen diunduh sebagai ${a.download}`);
    }, docFmt.OpenXml);
  }, [fileName, showToast]);

  const handlePrintPdf = useCallback(() => {
    const editor = instanceRef.current;
    const mod = moduleRef.current;
    if (!editor) return;
    editor.printDocument(mod?.PrintMode?.Html ?? 1);
  }, []);

  if (errorMsg) {
    return (
      <div className="flex flex-col items-center justify-center p-8 bg-panel/50 border border-border rounded-xl text-center">
        <div className="w-12 h-12 rounded-full bg-status-danger-bg flex items-center justify-center mb-3">
          <Info size={20} className="text-status-danger-text" />
        </div>
        <p className="text-sm font-semibold text-text">
          Editor Naskah Gagal Dimuat
        </p>
        <p className="text-xs text-text-muted mt-1 max-w-md">{errorMsg}</p>
      </div>
    );
  }

  const containerClasses = isFullscreen
    ? "fixed inset-0 z-[45] isolate bg-surface flex flex-col p-4 shadow-2xl"
    : "relative isolate z-0 flex flex-col border border-border rounded-xl bg-surface overflow-hidden shadow-sm";

  const activeHeight = isFullscreen
    ? FULLSCREEN_EDITOR_HEIGHT
    : DEFAULT_EDITOR_HEIGHT;

  return (
    <div ref={wrapperRef} className={containerClasses}>
      <LetterEditorToolbar
        appliedTemplateId={appliedTemplateId}
        onSelectTemplate={(newId) => {
          if (!confirmOverwrite()) return;
          loadTemplateIntoEditor(newId, true, false);
          setValue("templateId", newId, { shouldDirty: true });
        }}
        onSyncFormData={() => {
          if (!confirmOverwrite()) return;
          loadTemplateIntoEditor(appliedTemplateId, true, true);
        }}
        zoomLevel={zoomLevel}
        onZoomChange={applyZoom}
        statusNotification={statusNotification}
        isReady={isReady}
        isSaving={isSaving}
        isFullscreen={isFullscreen}
        onToggleFullscreen={() => setIsFullscreen((v) => !v)}
        onOpenDocx={handleOpenDocx}
        onDownloadDocx={handleDownloadDocx}
        onPrintPdf={handlePrintPdf}
        serverVersionNumber={serverVersionNumber}
        hasUnsavedDocumentChanges={hasUnsavedChanges}
      />

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
