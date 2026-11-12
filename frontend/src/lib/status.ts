// ─────────────────────────────────────────────
// lib/status.ts — Satu-satunya tempat label UI didefinisikan
// Label mengikuti PROJECT_CONTEXT bagian 5
// ─────────────────────────────────────────────

import type { LetterStatus } from "./types";

export interface StatusMeta {
  label: string;
  variant: "success" | "warn" | "danger" | "info" | "neutral";
}

export const STATUS_META: Record<LetterStatus, StatusMeta> = {
  DRAFT: {
    label: "Draft",
    variant: "info",
  },
  IN_REVIEW: {
    label: "Menunggu Review",
    variant: "warn",
  },
  RETURNED_TO_DRAFTER: {
    label: "Dikembalikan",
    variant: "danger",
  },
  IN_APPROVAL: {
    label: "Menunggu Approval",
    variant: "warn",
  },
  RETURNED_TO_REVIEWER: {
    label: "Dikembalikan",
    variant: "danger",
  },
  APPROVED: {
    label: "Menunggu Tanda Tangan",
    variant: "warn",
  },
  SENT: {
    label: "Disetujui",
    variant: "success",
  },
  RECEIVED: {
    label: "Surat Masuk Eksternal",
    variant: "info",
  },
  CANCELLED: {
    label: "Dibatalkan",
    variant: "danger",
  },
};

export function getStatusMeta(status: string): StatusMeta {
  return STATUS_META[status as LetterStatus] ?? { label: "Status belum dikenali", variant: "neutral" };
}
