"use client";

// ─────────────────────────────────────────────
// components/review/compact-audit-trail.tsx
// Komponen ringkas Alur Persetujuan & Timeline Riwayat Aksi (Audit Trail) di sidebar halaman detail surat
// ─────────────────────────────────────────────

import { useState } from "react";
import {
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
  Maximize2,
} from "lucide-react";
import type {
  LetterDetail,
  LetterHistoryEntry,
} from "@/lib/api/letter-detail";
import { LetterSteps } from "./letter-steps";
import { AuditTrailModal } from "./audit-trail-modal";
import {
  buildGroupedHistory,
  formatCompactTime,
  type HistoryFilterMode,
} from "./audit-trail-utils";
import { cn } from "@/lib/utils";

interface CompactAuditTrailProps {
  letter: LetterDetail;
  history: LetterHistoryEntry[];
}

function smallActionDot(action: string) {
  switch (action) {
    case "APPROVE":
    case "SIGN":
      return (
        <span className="w-4 h-4 rounded-full bg-status-success-bg border border-status-success-text/30 flex items-center justify-center shrink-0">
          <CheckCircle2 size={10} className="text-status-success-text" />
        </span>
      );
    case "RETURN":
      return (
        <span className="w-4 h-4 rounded-full bg-status-warn-bg border border-status-warn-text/30 flex items-center justify-center shrink-0">
          <RotateCcw size={9} className="text-status-warn-text" />
        </span>
      );
    case "SUBMIT":
      return (
        <span className="w-4 h-4 rounded-full bg-primary/10 border border-primary/30 flex items-center justify-center shrink-0">
          <Send size={9} className="text-primary" />
        </span>
      );
    case "SAVE":
      return (
        <span className="w-4 h-4 rounded-full bg-panel border border-border flex items-center justify-center shrink-0">
          <FileText size={9} className="text-text-muted" />
        </span>
      );
    case "READ":
      return (
        <span className="w-4 h-4 rounded-full bg-panel border border-border flex items-center justify-center shrink-0">
          <Eye size={9} className="text-text-muted" />
        </span>
      );
    case "CANCEL":
      return (
        <span className="w-4 h-4 rounded-full bg-status-danger-bg border border-status-danger-text/30 flex items-center justify-center shrink-0">
          <XCircle size={10} className="text-status-danger-text" />
        </span>
      );
    default:
      return (
        <span className="w-4 h-4 rounded-full bg-panel border border-border flex items-center justify-center shrink-0">
          <Clock size={9} className="text-text-muted" />
        </span>
      );
  }
}

export function CompactAuditTrail({
  letter,
  history,
}: CompactAuditTrailProps) {
  const [filterMode, setFilterMode] = useState<HistoryFilterMode>("milestone");
  const [showAllInline, setShowAllInline] = useState(false);
  const [openModal, setOpenModal] = useState(false);

  const grouped = buildGroupedHistory(history, filterMode);
  const visibleItems = showAllInline ? grouped : grouped.slice(0, 3);
  const hiddenCount = Math.max(0, grouped.length - 3);

  return (
    <>
      <div className="border border-border rounded-xl bg-surface overflow-hidden">
        {/* Header Kartu */}
        <div className="px-4 py-2.5 border-b border-border bg-panel/30 flex items-center justify-between gap-2">
          <div className="flex items-center gap-1.5">
            <GitCommitHorizontal size={15} className="text-primary shrink-0" />
            <h3 className="text-sm font-semibold text-text">
              Alur & Riwayat
            </h3>
          </div>
          <button
            type="button"
            onClick={() => setOpenModal(true)}
            className="inline-flex items-center gap-1 px-2 py-1 rounded-md text-[11px] font-semibold bg-primary/10 text-primary hover:bg-primary/15 transition-colors"
            title="Buka Visualisasi Diagram Alur & Riwayat Lengkap"
          >
            <Maximize2 size={11} />
            <span>Diagram</span>
          </button>
        </div>

        {/* Bagian 1: Stepper Tahap Persetujuan (Drafter -> Reviewer -> Approver) */}
        <div className="px-4 py-3.5">
          <LetterSteps
            steps={letter.steps}
            history={history}
            drafter={letter.drafter}
            status={letter.status}
            createdAt={letter.createdAt}
          />
        </div>

        {/* Bagian 2: Timeline Riwayat Aksi Ringkas (Compact Audit Trail) */}
        {history.length > 0 && (
          <div className="border-t border-border px-4 py-3 bg-panel/15 space-y-2.5">
            <div className="flex items-center justify-between gap-1">
              <span className="text-[11px] font-semibold uppercase tracking-wide text-text-muted">
                Jejak Aksi ({grouped.length})
              </span>
              <div className="inline-flex rounded-md border border-border bg-surface p-0.5 text-[10px]">
                <button
                  type="button"
                  onClick={() => {
                    setFilterMode("milestone");
                    setShowAllInline(false);
                  }}
                  className={cn(
                    "px-1.5 py-0.5 rounded font-semibold transition-colors",
                    filterMode === "milestone"
                      ? "bg-primary text-white"
                      : "text-text-muted hover:text-text"
                  )}
                >
                  Utama
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setFilterMode("all");
                    setShowAllInline(false);
                  }}
                  className={cn(
                    "px-1.5 py-0.5 rounded font-semibold transition-colors",
                    filterMode === "all"
                      ? "bg-primary text-white"
                      : "text-text-muted hover:text-text"
                  )}
                >
                  Semua ({history.length})
                </button>
              </div>
            </div>

            {/* Daftar Timeline Ringkas */}
            <ol className="relative space-y-2 before:absolute before:left-2 before:top-2 before:bottom-2 before:w-px before:bg-border">
              {visibleItems.map((item) => {
                const { entry, meta, repeatCount, versionNumbers } = item;
                return (
                  <li
                    key={item.id}
                    className="relative flex items-start gap-2 text-xs"
                  >
                    {smallActionDot(entry.action)}
                    <div className="min-w-0 flex-1 leading-snug">
                      <div className="flex items-center justify-between gap-1">
                        <div className="flex items-center gap-1 min-w-0 flex-wrap">
                          <span className="font-semibold text-text truncate">
                            {meta.shortLabel}
                          </span>
                          {repeatCount > 1 && (
                            <span className="text-[9px] font-semibold px-1 py-0.2 rounded bg-panel border border-border text-text-muted">
                              {repeatCount}×
                            </span>
                          )}
                          {versionNumbers.length > 0 && (
                            <span className="text-[9px] font-mono px-1 py-0.2 rounded bg-surface border border-border text-primary font-semibold">
                              v{versionNumbers[0]}
                            </span>
                          )}
                        </div>
                        <span className="text-[10px] text-text-muted shrink-0">
                          {formatCompactTime(entry.createdAt)}
                        </span>
                      </div>

                      <p className="text-[11px] text-text-muted truncate">
                        {entry.actor.name}
                        {entry.target && (
                          <span className="inline-flex items-center gap-0.5">
                            {" "}
                            <ArrowRight size={9} className="inline" />{" "}
                            {entry.target.name}
                          </span>
                        )}
                      </p>

                      {entry.note && (
                        <p className="mt-1 text-[11px] text-text bg-surface border border-border rounded px-2 py-1 italic line-clamp-2">
                          &ldquo;{entry.note}&rdquo;
                        </p>
                      )}
                    </div>
                  </li>
                );
              })}
            </ol>

            {/* Tombol Ekspansi Inline & Buka Diagram Lengkap */}
            <div className="pt-1 flex items-center justify-between gap-2 border-t border-border/60">
              {hiddenCount > 0 || showAllInline ? (
                <button
                  type="button"
                  onClick={() => setShowAllInline((v) => !v)}
                  className="inline-flex items-center gap-1 text-[11px] font-medium text-text-muted hover:text-text transition-colors"
                >
                  {showAllInline ? (
                    <>
                      <ChevronUp size={12} /> Ringkas (3 terbaru)
                    </>
                  ) : (
                    <>
                      <ChevronDown size={12} /> +{hiddenCount} aksi lainnya
                    </>
                  )}
                </button>
              ) : (
                <span className="text-[10px] text-text-muted">
                  Menampilkan semua aksi
                </span>
              )}

              <button
                type="button"
                onClick={() => setOpenModal(true)}
                className="text-[11px] font-semibold text-primary hover:underline ml-auto"
              >
                Lihat Visualisasi →
              </button>
            </div>
          </div>
        )}
      </div>

      {openModal && (
        <AuditTrailModal
          letter={letter}
          history={history}
          onClose={() => setOpenModal(false)}
        />
      )}
    </>
  );
}

