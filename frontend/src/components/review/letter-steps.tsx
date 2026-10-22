"use client";

// ─────────────────────────────────────────────
// components/review/letter-steps.tsx
// Stepper alur surat: Reviewer 1 → Reviewer 2 → Approver
// ─────────────────────────────────────────────

import { Check, Clock, Circle } from "lucide-react";
import type { LetterStep } from "@/lib/api/letter-detail";
import { cn } from "@/lib/utils";

interface LetterStepsProps {
  steps: LetterStep[];
}

export function LetterSteps({ steps }: LetterStepsProps) {
  return (
    <div className="space-y-0">
      {steps.map((step, idx) => {
        const isDone    = step.state === "DONE";
        const isActive  = step.state === "ACTIVE";
        const isPending = step.state === "PENDING";
        const label =
          step.role === "APPROVER"
            ? "Approver"
            : `Reviewer ${step.order}`;

        return (
          <div key={idx} className="flex gap-3">
            {/* Garis vertikal + ikon status */}
            <div className="flex flex-col items-center shrink-0">
              <div
                className={cn(
                  "w-7 h-7 rounded-full flex items-center justify-center text-white shrink-0 border-2",
                  isDone   && "bg-status-success-text border-status-success-text",
                  isActive && "bg-primary border-primary",
                  isPending && "bg-surface border-border text-text-muted"
                )}
              >
                {isDone   && <Check size={13} strokeWidth={2.5} />}
                {isActive && <Circle size={9} fill="white" strokeWidth={0} />}
                {isPending && <Clock size={11} />}
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

            {/* Konten */}
            <div className={cn("pb-4 min-w-0", idx === steps.length - 1 && "pb-0")}>
              <div className="flex items-center gap-1.5 flex-wrap">
                <span
                  className={cn(
                    "text-xs font-semibold",
                    isDone   && "text-status-success-text",
                    isActive && "text-primary",
                    isPending && "text-text-muted"
                  )}
                >
                  {label}
                </span>
                {isActive && (
                  <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-primary/10 text-primary font-semibold">
                    Saat ini
                  </span>
                )}
                {isDone && step.returnNote && (
                  <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-status-danger-bg text-status-danger-text font-semibold">
                    Dikembalikan
                  </span>
                )}
              </div>
              <div className="text-sm font-medium text-text mt-0.5 truncate">
                {step.user.name}
              </div>
              <div className="text-xs text-text-muted truncate">
                {step.user.positionName}
              </div>
              {isDone && step.returnNote && (
                <div className="mt-1.5 text-xs bg-status-danger-bg text-status-danger-text px-2.5 py-1.5 rounded-md border border-status-danger-text/20">
                  &quot;{step.returnNote}&quot;
                </div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
