"use client";

// ─────────────────────────────────────────────
// components/daftar-surat/pagination.tsx
// Footer pagination: info jumlah + kontrol halaman + pilihan per halaman
// ─────────────────────────────────────────────

import { ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface Props {
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
}

const PAGE_SIZES = [10, 20, 50];

function getPageNumbers(current: number, total: number): (number | "…")[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
  const pages: (number | "…")[] = [];
  if (current <= 4) {
    pages.push(1, 2, 3, 4, 5, "…", total);
  } else if (current >= total - 3) {
    pages.push(1, "…", total - 4, total - 3, total - 2, total - 1, total);
  } else {
    pages.push(1, "…", current - 1, current, current + 1, "…", total);
  }
  return pages;
}

export function Pagination({ page, pageSize, total, onPageChange, onPageSizeChange }: Props) {
  const totalPages = Math.ceil(total / pageSize);
  const start = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const end   = Math.min(page * pageSize, total);

  const pageNumbers = getPageNumbers(page, totalPages);

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 mt-4 text-sm">
      {/* Kiri: info + per halaman */}
      <div className="flex items-center gap-3 text-text-muted">
        <span>
          Menampilkan{" "}
          <span className="font-medium text-text">{start} – {end}</span>{" "}
          dari{" "}
          <span className="font-medium text-text">{total}</span> surat
        </span>

        <div className="flex items-center gap-1.5">
          <span className="text-xs">Tampilkan:</span>
          <select
            id="pagination-page-size"
            value={pageSize}
            onChange={(e) => {
              onPageSizeChange(Number(e.target.value));
              onPageChange(1);
            }}
            className="h-8 px-2 text-xs rounded-lg border border-border-input bg-surface text-text focus:outline-none focus:ring-2 focus:ring-primary/30 cursor-pointer"
          >
            {PAGE_SIZES.map((s) => (
              <option key={s} value={s}>{s} per halaman</option>
            ))}
          </select>
        </div>
      </div>

      {/* Kanan: navigasi halaman */}
      {totalPages > 1 && (
        <div className="flex items-center gap-1">
          {/* Sebelumnya */}
          <button
            id="btn-page-prev"
            onClick={() => onPageChange(page - 1)}
            disabled={page <= 1}
            className="inline-flex items-center gap-1 h-8 px-3 rounded-lg text-xs font-medium border border-border hover:bg-panel disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            <ChevronLeft size={13} /> Sebelumnya
          </button>

          {/* Angka halaman */}
          {pageNumbers.map((p, i) =>
            p === "…" ? (
              <span
                key={`ellipsis-${i}`}
                className="h-8 w-8 flex items-center justify-center text-text-muted text-xs"
              >
                …
              </span>
            ) : (
              <button
                key={p}
                id={`btn-page-${p}`}
                onClick={() => onPageChange(p as number)}
                className={cn(
                  "h-8 w-8 rounded-lg text-xs font-medium transition-colors",
                  p === page
                    ? "bg-primary text-white"
                    : "border border-border hover:bg-panel text-text"
                )}
              >
                {p}
              </button>
            )
          )}

          {/* Selanjutnya */}
          <button
            id="btn-page-next"
            onClick={() => onPageChange(page + 1)}
            disabled={page >= totalPages}
            className="inline-flex items-center gap-1 h-8 px-3 rounded-lg text-xs font-medium border border-border hover:bg-panel disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            Selanjutnya <ChevronRight size={13} />
          </button>
        </div>
      )}
    </div>
  );
}
