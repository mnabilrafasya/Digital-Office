"use client";

// ─────────────────────────────────────────────
// components/review/document-viewer/review-document-inner.tsx
// Pratinjau naskah surat di halaman detail / review / approval (/surat/[id]).
// SELALU mode baca (readOnly): reviewer dan approver menilai naskah, tidak mengetik.
// Di-load murni di browser lewat index.tsx (dynamic, ssr: false).
//
// Backend BELUM punya endpoint dokumen/versi, jadi naskah dibangun ulang dari template +
// data surat (perihal, penerima, approver, nomor). Editing naskah hanya di halaman Buat Surat.
// ─────────────────────────────────────────────

import { useEffect, useRef, useState } from "react";
import { Download, FileDown, Info, Maximize2, Minimize2 } from "lucide-react";

import "devextreme/dist/css/dx.light.css";
import "devexpress-richedit/dist/dx.richedit.css";

import type { RichEdit, Options } from "devexpress-richedit";
import type { LetterDetail } from "@/lib/api/letter-detail";
import {
  getTemplateById,
  type LetterTemplateContext,
} from "@/lib/templates/letter-templates";

type RichEditModule = typeof import("devexpress-richedit") & {
  license?: (key: string) => void;
};

const LICENSE_KEY = process.env.NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY ?? "";

export interface ReviewDocumentInnerProps {
  letter: LetterDetail;
}

/** Data surat -> konteks template naskah. */
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
    drafter: { name: letter.drafter.name, position: letter.drafter.positionName },
    date: letter.letterDate
      ? new Intl.DateTimeFormat("id-ID", {
          day: "numeric",
          month: "long",
          year: "numeric",
        }).format(new Date(letter.letterDate))
      : undefined,
    attachmentsCount: letter.attachments.length,
  };
}

export default function ReviewDocumentInner({ letter }: ReviewDocumentInnerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<RichEdit | null>(null);
  const moduleRef = useRef<RichEditModule | null>(null);

  const [isReady, setIsReady] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);

  // Inisialisasi editor sekali (mount). Konten dimuat di efek berikutnya.
  useEffect(() => {
    if (!containerRef.current) return;
    let disposed = false;

    (async () => {
      try {
        const mod = (await import("devexpress-richedit")) as RichEditModule;
        if (disposed || !containerRef.current) return;
        moduleRef.current = mod;
        if (LICENSE_KEY && mod.license) mod.license(LICENSE_KEY);

        const options: Options = mod.createOptions();
        options.width = "100%";
        options.height = "100%";
        options.readOnly = true; // mode baca: tidak bisa diketik
        options.ribbon.visible = false; // tanpa ribbon; unduh lewat toolbar di atas

        editorRef.current = mod.create(containerRef.current, options);
        setIsReady(true);
      } catch (err) {
        console.error("[ReviewDocumentViewer] Inisialisasi gagal:", err);
        setErrorMsg(
          err instanceof Error ? err.message : "Gagal memuat DevExpress RichEdit."
        );
      }
    })();

    return () => {
      disposed = true;
      editorRef.current?.dispose?.();
      editorRef.current = null;
      setIsReady(false);
    };
  }, []);

  // Muat naskah dari template + data surat bila editor siap / data surat berubah.
  useEffect(() => {
    const editor = editorRef.current;
    const mod = moduleRef.current;
    if (!isReady || !editor || !mod) return;

    const tpl = getTemplateById(letter.template?.code);
    const html = tpl
      .generateHtml(buildContext(letter))
      .replace(/###NoSurat###/g, letter.letterNumber ?? "###NoSurat###");
    const blob = new Blob([html], { type: "text/html;charset=utf-8" });
    editor.openDocument(blob, `${tpl.name}.html`, mod.DocumentFormat.Html);
  }, [isReady, letter]);

  // Sesuaikan ukuran saat fullscreen berubah.
  useEffect(() => {
    const t = setTimeout(() => editorRef.current?.adjust?.(), 150);
    return () => clearTimeout(t);
  }, [isFullscreen]);

  function handleDownload() {
    const editor = editorRef.current;
    const mod = moduleRef.current;
    if (!editor || !mod) return;
    editor.exportToFile((file: File) => {
      const url = URL.createObjectURL(file);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${letter.letterNumber ?? "Naskah_Dinas"}.docx`.replace(/[\\/]/g, "-");
      a.click();
      URL.revokeObjectURL(url);
    }, mod.DocumentFormat.OpenXml);
  }

  function handleDownloadPdf() {
    const editor = editorRef.current;
    if (!editor) return;
    const base = `${letter.letterNumber ?? "Naskah_Dinas"}`.replace(/[\\/]/g, "-");
    editor.downloadPdf(base);
  }

  if (errorMsg) {
    return (
      <div className="flex flex-col items-center justify-center p-8 bg-panel/50 border border-border rounded-xl text-center">
        <div className="w-12 h-12 rounded-full bg-status-danger-bg flex items-center justify-center mb-3">
          <Info size={20} className="text-status-danger-text" />
        </div>
        <p className="text-sm font-semibold text-text">Pratinjau naskah gagal dimuat</p>
        <p className="text-xs text-text-muted mt-1 max-w-md">{errorMsg}</p>
      </div>
    );
  }

  const wrapperClass = isFullscreen
    ? "fixed inset-0 z-[45] isolate bg-surface flex flex-col p-4 shadow-2xl"
    : // isolate + z-0: z-index internal DevExpress terkurung di sini, tidak menimpa
      // bar aksi bawah (z-20/z-30) maupun modal konfirmasi (z-40/z-50).
      "relative isolate z-0 flex flex-col border border-border rounded-xl bg-surface overflow-hidden shadow-sm";

  return (
    <div className={wrapperClass}>
      <div className="flex items-center gap-2 px-4 py-2.5 bg-panel border-b border-border flex-wrap shrink-0">
        <span className="text-xs font-semibold text-text">Naskah Surat</span>
        <span className="text-[11px] text-text-muted bg-surface border border-border px-2 py-0.5 rounded">
          Mode baca
        </span>
        <div className="flex items-center gap-2 ml-auto">
          <button
            type="button"
            onClick={handleDownload}
            disabled={!isReady}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-border bg-surface hover:bg-bg transition-colors text-text shadow-sm disabled:opacity-50"
          >
            <Download size={13} className="text-primary" />
            Unduh .docx
          </button>
          <button
            type="button"
            onClick={handleDownloadPdf}
            disabled={!isReady}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-border bg-surface hover:bg-bg transition-colors text-text shadow-sm disabled:opacity-50"
          >
            <FileDown size={13} className="text-primary" />
            Unduh PDF
          </button>
          <button
            type="button"
            onClick={() => setIsFullscreen((v) => !v)}
            className="p-1.5 rounded-lg border border-border bg-surface hover:bg-bg text-text-muted hover:text-text transition-colors shadow-sm"
            title={isFullscreen ? "Keluar layar penuh" : "Layar penuh"}
          >
            {isFullscreen ? <Minimize2 size={15} /> : <Maximize2 size={15} />}
          </button>
        </div>
      </div>

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