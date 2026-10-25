"use client";

// ─────────────────────────────────────────────
// app/dev/editor/rich-edit-wrapper.tsx
// Wrapper DevExpress RichEdit yang sesungguhnya.
// Diimpor HANYA via dynamic(..., { ssr: false }) — jangan diimpor langsung.
// ─────────────────────────────────────────────
// Cara mendapatkan kunci lisensi evaluasi DevExpress:
//   1. Daftar/masuk ke https://www.devexpress.com/
//   2. Pergi ke DevExpress Download Manager
//   3. Salin License Key
//   4. Buat file frontend/.env.local dan tambahkan:
//        NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY=<kunci_anda>
//   File .env.local sudah di-.gitignore — jangan commit kunci lisensi.
//   Tanpa kunci: editor tetap berjalan, muncul watermark "Unlicensed".
// ─────────────────────────────────────────────

import { useRef, useEffect, useCallback, useState } from "react";
// Impor tipe dari devexpress-richedit (akan diisi setelah install)
import type {
  RichEdit,
  Options,
} from "devexpress-richedit";
import { Upload, Save, FileText, Info } from "lucide-react";

// CSS DevExpress — wajib dimuat agar editor tampil benar
import "devextreme/dist/css/dx.light.css";
import "devexpress-richedit/dist/dx.richedit.css";

// ─── Tipe minimal yang kita butuhkan ───────────────────────────────
// devexpress-richedit diekspor sebagai ESM; TypeScript butuh cast
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type RichEditModule = typeof import("devexpress-richedit") & Record<string, any>;

// ─── Konstanta ──────────────────────────────────────────────────────
const LICENSE_KEY = process.env.NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY ?? "";

// ─── Komponen ───────────────────────────────────────────────────────
export default function RichEditWrapper() {
  const containerRef = useRef<HTMLDivElement>(null);
  const instanceRef = useRef<RichEdit | null>(null);
  const moduleRef = useRef<RichEditModule | null>(null);

  const [isReady, setIsReady]     = useState(false);
  const [fileName, setFileName]   = useState<string | null>(null);
  const [isSaving, setIsSaving]   = useState(false);
  const [errorMsg, setErrorMsg]   = useState<string | null>(null);

  // ── Inisialisasi editor ──────────────────────────────────────────
  useEffect(() => {
    if (!containerRef.current) return;

    let disposed = false;

    (async () => {
      try {
        // Impor modul secara dinamis (sudah dalam konteks browser)
        const mod = await import(
          // eslint-disable-next-line @typescript-eslint/ban-ts-comment
          // @ts-ignore — devexpress-richedit mungkin belum punya tipe lengkap di Next 16
          "devexpress-richedit"
        ) as RichEditModule;

        moduleRef.current = mod;

        // Daftarkan lisensi bila ada
        if (LICENSE_KEY && mod.license) {
          mod.license(LICENSE_KEY);
        }

        // Buat opsi default
        const options: Options = mod.createOptions();
        options.width  = "100%";
        options.height = "100%";

        // Aktifkan toolbar lengkap
        options.ribbon.visible = true;

        // Aktifkan save via API: format DOCX
        options.exportUrl = ""; // kosong = simpan via JS, bukan HTTP POST

        if (disposed) return;

        const editor = mod.create(containerRef.current!, options);
        instanceRef.current = editor;
        setIsReady(true);
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

  // ── Buka file DOCX dari komputer ────────────────────────────────
  const handleOpen = useCallback(() => {
    const input = document.createElement("input");
    input.type   = "file";
    input.accept = ".docx,.rtf,.txt";
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (!file || !instanceRef.current) return;

      try {
        const buffer = await file.arrayBuffer();
        const uint8  = new Uint8Array(buffer);

        // Tentukan format berdasarkan ekstensi
        const mod = moduleRef.current!;
        const ext = file.name.split(".").pop()?.toLowerCase();
        const DocumentFormat = mod.DocumentFormat;
        const fmt =
          ext === "rtf" ? DocumentFormat.Rtf :
          ext === "txt" ? DocumentFormat.PlainText :
          DocumentFormat.OpenXml; // DOCX default

        instanceRef.current.openDocument(uint8, file.name, fmt);
        setFileName(file.name);
      } catch (err) {
        console.error("[DevEditor] Gagal membuka file:", err);
        alert("Gagal membuka file: " + (err instanceof Error ? err.message : String(err)));
      }
    };
    input.click();
  }, []);

  // ── Simpan / unduh sebagai DOCX ─────────────────────────────────
  const handleSave = useCallback(async () => {
    if (!instanceRef.current || !moduleRef.current) return;
    setIsSaving(true);
    try {
      const mod = moduleRef.current;
      const DocumentFormat = mod.DocumentFormat;

      // Ekspor sebagai DOCX (ArrayBuffer)
      const buffer: ArrayBuffer = await new Promise((resolve, reject) => {
        instanceRef.current!.exportToFile(
          DocumentFormat.OpenXml,
          (blob: Blob) => {
            blob.arrayBuffer().then(resolve).catch(reject);
          }
        );
      });

      // Trigger unduhan
      const blob = new Blob([buffer], {
        type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
      });
      const url  = URL.createObjectURL(blob);
      const a    = document.createElement("a");
      a.href     = url;
      a.download = fileName ?? "dokumen.docx";
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error("[DevEditor] Gagal menyimpan:", err);
      alert("Gagal menyimpan: " + (err instanceof Error ? err.message : String(err)));
    } finally {
      setIsSaving(false);
    }
  }, [fileName]);

  // ── Render ──────────────────────────────────────────────────────
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
        <div className="bg-panel border border-border rounded-lg px-4 py-3 text-left text-xs font-mono text-text-muted max-w-md w-full">
          <p className="font-semibold text-text mb-1">Pastikan sudah install:</p>
          <p>npm install devextreme devexpress-richedit</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar aksi file */}
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

        {fileName && (
          <span className="flex items-center gap-1.5 text-xs text-text-muted">
            <FileText size={13} />
            {fileName}
          </span>
        )}

        {!LICENSE_KEY && (
          <span className="ml-auto text-xs text-amber-600 bg-amber-50 border border-amber-200 px-2 py-1 rounded">
            ⚠ Tanpa lisensi — tambahkan{" "}
            <code className="font-mono">NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY</code>{" "}
            di <code className="font-mono">.env.local</code>
          </span>
        )}
      </div>

      {/* Container editor */}
      <div
        ref={containerRef}
        className="flex-1"
        style={{ minHeight: 0 }} // penting agar flex-1 tidak overflow
      />
    </div>
  );
}
