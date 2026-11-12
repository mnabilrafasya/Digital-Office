"use client";

// ─────────────────────────────────────────────
// components/review/audit-trail-modal.tsx
// Modal interaktif Visualisasi Diagram Alur & Timeline Riwayat Aksi (Audit Trail) Lengkap
// ─────────────────────────────────────────────

import { useState } from "react";
import {
  X,
  GitCommitHorizontal,
  CheckCircle2,
  RotateCcw,
  Send,
  FileText,
  Eye,
  XCircle,
  Clock,
  ArrowRight,
  ChevronDown,
  ChevronUp,
  Search,
  UploadCloud,
  Inbox,
} from "lucide-react";
import type {
  LetterDetail,
  LetterHistoryEntry,
} from "@/lib/api/letter-detail";
import {
  buildGroupedHistory,
  formatFullDateTime,
  type HistoryFilterMode,
  type GroupedHistoryItem,
} from "./audit-trail-utils";
import { LetterFlowDiagram } from "./letter-flow-diagram";
import { cn } from "@/lib/utils";

interface AuditTrailModalProps {
  letter: LetterDetail;
  history: LetterHistoryEntry[];
  onClose: () => void;
}

function actionIcon(action: string) {
  switch (action) {
    case "APPROVE":
    case "SIGN":
      return <CheckCircle2 size={13} className="text-status-success-text" />;
    case "UPLOAD_SIGNED":
      return <UploadCloud size={13} className="text-status-success-text" />;
    case "RECEIVE":
    case "INCOMING_RECEIVE":
      return <Inbox size={12} className="text-status-info-text" />;
    case "RETURN":
      return <RotateCcw size={13} className="text-status-warn-text" />;
    case "SUBMIT":
      return <Send size={12} className="text-primary" />;
    case "SAVE":
      return <FileText size={12} className="text-text-muted" />;
    case "READ":
      return <Eye size={12} className="text-text-muted" />;
    case "CANCEL":
      return <XCircle size={13} className="text-status-danger-text" />;
    default:
      return <Clock size={12} className="text-primary" />;
  }
}

function TimelineRow({ item }: { item: GroupedHistoryItem }) {
  const [expanded, setExpanded] = useState(false);
  const { entry, meta, repeatCount, versionNumbers, groupedEntries } = item;

  const badgeClass =
    meta.tone === "success"
      ? "bg-status-success-bg text-status-success-text border-status-success-text/20"
      : meta.tone === "warn"
        ? "bg-status-warn-bg text-status-warn-text border-status-warn-text/20"
        : meta.tone === "danger"
          ? "bg-status-danger-bg text-status-danger-text border-status-danger-text/20"
          : meta.tone === "info"
            ? "bg-primary/10 text-primary border-primary/20"
            : "bg-panel text-text-muted border-border";

  return (
    <li className="relative pl-6 pb-3 last:pb-0">
      {/* Garis vertikal timeline */}
      <span className="absolute left-2.5 top-5 bottom-0 w-px bg-border last:hidden" />
      {/* Titik ikon */}
      <span className="absolute left-0 top-0.5 w-5 h-5 rounded-full bg-surface border border-border flex items-center justify-center shadow-2xs">
        {actionIcon(entry.action)}
      </span>

      <div className="rounded-lg border border-border bg-panel/25 px-3 py-2 text-xs">
        <div className="flex items-center justify-between gap-2 flex-wrap">
          <div className="flex items-center gap-1.5 flex-wrap">
            <span
              className={cn(
                "text-[10px] font-semibold px-2 py-0.5 rounded-full border",
                badgeClass
              )}
            >
              {meta.label}
            </span>
            {repeatCount > 1 && (
              <button
                type="button"
                onClick={() => setExpanded((v) => !v)}
                className="inline-flex items-center gap-0.5 text-[10px] font-semibold px-1.5 py-0.5 rounded bg-surface border border-border text-text-muted hover:text-text"
              >
                {repeatCount}× kejadian
                {expanded ? <ChevronUp size={10} /> : <ChevronDown size={10} />}
              </button>
            )}
            {versionNumbers.map((v) => (
              <span
                key={v}
                className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-surface border border-border text-primary font-semibold"
              >
                v{v}
              </span>
            ))}
          </div>
          <span className="text-[11px] text-text-muted">
            {formatFullDateTime(entry.createdAt)}
          </span>
        </div>

        <div className="mt-1 flex items-center gap-1 text-xs text-text flex-wrap">
          <span className="font-semibold">{entry.actor.name}</span>
          {entry.actor.positionName && (
            <span className="text-text-muted">
              ({entry.actor.positionName})
            </span>
          )}
          {entry.target && (
            <span className="inline-flex items-center gap-1 text-text-muted">
              <ArrowRight size={11} className="text-primary" />
              <strong className="text-text">{entry.target.name}</strong>
            </span>
          )}
        </div>

        {entry.note && (
          <p className="mt-1.5 px-2.5 py-1.5 rounded bg-surface border border-border text-text italic">
            &ldquo;{entry.note}&rdquo;
          </p>
        )}

        {expanded && repeatCount > 1 && (
          <ul className="mt-2 pt-2 border-t border-border/60 space-y-1 text-[11px] text-text-muted">
            {groupedEntries.map((g, i) => (
              <li key={i} className="flex items-center justify-between">
                <span>
                  {meta.shortLabel}
                  {g.versionNumber != null ? ` (v${g.versionNumber})` : ""}
                </span>
                <span>{formatFullDateTime(g.createdAt)}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </li>
  );
}

export function AuditTrailModal({
  letter,
  history,
  onClose,
}: AuditTrailModalProps) {
  const [filter, setFilter] = useState<HistoryFilterMode>("milestone");
  const [query, setQuery] = useState("");

  const grouped = buildGroupedHistory(history, filter).filter((item) => {
    if (!query.trim()) return true;
    const q = query.toLowerCase();
    return (
      item.entry.actor.name.toLowerCase().includes(q) ||
      item.meta.label.toLowerCase().includes(q) ||
      (item.entry.note ?? "").toLowerCase().includes(q) ||
      (item.entry.target?.name ?? "").toLowerCase().includes(q)
    );
  });

  return (
    <div
      className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="bg-surface border border-border rounded-2xl shadow-2xl w-full max-w-4xl max-h-[88vh] flex flex-col overflow-hidden">
        {/* Header Modal */}
        <div className="px-5 py-3.5 border-b border-border bg-panel/40 flex items-center justify-between gap-3 shrink-0">
          <div className="flex items-center gap-2 min-w-0">
            <div className="w-8 h-8 rounded-lg bg-primary/10 text-primary flex items-center justify-center shrink-0">
              <GitCommitHorizontal size={17} />
            </div>
            <div className="min-w-0">
              <h2 className="text-sm font-bold text-text truncate">
                Visualisasi Alur & Jejak Persetujuan (Audit Trail)
              </h2>
              <p className="text-xs text-text-muted truncate">
                {letter.letterNumber
                  ? `${letter.letterNumber} · `
                  : ""}
                {letter.title ?? "(Tanpa perihal)"}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="w-8 h-8 rounded-lg flex items-center justify-center text-text-muted hover:text-text hover:bg-panel transition-colors"
            title="Tutup"
          >
            <X size={17} />
          </button>
        </div>

        {/* Isi Modal Scrollable */}
        <div className="p-5 overflow-y-auto space-y-5 flex-1">
          {/* Bagian 1: Diagram Visual Perjalanan Surat */}
          <div className="rounded-xl border border-border bg-panel/20 p-4 space-y-2">
            <h3 className="text-xs font-bold uppercase tracking-wide text-text-muted">
              Diagram Perjalanan Surat (Hulu ke Hilir)
            </h3>
            <LetterFlowDiagram letter={letter} history={history} />
          </div>

          {/* Bagian 2: Timeline Riwayat Aksi Terkelompok */}
          <div className="space-y-3">
            <div className="flex items-center justify-between gap-2 flex-wrap">
              <div className="inline-flex rounded-lg border border-border bg-panel p-0.5 gap-0.5">
                {(
                  [
                    { id: "milestone", label: "Alur Utama" },
                    { id: "all", label: `Semua (${history.length})` },
                    { id: "notes", label: "Catatan Revisi" },
                  ] as const
                ).map((t) => (
                  <button
                    key={t.id}
                    type="button"
                    onClick={() => setFilter(t.id)}
                    className={cn(
                      "px-2.5 py-1 rounded-md text-xs font-semibold transition-colors",
                      filter === t.id
                        ? "bg-surface text-primary shadow-2xs"
                        : "text-text-muted hover:text-text"
                    )}
                  >
                    {t.label}
                  </button>
                ))}
              </div>

              <div className="relative min-w-52">
                <Search
                  size={13}
                  className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted"
                />
                <input
                  type="text"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="Cari nama, aksi, atau catatan..."
                  className="w-full pl-8 pr-3 py-1 rounded-lg border border-border bg-surface text-xs text-text outline-none focus:border-primary"
                />
              </div>
            </div>

            {grouped.length === 0 ? (
              <div className="py-8 text-center text-xs text-text-muted border border-dashed border-border rounded-xl">
                Tidak ada kejadian riwayat yang sesuai dengan filter.
              </div>
            ) : (
              <ol className="pt-1">
                {grouped.map((item) => (
                  <TimelineRow key={item.id} item={item} />
                ))}
              </ol>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

