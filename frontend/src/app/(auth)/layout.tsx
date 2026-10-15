import type { ReactNode } from "react";

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <main className="min-h-screen bg-bg flex flex-col items-center px-4 pt-16 pb-12 sm:pt-20">
      {children}
    </main>
  );
}
