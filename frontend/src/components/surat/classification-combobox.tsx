"use client";

// ─────────────────────────────────────────────
// components/surat/classification-combobox.tsx
// Combobox klasifikasi arsip (cari berdasarkan kode atau nama)
// ─────────────────────────────────────────────

import { useEffect, useRef, useState } from "react";
import { Search, X } from "lucide-react";
import { CLASSIFICATIONS } from "@/lib/mock-data";
import { cn } from "@/lib/utils";

interface Props {
  value: string | undefined;
  onChange: (id: string) => void;
  hasError?: boolean;
}

export function ClassificationCombobox({ value, onChange, hasError }: Props) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  const selected = CLASSIFICATIONS.find((c) => c.id === value);
  const q = query.toLowerCase();
  const filtered = CLASSIFICATIONS.filter(
    (c) => q === "" || c.code.toLowerCase().includes(q) || c.name.toLowerCase().includes(q),
  );

  // Tutup dropdown saat klik di luar
  useEffect(() => {
    function onDown(e: MouseEvent) {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, []);

  function pick(id: string) {
    onChange(id);
    setOpen(false);
    setQuery("");
  }

  return (
    <div ref={rootRef} className="relative">
      <div
        className={cn(
          "w-full flex items-center gap-2 px-3 py-2.5 text-sm border rounded-lg bg-surface cursor-pointer transition-colors",
          open ? "border-primary ring-2 ring-primary/15" : hasError ? "border-status-danger-text" : "border-border-input",
        )}
        onClick={() => setOpen((v) => !v)}
      >
        <Search size={14} className="text-text-muted shrink-0" />
        {selected ? (
          <span className="flex-1 text-text">
            <span className="font-mono text-text-muted text-xs mr-2">{selected.code}</span>
            {selected.name}
          </span>
        ) : (
          <span className="flex-1 text-text-muted">Cari berdasarkan kode atau nama klasifikasi...</span>
        )}
        {selected && (
          <button
            type="button"
            aria-label="Hapus klasifikasi"
            onClick={(e) => {
              e.stopPropagation();
              onChange("");
              setQuery("");
            }}
            className="text-text-muted hover:text-text"
          >
            <X size={13} />
          </button>
        )}
      </div>

      {open && (
        <div className="absolute top-full left-0 right-0 mt-1 z-20 bg-surface border border-border rounded-lg shadow-lg overflow-hidden">
          <div className="px-3 py-2 border-b border-border">
            <input
              id="classificationSearch"
              autoFocus
              type="text"
              placeholder="Ketik kode atau nama..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="w-full text-sm outline-none bg-transparent text-text placeholder-text-muted"
            />
          </div>
          <ul className="max-h-52 overflow-y-auto divide-y divide-border">
            {filtered.length === 0 ? (
              <li className="px-4 py-6 text-center text-text-muted text-sm">Tidak ditemukan.</li>
            ) : (
              filtered.map((c) => (
                <li key={c.id}>
                  <button
                    type="button"
                    onClick={() => pick(c.id)}
                    className={cn(
                      "w-full text-left px-4 py-2.5 hover:bg-panel transition-colors",
                      value === c.id && "bg-primary/5",
                    )}
                  >
                    <span className="font-mono text-text-muted text-xs mr-2">{c.code}</span>
                    <span className="text-sm text-text">{c.name}</span>
                  </button>
                </li>
              ))
            )}
          </ul>
        </div>
      )}
    </div>
  );
}
