"use client";

// ─────────────────────────────────────────────
// app/dev/editor/editor-client.tsx
// Client component: memuat DevExpress RichEdit via dynamic import (ssr:false).
// ─────────────────────────────────────────────

import dynamic from "next/dynamic";

// DevExpress RichEdit adalah library murni browser — WAJIB ssr:false
const RichEditWrapper = dynamic(() => import("./rich-edit-wrapper").then(m => ({ default: m.default })), {
  ssr: false,
  loading: () => (
    <div className="flex items-center justify-center h-full gap-3 text-text-muted">
      <span className="text-sm">Memuat editor...</span>
    </div>
  ),
});

export function DevEditorClient() {
  return <RichEditWrapper />;
}
