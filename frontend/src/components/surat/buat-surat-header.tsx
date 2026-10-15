"use client";

// ─────────────────────────────────────────────
// components/surat/buat-surat-header.tsx
// Breadcrumb, judul, indikator simpan otomatis, tombol Muat Draf Terakhir
// ─────────────────────────────────────────────

import { ChevronRight, Clock, RotateCcw } from "lucide-react";
import { formatTimeWIB } from "@/lib/utils";

interface Props {
  isSaving: boolean;
  isDirty: boolean;
  saveFailed: boolean;
  lastSaved: Date | null;
  loadingDraft: boolean;
  onLoadDraft: () => void;
}

function SaveIndicator({ isSaving, isDirty, saveFailed, lastSaved }: Omit<Props, "loadingDraft" | "onLoadDraft">) {
  if (isSaving) {
    return (
      <span className="flex items-center gap-1.5 text-status-warn-text">
        <Clock size={12} className="animate-pulse" /> Menyimpan...
      </span>
    );
  }
  if (saveFailed) return <span className="text-status-danger-text">Gagal menyimpan otomatis</span>;
  if (isDirty) {
    return (
      <span className="flex items-center gap-1.5 text-status-warn-text">
        <span className="w-2 h-2 rounded-full bg-status-warn-text animate-pulse" /> Belum tersimpan
      </span>
    );
  }
  if (lastSaved) {
    return (
      <span className="flex items-center gap-1.5">
        <span className="w-2 h-2 rounded-full bg-green-500" /> Tersimpan Otomatis {formatTimeWIB(lastSaved)}
      </span>
    );
  }
  return null;
}

export function BuatSuratHeader({ loadingDraft, onLoadDraft, ...save }: Props) {
  return (
    <div className="mb-6">
      <nav className="flex items-center gap-1 text-xs text-text-muted mb-2">
        <span>Beranda</span>
        <ChevronRight size={12} />
        <span>Penciptaan Surat</span>
        <ChevronRight size={12} />
        <span className="text-text font-medium">Buat Surat Baru</span>
      </nav>

      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-text">Buat Surat Baru</h1>
          <p className="text-text-muted text-sm mt-0.5">
            Isi detail surat, pilih reviewer &amp; approver, lalu lampirkan berkas pendukung.
          </p>
        </div>

        <div className="flex items-center gap-3 shrink-0 mt-1">
          <div className="text-xs text-text-muted" aria-live="polite">
            <SaveIndicator {...save} />
          </div>
          <button
            type="button"
            onClick={onLoadDraft}
            disabled={loadingDraft}
            className="flex items-center gap-1.5 text-xs text-text-muted border border-border px-3 py-1.5 rounded-lg hover:bg-panel hover:text-text transition-colors disabled:opacity-60"
          >
            <RotateCcw size={12} />
            {loadingDraft ? "Memuat..." : "Muat Draf Terakhir"}
          </button>
        </div>
      </div>
    </div>
  );
}
