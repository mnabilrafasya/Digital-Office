"use client";

// ─────────────────────────────────────────────
// components/surat/buat-surat-header.tsx
// Breadcrumb, judul
// ─────────────────────────────────────────────

import Link from "next/link";
import { ChevronRight, Clock } from "lucide-react";
import { formatTimeWIB } from "@/lib/utils";

interface Props {
  isSaving: boolean;
  isDirty: boolean;
  saveFailed: boolean;
  lastSaved: Date | null;
  isIncomingExternal?: boolean;
}

function SaveIndicator({
  isSaving,
  isDirty,
  saveFailed,
  lastSaved,
}: Omit<Props, "isIncomingExternal">) {
  if (isSaving) {
    return (
      <span className="flex items-center gap-1.5 text-status-warn-text">
        <Clock size={12} className="animate-pulse" /> Menyimpan...
      </span>
    );
  }
  if (saveFailed)
    return (
      <span className="text-status-danger-text">Gagal menyimpan draf</span>
    );
  if (isDirty) {
    return (
      <span className="flex items-center gap-1.5 text-status-warn-text">
        <span className="w-2 h-2 rounded-full bg-status-warn-text animate-pulse" />{" "}
        Belum tersimpan
      </span>
    );
  }
  if (lastSaved) {
    return (
      <span className="flex items-center gap-1.5">
        <span className="w-2 h-2 rounded-full bg-green-500" /> Draf tersimpan{" "}
        {formatTimeWIB(lastSaved)}
      </span>
    );
  }
  return null;
}

export function BuatSuratHeader({
  isIncomingExternal = false,
  ...save
}: Props) {
  return (
    <div className="mb-6">
      <nav className="flex items-center gap-1 text-xs text-text-muted mb-2">
        <Link href="/surat/daftar" className="hover:text-text transition-colors">
          Beranda
        </Link>
        <ChevronRight size={12} />
        <span>Penciptaan Surat</span>
        <ChevronRight size={12} />
        <span className="text-text font-medium">
          {isIncomingExternal ? "Catat Surat Masuk Eksternal" : "Buat Surat Baru"}
        </span>
      </nav>

      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-text">
            {isIncomingExternal ? "Catat Surat Masuk Eksternal" : "Buat Surat Baru"}
          </h1>
          <p className="text-text-muted text-sm mt-0.5">
            {isIncomingExternal
              ? "Catat metadata surat dari instansi luar, pilih tujuan disposisi, dan unggah scan PDF."
              : "Isi detail surat, pilih reviewer & approver, lalu lampirkan berkas pendukung."}
          </p>
        </div>

        {!isIncomingExternal && (
          <div className="flex items-center gap-3 shrink-0 mt-1">
            <div className="text-xs text-text-muted" aria-live="polite">
              <SaveIndicator {...save} />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
