"use client";

// ─────────────────────────────────────────────
// components/surat/step-picker.tsx
// Picker reviewer ('R') & approver ('A') — kandidat dari backend
// - Tampilan user/icon dibuat menyamping (seperti unit-picker)
// - Dibatasi maksimal 10 orang
// - Membaca "Kode Alias" / role: 'R' (Reviewer) & 'A' (Approver)
// ─────────────────────────────────────────────

import { useState, useEffect, useRef } from "react";
import { Search, X, Plus, User, CheckCircle2, AlertTriangle, Loader2 } from "lucide-react";
import { fetchCandidates, type CandidateItem } from "@/lib/api/form";
import type { StepEntry } from "@/lib/types";
import { cn, generateId } from "@/lib/utils";

export const MAX_STEP_LIMIT = 10;

interface StepPickerProps {
  /** Kode Alias ('R' / 'A') atau role ('REVIEWER' / 'APPROVER') dari Backend */
  kodeAlias?: "R" | "A" | string;
  role?: "REVIEWER" | "APPROVER" | "R" | "A" | string;
  value: StepEntry[];
  onChange: (entries: StepEntry[]) => void;
  error?: string;
  /** ID pegawai yang sudah dipilih di role lain (warning duplikat lintas role) */
  otherRoleIds?: string[];
}

export function StepPicker({
  kodeAlias,
  role = "REVIEWER",
  value = [],
  onChange,
  error,
  otherRoleIds = [],
}: StepPickerProps) {
  // FE membuat kondisi: Jika dapat 'R' tampilkan view Reviewer, jika dapat 'A' tampilkan view Approver
  const rawCode = (kodeAlias || role || "R").toUpperCase();
  const isApprover = rawCode === "A" || rawCode === "APPROVER";
  const roleLabel = isApprover ? "Approver" : "Reviewer";
  const roleType = isApprover ? "APPROVER" : "REVIEWER";

  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [candidates, setCandidates] = useState<CandidateItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const isLimitReached = value.length >= MAX_STEP_LIMIT;

  useEffect(() => {
    if (!open) return;

    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      setLoading(true);
      setFetchError(null);
      fetchCandidates({ type: roleType, q: query || undefined, limit: 30 })
        .then((data) => setCandidates(data))
        .catch((e: unknown) => {
          setFetchError(e instanceof Error ? e.message : "Gagal memuat kandidat.");
        })
        .finally(() => setLoading(false));
    }, 300);

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [open, query, roleType]);

  const selectedIds = new Set(value.map((e) => e.employeeId));
  const filtered = candidates.filter((c) => !selectedIds.has(c.id));

  function addCandidate(candidate: CandidateItem) {
    if (value.length >= MAX_STEP_LIMIT) return;

    const entry: StepEntry = {
      id: generateId(),
      employeeId: candidate.id,
      employeeName: candidate.name,
      jabatan: candidate.positionName,
      unitName: candidate.unitName,
      kodeAlias: isApprover ? "A" : "R",
    };
    onChange([...value, entry]);
    setQuery("");
    setOpen(false);
  }

  function removeEntry(entryId: string) {
    onChange(value.filter((e) => e.id !== entryId));
  }

  const duplicateIds = value
    .filter((e) => otherRoleIds.includes(e.employeeId))
    .map((e) => e.employeeId);

  return (
    <div className="space-y-2">
      {/* 1. Chip menyamping (flex-wrap gap-2) mirip unit-picker */}
      {value.length > 0 && (
        <div className="flex flex-wrap gap-2 items-center">
          {value.map((entry, idx) => {
            const isDuplicate = duplicateIds.includes(entry.employeeId);
            return (
              <span
                key={entry.id}
                className={cn(
                  "inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium transition-colors",
                  isDuplicate
                    ? "bg-status-warn-bg text-status-warn-text border border-status-warn-text/40"
                    : isApprover
                      ? "bg-emerald-50 text-emerald-800 border border-emerald-200"
                      : "bg-primary/10 text-primary border border-primary/20"
                )}
                title={`${entry.employeeName} • ${entry.jabatan} • ${entry.unitName}`}
              >
                {/* User / Approver Icon & Kode Tag */}
                {isApprover ? (
                  <CheckCircle2 size={12} className="text-emerald-600 shrink-0" />
                ) : (
                  <User size={12} className="text-primary shrink-0" />
                )}

                <span
                  className={cn(
                    "px-1.5 py-0.2 rounded-full text-[10px] font-bold leading-none shrink-0",
                    isApprover ? "bg-emerald-200 text-emerald-900" : "bg-primary/20 text-primary"
                  )}
                >
                  {isApprover ? `A${idx + 1}` : `R${idx + 1}`}
                </span>

                <span className="truncate max-w-40 sm:max-w-50 font-medium">
                  {entry.employeeName}
                </span>

                {isDuplicate && (
                  <span title={`Juga dipilih di role lain`} className="text-status-warn-text shrink-0">
                    <AlertTriangle size={11} />
                  </span>
                )}

                <button
                  type="button"
                  onClick={() => removeEntry(entry.id)}
                  className="hover:text-status-danger-text transition-colors ml-0.5"
                  aria-label={`Hapus ${entry.employeeName}`}
                >
                  <X size={12} />
                </button>
              </span>
            );
          })}
        </div>
      )}

      {/* 2. Tombol tambah & limitasi 10 orang */}
      {isLimitReached ? (
        <div className="flex items-center justify-between px-3 py-1.5 rounded-lg border border-dashed border-border bg-panel/40 text-xs text-text-muted">
          <span>Maksimal 10 {roleLabel} telah dipilih</span>
          <span className="font-semibold text-text">{value.length}/{MAX_STEP_LIMIT}</span>
        </div>
      ) : (
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className={cn(
            "flex items-center gap-1.5 text-sm px-3 py-2 rounded-lg w-full justify-center transition-colors border border-dashed",
            isApprover
              ? "text-emerald-700 border-emerald-300 hover:border-emerald-500 hover:bg-emerald-50/50"
              : "text-primary border-primary/40 hover:border-primary hover:bg-primary/5"
          )}
        >
          <Plus size={14} />
          Tambah {roleLabel}
          <span className="text-xs text-text-muted ml-0.5">({value.length}/{MAX_STEP_LIMIT})</span>
        </button>
      )}

      {/* 3. Panel pencarian kandidat */}
      {open && !isLimitReached && (
        <div className="border border-border rounded-lg shadow-md bg-surface overflow-hidden">
          <div className="flex items-center gap-2 px-3 py-2 border-b border-border">
            <Search size={14} className="text-text-muted shrink-0" />
            <input
              autoFocus
              type="text"
              placeholder={`Cari nama atau jabatan ${roleLabel.toLowerCase()}...`}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="flex-1 text-sm outline-none bg-transparent text-text placeholder-text-muted"
            />
            {query && (
              <button type="button" onClick={() => setQuery("")} className="text-text-muted hover:text-text">
                <X size={13} />
              </button>
            )}
          </div>

          <ul className="max-h-52 overflow-y-auto divide-y divide-border">
            {loading ? (
              <li className="px-4 py-6 flex items-center justify-center gap-2 text-sm text-text-muted">
                <Loader2 size={14} className="animate-spin" />
                Mencari kandidat...
              </li>
            ) : fetchError ? (
              <li className="px-4 py-6 text-center text-status-danger-text text-sm">{fetchError}</li>
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
                    className="w-full text-left px-4 py-2.5 hover:bg-panel transition-colors flex items-center gap-2"
                  >
                    {isApprover ? (
                      <CheckCircle2 size={13} className="text-emerald-600 shrink-0" />
                    ) : (
                      <User size={13} className="text-primary shrink-0" />
                    )}
                    <div className="min-w-0 flex-1">
                      <div className="text-sm font-medium text-text truncate">{c.name}</div>
                      <div className="text-xs text-text-muted truncate">
                        {c.positionName} &bull; {c.unitName}
                      </div>
                    </div>
                  </button>
                </li>
              ))
            )}
          </ul>

          <div className="px-3 py-2 border-t border-border flex justify-between items-center bg-panel/20">
            <span className="text-[11px] text-text-muted">Maks. {MAX_STEP_LIMIT} orang</span>
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

      {error && <p className="text-status-danger-text text-xs mt-1">{error}</p>}
    </div>
  );
}

// Alias VPPView untuk kemudahan penggunaan
export { StepPicker as VPPView };
