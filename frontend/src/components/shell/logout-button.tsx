"use client";

// ─────────────────────────────────────────────
// components/shell/logout-button.tsx
// Tombol logout — client component agar bisa
// memanggil logoutAction (Server Action).
// ─────────────────────────────────────────────

import { useTransition } from "react";
import { LogOut, LoaderCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import { logoutAction } from "@/app/actions/auth";

export function LogoutButton() {
  const [isPending, startTransition] = useTransition();

  return (
    <button
      id="btn-logout"
      disabled={isPending}
      onClick={() => startTransition(() => logoutAction())}
      className={cn(
        "flex items-center gap-1.5 text-sm text-slate-300 hover:text-white",
        "border border-white/20 hover:border-white/40",
        "px-3 py-1.5 rounded-md hover:bg-white/10 transition-colors",
        "disabled:opacity-60 disabled:cursor-not-allowed"
      )}
      aria-label="Keluar"
    >
      {isPending ? (
        <LoaderCircle size={14} className="animate-spin" />
      ) : (
        <LogOut size={14} />
      )}
      <span className="hidden sm:inline">Keluar</span>
    </button>
  );
}
