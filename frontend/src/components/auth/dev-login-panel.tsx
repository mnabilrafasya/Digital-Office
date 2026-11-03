"use client";

// ─────────────────────────────────────────────
// components/auth/dev-login-panel.tsx
// Panel Akses Cepat Developer untuk pengujian end-to-end persuratan:
// Buat Surat → Review/Paraf → Tanda Tangan → Inbox Masuk → Disposisi / Alihkan.
// Hanya aktif di mode development (SHOW_DEV_PANEL).
// ─────────────────────────────────────────────

import { useState, useMemo } from "react";
import {
  Zap,
  Search,
  X,
  ArrowRight,
  LoaderCircle,
  FileText,
  Inbox,
  Share2,
  Building,
  Shield,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { DEV_ACCOUNTS, type DevUserAccount } from "@/lib/dev-accounts";

const CATEGORIES = [
  { id: "all", label: "Semua Akun" },
  { id: "workflow", label: "Alur Surat TI", icon: FileText },
  { id: "inbox", label: "Inbox & Disposisi", icon: Inbox },
  { id: "direksi", label: "Direksi", icon: Building },
  { id: "admin", label: "Admin & Multi-Perusahaan", icon: Shield },
] as const;

interface DevLoginPanelProps {
  onDevLogin: (nik: string, label: string) => Promise<void>;
  onFillNik: (nik: string) => void;
  devLoading: string | null;
}

export function DevLoginPanel({
  onDevLogin,
  onFillNik,
  devLoading,
}: DevLoginPanelProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [activeCategory, setActiveCategory] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState("");

  const filteredAccounts = useMemo(() => {
    return DEV_ACCOUNTS.filter((acc) => {
      const matchCat =
        activeCategory === "all" || acc.category === activeCategory;
      if (!matchCat) return false;

      if (!searchQuery.trim()) return true;
      const q = searchQuery.toLowerCase().trim();
      return (
        acc.name.toLowerCase().includes(q) ||
        acc.nik.includes(q) ||
        acc.position.toLowerCase().includes(q) ||
        acc.unit.toLowerCase().includes(q) ||
        acc.badge.toLowerCase().includes(q) ||
        acc.flowNote.toLowerCase().includes(q)
      );
    });
  }, [activeCategory, searchQuery]);

  function getBadgeStyles(variant: DevUserAccount["badgeVariant"]) {
    switch (variant) {
      case "emerald":
        return "bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-300 dark:border-emerald-800";
      case "sky":
        return "bg-sky-50 text-sky-700 border-sky-200 dark:bg-sky-950/40 dark:text-sky-300 dark:border-sky-800";
      case "violet":
        return "bg-violet-50 text-violet-700 border-violet-200 dark:bg-violet-950/40 dark:text-violet-300 dark:border-violet-800";
      case "indigo":
        return "bg-indigo-50 text-indigo-700 border-indigo-200 dark:bg-indigo-950/40 dark:text-indigo-300 dark:border-indigo-800";
      case "rose":
        return "bg-rose-50 text-rose-700 border-rose-200 dark:bg-rose-950/40 dark:text-rose-300 dark:border-rose-800";
      case "amber":
        return "bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-950/40 dark:text-amber-300 dark:border-amber-800";
      case "gray":
      default:
        return "bg-gray-100 text-gray-600 border-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700";
    }
  }

  function getAvatarColor(variant: DevUserAccount["badgeVariant"]) {
    switch (variant) {
      case "emerald":
        return "bg-emerald-600 text-white";
      case "sky":
        return "bg-sky-600 text-white";
      case "violet":
        return "bg-violet-600 text-white";
      case "indigo":
        return "bg-indigo-600 text-white";
      case "rose":
        return "bg-rose-600 text-white";
      case "amber":
        return "bg-amber-600 text-white";
      case "gray":
      default:
        return "bg-gray-500 text-white";
    }
  }

  return (
    <div className="bg-surface border border-border rounded-xl shadow-sm overflow-hidden transition-all">
      {/* ── Toggle Header ── */}
      <button
        type="button"
        id="btn-dev-panel-toggle"
        onClick={() => setIsOpen((prev) => !prev)}
        className={cn(
          "w-full flex items-center gap-2.5 px-4 py-3 text-sm font-medium text-text-muted",
          "hover:bg-bg transition-colors select-none",
          isOpen && "border-b border-border bg-panel/30"
        )}
      >
        <div className="size-6 rounded-md bg-amber-500/10 flex items-center justify-center shrink-0">
          <Zap size={14} className="text-amber-500" />
        </div>
        <div className="flex-1 text-left">
          <span className="font-semibold text-text text-xs uppercase tracking-wider block">
            Akses Cepat Developer & Testing
          </span>
          <span className="text-[11px] text-text-muted block">
            {DEV_ACCOUNTS.length} akun database untuk uji alur persuratan, inbox & disposisi
          </span>
        </div>
        {isOpen ? (
          <ChevronUp size={16} className="text-text-muted" />
        ) : (
          <ChevronDown size={16} className="text-text-muted" />
        )}
      </button>

      {/* ── Konten Panel Saat Terbuka ── */}
      {isOpen && (
        <div className="p-3.5 space-y-3 bg-surface">
          {/* Petunjuk Alur Pengujian */}
          <div className="bg-primary-dark/5 border border-primary-dark/15 rounded-lg p-2.5 text-[11px] leading-relaxed text-text space-y-1">
            <div className="flex items-center gap-1.5 font-semibold text-primary-dark text-xs">
              <Share2 size={13} />
              <span>Panduan Alur Pengujian Lengkap:</span>
            </div>
            <p className="text-text-muted text-[11px]">
              <strong className="text-text">1. Buat Surat:</strong> Andi Kurniawan (Drafter) →{" "}
              <strong className="text-text">2. Review:</strong> Fajar Nugroho / Dewi Lestari →{" "}
              <strong className="text-text">3. Tanda Tangan:</strong> Agus Setiawan (Approver) →{" "}
              <strong className="text-text">4. Masuk Inbox:</strong> Sri Wahyuni (Pimpinan SDM) →{" "}
              <strong className="text-text">5. Disposisi:</strong> Maya Anggraini (Staf SDM).
            </p>
          </div>

          {/* Search Bar */}
          <div className="relative">
            <Search
              size={14}
              className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted"
            />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Cari nama, NIK, jabatan, atau peran..."
              className="w-full h-8 pl-8 pr-7 text-xs border border-border rounded-lg bg-surface placeholder:text-text-muted/60 focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary/20"
            />
            {searchQuery && (
              <button
                type="button"
                onClick={() => setSearchQuery("")}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-text-muted hover:text-text p-0.5"
                title="Hapus pencarian"
              >
                <X size={12} />
              </button>
            )}
          </div>

          {/* Kategori Tab Filter */}
          <div className="flex items-center gap-1 overflow-x-auto pb-1 no-scrollbar text-[11px]">
            {CATEGORIES.map((cat) => {
              const count =
                cat.id === "all"
                  ? DEV_ACCOUNTS.length
                  : DEV_ACCOUNTS.filter((a) => a.category === cat.id).length;
              const isActive = activeCategory === cat.id;

              return (
                <button
                  key={cat.id}
                  type="button"
                  onClick={() => setActiveCategory(cat.id)}
                  className={cn(
                    "whitespace-nowrap px-2.5 py-1 rounded-md font-medium transition-colors border",
                    isActive
                      ? "bg-primary-dark text-white border-primary-dark shadow-xs"
                      : "bg-surface text-text-muted hover:text-text border-border hover:bg-bg"
                  )}
                >
                  {cat.label} ({count})
                </button>
              );
            })}
          </div>

          {/* Daftar Akun (Scrollable) */}
          <div className="max-h-87.5 overflow-y-auto space-y-2 pr-1">
            {filteredAccounts.length === 0 ? (
              <div className="py-6 text-center text-xs text-text-muted border border-dashed border-border rounded-lg">
                Tidak ada user yang cocok dengan &quot;{searchQuery}&quot;
              </div>
            ) : (
              filteredAccounts.map((acc) => {
                const isLoading = devLoading === acc.nik;
                const initials = acc.name
                  .split(" ")
                  .filter((part) => !part.includes("."))
                  .slice(0, 2)
                  .map((p) => p[0])
                  .join("")
                  .toUpperCase() || acc.name.slice(0, 2).toUpperCase();

                return (
                  <div
                    key={acc.nik}
                    className="p-2.5 rounded-lg border border-border bg-surface hover:bg-bg/60 transition-all flex flex-col gap-1.5 shadow-2xs"
                  >
                    <div className="flex items-start justify-between gap-2">
                      {/* Avatar & Identitas */}
                      <div className="flex items-start gap-2.5 min-w-0">
                        <div
                          className={cn(
                            "size-8 rounded-lg flex items-center justify-center font-bold text-xs shrink-0 mt-0.5",
                            getAvatarColor(acc.badgeVariant)
                          )}
                        >
                          {initials}
                        </div>
                        <div className="min-w-0">
                          <div className="flex items-center gap-1.5 flex-wrap">
                            <span className="font-semibold text-xs text-text truncate">
                              {acc.name}
                            </span>
                            <span
                              className={cn(
                                "text-[10px] font-semibold px-1.5 py-0.5 rounded border",
                                getBadgeStyles(acc.badgeVariant)
                              )}
                            >
                              {acc.badge}
                            </span>
                          </div>
                          <div className="text-[11px] text-text-muted truncate">
                            {acc.position} · <span className="text-text-muted/80">{acc.unit}</span>
                          </div>
                        </div>
                      </div>

                      {/* Tombol Aksi */}
                      <div className="flex items-center gap-1 shrink-0">
                        <button
                          type="button"
                          onClick={() => onFillNik(acc.nik)}
                          title="Isi NIK ke form login di atas"
                          className="px-2 py-1 text-[11px] font-medium border border-border rounded hover:bg-bg text-text-muted hover:text-text transition-colors"
                        >
                          Isi NIK
                        </button>
                        <button
                          type="button"
                          disabled={devLoading !== null}
                          onClick={() => onDevLogin(acc.nik, acc.name)}
                          className={cn(
                            "flex items-center gap-1 px-2.5 py-1 text-[11px] font-semibold rounded transition-colors shadow-2xs",
                            "bg-primary-dark text-white hover:bg-primary disabled:opacity-50"
                          )}
                          title={`Login instan sebagai ${acc.name} (${acc.nik})`}
                        >
                          {isLoading ? (
                            <LoaderCircle size={12} className="animate-spin" />
                          ) : (
                            <>
                              Masuk
                              <ArrowRight size={11} />
                            </>
                          )}
                        </button>
                      </div>
                    </div>

                    {/* Catatan Alur Pengujian & NIK */}
                    <div className="flex items-center justify-between text-[10px] pt-1 border-t border-border/40 text-text-muted gap-2">
                      <span className="truncate italic flex items-center gap-1">
                        <span className="text-primary font-medium">Uji:</span>
                        {acc.flowNote}
                      </span>
                      <code className="shrink-0 font-mono text-[10px] bg-panel px-1 py-0.2 rounded border border-border/60">
                        NIK: {acc.nik}
                      </code>
                    </div>
                  </div>
                );
              })
            )}
          </div>

          <p className="text-[10px] text-text-muted/70 text-center pt-1">
            Password dev otomatis diisi dari <code>SEED_PASSWORD</code> di <code>.env.local</code>.
          </p>
        </div>
      )}
    </div>
  );
}
