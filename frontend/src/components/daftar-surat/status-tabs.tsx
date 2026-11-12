"use client";

// ─────────────────────────────────────────────
// components/daftar-surat/status-tabs.tsx
// Tab status dengan jumlah; nilai aktif dikelola lewat URL (query string)
// ─────────────────────────────────────────────

import { cn } from "@/lib/utils";

interface Counts {
  ALL: number;
  DRAFT: number;
  IN_REVIEW: number;
  RETURNED: number;
  IN_APPROVAL: number;
  SENT: number;
  CANCELLED: number;
}

interface Tab {
  label: string;
  /** Value yang dikirim ke query string `status`; "" = Semua */
  value: string;
  /** Daftar nilai URL yang dianggap mengaktifkan tab ini (mendukung gabungan koma) */
  aliases?: string[];
  count: number;
}

interface Props {
  counts: Counts;
  activeTab: string;
  onTabChange: (value: string) => void;
}

export function StatusTabs({ counts, activeTab, onTabChange }: Props) {
  const tabs: Tab[] = [
    { label: "Semua",             value: "",                    count: counts.ALL },
    { label: "Draft",             value: "DRAFT",               count: counts.DRAFT },
    { label: "Menunggu Review",   value: "IN_REVIEW",           aliases: ["IN_REVIEW,RETURNED_TO_REVIEWER"], count: counts.IN_REVIEW },
    { label: "Dikembalikan",      value: "RETURNED_TO_DRAFTER", aliases: ["RETURNED_TO_REVIEWER", "RETURNED_TO_DRAFTER,RETURNED_TO_REVIEWER"], count: counts.RETURNED },
    { label: "Menunggu Approval", value: "IN_APPROVAL",         aliases: ["APPROVED", "IN_APPROVAL,APPROVED"], count: counts.IN_APPROVAL },
    { label: "Disetujui & Masuk", value: "SENT",                aliases: ["RECEIVED", "SENT,RECEIVED"], count: counts.SENT },
    { label: "Dibatalkan",        value: "CANCELLED",           count: counts.CANCELLED },
  ];

  const normalizedActive = activeTab.trim().toUpperCase();

  return (
    <div
      role="tablist"
      aria-label="Filter status surat"
      className="flex items-center gap-0.5 border-b border-border overflow-x-auto mb-4 -mx-1 px-1"
    >
      {tabs.map((tab) => {
        const isActive =
          normalizedActive === tab.value ||
          (tab.aliases?.includes(normalizedActive) ?? false);
        return (
          <button
            key={tab.value}
            role="tab"
            aria-selected={isActive}
            id={`tab-${tab.value || "semua"}`}
            onClick={() => onTabChange(tab.value)}
            className={cn(
              "relative flex items-center gap-1.5 px-3 py-2.5 text-sm font-medium whitespace-nowrap transition-colors shrink-0",
              isActive
                ? "text-primary after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-primary"
                : "text-text-muted hover:text-text"
            )}
          >
            {tab.label}
            <span
              className={cn(
                "inline-flex items-center justify-center min-w-5 h-5 px-1.5 rounded-full text-xs font-semibold",
                isActive
                  ? "bg-primary text-white"
                  : "bg-panel text-text-muted"
              )}
            >
              {tab.count}
            </span>
          </button>
        );
      })}
    </div>
  );
}
