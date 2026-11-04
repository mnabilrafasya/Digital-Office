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

      {/* Panel pencarian */}
      {open && (
        <div className="border border-border rounded-lg shadow-md bg-surface overflow-hidden">
          <div className="flex items-center gap-2 px-3 py-2 border-b border-border">
            <Search size={14} className="text-text-muted shrink-0" />
            <input
              autoFocus
              type="text"
              placeholder="Cari nama unit atau alias jabatan (mis. Seluruh VP)..."
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
            ) : filteredAliases.length === 0 && filteredUnits.length === 0 ? (
              <li className="px-4 py-6 text-center text-text-muted text-sm">
                {query ? "Unit atau alias tidak ditemukan." : "Tidak ada unit tersedia."}
              </li>
            ) : (
              <>
                {filteredAliases.map((alias) => (
                  <li key={`alias-${alias.levelId}`}>
                    <button
                      type="button"
                      onClick={() =>
                        addEntry({
                          id: `alias-${alias.levelId}`,
                          name: alias.label,
                        })
                      }
                      className="w-full text-left px-4 py-2.5 hover:bg-panel transition-colors flex items-start gap-2 bg-primary/[0.02]"
                    >
                      <Users size={13} className="text-primary shrink-0 mt-0.5" />
                      <div>
                        <div className="text-sm font-medium text-text">
                          {alias.label}
                        </div>
                        <div className="text-xs text-text-muted">
                          Alias Jabatan &bull; {alias.memberCount} pimpinan unit aktif
                        </div>
                      </div>
                    </button>
                  </li>
                ))}
                {filteredUnits.map((unit) => (
                  <li key={unit.id}>
                    <button
                      type="button"
                      onClick={() => addEntry({ id: unit.id, name: unit.name })}
                      className="w-full text-left px-4 py-2.5 hover:bg-panel transition-colors flex items-start gap-2"
                    >
                      <Building2 size={13} className="text-text-muted shrink-0 mt-0.5" />
                      <div>
                        <div className="text-sm text-text">{unit.name}</div>
                        {unit.headPositionName && (
                          <div className="text-xs text-text-muted">
                            {unit.headPositionName}
                          </div>
                        )}
                      </div>
                    </button>
                  </li>
                ))}
              </>
            )}
          </ul>

          <div className="px-3 py-2 border-t border-border flex justify-end">
            <button
              type="button"
              onClick={() => {
                setOpen(false);
                setQuery("");
              }}
              className="text-xs text-text-muted hover:text-text"
            >
              Tutup
            </button>
          </div>
        </div>
      )}

      {/* Error validasi */}
      {error && <p className="text-status-danger-text text-xs mt-1">{error}</p>}
    </div>
  );
}
