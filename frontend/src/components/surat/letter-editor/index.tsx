"use client";

// ─────────────────────────────────────────────
// components/surat/letter-editor/index.tsx
// Dynamic import wrapper untuk DevExpress RichEdit di Section 3 Buat Surat.
// ssr: false WAJIB karena devexpress-richedit membutuhkan window & DOM.
// ─────────────────────────────────────────────

import dynamic from "next/dynamic";
import { Loader2, FileEdit } from "lucide-react";

const RichEditSuratInner = dynamic(
  () => import("./rich-edit-surat-inner").then((m) => ({ default: m.default })),
  {
    ssr: false,
    loading: () => (
      <div className="border border-border rounded-xl bg-panel/30 h-[550px] flex flex-col items-center justify-center gap-3 text-text-muted p-8">
        <div className="w-12 h-12 rounded-2xl bg-surface border border-border flex items-center justify-center shadow-sm">
          <Loader2 size={24} className="animate-spin text-primary" />
        </div>
        <div className="text-center">
          <p className="text-sm font-semibold text-text">Memuat Editor Tata Naskah Digital...</p>
          <p className="text-xs text-text-muted mt-1">
            Menyiapkan DevExpress Word RichEdit & template surat resmi
          </p>
        </div>
      </div>
    ),
  }
);

interface LetterEditorSectionProps {
  onDocumentChange?: () => void;
}

export function LetterEditorSection({ onDocumentChange }: LetterEditorSectionProps) {
  return <RichEditSuratInner onDocumentChange={onDocumentChange} />;
}
