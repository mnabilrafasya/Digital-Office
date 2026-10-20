"use client";

// ─────────────────────────────────────────────
// components/surat/classification-combobox.tsx
// Combobox klasifikasi arsip — data dari backend
// (GET /api/v1/lookups?category=CLASSIFICATION)
// ─────────────────────────────────────────────

import { useEffect, useRef, useState } from "react";
import { Search, X, Loader2 } from "lucide-react";
import { fetchLookups, type LookupItem } from "@/lib/api/form";
import { cn } from "@/lib/utils";

interface Props {
  value: string | undefined;
  onChange: (id: string) => void;
  hasError?: boolean;
}

export function ClassificationCombobox({ value, onChange, hasError }: Props) {
  const [items, setItems] = useState<LookupItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  // Fetch saat mount
  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setFetchError(null);
    fetchLookups("CLASSIFICATION")
      .then((data) => { if (!cancelled) setItems(data); })
      .catch((e: unknown) => {
        if (!cancelled)
          setFetchError(e instanceof Error ? e.message : "Gagal memuat klasifikasi.");
      })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, []);

  const selected = items.find((c) => c.id === value);
  const q = query.toLowerCase();
  const filtered = items.filter(
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

  // Loading state
  if (loading) {
    return (
      <div className="flex items-center gap-2 px-3 py-2.5 border border-border-input rounded-lg bg-surface text-sm text-text-muted">
        <Loader2 size={14} className="animate-spin shrink-0" />
        Memuat klasifikasi arsip...
      </div>
    );
  }

  // Error state
  if (fetchError) {
    return (
      <div className="flex items-center justify-between gap-2 px-3 py-2 border border-status-danger-text/40 rounded-lg bg-status-danger-bg text-xs text-status-danger-text">
        <span>{fetchError}</span>
        <button
          type="button"
          onClick={() => { setFetchError(null); setLoading(true); }}
          className="underline shrink-0"
        >
          Coba lagi
        </button>
      </div>
    );
  }

  return (
    <div ref={rootRef} className="relative">
      <div
        className={cn(
          "w-full flex items-center gap-2 px-3 py-2.5 text-sm border rounded-lg bg-surface cursor-pointer transition-colors",
          open
            ? "border-primary ring-2 ring-primary/15"
            : hasError
              ? "border-status-danger-text"
              : "border-border-input",
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
