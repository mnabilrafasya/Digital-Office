"use client";

import {
  ArrowUpRight,
  LockKeyhole,
  Building2,
  CheckCircle2,
  BookOpen,
} from "lucide-react";
import type { AuthUser } from "@/lib/auth";
import {
  CATEGORIES,
  canOpen,
  accessLabel,
  type MasterCategory,
} from "./categories";

const GROUP_META: Record<string, { title: string }> = {
  Organisasi: { title: "Struktur Organisasi & Akun" },
  Persuratan: { title: "Komponen & Template Persuratan" },
  Pengaturan: { title: "Pengaturan Perusahaan" },
};

export function MasterOverview({
  user,
  activeCompany,
  onSelect,
}: {
  user: AuthUser;
  activeCompany: { code: string; name: string };
  onSelect: (key: string) => void;
}) {
  const manageable = CATEGORIES.filter(
    (category) => accessLabel(category, user.permissions) === "Kelola data",
  ).length;

  return (
    <div className="space-y-5">
      {/* Compact Scope Bar */}
      <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-border bg-panel/45 px-4 py-2.5 text-xs">
        <div className="flex flex-wrap items-center gap-2 text-text-muted">
          <Building2 size={14} className="text-primary shrink-0" />
          <span>
            Lingkup data aktif:{" "}
            <strong className="text-text">
              {activeCompany.code} — {activeCompany.name}
            </strong>
          </span>
        </div>
        <span className="text-text-muted">
          <strong className="text-text">{manageable}</strong> dari{" "}
          {CATEGORIES.length} modul dapat dikelola
        </span>
      </div>

      {/* Grouped Compact Module Menus */}
      {["Organisasi", "Persuratan", "Pengaturan"].map((group) => {
        const meta = GROUP_META[group];
        const items = CATEGORIES.filter((item) => item.group === group);
        const gridClass =
          group === "Organisasi"
            ? "grid gap-2.5 sm:grid-cols-2 lg:grid-cols-4"
            : group === "Persuratan"
              ? "grid gap-2.5 sm:grid-cols-2 lg:grid-cols-3"
              : "grid gap-2.5 sm:grid-cols-2 lg:grid-cols-3";

        return (
          <section key={group} className="space-y-2.5">
            <div className="flex items-center justify-between gap-2 border-b border-border/80 pb-1.5">
              <h2 className="font-bold text-sm text-text">
                {meta?.title ?? group}
              </h2>
              <span className="text-[11px] text-text-muted">
                {items.length} modul
              </span>
            </div>

            <div className={gridClass}>
              {items.map((category) => (
                <CategoryCard
                  key={category.key}
                  category={category}
                  permissions={user.permissions}
                  onClick={() => onSelect(category.key)}
                />
              ))}
            </div>
          </section>
        );
      })}
    </div>
  );
}

function CategoryCard({
  category,
  permissions,
  onClick,
}: {
  category: MasterCategory;
  permissions: string[];
  onClick: () => void;
}) {
  const allowed = canOpen(category, permissions);
  const label = accessLabel(category, permissions);

  const badgeTone =
    label === "Kelola data"
      ? "bg-status-success-bg text-status-success-text"
      : label === "Referensi"
        ? "bg-status-info-bg text-status-info-text"
        : "bg-panel text-text-muted";

  const BadgeIcon =
    label === "Kelola data"
      ? CheckCircle2
      : label === "Referensi"
        ? BookOpen
        : LockKeyhole;

  return (
    <button
      type="button"
      onClick={onClick}
      className={`group text-left p-3.5 rounded-xl border transition-all flex items-start gap-3 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary ${
        allowed
          ? "border-border bg-surface hover:border-primary/40 hover:shadow-sm"
          : "border-border/80 bg-surface/60 opacity-85 hover:opacity-100 hover:border-border"
      }`}
    >
      <span
        className={`rounded-lg p-2 shrink-0 transition-colors ${
          allowed
            ? "bg-primary/10 text-primary group-hover:bg-primary group-hover:text-white"
            : "bg-panel text-text-muted"
        }`}
      >
        <category.icon size={17} />
      </span>

      <div className="min-w-0 flex-1 space-y-1">
        <div className="flex items-center justify-between gap-2">
          <h3 className="text-sm font-bold text-text group-hover:text-primary transition-colors truncate">
            {category.label}
          </h3>
          {allowed ? (
            <ArrowUpRight
              size={14}
              className="text-text-muted group-hover:text-primary transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5 shrink-0"
            />
          ) : null}
        </div>

        <p className="text-xs text-text-muted leading-snug line-clamp-2">
          {category.description}
        </p>

        <div className="pt-0.5">
          <span
            className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-semibold ${badgeTone}`}
          >
            <BadgeIcon size={10} />
            {label}
          </span>
        </div>
      </div>
    </button>
  );
}
