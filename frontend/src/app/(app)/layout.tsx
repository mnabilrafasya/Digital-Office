// ─────────────────────────────────────────────
// app/(app)/layout.tsx
// Shell layout: sidebar kiri + header atas + konten utama
// ─────────────────────────────────────────────

import { Suspense, type ReactNode } from "react";
import { Sidebar } from "@/components/shell/sidebar";
import { Header } from "@/components/shell/header";

interface AppLayoutProps {
  children: ReactNode;
}

export default function AppLayout({ children }: AppLayoutProps) {
  return (
    <div className="min-h-screen bg-bg">
      <Suspense fallback={<aside className="fixed inset-y-0 left-0 w-56 bg-primary-dark z-30" />}>
        <Sidebar />
      </Suspense>
      <Header />
      {/* Main content — offset untuk sidebar (w-56 = 224px) dan header (h-14 = 56px) */}
      <main className="ml-56 pt-14 min-h-screen">
        <div className="p-6">
          {children}
        </div>
      </main>
    </div>
  );
}
