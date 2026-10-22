"use client";

// ─────────────────────────────────────────────
// components/daftar-surat/letters-table.tsx
// Tabel utama daftar surat: 9 kolom sesuai mockup
// ─────────────────────────────────────────────

import Link from "next/link";
import { AlertTriangle, FilePen, Eye, FileEdit } from "lucide-react";
import type { LetterListItem } from "@/lib/types";
import { StatusBadge } from "./status-badge";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import { id as idLocale } from "date-fns/locale";

// ─── Helper: Tahap Saat Ini ──────────────────

function getCurrentStageLabel(item: LetterListItem): string {
  if (item.status === "DRAFT")      return "Drafter (Anda)";
  if (item.status === "SENT")       return "—";
  if (item.status === "CANCELLED")  return "—";
  if (!item.currentAssignee)        return "—";

  const a = item.currentAssignee;
  const roleLabel = a.role === "REVIEWER" ? `Reviewer ${a.order}` : "Approver";
  return `${roleLabel} · ${a.name}`;
}

// ─── Helper: Tombol Tindakan ─────────────────

interface ActionButtonProps {
  item: LetterListItem;
}

function ActionButton({ item }: ActionButtonProps) {
  const hasEdit = item.allowedActions.includes("EDIT");
  const isReturned =
    item.status === "RETURNED_TO_DRAFTER" ||
    item.status === "RETURNED_TO_REVIEWER";

  if (item.status === "DRAFT" && hasEdit) {
    return (
      <Link
        href={`/surat/baru?draft=${item.id}`}
        className="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
      >
        <FileEdit size={13} />
        Lanjutkan
      </Link>
    );
  }

  if (isReturned && hasEdit) {
    return (
      <Link
        href={`/surat/${item.id}`}
        title={item.lastReturnNote ?? undefined}
        className="inline-flex items-center gap-1 text-xs font-semibold text-status-danger-text hover:underline"
      >
        <FilePen size={13} />
        Revisi
      </Link>
    );
  }

  const isReview = item.status === "IN_REVIEW";
  const isApproval = item.status === "IN_APPROVAL";

  if (isReview || isApproval) {
    return (
      <Link
        href={`/surat/${item.id}`}
        className="inline-flex items-center gap-1 text-xs font-semibold text-primary hover:underline bg-primary/10 hover:bg-primary/20 px-2 py-1 rounded transition-colors"
      >
        <Eye size={13} />
        {isReview ? "Tinjau" : "Setujui"}
      </Link>
    );
  }

  return (
    <Link
      href={`/surat/${item.id}`}
      className="inline-flex items-center gap-1 text-xs font-medium text-text-muted hover:text-text hover:underline"
    >
      <Eye size={13} />
      Lihat
    </Link>
  );
}

// ─── Baris skeleton ──────────────────────────

function SkeletonRow() {
  return (
    <tr>
      {Array.from({ length: 9 }).map((_, i) => (
        <td key={i} className="px-4 py-3">
          <div className="h-3 bg-panel rounded animate-pulse w-full max-w-30" />
        </td>
      ))}
    </tr>
  );
}

// ─── Empty state ─────────────────────────────

function EmptyState({ isFiltered }: { isFiltered: boolean }) {
  return (
    <tr>
      <td colSpan={9}>
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <div className="w-16 h-16 rounded-2xl bg-panel flex items-center justify-center mb-4">
            <AlertTriangle size={28} className="text-text-muted" />
          </div>
          {isFiltered ? (
            <>
              <p className="text-sm font-medium text-text mb-1">Tidak ada surat yang sesuai filter</p>
              <p className="text-xs text-text-muted">Coba ubah kata kunci atau hapus beberapa filter</p>
            </>
          ) : (
            <>
              <p className="text-sm font-medium text-text mb-1">Belum ada surat</p>
              <p className="text-xs text-text-muted mb-4">Mulai buat surat pertama Anda</p>
              <Link
                href="/surat/baru"
                className="inline-flex items-center gap-1.5 text-xs font-semibold text-white bg-primary hover:bg-primary/90 px-4 py-2 rounded-lg transition-colors"
              >
                + Buat Surat Baru
              </Link>
            </>
          )}
        </div>
      </td>
    </tr>
  );
}

// ─── Error state ─────────────────────────────

function ErrorRow({ onRetry }: { onRetry: () => void }) {
  return (
    <tr>
      <td colSpan={9}>
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <p className="text-sm font-medium text-status-danger-text mb-2">Gagal memuat data surat</p>
          <button
            onClick={onRetry}
            className="text-xs text-primary hover:underline"
          >
            Coba lagi
          </button>
        </div>
      </td>
    </tr>
  );
}

// ─── Komponen utama ──────────────────────────

const HEADERS = [
  "Nomor Surat",
  "Judul / Perihal",
  "Pengirim",
  "Tujuan",
  "Jenis & Sifat",
  "Tanggal Dibuat",
  "Status",
  "Tahap Saat Ini",
  "Tindakan",
];

interface Props {
  items: LetterListItem[];
  loading: boolean;
  error: boolean;
  isFiltered: boolean;
  onRetry: () => void;
}

export function LettersTable({ items, loading, error, isFiltered, onRetry }: Props) {
  return (
    <div className="bg-surface rounded-xl border border-border overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-sm min-w-225">
          <thead>
            <tr className="border-b border-border bg-panel/50">
              {HEADERS.map((h) => (
                <th
                  key={h}
                  className="px-4 py-3 text-left text-xs font-semibold text-text-muted uppercase tracking-wide whitespace-nowrap"
                >
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {loading && Array.from({ length: 8 }).map((_, i) => <SkeletonRow key={i} />)}

            {!loading && error && <ErrorRow onRetry={onRetry} />}

            {!loading && !error && items.length === 0 && (
              <EmptyState isFiltered={isFiltered} />
            )}

            {!loading && !error && items.map((item) => {
              const isReturned =
                item.status === "RETURNED_TO_DRAFTER" ||
                item.status === "RETURNED_TO_REVIEWER";

              return (
                <tr
                  key={item.id}
                  className={cn(
                    "group hover:bg-bg transition-colors",
                    isReturned && "bg-status-danger-bg/30"
                  )}
                >
                  {/* Nomor Surat */}
                  <td className="px-4 py-3 whitespace-nowrap font-mono text-xs text-text-muted">
                    {item.letterNumber ?? (
                      <span className="text-text-muted/60 italic">Dibuat otomatis</span>
                    )}
                  </td>

                  {/* Judul */}
                  <td className="px-4 py-3 max-w-55">
                    <div className="flex items-start gap-1.5">
                      {isReturned && item.lastReturnNote && (
                        <span
                          title={`Catatan: ${item.lastReturnNote}`}
                          className="shrink-0 mt-0.5"
                        >
                          <AlertTriangle size={12} className="text-status-danger-text" />
                        </span>
                      )}
                      <span className="line-clamp-2 text-text font-medium leading-snug">
                        {item.title ?? (
                          <span className="text-text-muted/70 italic font-normal">(Tanpa perihal)</span>
                        )}
                      </span>
                    </div>
                  </td>

                  {/* Pengirim */}
                  <td className="px-4 py-3 text-text-muted text-xs max-w-35">
                    <span className="line-clamp-2">{item.senderUnit.name}</span>
                  </td>

                  {/* Tujuan */}
                  <td className="px-4 py-3 text-text-muted text-xs whitespace-nowrap">
                    {item.recipientsSummary.first ? (
                      <>
                        <span>{item.recipientsSummary.first}</span>
                        {item.recipientsSummary.total > 1 && (
                          <span className="ml-1 text-primary font-medium">
                            +{item.recipientsSummary.total - 1} lainnya
                          </span>
                        )}
                      </>
                    ) : (
                      <span className="text-text-muted/60 italic">—</span>
                    )}
                  </td>

                  {/* Jenis & Sifat */}
                  <td className="px-4 py-3">
                    <div className="flex flex-col gap-1">
                      <span className="text-xs text-text font-medium leading-tight">
                        {item.letterType?.name ?? "—"}
                      </span>
                      <span className={cn(
                        "inline-self text-[10px] font-semibold px-1.5 py-0.5 rounded w-fit",
                        item.confidentiality?.code === "RAHASIA"
                          ? "bg-status-danger-bg text-status-danger-text"
                          : item.confidentiality?.code === "TERBATAS"
                            ? "bg-status-warn-bg text-status-warn-text"
                            : "bg-status-neutral-bg text-status-neutral-text"
                      )}>
                        {item.confidentiality?.name ?? "Biasa"}
                      </span>
                    </div>
                  </td>

                  {/* Tanggal Dibuat */}
                  <td className="px-4 py-3 text-xs text-text-muted whitespace-nowrap">
                    {format(new Date(item.createdAt), "dd MMM yyyy", { locale: idLocale })}
                  </td>

                  {/* Status */}
                  <td className="px-4 py-3">
                    <StatusBadge status={item.status} />
                  </td>

                  {/* Tahap Saat Ini */}
                  <td className="px-4 py-3 text-xs text-text-muted max-w-40">
                    <span className="line-clamp-2">{getCurrentStageLabel(item)}</span>
                  </td>

                  {/* Tindakan */}
                  <td className="px-4 py-3 whitespace-nowrap">
                    <ActionButton item={item} />
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
