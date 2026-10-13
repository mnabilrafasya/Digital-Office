// ─────────────────────────────────────────────
// lib/utils.ts
// Utility functions umum
// ─────────────────────────────────────────────

import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/** Merge Tailwind classes dengan deduplication */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/** Format bytes ke string yang readable (KB, MB, dsb.) */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

/** Format tanggal ke string Indonesia (WIB) */
export function formatDateTimeWIB(iso: string): string {
  const date = new Date(iso);
  return date.toLocaleString("id-ID", {
    timeZone: "Asia/Jakarta",
    day: "2-digit",
    month: "long",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }) + " WIB";
}

/** Format waktu singkat untuk autosave indicator */
export function formatTimeWIB(date: Date): string {
  return date.toLocaleTimeString("id-ID", {
    timeZone: "Asia/Jakarta",
    hour: "2-digit",
    minute: "2-digit",
  }) + " WIB";
}

/** Generate UUID sederhana untuk ID lokal */
export function generateId(): string {
  return Math.random().toString(36).substring(2, 10) +
         Math.random().toString(36).substring(2, 10);
}

/** MIME type ke label singkat */
export function mimeToLabel(mimeType: string): string {
  const map: Record<string, string> = {
    "application/pdf":                                                           "PDF",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document":   "DOCX",
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "XLSX",
    "application/vnd.openxmlformats-officedocument.presentationml.presentation": "PPTX",
    "application/msword":                                                        "DOC",
  };
  return map[mimeType] ?? mimeType.split("/").pop()?.toUpperCase() ?? "FILE";
}
