"use client";

// ─────────────────────────────────────────────
// components/surat/unit-picker.tsx
// Picker tujuan surat / tembusan — unit & alias level dari backend
// GET /api/v1/units?q=... & GET /api/v1/recipients/aliases
// ─────────────────────────────────────────────

import { useState, useEffect, useRef } from "react";
import { Search, X, Plus, Building2, Users, Loader2 } from "lucide-react";
import {
  fetchUnits,
  fetchRecipientAliases,
  type UnitItem,
  type RecipientAliasItem,
} from "@/lib/api/form";
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
  /** Unit/alias yang tidak boleh dipilih (yang sudah ada di Tujuan tidak bisa di Tembusan) */
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
  const [pickerTab, setPickerTab] = useState<"unit" | "alias">("unit");
  const [query, setQuery] = useState("");
  const [units, setUnits] = useState<UnitItem[]>([]);
  const [aliases, setAliases] = useState<RecipientAliasItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const selectedIds = new Set(value.map((u) => u.id));

  // Fetch alias level sekali saat panel pertama kali dibuka
  useEffect(() => {
    if (!open || aliases.length > 0) return;
    fetchRecipientAliases()
      .then((data) => setAliases(data))
      .catch(() => {});
  }, [open, aliases.length]);

  // Fetch unit dari backend saat panel terbuka atau query berubah
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
    }, 300);

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [open, query]);

  const normalizedQuery = query.trim().toLowerCase();

  const filteredAliases = aliases.filter((a) => {
    const aliasId = `alias-${a.levelId}`;
    if (selectedIds.has(aliasId) || excludeIds.includes(aliasId)) return false;
    if (!normalizedQuery) return true;
    return (
      a.label.toLowerCase().includes(normalizedQuery) ||
      a.code.toLowerCase().includes(normalizedQuery)
    );
  });

  // Filter lokal: sembunyikan unit yang sudah dipilih dan yang di-exclude
  const filteredUnits = units.filter(
    (u) => !selectedIds.has(u.id) && !excludeIds.includes(u.id)
  );

  function addEntry(entry: UnitEntry) {
    onChange([...value, entry]);
    setQuery("");
    setOpen(false);
  }

  function removeUnit(unitId: string) {
    onChange(value.filter((u) => u.id !== unitId));
  }

  return (
    <div className="space-y-2">
      {/* Chip unit / alias yang dipilih */}
      {value.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {value.map((unit) => {
            const isAlias = unit.id.startsWith("alias-");
            return (
              <span
                key={unit.id}
                className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-primary/10 text-primary text-xs font-medium"
              >
                {isAlias ? <Users size={11} /> : <Building2 size={11} />}
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
          "px-3 py-2 rounded-lg w-full justify-center transition-colors"
        )}
      >
        <Plus size={14} />
        {label}
      </button>

      {/* Panel pencarian bertab (Unit / Departemen & Alias Jabatan) */}
      {open && (
        <div className="border border-border rounded-xl shadow-md bg-surface overflow-hidden">
          <div className="flex items-center justify-between border-b border-border bg-panel/40 px-3">
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => {
                  setPickerTab("unit");
                  setQuery("");
                }}
                className={cn(
                  "px-3 py-2 text-xs font-semibold border-b-2 -mb-px transition-colors flex items-center gap-1.5",
                  pickerTab === "unit"
                    ? "border-primary text-primary"
                    : "border-transparent text-text-muted hover:text-text"
                )}
              >
                <Building2 size={13} /> Unit / Departemen
              </button>
              <button
                type="button"
                onClick={() => {
                  setPickerTab("alias");
                  setQuery("");
                }}
                className={cn(
                  "px-3 py-2 text-xs font-semibold border-b-2 -mb-px transition-colors flex items-center gap-1.5",
                  pickerTab === "alias"
                    ? "border-primary text-primary"
                    : "border-transparent text-text-muted hover:text-text"
                )}
              >
                <Users size={13} /> Alias Jabatan ({filteredAliases.length})
              </button>
            </div>
            <button
              type="button"
              onClick={() => {
                setOpen(false);
                setQuery("");
              }}
              className="text-xs text-text-muted hover:text-text py-1.5"
            >
              Tutup
            </button>
          </div>

          <div className="p-2.5 border-b border-border">
            <div className="relative">
              <Search
                size={14}
                className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none"
              />
              <input
                autoFocus
                type="text"
                placeholder={
                  pickerTab === "unit"
                    ? "Cari nama unit / departemen tujuan..."
                    : "Cari alias jabatan (contoh: Seluruh VP)..."
                }
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                className="w-full pl-8 pr-8 py-1.5 text-sm border border-border rounded-lg bg-surface outline-none focus:border-primary"
              />
              {query && (
                <button
                  type="button"
                  onClick={() => setQuery("")}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-muted hover:text-text"
                >
                  <X size={13} />
                </button>
              )}
            </div>
          </div>

          <ul className="max-h-52 overflow-y-auto divide-y divide-border">
            {pickerTab === "unit" ? (
              loading ? (
                <li className="px-4 py-6 flex items-center justify-center gap-2 text-xs text-text-muted">
                  <Loader2 size={14} className="animate-spin" />
                  Mencari unit...
                </li>
              ) : fetchError ? (
                <li className="px-4 py-6 text-center text-status-danger-text text-xs">
                  {fetchError}
                </li>
              ) : filteredUnits.length === 0 ? (
                <li className="px-4 py-6 text-center text-text-muted text-xs">
                  {query ? "Unit tidak ditemukan." : "Tidak ada unit tersedia."}
                </li>
              ) : (
                filteredUnits.map((unit) => (
                  <li key={unit.id}>
                    <button
                      type="button"
                      onClick={() => addEntry({ id: unit.id, name: unit.name })}
                      className="w-full text-left px-3.5 py-2.5 hover:bg-panel transition-colors flex items-start gap-2.5 text-xs"
                    >
                      <Building2 size={14} className="text-primary shrink-0 mt-0.5" />
                      <div className="min-w-0">
                        <div className="font-medium text-text">{unit.name}</div>
                        <div className="text-[11px] text-text-muted">
                          {unit.headPositionName
                            ? `Pimpinan: ${unit.headPositionName}`
                            : unit.unitType}
                          {unit.code ? ` · Kode: ${unit.code}` : ""}
                        </div>
                      </div>
                    </button>
                  </li>
                ))
              )
            ) : filteredAliases.length === 0 ? (
              <li className="px-4 py-6 text-center text-text-muted text-xs">
                {query ? "Alias jabatan tidak ditemukan." : "Tidak ada alias jabatan tersedia."}
              </li>
            ) : (
              filteredAliases.map((alias) => (
                <li key={`alias-${alias.levelId}`}>
                  <button
                    type="button"
                    onClick={() =>
                      addEntry({
                        id: `alias-${alias.levelId}`,
                        name: alias.label,
                      })
                    }
                    className="w-full text-left px-3.5 py-2.5 hover:bg-panel transition-colors flex items-start gap-2.5 text-xs"
                  >
                    <Users size={14} className="text-primary shrink-0 mt-0.5" />
                    <div className="min-w-0">
                      <div className="font-medium text-text">{alias.label}</div>
                      <div className="text-[11px] text-text-muted">
                        Kode: {alias.code} &bull; {alias.memberCount} pimpinan unit aktif
                      </div>
                    </div>
                  </button>
                </li>
              ))
            )}
          </ul>
        </div>
      )}

      {/* Error validasi */}
      {error && <p className="text-status-danger-text text-xs mt-1">{error}</p>}
    </div>
  );
}
