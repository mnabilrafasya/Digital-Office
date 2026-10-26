"use client";

// ─────────────────────────────────────────────
// components/surat/letter-editor/rich-edit-surat-inner.tsx
// Implementasi DevExpress RichEdit untuk Section 3 Buat Surat.
// Di-load murni di sisi browser (ssr: false).
// ─────────────────────────────────────────────

import { useRef, useEffect, useCallback, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import {
  Upload,
  Save,
  FileText,
  FileDown,
  RefreshCw,
  Maximize2,
  Minimize2,
  Sparkles,
  Info,
  CheckCircle2,
} from "lucide-react";

import "devextreme/dist/css/dx.light.css";
import "devexpress-richedit/dist/dx.richedit.css";

import type { RichEdit, Options, DocumentFormat } from "devexpress-richedit";
import type { BuatSuratFormValues } from "@/lib/schemas";
import {
  LETTER_TEMPLATE_LIST,
  getTemplateById,
  type LetterTemplateContext,
} from "@/lib/templates/letter-templates";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type RichEditModule = typeof import("devexpress-richedit") & {
  license?: (key: string) => void;
};

const LICENSE_KEY = process.env.NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY ?? "";

interface RichEditSuratInnerProps {
  onDocumentChange?: () => void;
}

export default function RichEditSuratInner({ onDocumentChange }: RichEditSuratInnerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const instanceRef = useRef<RichEdit | null>(null);
  const moduleRef = useRef<RichEditModule | null>(null);
  const docFmtRef = useRef<typeof DocumentFormat | null>(null);
  // true saat dokumen dimuat lewat kode (template/buka file) supaya tidak dihitung "diketik user"
  const isLoadingDocRef = useRef(false);
  // true bila user sudah mengetik/mengubah isi naskah sejak pemuatan terakhir
  const userEditedRef = useRef(false);

  const [isReady, setIsReady] = useState(false);
  const [fileName, setFileName] = useState<string | null>("Naskah_Dinas_Draf.docx");
  const [isSaving, setIsSaving] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [appliedTemplateId, setAppliedTemplateId] = useState<string>("tmpl-dinas-standar");
  const [statusNotification, setStatusNotification] = useState<string | null>(null);

  // Ambil state form dari Buat Surat
  const { control, setValue } = useFormContext<BuatSuratFormValues>();
  const formTitle = useWatch({ control, name: "title" });
  const formSifat = useWatch({ control, name: "sifat" });
  const formKecepatan = useWatch({ control, name: "kecepatan" });
  const formTemplateId = useWatch({ control, name: "templateId" });
  const formToUnits = useWatch({ control, name: "toUnits" });
  const formApprovers = useWatch({ control, name: "approvers" });

  // Context dinamis dari form untuk dimasukkan ke template naskah
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

  // Mengganti template / sinkron form menimpa seluruh isi naskah: minta konfirmasi bila sudah diketik.
  const confirmOverwrite = useCallback(() => {
    if (!userEditedRef.current) return true;
    return window.confirm(
      "Isi naskah yang sudah Anda ketik akan diganti dengan template. Lanjutkan?"
    );
  }, []);

  // ─── Memuat template ke DevExpress RichEdit ───────────────────────
  const loadTemplateIntoEditor = useCallback(
    (templateId: string, showToast = true) => {
      const editor = instanceRef.current;
      const docFmt = docFmtRef.current;
      if (!editor || !docFmt) return;

      isLoadingDocRef.current = true;
      userEditedRef.current = false;
      const releaseLoading = () =>
        setTimeout(() => {
          isLoadingDocRef.current = false;
        }, 500);

      try {
        if (templateId === "tmpl-kosong") {
          editor.newDocument();
          releaseLoading();
          setAppliedTemplateId(templateId);
          setFileName("Dokumen_Kosong.docx");
          if (showToast) {
            setStatusNotification("Dokumen kosong baru dibuka");
            setTimeout(() => setStatusNotification(null), 3000);
          }
          editor.focus();
          onDocumentChange?.();
          return;
        }

        const tpl = getTemplateById(templateId);
        const ctx = getTemplateContext();
        const html = tpl.generateHtml(ctx);

        // DevExpress openDocument menerima Blob/File untuk format HTML
        const blob = new Blob([html], { type: "text/html;charset=utf-8" });

        editor.openDocument(blob, `${tpl.name}.html`, docFmt.Html, (success) => {
          releaseLoading();
          if (success) {
            setAppliedTemplateId(templateId);
            setFileName(`${tpl.name}.docx`);
            if (showToast) {
              setStatusNotification(`Template "${tpl.name}" berhasil dimuat`);
              setTimeout(() => setStatusNotification(null), 3000);
            }
            editor.focus();
            onDocumentChange?.();
          } else {
            console.warn("[LetterEditor] openDocument callback mengembalikan false");
          }
        });
      } catch (err) {
        isLoadingDocRef.current = false;
        console.error("[LetterEditor] Gagal memuat template:", err);
      }
    },
    [getTemplateContext, onDocumentChange]
  );

  // ─── Inisialisasi DevExpress RichEdit ─────────────────────────────
  useEffect(() => {
    if (!containerRef.current) return;
    let disposed = false;

    (async () => {
      try {
        const mod = (await import("devexpress-richedit")) as RichEditModule;
        moduleRef.current = mod;
        docFmtRef.current = mod.DocumentFormat;

        if (LICENSE_KEY && mod.license) {
          mod.license(LICENSE_KEY);
        }

        const options: Options = mod.createOptions();
        options.width = "100%";
        options.height = "100%";
        options.readOnly = false;
        options.ribbon.visible = true;
        options.events.documentChanged = () => {
          if (!isLoadingDocRef.current) userEditedRef.current = true;
        };

        if (disposed) return;

        const editor = mod.create(containerRef.current!, options);
        instanceRef.current = editor;
        setIsReady(true);

        // Setelah siap, muat template default (Surat Dinas Standar)
        setTimeout(() => {
          if (!disposed && instanceRef.current && mod.DocumentFormat) {
            isLoadingDocRef.current = true;
            const releaseLoading = () =>
              setTimeout(() => {
                isLoadingDocRef.current = false;
              }, 500);
            const initialTplId = formTemplateId || "tmpl-dinas-standar";
            if (initialTplId === "tmpl-kosong") {
              instanceRef.current.newDocument();
              releaseLoading();
              setAppliedTemplateId(initialTplId);
              instanceRef.current.focus();
            } else {
              const tpl = getTemplateById(initialTplId);
              const ctx = getTemplateContext();
              const html = tpl.generateHtml(ctx);
              const blob = new Blob([html], { type: "text/html;charset=utf-8" });
              instanceRef.current.openDocument(
                blob,
                `${tpl.name}.html`,
                mod.DocumentFormat.Html,
                (success) => {
                  releaseLoading();
                  if (success) {
                    instanceRef.current?.focus();
                  }
                }
              );
              setAppliedTemplateId(initialTplId);
            }
          }
        }, 150);
      } catch (err) {
        console.error("[LetterEditor] Inisialisasi gagal:", err);
        setErrorMsg(
          err instanceof Error
            ? err.message
            : "Gagal memuat DevExpress RichEdit. Pastikan dependensi devexpress-richedit terinstall."
        );
      }
    })();

    return () => {
      disposed = true;
      instanceRef.current?.dispose?.();
      instanceRef.current = null;
      setIsReady(false);
    };
  }, []); // Mount sekali

  // Resize / adjust editor saat toggle fullscreen
  useEffect(() => {
    if (instanceRef.current) {
      const timer = setTimeout(() => {
        instanceRef.current?.adjust?.();
      }, 150);
      return () => clearTimeout(timer);
    }
  }, [isFullscreen]);

  // Jika user memilih template di Section 1 (formTemplateId berubah)
  useEffect(() => {
    if (isReady && formTemplateId && formTemplateId !== appliedTemplateId) {
      if (!confirmOverwrite()) {
        // batal: kembalikan pilihan template di form ke template yang sedang dipakai
        setValue("templateId", appliedTemplateId);
        return;
      }
      loadTemplateIntoEditor(formTemplateId, false);
    }
  }, [formTemplateId, isReady, appliedTemplateId, loadTemplateIntoEditor, confirmOverwrite, setValue]);

  // ─── Sinkronkan data form ke naskah surat ─────────────────────────
  const handleSyncFormData = useCallback(() => {
    if (!confirmOverwrite()) return;
    loadTemplateIntoEditor(appliedTemplateId, true);
  }, [appliedTemplateId, loadTemplateIntoEditor, confirmOverwrite]);

  // ─── Buka file DOCX dari komputer ────────────────────────────────
  const handleOpenDocx = useCallback(() => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".docx,.rtf,.txt";

    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (!file || !instanceRef.current || !docFmtRef.current) return;

      try {
        const buffer = await file.arrayBuffer();
        const ext = file.name.split(".").pop()?.toLowerCase();
        const fmt: DocumentFormat =
          ext === "rtf"
            ? docFmtRef.current.Rtf
            : ext === "txt"
            ? docFmtRef.current.PlainText
            : docFmtRef.current.OpenXml;

        isLoadingDocRef.current = true;
        userEditedRef.current = false;
        instanceRef.current.openDocument(buffer, file.name, fmt, (success) => {
          setTimeout(() => {
            isLoadingDocRef.current = false;
          }, 500);
          if (success) {
            setFileName(file.name);
            setStatusNotification(`File "${file.name}" berhasil dibuka`);
            setTimeout(() => setStatusNotification(null), 3000);
            instanceRef.current?.focus();
            onDocumentChange?.();
          }
        });
      } catch (err) {
        console.error("[LetterEditor] Gagal membuka file:", err);
        alert("Gagal membuka file: " + (err instanceof Error ? err.message : String(err)));
      }
    };

    input.click();
  }, [onDocumentChange]);

  // ─── Simpan / unduh sebagai DOCX ─────────────────────────────────
  const handleSaveDocx = useCallback(() => {
    if (!instanceRef.current || !docFmtRef.current) return;
    setIsSaving(true);

    try {
      instanceRef.current.exportToFile((file: File) => {
        const url = URL.createObjectURL(file);
        const a = document.createElement("a");
        a.href = url;
        a.download = fileName || "Naskah_Dinas.docx";
        a.click();
        URL.revokeObjectURL(url);
        setIsSaving(false);
        setStatusNotification(`Dokumen berhasil diunduh sebagai ${a.download}`);
        setTimeout(() => setStatusNotification(null), 3000);
      }, docFmtRef.current.OpenXml);
    } catch (err) {
      console.error("[LetterEditor] Gagal menyimpan file:", err);
      alert("Gagal menyimpan dokumen: " + (err instanceof Error ? err.message : String(err)));
      setIsSaving(false);
    }
  }, [fileName]);

  // ─── Unduh sebagai PDF (output final; salinan yang bisa diedit tetap .docx) ───
  const handleDownloadPdf = useCallback(() => {
    const editor = instanceRef.current;
    if (!editor) return;
    try {
      const baseName = (fileName || "Naskah_Dinas").replace(/\.[^.]+$/, "");
      editor.downloadPdf(baseName);
    } catch (err) {
      console.error("[LetterEditor] Gagal membuat PDF:", err);
      alert("Gagal membuat PDF: " + (err instanceof Error ? err.message : String(err)));
    }
  }, [fileName]);

  // ─── Error State ──────────────────────────────────────────────────
  if (errorMsg) {
    return (
      <div className="flex flex-col items-center justify-center p-8 bg-panel/50 border border-border rounded-xl text-center">
        <div className="w-12 h-12 rounded-full bg-status-danger-bg flex items-center justify-center mb-3">
          <Info size={20} className="text-status-danger-text" />
        </div>
        <p className="text-sm font-semibold text-text">Editor Naskah Gagal Dimuat</p>
        <p className="text-xs text-text-muted mt-1 max-w-md">{errorMsg}</p>
      </div>
    );
  }

  const containerClasses = isFullscreen
    ? "fixed inset-0 z-[45] isolate bg-surface flex flex-col p-4 shadow-2xl"
    : // isolate + z-0: z-index internal DevExpress (hingga ribuan) terkurung di sini,
      // jadi tidak menimpa action bar (z-20) dan dialog konfirmasi (z-50).
      "relative isolate z-0 flex flex-col border border-border rounded-xl bg-surface overflow-hidden shadow-sm";

  return (
    <div className={containerClasses}>
      {/* ── Toolbar Atas: Template, Aksi Dokumen, Fullscreen ── */}
      <div className="flex items-center justify-between gap-3 px-4 py-2.5 bg-panel border-b border-border flex-wrap shrink-0">
        {/* Pilihan Template Tata Naskah */}
        <div className="flex items-center gap-2">
          <span className="flex items-center gap-1.5 text-xs font-semibold text-text whitespace-nowrap">
            <Sparkles size={14} className="text-primary" />
            Template Naskah:
          </span>
          <select
            value={appliedTemplateId}
            onChange={(e) => {
              const newTplId = e.target.value;
              if (!confirmOverwrite()) return; // select tetap di template lama
              loadTemplateIntoEditor(newTplId, true);
              setValue("templateId", newTplId, { shouldDirty: true });
            }}
            className="text-xs py-1.5 px-2.5 rounded-lg border border-border bg-surface text-text font-medium focus:outline-none focus:ring-1 focus:ring-primary shadow-sm"
          >
            {LETTER_TEMPLATE_LIST.map((tpl) => (
              <option key={tpl.id} value={tpl.id}>
                {tpl.name}
              </option>
            ))}
          </select>

          {/* Tombol Sinkronkan Data dari Section 1 */}
          <button
            type="button"
            onClick={handleSyncFormData}
            title="Perbarui naskah dengan data terbaru dari Section 1 (Perihal, Sifat, Penerima, Penandatangan)"
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium border border-border bg-surface hover:bg-bg transition-colors text-text shadow-sm"
          >
            <RefreshCw size={12} className="text-text-muted" />
            Sinkronkan Form
          </button>
        </div>

        {/* Status Notifikasi Toast Kecil */}
        {statusNotification && (
          <div className="flex items-center gap-1.5 text-xs text-emerald-700 bg-emerald-50 border border-emerald-200 px-2.5 py-1 rounded-md">
            <CheckCircle2 size={13} />
            <span>{statusNotification}</span>
          </div>
        )}

        {/* Aksi File & Fullscreen */}
        <div className="flex items-center gap-2 ml-auto">
          <button
            type="button"
            onClick={handleOpenDocx}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-border bg-surface hover:bg-bg transition-colors text-text shadow-sm"
            title="Buka dokumen Word (.docx) dari komputer"
          >
            <Upload size={13} className="text-primary" />
            Buka .docx
          </button>

          <button
            type="button"
            onClick={handleSaveDocx}
            disabled={!isReady || isSaving}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-primary text-white hover:bg-primary-dark transition-colors shadow-sm disabled:opacity-50"
            title="Unduh naskah saat ini sebagai file Microsoft Word (.docx)"
          >
            <Save size={13} />
            {isSaving ? "Menyimpan..." : "Unduh .docx"}
          </button>

          <button
            type="button"
            onClick={handleDownloadPdf}
            disabled={!isReady}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-border bg-surface hover:bg-bg transition-colors text-text shadow-sm disabled:opacity-50"
            title="Unduh naskah saat ini sebagai PDF"
          >
            <FileDown size={13} className="text-primary" />
            Unduh PDF
          </button>

          {/* Toggle Fullscreen */}
          <button
            type="button"
            onClick={() => setIsFullscreen((prev) => !prev)}
            className="p-1.5 rounded-lg border border-border bg-surface hover:bg-bg text-text-muted hover:text-text transition-colors shadow-sm"
            title={isFullscreen ? "Keluar Fullscreen" : "Layar Penuh (Fullscreen)"}
          >
            {isFullscreen ? <Minimize2 size={15} /> : <Maximize2 size={15} />}
          </button>
        </div>
      </div>

      {/* ── Info Bar / Petunjuk Variabel ── */}
      <div className="flex items-center justify-between px-4 py-1.5 bg-surface border-b border-border/60 text-[11px] text-text-muted shrink-0 flex-wrap gap-2">
        <div className="flex items-center gap-2">
          <FileText size={12} className="text-text-muted" />
          <span>
            Variabel otomatis: <code className="bg-panel px-1 py-0.5 rounded font-mono text-[10px]">###NoSurat###</code> akan diisi nomor otomatis saat surat disetujui Approver.
          </span>
        </div>
        <span className="text-amber-700">
          Isi naskah belum tersimpan di server. &quot;Simpan Draft&quot; hanya menyimpan data form; unduh .docx bila perlu menyimpan naskah.
        </span>
      </div>

      {/* ── Container DevExpress RichEdit ── */}
      <div
        ref={containerRef}
        style={{
          height: isFullscreen ? "calc(100vh - 100px)" : "640px",
          width: "100%",
          minHeight: "450px",
        }}
        className="w-full bg-white relative"
      />
    </div>
  );
}