"use client";

// app/dev/editor/rich-edit-wrapper.tsx
// Halaman sandbox uji DevExpress RichEdit (/dev/editor).
// Menggunakan konfigurasi Word yang sama dengan halaman Buat Surat.

import { useRef, useEffect, useCallback, useState } from "react";
import { Upload, Save, FileText, Info } from "lucide-react";
import {
  WordZoomControl,
  useWordZoomInteractivity,
} from "@/components/ui/word-zoom-control";
import { useSelectOnFocus } from "@/components/surat/letter-editor/use-select-on-focus";
import {
  createWordEditorOptions,
  type RichEditModule,
} from "@/components/surat/letter-editor/word-editor-config";
import { resolveTemplateDocxBlob } from "@/lib/templates/letter-templates";

import "devextreme/dist/css/dx.light.css";
import "devexpress-richedit/dist/dx.richedit.css";

import type { RichEdit, DocumentFormat } from "devexpress-richedit";

const LICENSE_KEY = process.env.NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY ?? "";

export default function RichEditWrapper() {
  const wrapperRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const instanceRef = useRef<RichEdit | null>(null);
  const docFmtRef = useRef<typeof DocumentFormat | null>(null);

  const [isReady, setIsReady] = useState(false);
  const [fileName, setFileName] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const [zoomLevel, setZoomLevel] = useState<number>(1);
  const zoomLevelRef = useRef<number>(1);

  useEffect(() => {
    zoomLevelRef.current = zoomLevel;
  }, [zoomLevel]);

  const applyZoom = useCallback((level: number) => {
    const clamped = Math.min(2.0, Math.max(0.5, Math.round(level * 100) / 100));
    setZoomLevel(clamped);
    zoomLevelRef.current = clamped;
    if (instanceRef.current) {
      instanceRef.current.zoomLevel = clamped;
      instanceRef.current.adjust();
    }
  }, []);

  useSelectOnFocus(containerRef, isReady);

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

  useEffect(() => {
    if (!containerRef.current) return;
    let disposed = false;

    (async () => {
      try {
        const mod = (await import("devexpress-richedit")) as RichEditModule;
        if (disposed || !containerRef.current) return;
        docFmtRef.current = mod.DocumentFormat;

        if (LICENSE_KEY && mod.license) {
          mod.license(LICENSE_KEY);
        }

        const options = createWordEditorOptions({
          mod,
          readOnly: false,
          showRibbon: true,
          initialZoom: 1,
          height: "840px",
          onKeyDown: (s, args) => keyDownRef.current(s, args),
        });

        const editor = mod.create(containerRef.current, options);
        instanceRef.current = editor;
        setIsReady(true);

        // Muat dokumen kosong A4 standar Word agar margin & penggaris langsung presisi
        const { blob } = await resolveTemplateDocxBlob("tmpl-kosong");
        if (!disposed && instanceRef.current) {
          instanceRef.current.openDocument(
            blob,
            "Dokumen_Kosong.docx",
            mod.DocumentFormat.OpenXml,
            (success) => {
              if (success && instanceRef.current) {
                instanceRef.current.zoomLevel = zoomLevelRef.current;
                instanceRef.current.adjust();
              }
            }
          );
        }
      } catch (err) {
        console.error("[DevEditor] Gagal inisialisasi:", err);
        setErrorMsg(
          err instanceof Error
            ? err.message
            : "Gagal memuat editor. Pastikan devexpress-richedit sudah terinstall."
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

  const handleOpen = useCallback(() => {
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
      } catch (err) {
        console.error("[DevEditor] Gagal membuka file:", err);
        alert(
          "Gagal membuka file: " +
            (err instanceof Error ? err.message : String(err))
        );
      }
    };

    input.click();
  }, []);

  const handleSave = useCallback(() => {
    if (!instanceRef.current || !docFmtRef.current) return;
    setIsSaving(true);

    try {
      instanceRef.current.exportToFile((file: File) => {
        const url = URL.createObjectURL(file);
        const a = document.createElement("a");
        a.href = url;
        a.download = fileName ?? "dokumen.docx";
        a.click();
        URL.revokeObjectURL(url);
        setIsSaving(false);
      }, docFmtRef.current.OpenXml);
    } catch (err) {
      console.error("[DevEditor] Gagal menyimpan:", err);
      alert(
        "Gagal menyimpan: " + (err instanceof Error ? err.message : String(err))
      );
      setIsSaving(false);
    }
  }, [fileName]);

  if (errorMsg) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-4 text-center px-6">
        <div className="w-12 h-12 rounded-full bg-status-danger-bg flex items-center justify-center">
          <Info size={20} className="text-status-danger-text" />
        </div>
        <div>
          <p className="text-sm font-semibold text-text">Editor gagal dimuat</p>
          <p className="text-xs text-text-muted mt-1 max-w-md">{errorMsg}</p>
        </div>
      </div>
    );
  }

  return (
    <div ref={wrapperRef} className="flex flex-col h-full relative">
      <div className="flex items-center gap-2 px-4 py-2 border-b border-border bg-panel/40 shrink-0 flex-wrap">
        <button
          type="button"
          onClick={handleOpen}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-primary text-white hover:bg-primary-dark transition-colors"
        >
          <Upload size={13} />
          Buka File (.docx / .rtf / .txt)
        </button>

        <button
          type="button"
          onClick={handleSave}
          disabled={!isReady || isSaving}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-border bg-surface hover:bg-bg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Save size={13} />
          {isSaving ? "Menyimpan..." : "Simpan sebagai .docx"}
        </button>

        <WordZoomControl
          zoomLevel={zoomLevel}
          onZoomChange={applyZoom}
          className="ml-2"
        />

        {fileName && (
          <span className="flex items-center gap-1.5 text-xs text-text-muted">
            <FileText size={13} />
            {fileName}
          </span>
        )}
      </div>

      <div
        ref={containerRef}
        className="w-full bg-white relative"
        style={{ height: "840px", minHeight: "840px", width: "100%" }}
      />

      {feedbackPercent !== null && (
        <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-50 pointer-events-none bg-neutral-900/90 text-white text-xs px-3.5 py-1.5 rounded-full font-medium shadow-lg flex items-center gap-1.5 animate-in fade-in duration-150">
          <span>Zoom: {feedbackPercent}%</span>
        </div>
      )}
    </div>
  );
}
