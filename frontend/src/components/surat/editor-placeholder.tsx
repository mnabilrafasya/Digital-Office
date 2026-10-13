"use client";

// ─────────────────────────────────────────────
// components/surat/editor-placeholder.tsx
// Section 3: Placeholder editor — komponen dipisah agar
// mudah diganti editor sesungguhnya nanti (TipTap/OnlyOffice)
// Lihat PROJECT_CONTEXT bagian 7 (Section 3) & bagian 13
// ─────────────────────────────────────────────

import { FileText } from "lucide-react";

export function EditorPlaceholder() {
  return (
    <div className="rounded-xl border-2 border-dashed border-border bg-panel/50 flex flex-col items-center justify-center py-16 px-8 text-center">
      <div className="w-14 h-14 rounded-2xl bg-surface border border-border flex items-center justify-center mb-4 shadow-sm">
        <FileText size={26} className="text-text-muted" />
      </div>
      <h3 className="text-base font-semibold text-text mb-1">
        Editor Naskah Belum Tersedia
      </h3>
      <p className="text-sm text-text-muted max-w-sm">
        Editor naskah menyusul. Isi surat akan disusun dari template yang dipilih
        pada field di atas setelah fitur ini aktif.
      </p>
      <p className="text-xs text-text-muted mt-3 italic">
        Komponen <code className="bg-border px-1 rounded text-xs">EditorPlaceholder</code> — akan diganti editor Word-like pada iterasi berikutnya.
      </p>
    </div>
  );
}
