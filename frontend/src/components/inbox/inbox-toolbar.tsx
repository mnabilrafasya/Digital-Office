"use client";

// ─────────────────────────────────────────────
// components/inbox/inbox-toolbar.tsx
// Bilah alat atas & pencari pesan bergaya Gmail untuk halaman Inbox
// ─────────────────────────────────────────────

import { useRef, useEffect } from "react";
import {
  Search,
  X,
  SlidersHorizontal,
  RefreshCw,
  ChevronLeft,
  ChevronRight,
  Columns2,
  List,
  Rows3,
  Rows4,
  Star,
  RotateCcw,
  Share2,
  MailOpen,
} from "lucide-react";
import type { InboxCounts } from "@/lib/api/inbox";
import { cn } from "@/lib/utils";

export type TabStatus = "" | "UNREAD" | "READ" | "DISPOSED" | "REDIRECTED";
export type SourceFilter =
  | "ALL"
  | "DIRECT_TO"
  | "DIRECT_CC"
  | "DISPOSITION"
  | "REDIRECT";
export type DensityMode = "comfortable" | "compact";

export interface InboxClientFilters {
  source: SourceFilter;
  confidentiality: string;
  urgency: string;
  starredOnly: boolean;
}

export const DEFAULT_CLIENT_FILTERS: InboxClientFilters = {
  source: "ALL",
  confidentiality: "ALL",
  urgency: "ALL",
  starredOnly: false,
};

interface InboxToolbarProps {
  query: string;
  onQueryChange: (q: string) => void;
  activeTab: TabStatus;
  onTabChange: (tab: TabStatus) => void;
  counts: InboxCounts;
  clientFilters: InboxClientFilters;
  onClientFiltersChange: (next: InboxClientFilters) => void;
  showFilterPanel: boolean;
  onToggleFilterPanel: () => void;
  loading: boolean;
  onRefresh: () => void;
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (nextPage: number) => void;
  splitView: boolean;
  onToggleSplitView: () => void;
  density: DensityMode;
  onToggleDensity: () => void;
  selectedCount: number;
  allVisibleSelected: boolean;
  onToggleSelectAll: () => void;
  onClearSelection: () => void;
  onOpenSelectedFirst?: () => void;
  onDisposeSelectedFirst?: () => void;
}

const TABS: { value: TabStatus; label: string }[] = [
  { value: "", label: "Utama (Semua)" },
  { value: "UNREAD", label: "Belum Dibaca" },
  { value: "READ", label: "Sudah Dibaca" },
  { value: "DISPOSED", label: "Didisposisikan" },
  { value: "REDIRECTED", label: "Dialihkan" },
];

export function InboxToolbar({
  query,
  onQueryChange,
  activeTab,
  onTabChange,
  counts,
  clientFilters,
  onClientFiltersChange,
  showFilterPanel,
  onToggleFilterPanel,
  loading,
  onRefresh,
  page,
  pageSize,
  total,
  onPageChange,
  splitView,
  onToggleSplitView,
  density,
  onToggleDensity,
  selectedCount,
  allVisibleSelected,
  onToggleSelectAll,
  onClearSelection,
  onOpenSelectedFirst,
  onDisposeSelectedFirst,
}: InboxToolbarProps) {
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Pintasan keyboard '/' atau 'Ctrl+K' untuk fokus ke kotak pencarian seperti Gmail
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const target = e.target as HTMLElement | null;
      const isInput =
        target?.tagName === "INPUT" ||
        target?.tagName === "TEXTAREA" ||
        target?.isContentEditable;

      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        searchInputRef.current?.focus();
        return;
      }
      if (!isInput && e.key === "/") {
        e.preventDefault();
        searchInputRef.current?.focus();
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  const totalAll =
    counts.UNREAD + counts.READ + counts.DISPOSED + counts.REDIRECTED;
  const startRange = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const endRange = Math.min(page * pageSize, total);

  const hasActiveClientFilter =
    clientFilters.source !== "ALL" ||
    clientFilters.confidentiality !== "ALL" ||
    clientFilters.urgency !== "ALL" ||
    clientFilters.starredOnly;

  return (
    <div className="bg-surface border-b border-border shrink-0">
      {/* ── Baris 1: Omnibox Search Bar ala Gmail + Utilitas Tampilan & Paginasi ── */}
      <div className="px-4 py-2.5 flex items-center justify-between gap-3 flex-wrap">
        {/* Kiri: Checkbox Pilih Semua + Aksi Massal + Refresh */}
        <div className="flex items-center gap-1.5">
          <label
            className="inline-flex items-center justify-center w-8 h-8 rounded-lg hover:bg-panel cursor-pointer transition-colors"
            title={
              allVisibleSelected
                ? "Batalkan semua pilihan"
                : "Pilih semua surat di halaman ini"
            }
          >
            <input
              type="checkbox"
              checked={allVisibleSelected}
              onChange={onToggleSelectAll}
              className="w-4 h-4 rounded border-border-input text-primary focus:ring-primary/30 cursor-pointer accent-primary"
            />
          </label>

          {selectedCount > 0 ? (
            <div className="flex items-center gap-1.5 pl-1 pr-2 py-1 rounded-lg bg-primary/10 text-primary text-xs font-semibold">
              <span>{selectedCount} dipilih</span>
              {selectedCount === 1 && onOpenSelectedFirst && (
                <button
                  type="button"
                  onClick={onOpenSelectedFirst}
                  className="ml-1 px-2 py-0.5 rounded bg-surface border border-border text-text hover:bg-panel inline-flex items-center gap-1 font-medium transition-colors"
                  title="Buka surat terpilih"
                >
                  <MailOpen size={12} /> Buka
                </button>
              )}
              {selectedCount === 1 && onDisposeSelectedFirst && (
                <button
                  type="button"
                  onClick={onDisposeSelectedFirst}
                  className="px-2 py-0.5 rounded bg-primary text-white hover:bg-primary-dark inline-flex items-center gap-1 font-medium transition-colors"
                  title="Disposisikan surat terpilih"
                >
                  <Share2 size={12} /> Disposisi
                </button>
              )}
              <button
                type="button"
                onClick={onClearSelection}
                className="p-0.5 rounded hover:bg-primary/20"
                title="Batalkan pilihan"
              >
                <X size={13} />
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={onRefresh}
              disabled={loading}
              title="Perbarui kotak masuk"
              className="w-8 h-8 rounded-lg flex items-center justify-center text-text-muted hover:text-text hover:bg-panel transition-colors disabled:opacity-50"
            >
              <RefreshCw size={15} className={cn(loading && "animate-spin")} />
            </button>
          )}
        </div>

        {/* Tengah: Kotak Pencarian Omnibox ala Gmail */}
        <div className="flex-1 max-w-2xl min-w-60">
          <div className="relative flex items-center bg-panel/70 hover:bg-panel focus-within:bg-surface border border-transparent focus-within:border-primary focus-within:shadow-xs rounded-full transition-all">
            <Search
              size={15}
              className="ml-3.5 text-text-muted shrink-0 pointer-events-none"
            />
            <input
              ref={searchInputRef}
              type="text"
              value={query}
              onChange={(e) => onQueryChange(e.target.value)}
              placeholder="Telusuri surat masuk (perihal, nomor surat, pengirim)... [tekan /]"
              className="w-full py-2 pl-2.5 pr-20 text-sm bg-transparent text-text placeholder:text-text-muted outline-none"
            />
            <div className="absolute right-2 flex items-center gap-1">
              {query && (
                <button
                  type="button"
                  onClick={() => onQueryChange("")}
                  title="Hapus pencarian"
                  className="p-1 rounded-full text-text-muted hover:text-text hover:bg-surface transition-colors"
                >
                  <X size={14} />
                </button>
              )}
              <button
                type="button"
                onClick={onToggleFilterPanel}
                title="Opsi filter lanjutan"
                className={cn(
                  "p-1.5 rounded-full transition-colors relative",
                  showFilterPanel || hasActiveClientFilter
                    ? "bg-primary/15 text-primary"
                    : "text-text-muted hover:text-text hover:bg-surface"
                )}
              >
                <SlidersHorizontal size={14} />
                {hasActiveClientFilter && (
                  <span className="absolute top-1 right-1 w-1.5 h-1.5 rounded-full bg-primary" />
                )}
              </button>
            </div>
          </div>
        </div>

        {/* Kanan: Filter Berbintang Cepat, Kerapatan, Mode Panel Baca, & Paginasi ala Gmail */}
        <div className="flex items-center gap-1.5 ml-auto">
          <button
            type="button"
            onClick={() =>
              onClientFiltersChange({
                ...clientFilters,
                starredOnly: !clientFilters.starredOnly,
              })
            }
            title={
              clientFilters.starredOnly
                ? "Tampilkan semua surat"
                : "Hanya tampilkan surat berbintang"
            }
            className={cn(
              "inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium border transition-colors",
              clientFilters.starredOnly
                ? "bg-status-warn-bg text-status-warn-text border-status-warn-text/30"
                : "border-border text-text-muted hover:text-text hover:bg-panel"
            )}
          >
            <Star
              size={13}
              className={cn(clientFilters.starredOnly && "fill-current")}
            />
            <span className="hidden sm:inline">Berbintang</span>
          </button>

          <button
            type="button"
            onClick={onToggleDensity}
            title={
              density === "comfortable"
                ? "Ubah ke tampilan ringkas (Compact)"
                : "Ubah ke tampilan nyaman (Default)"
            }
            className="w-8 h-8 rounded-lg border border-border flex items-center justify-center text-text-muted hover:text-text hover:bg-panel transition-colors"
          >
            {density === "comfortable" ? (
              <Rows3 size={15} />
            ) : (
              <Rows4 size={15} />
            )}
          </button>

          <button
            type="button"
            onClick={onToggleSplitView}
            title={
              splitView
                ? "Nonaktifkan Panel Baca Samping (Tampilan Daftar Penuh)"
                : "Aktifkan Panel Baca Samping ala Gmail (Split View)"
            }
            className={cn(
              "inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium border transition-colors",
              splitView
                ? "bg-primary/10 border-primary/30 text-primary"
                : "border-border text-text-muted hover:text-text hover:bg-panel"
            )}
          >
            {splitView ? <Columns2 size={14} /> : <List size={14} />}
            <span className="hidden md:inline">
              {splitView ? "Panel Baca" : "Daftar Penuh"}
            </span>
          </button>

          {/* Indikator jumlah & tombol < > ala Gmail */}
          <div className="flex items-center gap-1 pl-2 border-l border-border text-xs text-text-muted">
            <span className="px-1.5 whitespace-nowrap font-medium">
              {startRange}–{endRange} dari {total}
            </span>
            <button
              type="button"
              disabled={page <= 1 || loading}
              onClick={() => onPageChange(page - 1)}
              title="Halaman sebelumnya"
              className="w-7 h-7 rounded-full flex items-center justify-center hover:bg-panel disabled:opacity-35 disabled:pointer-events-none transition-colors"
            >
              <ChevronLeft size={15} />
            </button>
            <button
              type="button"
              disabled={page * pageSize >= total || loading}
              onClick={() => onPageChange(page + 1)}
              title="Halaman berikutnya"
              className="w-7 h-7 rounded-full flex items-center justify-center hover:bg-panel disabled:opacity-35 disabled:pointer-events-none transition-colors"
            >
              <ChevronRight size={15} />
            </button>
          </div>
        </div>
      </div>

      {/* ── Baris Filter Lanjutan (muncul saat ikon filter diklik) ── */}
      {showFilterPanel && (
        <div className="px-4 py-3 bg-panel/40 border-t border-border flex flex-wrap items-center gap-3 text-xs">
          <div className="flex items-center gap-1.5">
            <span className="text-text-muted font-semibold">Asal Surat:</span>
            <select
              value={clientFilters.source}
              onChange={(e) =>
                onClientFiltersChange({
                  ...clientFilters,
                  source: e.target.value as SourceFilter,
                })
              }
              className="px-2.5 py-1 rounded-lg border border-border bg-surface text-text font-medium outline-none focus:border-primary"
            >
              <option value="ALL">Semua Asal</option>
              <option value="DIRECT_TO">Tujuan Utama (Kepada)</option>
              <option value="DIRECT_CC">Tembusan (CC)</option>
              <option value="DISPOSITION">Disposisi Masuk</option>
              <option value="REDIRECT">Pengalihan Masuk</option>
            </select>
          </div>

          <div className="flex items-center gap-1.5">
            <span className="text-text-muted font-semibold">Sifat:</span>
            <select
              value={clientFilters.confidentiality}
              onChange={(e) =>
                onClientFiltersChange({
                  ...clientFilters,
                  confidentiality: e.target.value,
                })
              }
              className="px-2.5 py-1 rounded-lg border border-border bg-surface text-text font-medium outline-none focus:border-primary"
            >
              <option value="ALL">Semua Sifat</option>
              <option value="BIASA">Biasa</option>
              <option value="TERBATAS">Terbatas</option>
              <option value="RAHASIA">Rahasia</option>
            </select>
          </div>

          <div className="flex items-center gap-1.5">
            <span className="text-text-muted font-semibold">Kecepatan:</span>
            <select
              value={clientFilters.urgency}
              onChange={(e) =>
                onClientFiltersChange({
                  ...clientFilters,
                  urgency: e.target.value,
                })
              }
              className="px-2.5 py-1 rounded-lg border border-border bg-surface text-text font-medium outline-none focus:border-primary"
            >
              <option value="ALL">Semua Kecepatan</option>
              <option value="BIASA">Biasa</option>
              <option value="SEGERA">Segera</option>
              <option value="SANGAT_SEGERA">Sangat Segera</option>
            </select>
          </div>

          {hasActiveClientFilter && (
            <button
              type="button"
              onClick={() => onClientFiltersChange(DEFAULT_CLIENT_FILTERS)}
              className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-primary hover:bg-primary/10 font-semibold transition-colors ml-auto"
            >
              <RotateCcw size={12} /> Reset Filter
            </button>
          )}
        </div>
      )}

      {/* ── Baris 2: Tab Kategori ala Gmail (Primary / Unread / Read / Disposed / Redirected) ── */}
      <div className="flex items-center px-2 border-t border-border overflow-x-auto">
        {TABS.map((tab) => {
          const count = tab.value ? counts[tab.value] : totalAll;
          const isActive = activeTab === tab.value;
          const isUnreadTab = tab.value === "UNREAD";
          return (
            <button
              key={tab.value}
              type="button"
              onClick={() => onTabChange(tab.value)}
              className={cn(
                "relative flex items-center gap-2 px-4 py-2.5 text-xs sm:text-sm font-semibold whitespace-nowrap transition-colors",
                isActive
                  ? "text-primary"
                  : "text-text-muted hover:text-text hover:bg-panel/40"
              )}
            >
              <span>{tab.label}</span>
              {count > 0 && (
                <span
                  className={cn(
                    "text-[10px] font-bold px-1.5 py-0.2 rounded-full leading-relaxed",
                    isUnreadTab && count > 0
                      ? "bg-status-danger-bg text-status-danger-text"
                      : isActive
                        ? "bg-primary/15 text-primary"
                        : "bg-panel text-text-muted"
                  )}
                >
                  {count}
                </span>
              )}
              {isActive && (
                <span className="absolute bottom-0 left-3 right-3 h-0.5 rounded-t-full bg-primary" />
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}

