"use client";

import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  X,
  Save,
  Upload,
  Download,
  FileText,
  Maximize2,
  Minimize2,
  Hash,
  Copy,
  Check,
  Loader2,
  AlertCircle,
} from "lucide-react";

import "devextreme/dist/css/dx.light.css";
import "devexpress-richedit/dist/dx.richedit.css";

import type { RichEdit, DocumentFormat } from "devexpress-richedit";
import {
  createWordEditorOptions,
  loadRichEdit,
  type RichEditModule,
} from "@/components/surat/letter-editor/word-editor-config";
import {
  LETTER_TEMPLATE_LIST,
  SYSTEM_PLACEHOLDERS,
  createTemplate,
  updateTemplate,
  invalidateTemplateCache,
} from "@/lib/templates/letter-templates";
import { WordZoomControl, useWordZoomInteractivity } from "@/components/ui/word-zoom-control";
import type { MasterRow } from "@/lib/api/master-data";
import { loadChoices } from "./config";
import {
  buttonClass,
  primaryButtonClass,
  softPrimaryButtonClass,
  inputClass,
} from "./master-form";

const LICENSE_KEY = process.env.NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY ?? "";
const DEFAULT_EDITOR_HEIGHT = "clamp(420px, 52vh, 640px)";
const FULLSCREEN_EDITOR_HEIGHT = "calc(100vh - 180px)";

export function TemplateDocxEditorModal({
  mode,
  row,
  onClose,
  onSaved,
}: {
  mode: "create" | "edit";
  row?: MasterRow;
  onClose: () => void;
  onSaved: (message: string) => Promise<void> | void;
}) {
  const wrapperRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const instanceRef = useRef<RichEdit | null>(null);
  const moduleRef = useRef<RichEditModule | null>(null);
  const docFmtRef = useRef<typeof DocumentFormat | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [name, setName] = useState(() => String(row?.name ?? ""));
  const [letterTypeId, setLetterTypeId] = useState(() =>
    String((row?.letterType as MasterRow | null)?.id ?? ""),
  );
  const [description, setDescription] = useState(() =>
    String(row?.description ?? ""),
  );

  const [isReady, setIsReady] = useState(false);
  const [isLoadingDoc, setIsLoadingDoc] = useState(true);
  const [sourceLabel, setSourceLabel] = useState<string>(
    mode === "edit"
      ? `Berkas server: ${String(row?.name ?? "Template")}.docx`
      : "Dokumen Kosong A4 (.docx)",
  );
  const [uploadedFallbackFile, setUploadedFallbackFile] = useState<File | null>(
    null,
  );
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [zoomLevel, setZoomLevel] = useState(1);
  const zoomLevelRef = useRef(1);

  const [copiedCode, setCopiedCode] = useState<string | null>(null);
  const [toast, setToast] = useState<string | null>(null);
  const [error, setError] = useState<string>("");
  const [busy, setBusy] = useState(false);

  const letterTypesQuery = useQuery({
    queryKey: ["master", "choices", "letterTypes", undefined],
    queryFn: () => loadChoices("letterTypes"),
  });

  const showToast = useCallback((msg: string) => {
    setToast(msg);
    setTimeout(() => {
      setToast((prev) => (prev === msg ? null : prev));
    }, 3000);
  }, []);

  const applyZoom = useCallback((level: number) => {
    const clamped = Math.min(2.0, Math.max(0.5, Math.round(level * 100) / 100));
    setZoomLevel(clamped);
    zoomLevelRef.current = clamped;
    const editor = instanceRef.current;
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
    isReady,
  });

  const keyDownHandlerRef = useRef(handleDevExpressKeyDown);
  useEffect(() => {
    keyDownHandlerRef.current = handleDevExpressKeyDown;
  }, [handleDevExpressKeyDown]);

  // Muat Blob atau ArrayBuffer ke dalam DevExpress RichEdit
  const openBlobInEditor = useCallback(
    (data: Blob | ArrayBuffer, docName: string, label: string) => {
      const editor = instanceRef.current;
      const docFmt = docFmtRef.current;
      if (!editor || editor.isDisposed || !docFmt) return;

      editor.openDocument(data, docName, docFmt.OpenXml, (success) => {
        setIsLoadingDoc(false);
        const activeEditor = instanceRef.current;
        if (success && activeEditor && !activeEditor.isDisposed) {
          setSourceLabel(label);
          try {
            activeEditor.zoomLevel = zoomLevelRef.current;
            activeEditor.adjust();
          } catch {
            // Abaikan
          }
        } else if (!success) {
          setError("Gagal membuka dokumen .docx di editor.");
        }
      });
    },
    [],
  );

  // Muat template preset / kosong dari URL
  const loadFromUrl = useCallback(
    (url: string, docName: string, label: string, fallbackUrl?: string) => {
      fetch(url, {
        cache: "no-store",
        credentials: "include",
      })
        .then((res) => {
          if (!res.ok && fallbackUrl) {
            return fetch(fallbackUrl, { cache: "no-store" });
          }
          return res;
        })
        .then((res) => {
          if (!res.ok) {
            throw new Error(`Gagal memuat dokumen (${res.status})`);
          }
          return res.blob();
        })
        .then((blob) => {
          openBlobInEditor(blob, docName, label);
        })
        .catch((err) => {
          setIsLoadingDoc(false);
          setError(
            err instanceof Error
              ? err.message
              : "Gagal memuat dokumen awal ke editor.",
          );
        });
    },
    [openBlobInEditor],
  );

  // Inisialisasi DevExpress RichEdit
  useEffect(() => {
    const outerContainer = containerRef.current;
    if (!outerContainer) return;
    let disposed = false;

    const hostEl = document.createElement("div");
    hostEl.style.width = "100%";
    hostEl.style.height = "100%";
    outerContainer.innerHTML = "";
    outerContainer.appendChild(hostEl);

    (async () => {
      try {
        const mod = await loadRichEdit();
        if (disposed || !hostEl.isConnected) return;
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
          onKeyDown: (s, args) => keyDownHandlerRef.current(s, args),
        });

        const editor = mod.create(hostEl, options);
        if (disposed) {
          try {
            if (!editor.isDisposed) editor.dispose();
          } catch {
            // Abaikan
          }
          return;
        }
        instanceRef.current = editor;
        setIsReady(true);
      } catch (err) {
        if (!disposed) {
          setError(
            err instanceof Error
              ? err.message
              : "Gagal memuat DevExpress RichEdit.",
          );
          setIsLoadingDoc(false);
        }
      }
    })();

    return () => {
      disposed = true;
      const inst = instanceRef.current;
      instanceRef.current = null;
      if (inst) {
        try {
          if (!inst.isDisposed) inst.dispose();
        } catch {
          // Abaikan
        }
      }
      if (outerContainer) {
        outerContainer.innerHTML = "";
      }
    };
  }, []);

  // Setelah editor siap, muat berkas template (saat edit) atau dokumen kosong (saat create)
  useEffect(() => {
    if (!isReady) return;
    if (mode === "edit" && row?.id) {
      const id = encodeURIComponent(String(row.id));
      const tplName = String(row.name ?? "Template");
      const rowTypeCode = String(
        (row.letterType as MasterRow | null)?.code ?? "",
      ).toLowerCase();
      const matchedStarter =
        LETTER_TEMPLATE_LIST.find(
          (item) => item.name.trim().toLowerCase() === tplName.trim().toLowerCase(),
        ) ??
        (rowTypeCode
          ? LETTER_TEMPLATE_LIST.find((item) =>
              item.letterTypeKeywords.some((kw) => rowTypeCode.includes(kw)),
            )
          : undefined);
      loadFromUrl(
        `/api/v1/templates/${id}/file`,
        `${tplName.replace(/\s+/g, "_")}.docx`,
        `Berkas server: ${tplName}.docx`,
        matchedStarter?.fallbackFileUrl ?? "/templates/dokumen-kosong.docx",
      );
    } else {
      loadFromUrl(
        "/templates/dokumen-kosong.docx",
        "Template_Baru.docx",
        "Dokumen Kosong A4 (.docx)",
      );
    }
  }, [isReady, mode, row?.id, row?.name, row?.letterType, loadFromUrl]);

  // Sesuaikan ukuran saat toggle fullscreen
  useEffect(() => {
    const targetHeight = isFullscreen
      ? FULLSCREEN_EDITOR_HEIGHT
      : DEFAULT_EDITOR_HEIGHT;
    if (containerRef.current) {
      containerRef.current.style.height = targetHeight;
      containerRef.current.style.minHeight = targetHeight;
    }
    const timer = setTimeout(() => {
      const editor = instanceRef.current;
      if (editor && !editor.isDisposed) {
        try {
          editor.zoomLevel = zoomLevelRef.current;
          editor.adjust();
        } catch {
          // Abaikan
        }
      }
    }, 150);
    return () => clearTimeout(timer);
  }, [isFullscreen]);

  // Unggah / buka file .docx dari komputer ke dalam DevExpress RichEdit
  async function handleFileUpload(file: File | undefined) {
    if (!file) return;
    if (!file.name.toLowerCase().endsWith(".docx")) {
      setError("Pilih berkas Microsoft Word dengan ekstensi .docx.");
      return;
    }
    setError("");
    setUploadedFallbackFile(file);
    if (!name.trim()) {
      setName(file.name.replace(/\.docx$/i, "").replace(/[_-]+/g, " "));
    }

    const editor = instanceRef.current;
    const docFmt = docFmtRef.current;
    if (!editor || editor.isDisposed || !docFmt) return;

    const buffer = await file.arrayBuffer();
    setIsLoadingDoc(true);
    openBlobInEditor(buffer, file.name, `File lokal: ${file.name}`);
    showToast(`Berkas "${file.name}" dimuat ke editor dan siap diedit`);
  }

  // Sisipkan penanda sistem ke posisi kursor di DevExpress RichEdit + salin ke clipboard
  function handleInsertPlaceholder(code: string) {
    navigator.clipboard?.writeText(code);
    setCopiedCode(code);
    setTimeout(() => {
      setCopiedCode((prev) => (prev === code ? null : prev));
    }, 2000);

    const editor = instanceRef.current;
    if (editor && !editor.isDisposed) {
      try {
        const subDoc = editor.document?.subDocuments?.main;
        const pos = editor.selection?.active ?? 0;
        if (subDoc) {
          editor.beginUpdate();
          subDoc.insertText(pos, code);
          editor.endUpdate();
          editor.focus();
          showToast(`Penanda ${code} disisipkan ke naskah`);
          return;
        }
      } catch {
        // Fallback ke clipboard saja
      }
    }
    showToast(`Penanda ${code} disalin ke clipboard`);
  }

  function handleDownloadCurrentDocx() {
    const editor = instanceRef.current;
    const docFmt = docFmtRef.current;
    if (!editor || editor.isDisposed || !docFmt) return;
    const safeName =
      (name.trim() || "Template_Surat").replace(/\s+/g, "_") + ".docx";
    editor.exportToFile((file: File) => {
      const url = URL.createObjectURL(file);
      const a = document.createElement("a");
      a.href = url;
      a.download = safeName;
      a.click();
      URL.revokeObjectURL(url);
      showToast(`Diunduh sebagai ${safeName}`);
    }, docFmt.OpenXml);
  }

  function exportEditorToDocxFile(fileName: string): Promise<File | null> {
    return new Promise((resolve) => {
      const editor = instanceRef.current;
      const docFmt = docFmtRef.current;
      if (!editor || editor.isDisposed || !docFmt) {
        resolve(uploadedFallbackFile);
        return;
      }
      try {
        editor.exportToFile((exported: File) => {
          const renamed = new File([exported], fileName, {
            type:
              exported.type ||
              "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
          });
          resolve(renamed);
        }, docFmt.OpenXml);
      } catch {
        resolve(uploadedFallbackFile);
      }
    });
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const cleanName = name.trim();
    if (!cleanName) {
      setError("Nama template surat wajib diisi.");
      return;
    }

    setBusy(true);
    setError("");
    try {
      const safeFileName = `${cleanName.replace(/[^A-Za-z0-9_.-]+/g, "_")}.docx`;
      const docxFile = await exportEditorToDocxFile(safeFileName);
      if (!docxFile) {
        throw new Error(
          "Dokumen template belum siap. Unggah berkas .docx atau tunggu editor selesai memuat.",
        );
      }

      const payload = {
        name: cleanName,
        description: description.trim(),
        letterTypeId: letterTypeId.trim(),
        file: docxFile,
      };

      if (mode === "create") {
        await createTemplate(payload);
      } else {
        await updateTemplate(String(row?.id), payload);
      }

      invalidateTemplateCache();
      await onSaved(
        mode === "create"
          ? `Template "${cleanName}" berhasil dibuat dan disimpan.`
          : `Template "${cleanName}" berhasil diperbarui.`,
      );
      onClose();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Gagal menyimpan template surat.",
      );
    } finally {
      setBusy(false);
    }
  }

  const activeHeight = isFullscreen
    ? FULLSCREEN_EDITOR_HEIGHT
    : DEFAULT_EDITOR_HEIGHT;

  return (
    <div className="fixed inset-0 z-50 bg-black/55 backdrop-blur-[1px] flex items-center justify-center p-2 sm:p-4 overflow-y-auto">
      <div
        className={`bg-surface border border-border rounded-2xl shadow-2xl flex flex-col overflow-hidden transition-all ${
          isFullscreen
            ? "fixed inset-2 z-50 max-w-none max-h-none"
            : "w-full max-w-6xl max-h-[94vh]"
        }`}
      >
        {/* Top Modal Header */}
        <div className="flex flex-wrap items-center justify-between gap-3 px-5 py-3.5 border-b border-border bg-panel/40 shrink-0">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-xl bg-primary/10 text-primary">
              <FileText size={19} />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-base font-bold text-text">
                  {mode === "create"
                    ? "Tambah & Desain Template Surat (.docx)"
                    : `Edit Template Surat — ${String(row?.name ?? "")}`}
                </h2>
                <span className="inline-flex items-center gap-1 rounded-full bg-primary/10 text-primary px-2.5 py-0.5 text-[11px] font-semibold">
                  DevExpress Word Editor
                </span>
              </div>
              <p className="text-xs text-text-muted">
                Unggah berkas .docx dari komputer atau ketik dan sesuaikan
                langsung tata naskah di editor Word di bawah ini.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setIsFullscreen((v) => !v)}
              className={buttonClass}
              title={isFullscreen ? "Keluar layar penuh" : "Layar penuh"}
            >
              {isFullscreen ? <Minimize2 size={14} /> : <Maximize2 size={14} />}
              <span className="hidden sm:inline">
                {isFullscreen ? "Perkecil" : "Layar Penuh"}
              </span>
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={onClose}
              aria-label="Tutup editor template"
              className="p-2 rounded-lg hover:bg-panel text-text-muted hover:text-text transition-colors"
            >
              <X size={18} />
            </button>
          </div>
        </div>

        {/* Scrollable Body */}
        <form
          onSubmit={handleSubmit}
          className="flex-1 overflow-y-auto p-4 sm:p-5 space-y-4"
        >
          {error && (
            <div
              role="alert"
              className="flex items-center gap-2.5 rounded-xl bg-status-danger-bg px-4 py-3 text-xs font-medium text-status-danger-text"
            >
              <AlertCircle size={16} className="shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {/* Metadata Form Row */}
          <div className="grid gap-3.5 md:grid-cols-3 rounded-xl border border-border bg-panel/30 p-4">
            <label className="block space-y-1.5 text-xs font-semibold text-text">
              <span>
                Nama Template Surat{" "}
                <span className="text-status-danger-text">*</span>
              </span>
              <input
                required
                maxLength={150}
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="Contoh: Nota Dinas Internal"
                className={inputClass}
              />
            </label>

            <label className="block space-y-1.5 text-xs font-semibold text-text">
              <span>Peruntukan Jenis Surat</span>
              <select
                value={letterTypeId}
                onChange={(event) => setLetterTypeId(event.target.value)}
                className={inputClass}
              >
                <option value="">Semua jenis surat (Umum)</option>
                {(letterTypesQuery.data ?? []).map((choice) => (
                  <option key={choice.value} value={choice.value}>
                    {choice.label}
                  </option>
                ))}
              </select>
            </label>

            <label className="block space-y-1.5 text-xs font-semibold text-text">
              <span>Deskripsi Penggunaan</span>
              <input
                maxLength={500}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                placeholder="Keterangan singkat kapan template ini dipakai..."
                className={inputClass}
              />
            </label>
          </div>

          {/* Document Source & Placeholder Action Bar */}
          <div
            ref={wrapperRef}
            className="rounded-xl border border-border bg-surface overflow-hidden shadow-xs flex flex-col"
          >
            {/* Editor Command Bar */}
            <div className="flex flex-wrap items-center justify-between gap-2 px-3.5 py-2.5 bg-panel border-b border-border">
              <div className="flex flex-wrap items-center gap-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".docx,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
                  onChange={(event) => {
                    const selected = event.target.files?.[0];
                    handleFileUpload(selected);
                    event.target.value = "";
                  }}
                  className="hidden"
                />
                <button
                  type="button"
                  disabled={!isReady || busy}
                  onClick={() => fileInputRef.current?.click()}
                  className={softPrimaryButtonClass}
                >
                  <Upload size={14} />
                  Unggah / Buka .docx dari Komputer
                </button>

                {/* Starter Template Selector */}
                <select
                  aria-label="Muat format dasar contoh"
                  disabled={!isReady || busy}
                  defaultValue=""
                  onChange={(event) => {
                    const tplId = event.target.value;
                    if (!tplId) return;
                    const starter = LETTER_TEMPLATE_LIST.find(
                      (item) => item.id === tplId,
                    );
                    if (starter) {
                      if (mode === "create" && !name.trim() && starter.id !== "tmpl-kosong") {
                        setName(starter.name);
                        if (!description.trim() && starter.description) {
                          setDescription(starter.description);
                        }
                      }
                      setIsLoadingDoc(true);
                      loadFromUrl(
                        starter.fileUrl,
                        `${starter.name.replace(/\s+/g, "_")}.docx`,
                        `Contoh dasar: ${starter.name}`,
                      );
                    }
                    event.target.value = "";
                  }}
                  className="rounded-lg border border-border bg-surface px-2.5 py-1.5 text-xs font-medium text-text focus:outline-none focus:ring-2 focus:ring-primary/30"
                >
                  <option value="">Muat format dasar...</option>
                  {LETTER_TEMPLATE_LIST.map((starter) => (
                    <option key={starter.id} value={starter.id}>
                      {starter.name}
                    </option>
                  ))}
                </select>

                <button
                  type="button"
                  disabled={!isReady || busy}
                  onClick={handleDownloadCurrentDocx}
                  className={buttonClass}
                  title="Unduh hasil edit saat ini sebagai file .docx"
                >
                  <Download size={13} />
                  Unduh .docx
                </button>

                <span className="text-[11px] text-text-muted px-2 py-1 rounded bg-surface border border-border truncate max-w-64">
                  {sourceLabel}
                </span>
              </div>

              <div className="flex items-center gap-2">
                {toast && (
                  <span className="text-xs font-medium text-status-success-text bg-status-success-bg px-2.5 py-1 rounded-lg">
                    {toast}
                  </span>
                )}
                <WordZoomControl
                  zoomLevel={zoomLevel}
                  onZoomChange={applyZoom}
                  showPresets={false}
                />
              </div>
            </div>

            {/* Quick Insert System Placeholders Strip */}
            <div className="px-3.5 py-2 bg-panel/45 border-b border-border flex items-center gap-1.5 overflow-x-auto no-scrollbar">
              <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-text-muted shrink-0 mr-1">
                <Hash size={12} className="text-primary" />
                Sisipkan 4 Penanda Sistem Otomatis:
              </span>
              {SYSTEM_PLACEHOLDERS.map((item) => {
                const isCopied = copiedCode === item.code;
                return (
                  <button
                    key={item.code}
                    type="button"
                    disabled={!isReady}
                    onClick={() => handleInsertPlaceholder(item.code)}
                    title={`${item.label}: ${item.description} (Klik untuk menyisipkan ke posisi kursor)`}
                    className="inline-flex items-center gap-1 rounded-md border border-border bg-surface hover:border-primary/40 px-2 py-1 text-[11px] font-mono text-text hover:text-primary transition-colors shrink-0"
                  >
                    <span>{item.code}</span>
                    {isCopied ? (
                      <Check
                        size={11}
                        className="text-status-success-text shrink-0"
                      />
                    ) : (
                      <Copy size={10} className="text-text-muted shrink-0" />
                    )}
                  </button>
                );
              })}
              <span className="text-[11px] text-text-muted ml-2 shrink-0">
                (Wajib memuat <code className="font-mono text-text">###*QRCode*###</code> untuk surat internal)
              </span>
            </div>

            {/* DevExpress RichEdit Host Canvas */}
            <div className="relative w-full bg-white">
              {(!isReady || isLoadingDoc) && (
                <div className="absolute inset-0 z-10 bg-surface/85 backdrop-blur-[1px] flex flex-col items-center justify-center gap-2.5 text-text-muted">
                  <Loader2 size={24} className="animate-spin text-primary" />
                  <p className="text-xs font-semibold text-text">
                    {!isReady
                      ? "Menyiapkan DevExpress Word RichEdit..."
                      : "Memuat dokumen template (.docx)..."}
                  </p>
                </div>
              )}

              <div
                ref={containerRef}
                style={{
                  height: activeHeight,
                  minHeight: activeHeight,
                  width: "100%",
                }}
                className="w-full bg-white"
              />

              {feedbackPercent !== null && (
                <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-30 pointer-events-none bg-neutral-900/90 text-white text-xs px-3.5 py-1.5 rounded-full font-medium shadow-lg">
                  Zoom: {feedbackPercent}%
                </div>
              )}
            </div>
          </div>

          {/* Modal Footer */}
          <div className="flex flex-wrap items-center justify-between gap-3 pt-2 border-t border-border">
            <p className="text-xs text-text-muted">
              Saat disimpan, naskah di dalam editor DevExpress otomatis diekspor
              menjadi berkas <code className="font-mono text-text">.docx</code>{" "}
              dan disimpan ke penyimpanan template perusahaan.
            </p>
            <div className="flex items-center gap-2">
              <button
                type="button"
                disabled={busy}
                onClick={onClose}
                className={buttonClass}
              >
                Batal
              </button>
              <button
                type="submit"
                disabled={busy || !isReady || isLoadingDoc}
                className={primaryButtonClass}
              >
                {busy ? (
                  <Loader2 size={14} className="animate-spin" />
                ) : (
                  <Save size={14} />
                )}
                {busy
                  ? "Menyimpan Template..."
                  : mode === "create"
                    ? "Simpan Template Baru"
                    : "Simpan Perubahan Template"}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}

