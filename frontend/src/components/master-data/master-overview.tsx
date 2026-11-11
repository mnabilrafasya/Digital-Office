"use client";

import {
  ArrowUpRight,
  LockKeyhole,
  Building2,
  ShieldCheck,
  Layers3,
  CircleHelp,
  CheckCircle2,
  BookOpen,
  Stamp,
  Sparkles,
} from "lucide-react";
import type { AuthUser } from "@/lib/auth";
import {
  CATEGORIES,
  canOpen,
  accessLabel,
  type MasterCategory,
} from "./categories";

const GROUP_META: Record<
  string,
  { title: string; subtitle: string; badge: string }
> = {
  Organisasi: {
    title: "Struktur Organisasi & Akun",
    subtitle:
      "Kelola entitas perusahaan, susunan unit kerja, jabatan struktural/fungsional, dan akun pengguna.",
    badge: "4 modul",
  },
  Persuratan: {
    title: "Komponen & Template Persuratan",
    subtitle:
      "Atur template dokumen .docx serta referensi jenis surat, klasifikasi arsip, sifat, dan kecepatan surat.",
    badge: "5 modul",
  },
  Pengaturan: {
    title: "Identitas Naskah, Penomoran & Watermark",
    subtitle:
      "Konfigurasi pola penomoran surat keluar otomatis serta teks watermark diagonal pada PDF naskah resmi.",
    badge: "Konfigurasi PDF & Nomor",
  },
};

export function MasterOverview({
  user,
  onSelect,
}: {
  user: AuthUser;
  onSelect: (key: string) => void;
}) {
  const available = CATEGORIES.filter((category) =>
    canOpen(category, user.permissions),
  ).length;
  const manageable = CATEGORIES.filter(
    (category) => accessLabel(category, user.permissions) === "Kelola data",
  ).length;
  const canConfigureSettings = user.permissions.includes("COMPANY_SETTINGS");

  const summaryCards = [
    {
      icon: Building2,
      label: "Lingkup Perusahaan",
      value: user.company.code,
      sub: user.company.name,
      tone: "bg-primary/10 text-primary",
    },
    {
      icon: ShieldCheck,
      label: "Peran & Hak Akses",
      value: user.role.name,
      sub: `${manageable} modul dapat dikelola langsung`,
      tone: "bg-status-success-bg text-status-success-text",
    },
    {
      icon: Layers3,
      label: "Modul Tersedia",
      value: `${available} dari ${CATEGORIES.length}`,
      sub: "Organisasi, komponen surat, dan watermark",
      tone: "bg-status-info-bg text-status-info-text",
    },
  ];

  return (
    <div className="space-y-8">
      {/* Summary KPI Cards */}
      <div className="grid gap-3.5 sm:grid-cols-3">
        {summaryCards.map((card) => (
          <div
            key={card.label}
            className="bg-surface border border-border rounded-xl p-4 sm:p-5 flex items-start justify-between gap-3 shadow-xs"
          >
            <div className="min-w-0">
              <p className="text-xs text-text-muted font-medium">
                {card.label}
              </p>
              <p
                className="text-xl font-bold mt-1.5 tracking-tight truncate"
                title={card.value}
              >
                {card.value}
              </p>
              <p className="text-xs text-text-muted mt-1 truncate" title={card.sub}>
                {card.sub}
              </p>
            </div>
            <div className={`p-2.5 rounded-xl shrink-0 ${card.tone}`}>
              <card.icon size={19} />
            </div>
          </div>
        ))}
      </div>

      {/* Quick Banner for Watermark & Number Format */}
      <div className="rounded-xl border border-primary/20 bg-linear-to-r from-primary/8 via-primary/4 to-transparent p-4 sm:p-5 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-start gap-3.5">
          <div className="p-2.5 rounded-xl bg-primary text-white shrink-0 mt-0.5">
            <Stamp size={19} />
          </div>
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="text-sm font-bold text-text">
                Watermark PDF & Format Nomor Surat
              </h2>
              <span className="inline-flex items-center gap-1 rounded-full bg-primary/15 text-primary px-2 py-0.5 text-[11px] font-semibold">
                <Sparkles size={11} />
                Identitas Naskah Resmi
              </span>
            </div>
            <p className="text-xs text-text-muted mt-1 leading-relaxed max-w-2xl">
              Teks watermark diagonal pada dokumen PDF resmi serta format
              penomoran surat otomatis (token{" "}
              <code className="font-mono text-text bg-surface/80 px-1 py-0.5 rounded border border-border">
                {"{SEQ}/{COMP}/{MM_ROMAN}/{YYYY}"}
              </code>
              ) diatur melalui menu{" "}
              <strong className="text-text">Pengaturan Perusahaan</strong>.
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => onSelect("settings")}
          className="inline-flex items-center justify-center gap-1.5 rounded-lg bg-surface border border-border hover:border-primary/50 px-3.5 py-2 text-xs font-semibold text-text hover:text-primary transition-colors shrink-0 shadow-2xs"
        >
          {canConfigureSettings
            ? "Atur Watermark & Nomor"
            : "Lihat Pengaturan"}
          <ArrowUpRight size={14} />
        </button>
      </div>

      {/* Grouped Category Cards */}
      {["Organisasi", "Persuratan", "Pengaturan"].map((group) => {
        const meta = GROUP_META[group];
        const items = CATEGORIES.filter((item) => item.group === group);
        return (
          <section key={group} className="space-y-3.5">
            <div className="flex flex-wrap items-end justify-between gap-2 border-b border-border pb-2.5">
              <div>
                <h2 className="font-bold text-base text-text">
                  {meta?.title ?? group}
                </h2>
                <p className="text-xs text-text-muted mt-0.5">
                  {meta?.subtitle}
                </p>
              </div>
              <span className="text-[11px] font-medium text-text-muted bg-panel px-2.5 py-1 rounded-full border border-border">
                {meta?.badge}
              </span>
            </div>

            <div className="grid gap-3.5 sm:grid-cols-2 xl:grid-cols-3">
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

      {/* Recommended Workflow Footer */}
      <div className="rounded-xl border border-border bg-panel/40 p-4 sm:p-5">
        <div className="flex items-start gap-3">
          <div className="p-2 rounded-lg bg-primary/10 text-primary shrink-0">
            <CircleHelp size={17} />
          </div>
          <div className="space-y-2 text-xs text-text-muted leading-relaxed">
            <p className="font-semibold text-text text-sm">
              Alur Pengelolaan Master Data yang Disarankan
            </p>
            <p>
              1. Siapkan <strong className="text-text">Unit Kerja</strong> dan{" "}
              <strong className="text-text">Jabatan</strong> terlebih dahulu
              sebelum mendaftarkan akun <strong className="text-text">User</strong>.{" "}
              2. Sesuaikan <strong className="text-text">Jenis Surat</strong>,{" "}
              <strong className="text-text">Klasifikasi Arsip</strong>,{" "}
              <strong className="text-text">Template Surat (.docx)</strong>, dan{" "}
              <strong className="text-text">Watermark PDF</strong> sesuai
              standar tata naskah perusahaan.{" "}
              3. Gunakan tombol <strong className="text-text">Nonaktifkan</strong>{" "}
              bila data sudah tidak dipakai agar histori surat dan jejak audit
              tetap terjaga.
            </p>
          </div>
        </div>
      </div>
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
      className={`group text-left p-4 sm:p-5 rounded-xl border transition-all flex flex-col justify-between focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary ${
        allowed
          ? "border-border bg-surface hover:border-primary/40 hover:shadow-md"
          : "border-border/80 bg-surface/60 opacity-85 hover:opacity-100 hover:border-border"
      }`}
    >
      <div>
        <div className="flex items-center justify-between gap-2 mb-3.5">
          <span
            className={`rounded-xl p-2.5 transition-colors ${
              allowed
                ? "bg-primary/10 text-primary group-hover:bg-primary group-hover:text-white"
                : "bg-panel text-text-muted"
            }`}
          >
            <category.icon size={20} />
          </span>

          <span
            className={`inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-[11px] font-semibold ${badgeTone}`}
          >
            <BadgeIcon size={12} />
            {label}
          </span>
        </div>

        <div className="flex items-center justify-between gap-2">
          <h3 className="text-base font-bold text-text group-hover:text-primary transition-colors">
            {category.label}
          </h3>
          {allowed ? (
            <ArrowUpRight
              size={16}
              className="text-text-muted group-hover:text-primary group-hover:translate-x-0.5 group-hover:-translate-y-0.5 transition-transform shrink-0"
            />
          ) : null}
        </div>

        <p className="text-xs text-text-muted mt-1.5 leading-relaxed">
          {category.description}
        </p>
      </div>

      {category.highlights && category.highlights.length > 0 && (
        <div className="mt-4 pt-3 border-t border-border/70 flex flex-wrap gap-1.5">
          {category.highlights.map((tag) => (
            <span
              key={tag}
              className="rounded-md bg-panel/80 px-2 py-0.5 text-[11px] font-medium text-text-muted"
            >
              {tag}
            </span>
          ))}
        </div>
      )}
    </button>
  );
}
