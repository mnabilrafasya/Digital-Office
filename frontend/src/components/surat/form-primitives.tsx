// ─────────────────────────────────────────────
// components/surat/form-primitives.tsx
// Potongan UI kecil yang dipakai berulang di form Buat Surat
// ─────────────────────────────────────────────

import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function FieldLabel({
  label,
  required,
  htmlFor,
}: {
  label: string;
  required?: boolean;
  htmlFor?: string;
}) {
  return (
    <label htmlFor={htmlFor} className="block text-sm font-semibold text-text mb-1.5">
      {label}
      {required && <span className="text-status-danger-text ml-0.5">*</span>}
    </label>
  );
}

export function FieldError({ message }: { message?: string }) {
  if (!message) return null;
  return <p className="text-status-danger-text text-xs mt-1">{message}</p>;
}

/** Kelas untuk input/select/textarea. */
export function inputClass(hasError = false) {
  return cn(
    "w-full px-3 py-2.5 text-sm border rounded-lg bg-surface outline-none transition-colors",
    "focus:border-primary focus:ring-2 focus:ring-primary/15",
    hasError ? "border-status-danger-text" : "border-border-input",
  );
}

export function SectionCard({
  number,
  title,
  children,
}: {
  number: number;
  title: string;
  children: ReactNode;
}) {
  return (
    <div className="bg-surface rounded-2xl border border-border shadow-sm overflow-hidden">
      <div className="flex items-center gap-3 px-6 py-4 border-b border-border bg-panel/60">
        <div className="w-7 h-7 rounded-full bg-primary flex items-center justify-center shrink-0">
          <span className="text-white text-xs font-bold">{number}</span>
        </div>
        <h2 className="text-base font-bold text-text">{title}</h2>
      </div>
      <div className="px-6 py-5">{children}</div>
    </div>
  );
}
