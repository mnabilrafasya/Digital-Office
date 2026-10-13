"use client";

// ─────────────────────────────────────────────
// components/shell/header.tsx
// Header navy — mengikuti tampilan di PROJECT_CONTEXT bagian 8
// ─────────────────────────────────────────────

import Link from "next/link";
import { Bell, Home, LogOut, FileText, ChevronDown } from "lucide-react";
import { CURRENT_USER } from "@/lib/mock-data";
import { cn } from "@/lib/utils";

export function Header() {
  return (
    <header className="fixed top-0 left-56 right-0 h-14 bg-primary-dark border-b border-white/10 z-20 flex items-center px-5 gap-4">
      {/* Nav links */}
      <nav className="flex items-center gap-1">
        <Link
          href="/dashboard"
          className="flex items-center gap-1.5 text-slate-300 hover:text-white text-sm px-3 py-1.5 rounded-md hover:bg-white/10 transition-colors"
        >
          <Home size={14} />
          <span>Beranda</span>
        </Link>
        <Link
          href="#"
          className="flex items-center gap-1.5 text-slate-300 hover:text-white text-sm px-3 py-1.5 rounded-md hover:bg-white/10 transition-colors"
        >
          <FileText size={14} />
          <span>Supporting Documents</span>
        </Link>
      </nav>

      {/* Spacer */}
      <div className="flex-1" />

      {/* Actions */}
      <div className="flex items-center gap-2">
        {/* Notifikasi */}
        <button
          className="relative text-slate-300 hover:text-white p-2 rounded-md hover:bg-white/10 transition-colors"
          aria-label="Notifikasi"
        >
          <Bell size={18} />
          <span className="absolute top-1 right-1 w-2 h-2 bg-accent rounded-full" />
        </button>

        {/* User */}
        <div className="flex items-center gap-2 text-sm text-slate-300">
          <div className="w-7 h-7 rounded-full bg-primary flex items-center justify-center text-white font-bold text-xs">
            {CURRENT_USER.name.charAt(0)}
          </div>
          <div className="hidden sm:block">
            <div className="text-white text-xs font-medium leading-tight">
              {CURRENT_USER.name}
            </div>
            <div className="text-slate-400 text-[10px] leading-tight">
              {CURRENT_USER.nik}
            </div>
          </div>
          <ChevronDown size={13} className="text-slate-400" />
        </div>

        {/* Logout */}
        <button
          className={cn(
            "flex items-center gap-1.5 text-sm text-slate-300 hover:text-white",
            "border border-white/20 hover:border-white/40",
            "px-3 py-1.5 rounded-md hover:bg-white/10 transition-colors",
          )}
          aria-label="Keluar"
        >
          <LogOut size={14} />
          <span className="hidden sm:inline">Keluar</span>
        </button>
      </div>
    </header>
  );
}
