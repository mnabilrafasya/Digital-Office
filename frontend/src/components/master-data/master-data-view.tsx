"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import {
  QueryClient,
  QueryClientProvider,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import {
  ChevronRight,
  LayoutGrid,
  LockKeyhole,
  ArrowLeft,
  Building2,
  ShieldCheck,
  Stamp,
  RotateCcw,
} from "lucide-react";
import type { AuthUser } from "@/lib/auth";
import { masterRequest, type MasterList } from "@/lib/api/master-data";
import { invalidateTemplateCache } from "@/lib/templates/letter-templates";
import { RESOURCES } from "./config";
import { ResourcePanel } from "./resource-panel";
import { CompanySettingsPanel } from "./company-settings-panel";
import { MasterOverview } from "./master-overview";
import { CATEGORIES, canOpen, accessLabel } from "./categories";
import { LookupPanel } from "./lookup-panel";
import { TemplatesPanel } from "./templates-panel";

function MasterDataContent({ user }: { user: AuthUser }) {
  const search = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const cache = useQueryClient();
  const isSuperadmin = user.permissions.includes("COMPANY_MANAGE");
  const active = CATEGORIES.find((item) => item.key === search.get("tab"));

  const companiesQuery = useQuery({
    queryKey: ["master", "company-switcher"],
    queryFn: () =>
      masterRequest<MasterList>("/admin/companies?page=1&pageSize=100"),
    enabled: isSuperadmin,
  });

  const companies = (companiesQuery.data?.data ?? []).map((row) => ({
    code: String(row.code ?? ""),
    name: String(row.name ?? ""),
    isActive: Boolean(row.isActive),
  }));

  const requestedCode = (search.get("company") ?? "").trim().toUpperCase();
  const matchedCompany = isSuperadmin && requestedCode
    ? companies.find((c) => c.code === requestedCode)
    : undefined;
  const activeCompany = isSuperadmin && requestedCode
    ? {
        code: matchedCompany?.code ?? requestedCode,
        name:
          matchedCompany?.name ??
          (requestedCode === user.company.code
            ? user.company.name
            : requestedCode),
      }
    : user.company;
  const isCrossCompany =
    isSuperadmin && activeCompany.code !== user.company.code;

  function updateUrl(mutator: (params: URLSearchParams) => void) {
    const params = new URLSearchParams(search.toString());
    mutator(params);
    router.replace(`${pathname}${params.size ? `?${params}` : ""}`, {
      scroll: false,
    });
  }

  function select(key: string) {
    updateUrl((params) => {
      if (key === "overview") {
        params.delete("tab");
      } else {
        params.set("tab", key);
      }
    });
  }

  function selectCompany(code: string, nextTab?: string) {
    const normalized = code.trim().toUpperCase() || user.company.code;
    updateUrl((params) => {
      if (normalized === user.company.code) {
        params.delete("company");
      } else {
        params.set("company", normalized);
      }
      if (nextTab) {
        if (nextTab === "overview") {
          params.delete("tab");
        } else {
          params.set("tab", nextTab);
        }
      }
    });
    invalidateTemplateCache();
    void cache.invalidateQueries({ queryKey: ["master"] });
  }

  const resource = RESOURCES.find((item) => item.key === active?.key);
  const allowed = active ? canOpen(active, user.permissions) : true;
  const ActiveIcon = active?.icon ?? LayoutGrid;

  return (
    <div className="text-text space-y-6 min-w-0">
      {/* Page Header */}
      <header className="space-y-3">
        <nav
          aria-label="Breadcrumb"
          className="flex items-center gap-1.5 text-xs text-text-muted"
        >
          <Link href="/dashboard" className="hover:text-primary transition-colors">
            Beranda
          </Link>
          <ChevronRight size={12} />
          <button
            type="button"
            onClick={() => select("overview")}
            className={
              active
                ? "hover:text-primary transition-colors"
                : "text-text font-semibold"
            }
          >
            Master Data
          </button>
          {active && (
            <>
              <ChevronRight size={12} />
              <span className="text-text font-semibold">{active.label}</span>
            </>
          )}
        </nav>

        <div className="flex flex-wrap justify-between items-start gap-4">
          <div className="flex items-start gap-3.5">
            {active && (
              <button
                type="button"
                onClick={() => select("overview")}
                title="Kembali ke ringkasan Master Data"
                aria-label="Kembali ke ringkasan Master Data"
                className="mt-0.5 p-2.5 rounded-xl border border-border bg-surface hover:border-primary/40 hover:text-primary transition-colors shrink-0 shadow-2xs"
              >
                <ArrowLeft size={18} />
              </button>
            )}
            <div>
              <div className="flex flex-wrap items-center gap-2.5">
                <h1 className="text-2xl font-bold tracking-tight">
                  {active?.label ?? "Master Data"}
                </h1>
                {active && (
                  <span
                    className={`rounded-full px-2.5 py-0.5 text-[11px] font-semibold ${
                      accessLabel(active, user.permissions) === "Kelola data"
                        ? "bg-status-success-bg text-status-success-text"
                        : accessLabel(active, user.permissions) === "Referensi"
                          ? "bg-status-info-bg text-status-info-text"
                          : "bg-panel text-text-muted"
                    }`}
                  >
                    {accessLabel(active, user.permissions)}
                  </span>
                )}
              </div>
              <p className="text-sm text-text-muted mt-1">
                {active?.description ??
                  "Pusat pengelolaan struktur organisasi, akun pengguna, komponen surat, template, dan watermark naskah resmi."}
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {active?.key !== "settings" && (
              <button
                type="button"
                onClick={() => select("settings")}
                className="inline-flex items-center gap-1.5 rounded-xl border border-border bg-surface hover:border-primary/40 px-3 py-2 text-xs font-semibold text-text hover:text-primary transition-colors shadow-2xs"
              >
                <Stamp size={14} className="text-primary" />
                Watermark & Nomor Surat
              </button>
            )}
            <div className="inline-flex items-center gap-2 border border-border rounded-xl bg-surface px-3 py-1.5 text-xs text-text-muted shadow-2xs">
              <Building2 size={14} className="text-primary shrink-0" />
              {isSuperadmin ? (
                <label className="inline-flex items-center gap-1.5">
                  <span className="sr-only">Pilih perusahaan</span>
                  <select
                    aria-label="Pilih perusahaan yang dikelola"
                    value={activeCompany.code}
                    onChange={(e) => selectCompany(e.target.value)}
                    className="bg-transparent font-semibold text-text focus:outline-none cursor-pointer pr-1 max-w-52 truncate"
                  >
                    {companies.length > 0 ? (
                      companies.map((c) => (
                        <option key={c.code} value={c.code}>
                          {c.code} — {c.name}
                          {!c.isActive ? " (Nonaktif)" : ""}
                        </option>
                      ))
                    ) : (
                      <option value={activeCompany.code}>
                        {activeCompany.code} — {activeCompany.name}
                      </option>
                    )}
                  </select>
                </label>
              ) : (
                <span className="font-semibold text-text">
                  {user.company.code}
                </span>
              )}
              <span className="h-3.5 w-px bg-border" />
              <ShieldCheck
                size={14}
                className="text-status-success-text shrink-0"
              />
              <span className="whitespace-nowrap">{user.role.name}</span>
            </div>
          </div>
        </div>
        {isCrossCompany && active?.key !== "companies" && (
          <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-primary/25 bg-primary/5 px-3.5 py-2 text-xs">
            <div className="flex items-center gap-2 text-text">
              <Building2 size={14} className="text-primary shrink-0" />
              <span>
                Mode Superadmin: Anda sedang mengelola master data perusahaan{" "}
                <strong className="text-primary">
                  {activeCompany.code} — {activeCompany.name}
                </strong>
              </span>
            </div>
            <button
              type="button"
              onClick={() => selectCompany(user.company.code)}
              className="inline-flex items-center gap-1 rounded-lg border border-border bg-surface px-2.5 py-1 text-[11px] font-semibold text-text hover:text-primary hover:border-primary/40 transition-colors"
            >
              <RotateCcw size={12} />
              Kembali ke {user.company.code}
            </button>
          </div>
        )}
      </header>

      {!active ? (
        <MasterOverview
          user={user}
          activeCompany={activeCompany}
          onSelect={select}
        />
      ) : (
        <div className="space-y-4">
          {/* Mobile / Tablet Quick Category Strip */}
          <div className="xl:hidden flex items-center gap-1.5 overflow-x-auto pb-1 no-scrollbar">
            <button
              type="button"
              onClick={() => select("overview")}
              className="inline-flex items-center gap-1.5 rounded-lg border border-border bg-surface px-3 py-2 text-xs font-medium text-text-muted hover:text-text shrink-0"
            >
              <LayoutGrid size={14} />
              Ringkasan
            </button>
            {CATEGORIES.map((item) => {
              const isCurrent = item.key === active.key;
              return (
                <button
                  key={item.key}
                  type="button"
                  onClick={() => select(item.key)}
                  aria-current={isCurrent ? "page" : undefined}
                  className={`inline-flex items-center gap-1.5 rounded-lg border px-3 py-2 text-xs shrink-0 transition-colors ${
                    isCurrent
                      ? "border-primary bg-primary text-white font-semibold"
                      : "border-border bg-surface text-text-muted hover:text-text"
                  }`}
                >
                  <item.icon size={14} />
                  <span>{item.label}</span>
                  {!canOpen(item, user.permissions) && (
                    <LockKeyhole size={11} className="opacity-75" />
                  )}
                </button>
              );
            })}
          </div>

          <div className="flex flex-col xl:flex-row items-start gap-4">
            {/* Desktop Sticky Navigation Sidebar */}
            <aside className="hidden xl:block w-56 shrink-0 rounded-xl border border-border bg-surface p-2 sticky top-20 shadow-2xs">
              <button
                type="button"
                onClick={() => select("overview")}
                className="flex items-center gap-2.5 w-full text-xs font-semibold text-text-muted hover:text-text hover:bg-panel rounded-lg px-3 py-2.5 transition-colors"
              >
                <LayoutGrid size={15} className="text-primary" />
                Semua Master Data
              </button>

              <div className="mt-1 space-y-3">
                {["Organisasi", "Persuratan", "Pengaturan"].map((group) => (
                  <div key={group} className=" border-t border-border/60 pt-2.5">
                    <p className="px-3 pb-1.5 text-[10px] uppercase tracking-wider font-bold text-text-muted">
                      {group}
                    </p>
                    <div className="space-y-0.5">
                      {CATEGORIES.filter((item) => item.group === group).map(
                        (item) => {
                          const isCurrent = item.key === active.key;
                          const itemAllowed = canOpen(item, user.permissions);
                          return (
                            <button
                              key={item.key}
                              type="button"
                              onClick={() => select(item.key)}
                              aria-current={isCurrent ? "page" : undefined}
                              className={`w-full text-left flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs transition-colors ${
                                isCurrent
                                  ? "bg-primary/10 text-primary font-semibold"
                                  : "text-text-muted hover:bg-panel hover:text-text"
                              }`}
                            >
                              <item.icon
                                size={15}
                                className={
                                  isCurrent ? "text-primary" : "text-text-muted"
                                }
                              />
                              <span className="flex-1 truncate">
                                {item.label}
                              </span>
                              {!itemAllowed && (
                                <LockKeyhole
                                  size={12}
                                  className="text-text-muted/70 shrink-0"
                                />
                              )}
                            </button>
                          );
                        },
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </aside>

            {/* Main Workspace Card */}
            <div className="flex-1 min-w-0 w-full rounded-xl border border-border bg-surface p-4 sm:p-5 shadow-2xs">
              {!allowed ? (
                <div className="text-center py-16 max-w-md mx-auto">
                  <span className="inline-flex rounded-2xl p-4 bg-panel text-text-muted mb-4 border border-border">
                    <ActiveIcon size={28} />
                  </span>
                  <h2 className="text-lg font-bold">
                    Akses Pengelolaan Terbatas
                  </h2>
                  <p className="text-sm text-text-muted mt-2 leading-relaxed">
                    {active.key === "companies"
                      ? "Master perusahaan dikelola khusus oleh Super Administrator."
                      : `Modul ${active.label} dikelola oleh administrator perusahaan yang memiliki izin terkait.`}{" "}
                    Akun Anda saat ini masuk sebagai{" "}
                    <strong className="text-text">{user.role.name}</strong> di{" "}
                    <strong className="text-text">{user.company.name}</strong>.
                  </p>
                  <button
                    type="button"
                    onClick={() => select("overview")}
                    className="inline-flex items-center gap-2 mt-6 rounded-lg border border-border bg-panel hover:border-primary/40 px-4 py-2 text-xs font-semibold text-text hover:text-primary transition-colors"
                  >
                    <ArrowLeft size={14} />
                    Kembali ke Semua Master Data
                  </button>
                </div>
              ) : active.key === "templates" ? (
                <TemplatesPanel
                  key={`templates-${activeCompany.code}`}
                  permissions={user.permissions}
                  companyCode={activeCompany.code}
                />
              ) : active.category ? (
                <LookupPanel
                  key={`${active.category}-${activeCompany.code}`}
                  category={active.category}
                  label={active.label}
                  permissions={user.permissions}
                  userId={user.id}
                  companyCode={activeCompany.code}
                />
              ) : active.key === "settings" ? (
                <CompanySettingsPanel
                  key={`settings-${activeCompany.code}`}
                  companyCode={activeCompany.code}
                />
              ) : resource ? (
                <ResourcePanel
                  key={`${resource.key}-${activeCompany.code}`}
                  resource={resource}
                  userId={user.id}
                  companyCode={activeCompany.code}
                  ownCompanyCode={user.company.code}
                  onManageCompany={
                    isSuperadmin
                      ? (code) => selectCompany(code, "units")
                      : undefined
                  }
                  canManageSuperadmin={isSuperadmin}
                />
              ) : null}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export function MasterDataView({ user }: { user: AuthUser }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { retry: false, refetchOnWindowFocus: false },
        },
      }),
  );

  return (
    <QueryClientProvider client={client}>
      <MasterDataContent user={user} />
    </QueryClientProvider>
  );
}
