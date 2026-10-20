"use client";

// ─────────────────────────────────────────────
// components/surat/unit-picker.tsx
// Picker tujuan surat / tembusan — unit dari backend
// GET /api/v1/units?q=... (dengan debounce)
// ─────────────────────────────────────────────

import { useState, useEffect, useRef } from "react";
import { Search, X, Plus, Building2, Loader2 } from "lucide-react";
import { fetchUnits, type UnitItem } from "@/lib/api/form";
import { cn } from "@/lib/utils";

interface UnitEntry {
  id: string;
  name: string;
}

interface UnitPickerProps {
  label: string;
  value: UnitEntry[];
  onChange: (entries: UnitEntry[]) => void;
  error?: string;
  /** Unit yang tidak boleh dipilih (unit yang sudah ada di Tujuan tidak bisa di Tembusan) */
  excludeIds?: string[];
}

export function UnitPicker({
  label,
  value,
  onChange,
  error,
  excludeIds = [],
}: UnitPickerProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [units, setUnits] = useState<UnitItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const selectedIds = new Set(value.map((u) => u.id));

  // Fetch dari backend saat panel terbuka atau query berubah
  useEffect(() => {
    if (!open) return;

    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      setLoading(true);
      setFetchError(null);
      fetchUnits(query || undefined)
        .then((data) => setUnits(data))
        .catch((e: unknown) => {
          setFetchError(e instanceof Error ? e.message : "Gagal memuat unit.");
        })
        .finally(() => setLoading(false));
    }, 300); // debounce 300ms

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [open, query]);

  // Filter lokal: sembunyikan yang sudah dipilih dan yang di-exclude
  const filtered = units.filter(
    (u) => !selectedIds.has(u.id) && !excludeIds.includes(u.id),
  );

  function addUnit(unit: UnitItem) {
    onChange([...value, { id: unit.id, name: unit.name }]);
    setQuery("");
    setOpen(false);
  }

  function removeUnit(unitId: string) {
    onChange(value.filter((u) => u.id !== unitId));
  }

  return (
    <div className="space-y-2">
      {/* Chip unit yang dipilih */}
      {value.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {value.map((unit) => (
            <span
              key={unit.id}
              className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-primary/10 text-primary text-xs font-medium"
            >
              <Building2 size={11} />
              {unit.name}
              <button
                type="button"
                onClick={() => removeUnit(unit.id)}
                className="text-primary/60 hover:text-status-danger-text transition-colors ml-0.5"
                aria-label={`Hapus ${unit.name}`}
              >
                <X size={11} />
              </button>
            </span>
          ))}
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
        {label}
      </button>

      {/* Panel pencarian */}
      {open && (
        <div className="border border-border rounded-lg shadow-md bg-surface overflow-hidden">
          <div className="flex items-center gap-2 px-3 py-2 border-b border-border">
            <Search size={14} className="text-text-muted shrink-0" />
            <input
              autoFocus
              type="text"
              placeholder="Cari nama unit..."
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

          <ul className="max-h-52 overflow-y-auto divide-y divide-border">
            {loading ? (
              <li className="px-4 py-6 flex items-center justify-center gap-2 text-sm text-text-muted">
                <Loader2 size={14} className="animate-spin" />
                Mencari unit...
              </li>
            ) : fetchError ? (
              <li className="px-4 py-6 text-center text-status-danger-text text-sm">
                {fetchError}
              </li>
            ) : filtered.length === 0 ? (
              <li className="px-4 py-6 text-center text-text-muted text-sm">
                {query ? "Unit tidak ditemukan." : "Tidak ada unit tersedia."}
              </li>
            ) : (
              filtered.map((unit) => (
                <li key={unit.id}>
                  <button
                    type="button"
                    onClick={() => addUnit(unit)}
                    className="w-full text-left px-4 py-2.5 hover:bg-panel transition-colors flex items-start gap-2"
                  >
                    <Building2 size={13} className="text-text-muted shrink-0 mt-0.5" />
                    <div>
                      <div className="text-sm text-text">{unit.name}</div>
                      {unit.headPositionName && (
                        <div className="text-xs text-text-muted">{unit.headPositionName}</div>
                      )}
                    </div>
                  </button>
                </li>
              ))
            )}
          </ul>

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
