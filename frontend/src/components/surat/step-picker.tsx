"use client";

// ─────────────────────────────────────────────
// components/surat/step-picker.tsx
// Picker reviewer / approver — kandidat dari backend
// GET /api/v1/users/candidates?type=REVIEWER|APPROVER&q=...
// ─────────────────────────────────────────────

import { useState, useEffect, useRef } from "react";
import { Search, X, Plus, GripVertical, AlertTriangle, Loader2 } from "lucide-react";
import { fetchCandidates, type CandidateItem } from "@/lib/api/form";
import type { StepEntry } from "@/lib/types";
import { cn, generateId } from "@/lib/utils";

interface StepPickerProps {
  role: "REVIEWER" | "APPROVER";
  value: StepEntry[];
  onChange: (entries: StepEntry[]) => void;
  error?: string;
  /** ID pegawai yang sudah dipilih di role lain (warning duplikat lintas role) */
  otherRoleIds?: string[];
}

export function StepPicker({
  role,
  value,
  onChange,
  error,
  otherRoleIds = [],
}: StepPickerProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [candidates, setCandidates] = useState<CandidateItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const roleLabel = role === "REVIEWER" ? "Reviewer" : "Approver";

  // Fetch kandidat dari backend, dipicu saat panel terbuka atau query berubah
  useEffect(() => {
    if (!open) return;

    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      setLoading(true);
      setFetchError(null);
      fetchCandidates({ type: role, q: query || undefined, limit: 30 })
        .then((data) => setCandidates(data))
        .catch((e: unknown) => {
          setFetchError(e instanceof Error ? e.message : "Gagal memuat kandidat.");
        })
        .finally(() => setLoading(false));
    }, 300); // debounce 300ms

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [open, query, role]);

  const selectedIds = new Set(value.map((e) => e.employeeId));

  // Filter lokal: sembunyikan yang sudah dipilih
  const filtered = candidates.filter((c) => !selectedIds.has(c.id));

  function addCandidate(candidate: CandidateItem) {
    const entry: StepEntry = {
      id: generateId(),
      employeeId: candidate.id,
      employeeName: candidate.name,
      jabatan: candidate.positionName,
      unitName: candidate.unitName,
    };
    onChange([...value, entry]);
    setQuery("");
    setOpen(false);
  }

  function removeEntry(entryId: string) {
    onChange(value.filter((e) => e.id !== entryId));
  }

  // Warning: orang yang sama ada di kedua role
  const duplicateIds = value
    .filter((e) => otherRoleIds.includes(e.employeeId))
    .map((e) => e.employeeId);

  return (
    <div className="space-y-2">
      {/* Daftar tahap yang sudah dipilih */}
      {value.length > 0 && (
        <div className="space-y-2">
          {value.map((entry, idx) => {
            const isDuplicate = duplicateIds.includes(entry.employeeId);
            return (
              <div
                key={entry.id}
                className={cn(
                  "flex items-center gap-2 px-3 py-2 rounded-lg border text-sm",
                  isDuplicate
                    ? "border-status-warn-text/40 bg-status-warn-bg"
                    : "border-border bg-surface",
                )}
              >
                {/* Grip (visual only) */}
                <GripVertical size={14} className="text-text-muted shrink-0 cursor-grab" />

                {/* Chip tahap */}
                <span className="inline-flex items-center px-2 py-0.5 rounded-full bg-status-info-bg text-status-info-text text-[11px] font-semibold shrink-0">
                  Tahap {idx + 1}
                </span>

                {/* Info */}
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-text truncate">{entry.employeeName}</div>
                  <div className="text-text-muted text-xs truncate">
                    {entry.jabatan} &bull; {entry.unitName}
                  </div>
                </div>

                {/* Warning duplikat */}
                {isDuplicate && (
                  <div
                    title={`Orang yang sama juga ada sebagai ${role === "REVIEWER" ? "Approver" : "Reviewer"}`}
                    className="text-status-warn-text shrink-0"
                  >
                    <AlertTriangle size={14} />
                  </div>
                )}

                {/* Hapus */}
                <button
                  type="button"
                  onClick={() => removeEntry(entry.id)}
                  className="text-text-muted hover:text-status-danger-text transition-colors shrink-0"
                  aria-label={`Hapus ${entry.employeeName} dari daftar ${roleLabel}`}
                >
                  <X size={14} />
                </button>
              </div>
            );
          })}
        </div>
      )}

      {/* Tombol tambah */}
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={cn(
          "flex items-center gap-1.5 text-sm text-primary hover:text-primary-dark",
          "border border-dashed border-primary/40 hover:border-primary",
          "px-3 py-2 rounded-lg w-full justify-center transition-colors",
        )}
      >
        <Plus size={14} />
        Tambah {roleLabel}
      </button>

      {/* Panel pencarian */}
      {open && (
        <div className="border border-border rounded-lg shadow-md bg-surface overflow-hidden">
          {/* Search input */}
          <div className="flex items-center gap-2 px-3 py-2 border-b border-border">
            <Search size={14} className="text-text-muted shrink-0" />
            <input
              autoFocus
              type="text"
              placeholder="Cari nama atau jabatan..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="flex-1 text-sm outline-none bg-transparent text-text placeholder-text-muted"
            />
            {query && (
              <button
                type="button"
                onClick={() => setQuery("")}
                className="text-text-muted hover:text-text"
              >
                <X size={13} />
              </button>
            )}
          </div>

          {/* Hasil */}
          <ul className="max-h-52 overflow-y-auto divide-y divide-border">
            {loading ? (
              <li className="px-4 py-6 flex items-center justify-center gap-2 text-sm text-text-muted">
                <Loader2 size={14} className="animate-spin" />
                Mencari kandidat...
              </li>
            ) : fetchError ? (
              <li className="px-4 py-6 text-center text-status-danger-text text-sm">
                {fetchError}
              </li>
            ) : filtered.length === 0 ? (
              <li className="px-4 py-6 text-center text-text-muted text-sm">
                {query ? "Tidak ditemukan." : `Tidak ada kandidat ${roleLabel} tersedia.`}
              </li>
            ) : (
              filtered.map((c) => (
                <li key={c.id}>
                  <button
                    type="button"
                    onClick={() => addCandidate(c)}
                    className="w-full text-left px-4 py-2.5 hover:bg-panel transition-colors"
                  >
                    <div className="text-sm font-medium text-text">{c.name}</div>
                    <div className="text-xs text-text-muted">
                      {c.positionName} &bull; {c.unitName}
                    </div>
                  </button>
                </li>
              ))
            )}
          </ul>

          {/* Close */}
          <div className="px-3 py-2 border-t border-border flex justify-end">
            <button
              type="button"
              onClick={() => { setOpen(false); setQuery(""); }}
              className="text-xs text-text-muted hover:text-text"
            >
              Tutup
            </button>
          </div>
        </div>
      )}

      {/* Error validasi */}
      {error && (
        <p className="text-status-danger-text text-xs mt-1">{error}</p>
      )}
    </div>
  );
}
