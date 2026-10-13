"use client";

// ─────────────────────────────────────────────
// components/ui/status-badge.tsx
// Badge berlabel teks untuk status surat
// Selalu menampilkan label, tidak hanya warna
// ─────────────────────────────────────────────

import type { LetterStatus } from "@/lib/types";
import { getStatusMeta } from "@/lib/status";
import { cn } from "@/lib/utils";

interface StatusBadgeProps {
  status: LetterStatus;
  className?: string;
}

const variantClass: Record<string, string> = {
  success: "bg-status-success-bg text-status-success-text",
  warn:    "bg-status-warn-bg    text-status-warn-text",
  danger:  "bg-status-danger-bg  text-status-danger-text",
  info:    "bg-status-info-bg    text-status-info-text",
  neutral: "bg-status-neutral-bg text-status-neutral-text",
};

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const { label, variant } = getStatusMeta(status);

  return (
    <span
      className={cn(
        "inline-flex items-center rounded px-2 py-0.5 text-xs font-semibold tracking-wide",
        variantClass[variant],
        className,
      )}
    >
      {label}
    </span>
  );
}
