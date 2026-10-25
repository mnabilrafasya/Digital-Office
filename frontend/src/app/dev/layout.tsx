// ─────────────────────────────────────────────
// app/dev/layout.tsx
// Layout untuk halaman /dev/* — tanpa sidebar dan header.
// Seluruh halaman /dev hanya tersedia di development.
// ─────────────────────────────────────────────

import type { ReactNode } from "react";
import { notFound } from "next/navigation";
import type { Metadata } from "next";

export const metadata: Metadata = {
  robots: "noindex, nofollow",
};

export default function DevLayout({ children }: { children: ReactNode }) {
  if (process.env.NODE_ENV === "production") {
    notFound();
  }

  return (
    <div className="min-h-screen bg-bg text-text">
      {children}
    </div>
  );
}
