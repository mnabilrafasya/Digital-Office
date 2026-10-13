"use client";

// ─────────────────────────────────────────────
// components/surat/unit-picker.tsx
// Picker tujuan surat / tembusan (unit, bukan orang)
// Static mock dulu
// ─────────────────────────────────────────────

import { useState } from "react";
import { Search, X, Plus, Building2 } from "lucide-react";
import { ORG_UNITS } from "@/lib/mock-data";
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
  /** Unit yang tidak boleh dipilih (contoh: unit yang sudah ada di Tujuan tidak bisa ada di Tembusan) */
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

  const selectedIds = new Set(value.map((u) => u.id));

  const filtered = ORG_UNITS.filter(
    (u) =>
      !selectedIds.has(u.id) &&
      !excludeIds.includes(u.id) &&
      (query === "" || u.name.toLowerCase().includes(query.toLowerCase())),
  );

  function addUnit(unitId: string) {
    const unit = ORG_UNITS.find((u) => u.id === unitId);
    if (!unit) return;
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
            {filtered.length === 0 ? (
              <li className="px-4 py-6 text-center text-text-muted text-sm">
                Tidak ada unit yang tersedia.
              </li>
            ) : (
              filtered.map((unit) => (
                <li key={unit.id}>
                  <button
                    type="button"
                    onClick={() => addUnit(unit.id)}
                    className="w-full text-left px-4 py-2.5 hover:bg-panel transition-colors flex items-center gap-2"
                  >
                    <Building2 size={13} className="text-text-muted shrink-0" />
                    <span className="text-sm text-text">{unit.name}</span>
                  </button>
                </li>
              ))
            )}
          </ul>

          <div className="px-3 py-2 border-t border-border flex justify-end">
            <button
              type="button"
              onClick={() => setOpen(false)}
              className="text-xs text-text-muted hover:text-text"
            >
              Tutup
            </button>
          </div>
        </div>
      )}

      {/* Error */}
      {error && (
        <p className="text-status-danger-text text-xs mt-1">{error}</p>
      )}
    </div>
  );
}
