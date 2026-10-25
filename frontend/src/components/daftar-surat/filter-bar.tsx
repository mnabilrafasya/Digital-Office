"use client";

// Baris filter: hanya pencarian (q). Jenis/sifat/tanggal disembunyikan sampai backend mendukung.

import { Search, SlidersHorizontal, RotateCcw } from "lucide-react";
import { useRef, useEffect } from "react";

export interface FilterValues {
  q: string;
  letterTypeId: string;
  confidentialityId: string;
  from: string;
  to: string;
}

interface Props {
  values: FilterValues;
  onChange: (values: FilterValues) => void;
  onApply: () => void;
  onReset: () => void;
}

export function FilterBar({ values, onChange, onApply, onReset }: Props) {
  const searchRef = useRef<HTMLInputElement>(null);

  // Ctrl+K shortcut
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key === "k") {
        e.preventDefault();
        searchRef.current?.focus();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  function set<K extends keyof FilterValues>(key: K, val: FilterValues[K]) {
    onChange({ ...values, [key]: val });
  }

  return (
    <div className="flex flex-wrap items-center gap-2 mb-4">
      <div className="relative flex-1 min-w-48">
        <Search
          size={14}
          className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none"
        />
        <input
          ref={searchRef}
          id="filter-search"
          type="text"
          value={values.q}
          onChange={(e) => set("q", e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") onApply();
          }}
          placeholder="Cari nomor, judul, atau jenis surat…"
          className="w-full h-9 pl-9 pr-16 text-sm rounded-lg border border-border-input bg-surface text-text placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-colors"
        />
        <span className="absolute right-3 top-1/2 -translate-y-1/2 text-[10px] text-text-muted bg-panel px-1.5 py-0.5 rounded font-mono leading-tight select-none">
          Ctrl+K
        </span>
      </div>

      <button
        id="btn-terapkan-filter"
        onClick={onApply}
        className="inline-flex items-center gap-1.5 h-9 px-4 rounded-lg text-sm font-semibold bg-primary text-white hover:bg-primary/90 transition-colors"
      >
        <SlidersHorizontal size={13} />
        Cari
      </button>

      <button
        id="btn-reset-filter"
        onClick={onReset}
        className="inline-flex items-center gap-1.5 h-9 px-3 rounded-lg text-sm text-text-muted border border-border hover:bg-panel hover:text-text transition-colors"
      >
        <RotateCcw size={12} />
        Reset
      </button>
    </div>
  );
}
