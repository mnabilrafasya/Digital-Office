"use client";

// ─────────────────────────────────────────────
// components/review/letter-steps.tsx
// Stepper visual ringkas alur persetujuan: Pembuat → Reviewer 1..N → Approver
// ─────────────────────────────────────────────

import { useState } from "react";
import {
  Check,
  Clock,
  Circle,
  RotateCcw,
  FileEdit,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import type {
  LetterStep,
  LetterHistoryEntry,
  PersonRef,
} from "@/lib/api/letter-detail";
import { formatCompactTime } from "./audit-trail-utils";
import { cn } from "@/lib/utils";

interface LetterStepsProps {
  steps: LetterStep[];
  /** Riwayat aksi dari `GET /api/v1/letters/{id}/history` untuk waktu & catatan pengembalian per step */
  history?: LetterHistoryEntry[];
  /** Pembuat surat (opsional, ditampilkan sebagai simpul awal alur) */
  drafter?: PersonRef;
  status?: string;
  direction?: string;
  createdAt?: string;
}

export function LetterSteps({
  steps,
  history = [],
  drafter,
  status,
  direction,
  createdAt,
}: LetterStepsProps) {
  const [expandedNoteIdx, setExpandedNoteIdx] = useState<number | null>(null);

  const isReturnedToDrafter = status === "RETURNED_TO_DRAFTER";
  const isDraft = status === "DRAFT";
  const totalApprovers = steps.filter((s) => s.role === "APPROVER").length;

  return (
    <div className="space-y-0">
      {/* Simpul Awal: Pembuat Surat (Drafter) bila disertakan */}
      {drafter && (
        <div className="flex gap-2.5">
          <div className="flex flex-col items-center shrink-0">
            <div
              className={cn(
                "w-6 h-6 rounded-full flex items-center justify-center text-white shrink-0 border",
                isReturnedToDrafter
                  ? "bg-status-warn-text border-status-warn-text"
                  : isDraft
                    ? "bg-primary border-primary"
                    : "bg-status-success-text border-status-success-text"
              )}
            >
              {isReturnedToDrafter ? (
                <RotateCcw size={11} />
              ) : isDraft ? (
                <FileEdit size={11} />
              ) : (
                <Check size={12} strokeWidth={2.5} />
              )}
            </div>
            {steps.length > 0 && (
              <div className="w-px flex-1 my-1 bg-status-success-text/40" />
            )}
          </div>
          <div className="pb-3 min-w-0 flex-1">
            <div className="flex items-center justify-between gap-1">
              <span className="text-[11px] font-semibold text-status-success-text">
                {direction === "INCOMING"
                  ? "Pencatat (Surat Masuk)"
                  : "Pembuat (Drafter)"}
              </span>
              {createdAt && (
                <span className="text-[10px] text-text-muted shrink-0">
                  {formatCompactTime(createdAt)}
                </span>
              )}
            </div>
            <p className="text-xs font-semibold text-text truncate">
              {drafter.name}
            </p>
            <p className="text-[11px] text-text-muted truncate">
              {drafter.positionName}
            </p>
          </div>
        </div>
      )}

      {/* Tahap Reviewer 1..N & Approver */}
      {steps.map((step, idx) => {
        const isDone = step.state === "DONE";
        const isActive = step.state === "ACTIVE";
        const isPending = step.state === "PENDING";
        const isReturnedReviewer =
          status === "RETURNED_TO_REVIEWER" &&
          isActive &&
          step.role === "REVIEWER";

        const label =
          step.role === "APPROVER"
            ? totalApprovers > 1
              ? `Approver ${step.order}${step.order === totalApprovers ? " (Penandatangan)" : ""}`
              : "Approver"
            : `Reviewer ${step.order}`;

        const stepActions = history.filter(
          (h) =>
            h.actor.id === step.user.id &&
            (!h.actorRole || h.actorRole === step.role)
        );
        const lastAction = stepActions[stepActions.length - 1];
        const latestReturnEntry = [...stepActions]
          .reverse()
          .find((h) => h.action === "RETURN" && Boolean(h.note));
        const returnNote = step.returnNote ?? latestReturnEntry?.note ?? null;
        const isNoteOpen = expandedNoteIdx === idx;

        return (
          <div key={idx} className="flex gap-2.5">
            {/* Garis vertikal + ikon status */}
            <div className="flex flex-col items-center shrink-0">
              <div
                className={cn(
                  "w-6 h-6 rounded-full flex items-center justify-center text-white shrink-0 border",
                  isDone &&
                    "bg-status-success-text border-status-success-text",
                  isActive &&
                    !isReturnedReviewer &&
                    "bg-primary border-primary ring-2 ring-primary/15",
                  isReturnedReviewer &&
                    "bg-status-warn-text border-status-warn-text ring-2 ring-status-warn-text/15",
                  isPending && "bg-surface border-border text-text-muted"
                )}
              >
                {isDone && <Check size={12} strokeWidth={2.5} />}
                {isActive && !isReturnedReviewer && (
                  <Circle size={8} fill="white" strokeWidth={0} />
                )}
                {isReturnedReviewer && <RotateCcw size={11} />}
                {isPending && <Clock size={10} />}
              </div>
              {idx < steps.length - 1 && (
                <div
                  className={cn(
                    "w-px flex-1 my-1",
                    isDone ? "bg-status-success-text/40" : "bg-border"
                  )}
                />
              )}
            </div>

            {/* Konten Tahap Ringkas */}
            <div
              className={cn(
                "pb-3 min-w-0 flex-1",
                idx === steps.length - 1 && "pb-0"
              )}
            >
              <div className="flex items-center justify-between gap-1 flex-wrap">
                <div className="flex items-center gap-1.5 flex-wrap">
                  <span
                    className={cn(
                      "text-[11px] font-semibold",
                      isDone && "text-status-success-text",
                      isActive && !isReturnedReviewer && "text-primary",
                      isReturnedReviewer && "text-status-warn-text",
                      isPending && "text-text-muted"
                    )}
                  >
                    {label}
                  </span>
                  {isActive && (
                    <span
                      className={cn(
                        "text-[9px] px-1.5 py-0.2 rounded-full font-semibold",
                        isReturnedReviewer
                          ? "bg-status-warn-bg text-status-warn-text"
                          : "bg-primary/10 text-primary"
                      )}
                    >
                      {isReturnedReviewer ? "Revisi" : "Saat ini"}
                    </span>
                  )}
                </div>
                {lastAction && (
                  <span className="text-[10px] text-text-muted shrink-0">
                    {formatCompactTime(lastAction.createdAt)}
                  </span>
                )}
              </div>

              <div className="text-xs font-semibold text-text truncate">
                {step.user.name}
              </div>
              <div className="text-[11px] text-text-muted truncate">
                {step.user.positionName}
              </div>

              {returnNote && (
                <div className="mt-1">
                  <button
                    type="button"
                    onClick={() =>
                      setExpandedNoteIdx(isNoteOpen ? null : idx)
                    }
                    className="inline-flex items-center gap-1 text-[10px] font-semibold px-2 py-0.5 rounded bg-status-warn-bg text-status-warn-text hover:opacity-90 transition-opacity"
                  >
                    <RotateCcw size={9} />
                    <span>Catatan Revisi</span>
                    {isNoteOpen ? (
                      <ChevronUp size={10} />
                    ) : (
                      <ChevronDown size={10} />
                    )}
                  </button>
                  {isNoteOpen && (
                    <div className="mt-1 text-[11px] bg-status-warn-bg/60 text-status-warn-text px-2.5 py-1.5 rounded-md border border-status-warn-text/20 italic">
                      &ldquo;{returnNote}&rdquo;
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
