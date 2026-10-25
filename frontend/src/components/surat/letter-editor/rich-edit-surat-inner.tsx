"use client";

// ─────────────────────────────────────────────
// components/surat/letter-editor/rich-edit-surat-inner.tsx
// Implementasi DevExpress RichEdit khusus untuk Section 3 Buat Surat.
// HANYA di-load secara dinamis via ssr: false.
// ─────────────────────────────────────────────

import { useRef, useEffect, useCallback, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import {
  Upload,
  Save,
  FileText,
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

  const [isReady, setIsReady] = useState(false);
  const [fileName, setFileName] = useState<string | null>("Naskah_Dinas_Draf.docx");
  const [isSaving, setIsSaving] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [appliedTemplateId, setAppliedTemplateId] = useState<string>("tmpl-dinas-standar");
  const [statusNotification, setStatusNotification] = useState<string | null>(null);

  // Ambil state data form dari FormProvider Buat Surat
  const { control, setValue } = useFormContext<BuatSuratFormValues>();
  const formTitle = useWatch({ control, name: "title" });
  const formSifat = useWatch({ control, name: "sifat" });
  const formKecepatan = useWatch({ control, name: "kecepatan" });
  const formTemplateId = useWatch({ control, name: "templateId" });
  const formToUnits = useWatch({ control, name: "toUnits" });
  const formApprovers = useWatch({ control, name: "approvers" });

  // Helper untuk membuat context template dari form
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

  // Fungsi untuk memuat template ke dalam editor
  const loadTemplateIntoEditor = useCallback(
    (templateId: string, showToast = true) => {
      const editor = instanceRef.current;
      const docFmt = docFmtRef.current;
      if (!editor || !docFmt) return;

      try {
        const tpl = getTemplateById(templateId);
        const ctx = getTemplateContext();
        const html = tpl.generateHtml(ctx);

        // Buka HTML template di DevExpress RichEdit
        editor.openDocument(html, `${tpl.name}.html`, docFmt.Html, (success) => {
          if (success) {
            setAppliedTemplateId(templateId);
            setFileName(`${tpl.name}.docx`);
            if (showToast) {
              setStatusNotification(`Template "${tpl.name}" berhasil diterapkan`);
              setTimeout(() => setStatusNotification(null), 3500);
            }
            onDocumentChange?.();
          }
        });
      } catch (err) {
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
        options.ribbon.visible = true;

        if (disposed) return;

        const editor = mod.create(containerRef.current!, options);
        instanceRef.current = editor;
        setIsReady(true);

        // Setelah inisialisasi selesai, muat template default (Surat Dinas Standar)
        setTimeout(() => {
          if (!disposed && instanceRef.current && mod.DocumentFormat) {
            const initialTplId = formTemplateId || "tmpl-dinas-standar";
            const tpl = getTemplateById(initialTplId);
            const ctx = getTemplateContext();
            const html = tpl.generateHtml(ctx);
            instanceRef.current.openDocument(html, "Draft.html", mod.DocumentFormat.Html);
            setAppliedTemplateId(initialTplId);
          }
        }, 200);
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
  }, []); // Run once on mount

  // Jika user memilih template berbeda di Section 1 (formTemplateId berubah)
  useEffect(() => {
    if (isReady && formTemplateId && formTemplateId !== appliedTemplateId) {
      loadTemplateIntoEditor(formTemplateId, false);
    }
  }, [formTemplateId, isReady, appliedTemplateId, loadTemplateIntoEditor]);

  // ─── Sinkronkan data form ke template naskah ──────────────────────
  const handleSyncFormData = useCallback(() => {
    loadTemplateIntoEditor(appliedTemplateId, true);
  }, [appliedTemplateId, loadTemplateIntoEditor]);

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

        instanceRef.current.openDocument(buffer, file.name, fmt);
        setFileName(file.name);
        setStatusNotification(`File "${file.name}" berhasil dibuka`);
        setTimeout(() => setStatusNotification(null), 3000);
        onDocumentChange?.();
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
    ? "fixed inset-0 z-50 bg-surface flex flex-col p-4 shadow-2xl"
    : "flex flex-col border border-border rounded-xl bg-surface overflow-hidden shadow-sm";

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
          <div className="flex items-center gap-1.5 text-xs text-emerald-700 bg-emerald-50 border border-emerald-200 px-2.5 py-1 rounded-md animate-fade-in">
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

        {!LICENSE_KEY ? (
          <span className="text-[11px] text-amber-600 bg-amber-50 border border-amber-200 px-2 py-0.5 rounded">
            Evaluasi DevExpress (Unlicensed Watermark)
          </span>
        ) : (
          <span className="text-[11px] text-emerald-600 bg-emerald-50 border border-emerald-200 px-2 py-0.5 rounded">
            DevExpress Terlisensi
          </span>
        )}
      </div>

      {/* ── Container DevExpress RichEdit ── */}
      <div
        ref={containerRef}
        style={{
          height: isFullscreen ? "calc(100vh - 110px)" : "640px",
          width: "100%",
          minHeight: "450px",
        }}
        className="w-full bg-white relative"
      />
    </div>
  );
}
