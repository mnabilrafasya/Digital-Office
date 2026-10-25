// ─────────────────────────────────────────────
// app/dev/editor/page.tsx
// Halaman uji DevExpress Rich Text Editor.
// HANYA DEVELOPMENT — akses di production diblokir.
// ─────────────────────────────────────────────

import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { DevEditorClient } from "./editor-client";

export const metadata: Metadata = {
  title: "Dev: Word Editor — Digital Office",
};

export default function DevEditorPage() {
  // Blokir di production secara server-side
  if (process.env.NODE_ENV === "production") {
    notFound();
  }

  return (
    <div className="flex flex-col h-screen bg-bg">
      {/* Header halaman uji */}
      <div className="flex items-center gap-3 px-5 py-3 border-b border-border bg-panel/60 shrink-0">
        <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-amber-100 text-amber-800 border border-amber-300">
          DEV ONLY
        </span>
        <h1 className="text-sm font-semibold text-text">
          Editor Dokumen (DevExpress Spike)
        </h1>
        <span className="text-xs text-text-muted ml-auto">
          Halaman ini hanya muncul saat{" "}
          <code className="font-mono bg-panel px-1 rounded">NODE_ENV=development</code>
        </span>
      </div>

      {/* Editor — full-height client component */}
      <div className="flex-1 overflow-hidden">
        <DevEditorClient />
      </div>
    </div>
  );
}
