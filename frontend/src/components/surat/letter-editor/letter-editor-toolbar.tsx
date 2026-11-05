"use client";

// components/surat/letter-editor/letter-editor-toolbar.tsx
// Command bar terpadu gaya Microsoft Word / Office 365 di bagian atas editor naskah:
// - Grup 1: Pilihan Template Naskah (.docx dari /api/v1/templates) & Sinkronkan Data Form
// - Grup 2: Operasi Berkas terintegrasi (Buka .docx, Unduh .docx, Cetak / PDF)
// - Grup 3: Chip interaktif salin penanda `###NoSurat###` & indikator versi dokumen (.docx)
// - Grup 4: Kontrol Zoom ringkas, Panduan Pintasan Keyboard, & Layar Penuh

import { useState, useRef, useEffect } from "react";
import {
  Upload,
  Download,
  FileText,
  Printer,
  RefreshCw,
  Maximize2,
  Minimize2,
  CheckCircle2,
  Keyboard,
  CloudUpload,
  Copy,
  Check,
} from "lucide-react";

import { WordZoomControl } from "@/components/ui/word-zoom-control";
import {
  fetchTemplates,
  getCachedTemplates,
  type LetterTemplateDefinition,
} from "@/lib/templates/letter-templates";

export interface LetterEditorToolbarProps {
  appliedTemplateId: string;
  onSelectTemplate: (templateId: string) => void;
  onSyncFormData: () => void;
  zoomLevel: number;
  onZoomChange: (level: number) => void;
  statusNotification: string | null;
  isReady: boolean;
  isSaving: boolean;
  isFullscreen: boolean;
  onToggleFullscreen: () => void;
  onOpenDocx: () => void;
  onDownloadDocx: () => void;
  onPrintPdf: () => void;
  serverVersionNumber?: number | null;
  hasUnsavedDocumentChanges?: boolean;
}

const WORD_SHORTCUTS = [
  { keys: "Ctrl + B / I / U", desc: "Tebal / Miring / Garis Bawah" },
  { keys: "Ctrl + Z / Y", desc: "Urungkan (Undo) / Ulangi (Redo)" },
  { keys: "Ctrl + L / E / R / J", desc: "Rata Kiri / Tengah / Kanan / Kiri-Kanan" },
  { keys: "Ctrl + F / H", desc: "Cari Teks / Cari & Ganti" },
  { keys: "Ctrl + + / - / 0", desc: "Perbesar / Perkecil / Reset Zoom 100%" },
  { keys: "Ctrl + Scroll Mouse", desc: "Zoom lembar kerja secara halus" },
  { keys: "Ctrl + S", desc: "Simpan Draf & Naskah (.docx)" },
];

export function LetterEditorToolbar({
  appliedTemplateId,
  onSelectTemplate,
  onSyncFormData,
  zoomLevel,
  onZoomChange,
  statusNotification,
  isReady,
  isSaving,
  isFullscreen,
  onToggleFullscreen,
  onOpenDocx,
  onDownloadDocx,
  onPrintPdf,
  serverVersionNumber,
  hasUnsavedDocumentChanges,
}: LetterEditorToolbarProps) {
  const [showShortcuts, setShowShortcuts] = useState(false);
  const [copiedPlaceholder, setCopiedPlaceholder] = useState(false);
  const [templates, setTemplates] = useState<LetterTemplateDefinition[]>(() =>
    getCachedTemplates()
  );
  const shortcutRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    fetchTemplates().then((items) => {
      if (!cancelled) {
        setTemplates(items);
      }
    });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!showShortcuts) return;
    const handleOutside = (e: MouseEvent) => {
      if (
        shortcutRef.current &&
        !shortcutRef.current.contains(e.target as Node)
      ) {
        setShowShortcuts(false);
      }
    };
    document.addEventListener("mousedown", handleOutside);
    return () => document.removeEventListener("mousedown", handleOutside);
  }, [showShortcuts]);

  const handleCopyPlaceholder = () => {
    navigator.clipboard?.writeText("###NoSurat###");
    setCopiedPlaceholder(true);
    setTimeout(() => setCopiedPlaceholder(false), 2000);
  };

  const resolvedSelectedId =
    templates.find(
      (t) => t.id === appliedTemplateId || t.legacyId === appliedTemplateId
    )?.id ?? appliedTemplateId;

  return (
    <div className="flex items-center justify-between gap-1.5 min-[1600px]:gap-2 px-2.5 min-[1600px]:px-3.5 py-2 bg-panel border-b border-border flex-wrap xl:flex-nowrap shrink-0">
      {/* Kiri: Grup Template + Grup Aksi Berkas */}
      <div className="flex items-center gap-1.5 min-[1600px]:gap-2 flex-wrap xl:flex-nowrap shrink-0">
        {/* Kapsul Template & Sinkronisasi Form */}
        <div className="inline-flex items-center rounded-lg border border-border bg-surface p-0.5 shadow-2xs shrink-0">
          <div className="flex items-center gap-1.5 pl-2 min-[1600px]:pl-2.5 pr-1 text-xs font-semibold text-text-muted">
            <FileText size={13} className="text-primary shrink-0" />
            <span className="hidden min-[1600px]:inline">Template:</span>
          </div>
          <select
            value={resolvedSelectedId}
            onChange={(e) => onSelectTemplate(e.target.value)}
            aria-label="Pilih template naskah"
            className="text-xs py-1 pl-1 pr-1.5 min-[1600px]:pr-2 max-w-45 min-[1600px]:max-w-none truncate rounded-md bg-transparent text-text font-semibold focus:outline-none cursor-pointer"
          >
            {templates.map((tpl) => (
              <option key={tpl.id} value={tpl.id} className="bg-surface text-text">
                {tpl.name}
              </option>
            ))}
          </select>
          <span className="h-4 w-px bg-border mx-0.5" />
          <button
            type="button"
            onClick={onSyncFormData}
            disabled={!isReady}
            title="Isi otomatis Perihal, Penerima, dan Penandatangan dari Form Section 1 ke dalam naskah"
            className="flex items-center gap-1.5 px-2 min-[1600px]:px-2.5 py-1 rounded-md text-xs font-medium text-text hover:bg-panel transition-colors disabled:opacity-50 whitespace-nowrap"
          >
            <RefreshCw size={12} className="text-primary shrink-0" />
            <span>Sinkronkan Form</span>
          </button>
        </div>

        {/* Grup Segmented: Buka .docx | Unduh .docx | Cetak / PDF */}
        <div className="inline-flex items-center rounded-lg border border-border bg-surface p-0.5 shadow-2xs shrink-0">
          <button
            type="button"
            onClick={onOpenDocx}
            disabled={!isReady}
            className="flex items-center gap-1.5 px-2 min-[1600px]:px-2.5 py-1 rounded-md text-xs font-medium text-text hover:bg-panel transition-colors disabled:opacity-50 whitespace-nowrap"
            title="Buka dokumen Microsoft Word (.docx) dari komputer Anda"
          >
            <Upload size={13} className="text-primary shrink-0" />
            <span>Buka .docx</span>
          </button>

          <span className="h-4 w-px bg-border" />

          <button
            type="button"
            onClick={onDownloadDocx}
            disabled={!isReady || isSaving}
            className="flex items-center gap-1.5 px-2 min-[1600px]:px-2.5 py-1 rounded-md text-xs font-medium text-text hover:bg-panel transition-colors disabled:opacity-50 whitespace-nowrap"
            title="Unduh salinan naskah saat ini ke komputer sebagai file .docx"
          >
            <Download size={13} className="text-primary shrink-0" />
            <span>{isSaving ? "Mengunduh..." : "Unduh .docx"}</span>
          </button>

          <span className="h-4 w-px bg-border" />

          <button
            type="button"
            onClick={onPrintPdf}
            disabled={!isReady}
            className="flex items-center gap-1.5 px-2 min-[1600px]:px-2.5 py-1 rounded-md text-xs font-medium text-text hover:bg-panel transition-colors disabled:opacity-50 whitespace-nowrap"
            title="Cetak naskah atau simpan sebagai PDF"
          >
            <Printer size={13} className="text-primary shrink-0" />
            <span>Cetak / PDF</span>
          </button>
        </div>
      </div>

      {/* Kanan-Tengah & Kanan: Status / Penanda Nomor + Zoom & Utilitas */}
      <div className="flex items-center gap-1.5 min-[1600px]:gap-2 ml-auto flex-wrap xl:flex-nowrap shrink-0">
        {statusNotification ? (
          <div className="inline-flex items-center gap-1.5 text-xs font-medium text-status-success-text bg-status-success-bg border border-status-success-text/20 px-2.5 py-1 rounded-lg whitespace-nowrap">
            <CheckCircle2 size={13} className="shrink-0" />
            <span>{statusNotification}</span>
          </div>
        ) : (
          <div className="hidden xl:flex items-center gap-1.5 shrink-0">
            {/* Chip Interaktif ###NoSurat### */}
            <button
              type="button"
              onClick={handleCopyPlaceholder}
              title="Klik untuk menyalin ###NoSurat###. Penanda ini otomatis diganti nomor resmi oleh server saat disetujui Approver."
              className="inline-flex items-center gap-1.5 px-2 min-[1600px]:px-2.5 py-1 rounded-lg border border-border bg-surface hover:bg-bg text-[11px] text-text-muted hover:text-text transition-colors shadow-2xs whitespace-nowrap"
            >
              <span className="hidden min-[1600px]:inline">Penanda:</span>
              <code className="font-mono font-semibold text-text">
                ###NoSurat###
              </code>
              {copiedPlaceholder ? (
                <Check size={11} className="text-status-success-text shrink-0" />
              ) : (
                <Copy size={11} className="text-text-muted shrink-0" />
              )}
            </button>

            {/* Badge Status Versi Server */}
            <div
              className="hidden min-[1600px]:inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg border border-border bg-surface text-[11px] text-text-muted shadow-2xs whitespace-nowrap"
              title="Naskah .docx otomatis diunggah ke server saat klik Simpan Draft atau Kirim Surat"
            >
              <CloudUpload size={12} className="text-primary shrink-0" />
              {serverVersionNumber && serverVersionNumber > 0 ? (
                <span className="text-text font-medium">
                  Versi {serverVersionNumber} (.docx)
                  {hasUnsavedDocumentChanges ? " • Belum disimpan" : ""}
                </span>
              ) : (
                <span>Auto-sync .docx</span>
              )}
            </div>
          </div>
        )}

        {/* Kontrol Zoom Ringkas */}
        <WordZoomControl
          zoomLevel={zoomLevel}
          onZoomChange={onZoomChange}
          showPresets={false}
        />

        {/* Grup Utilitas: Pintasan Keyboard & Layar Penuh */}
        <div
          ref={shortcutRef}
          className="relative inline-flex items-center rounded-lg border border-border bg-surface p-0.5 shadow-2xs"
        >
          <button
            type="button"
            onClick={() => setShowShortcuts((v) => !v)}
            className="p-1.5 rounded-md text-text-muted hover:text-text hover:bg-panel transition-colors"
            title="Pintasan Keyboard Microsoft Word"
            aria-label="Pintasan Keyboard"
          >
            <Keyboard size={14} />
          </button>

          <span className="h-4 w-px bg-border" />

          <button
            type="button"
            onClick={onToggleFullscreen}
            className="p-1.5 rounded-md text-text-muted hover:text-text hover:bg-panel transition-colors"
            title={
              isFullscreen
                ? "Keluar Layar Penuh"
                : "Layar Penuh (Fullscreen)"
            }
            aria-label="Toggle Layar Penuh"
          >
            {isFullscreen ? <Minimize2 size={14} /> : <Maximize2 size={14} />}
          </button>

          {showShortcuts && (
            <div className="absolute right-0 top-full mt-1.5 z-50 w-72 bg-surface border border-border rounded-xl shadow-lg p-3 text-xs">
              <div className="flex items-center justify-between pb-2 mb-2 border-b border-border">
                <span className="font-semibold text-text">
                  Pintasan Keyboard Word
                </span>
                <button
                  type="button"
                  onClick={() => setShowShortcuts(false)}
                  className="text-text-muted hover:text-text text-[11px]"
                >
                  Tutup
                </button>
              </div>
              <ul className="space-y-1.5">
                {WORD_SHORTCUTS.map((s) => (
                  <li
                    key={s.keys}
                    className="flex items-center justify-between gap-2"
                  >
                    <span className="text-text-muted">{s.desc}</span>
                    <kbd className="px-1.5 py-0.5 rounded bg-panel border border-border font-mono text-[10px] text-text shrink-0">
                      {s.keys}
                    </kbd>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
