// ─────────────────────────────────────────────
// components/daftar-surat/status-badge.tsx
// Badge status surat — selalu berlabel teks, sesuai lib/status.ts
// ─────────────────────────────────────────────

import { getStatusMeta } from "@/lib/status";
import type { LetterStatus } from "@/lib/types";
import { cn } from "@/lib/utils";

interface Props {
  status: LetterStatus;
  className?: string;
}

const VARIANT_CLASSES: Record<string, string> = {
  success: "bg-status-success-bg text-status-success-text",
  warn:    "bg-status-warn-bg text-status-warn-text",
  danger:  "bg-status-danger-bg text-status-danger-text",
  info:    "bg-status-info-bg text-status-info-text",
  neutral: "bg-status-neutral-bg text-status-neutral-text",
};

export function StatusBadge({ status, className }: Props) {
  const meta = getStatusMeta(status);
  return (
    <span
      className={cn(
        "inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold whitespace-nowrap",
        VARIANT_CLASSES[meta.variant],
        className
      )}
    >
      {meta.label}
    </span>
  );
}
