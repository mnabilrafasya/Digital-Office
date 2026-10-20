"use client";

// ─────────────────────────────────────────────
// components/daftar-surat/daftar-surat-view.tsx
// Komponen Client utama untuk Daftar Surat:
// Mengelola state URL query params, fetch data, koordinasi tab, filter, tabel, pagination.
// ─────────────────────────────────────────────

import { useCallback, useEffect, useState, useTransition } from "react";
import { useRouter, useSearchParams, usePathname } from "next/navigation";
import Link from "next/link";
import { ChevronRight, Plus } from "lucide-react";

import { listLetters } from "@/lib/api/letters";
import type {
  LetterListItem,
  ListLettersResponse,
} from "@/lib/types";

import { SummaryCards }  from "@/components/daftar-surat/summary-cards";
import { StatusTabs }    from "@/components/daftar-surat/status-tabs";
import { FilterBar }     from "@/components/daftar-surat/filter-bar";
import type { FilterValues } from "@/components/daftar-surat/filter-bar";
import { LettersTable }  from "@/components/daftar-surat/letters-table";
import { Pagination }    from "@/components/daftar-surat/pagination";

// ─── Default state ────────────────────────────

const DEFAULT_PAGE_SIZE = 10;

const EMPTY_COUNTS = {
  ALL: 0, DRAFT: 0, IN_REVIEW: 0, RETURNED: 0,
  IN_APPROVAL: 0, SENT: 0, CANCELLED: 0,
};

const EMPTY_FILTER: FilterValues = {
  q: "", letterTypeId: "", confidentialityId: "", from: "", to: "",
};

// ─── Helpers URL ──────────────────────────────

function buildSearch(params: Record<string, string | number>) {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== "" && v !== 0 && v !== undefined) sp.set(k, String(v));
  }
  return sp.toString() ? `?${sp.toString()}` : "";
}

// ─── Skeleton fallback Suspense ──────────────

export function DaftarSuratSkeleton() {
  return (
    <div className="animate-pulse space-y-5">
      <div className="h-8 w-48 bg-panel rounded" />
      <div className="grid grid-cols-4 gap-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-24 bg-surface border border-border rounded-xl" />
        ))}
      </div>
      <div className="h-10 bg-panel rounded" />
      <div className="h-64 bg-surface border border-border rounded-xl" />
    </div>
  );
}

// ─── Main View ───────────────────────────────

export function DaftarSuratView() {
  const router       = useRouter();
  const pathname     = usePathname();
  const searchParams = useSearchParams();
  const [isPending, startTransition] = useTransition();

  // Baca state dari URL
  const activeTab  = searchParams.get("status") ?? "";
  const page       = Number(searchParams.get("page") ?? "1");
  const pageSize   = Number(searchParams.get("pageSize") ?? String(DEFAULT_PAGE_SIZE));

  // Filter lokal (belum dikirim ke URL sampai klik Terapkan)
  const [pendingFilter, setPendingFilter] = useState<FilterValues>(() => ({
    q:                searchParams.get("q")                ?? "",
    letterTypeId:     searchParams.get("letterTypeId")     ?? "",
    confidentialityId: searchParams.get("confidentialityId") ?? "",
    from:             searchParams.get("from")             ?? "",
    to:               searchParams.get("to")               ?? "",
  }));

  // Data
  const [data,    setData]    = useState<LetterListItem[]>([]);
  const [counts,  setCounts]  = useState(EMPTY_COUNTS);
  const [meta,    setMeta]    = useState({ page: 1, pageSize: DEFAULT_PAGE_SIZE, total: 0 });
  const [loading, setLoading] = useState(true);
  const [error,   setError]   = useState(false);

  // ── Fetch ─────────────────────────────────

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(false);
    try {
      const resp = await listLetters({
        status: (activeTab as ListLettersResponse["data"][0]["status"]) || undefined,
        q:                searchParams.get("q")                ?? "",
        letterTypeId:     searchParams.get("letterTypeId")     ?? "",
        confidentialityId: searchParams.get("confidentialityId") ?? "",
        from:             searchParams.get("from")             ?? "",
        to:               searchParams.get("to")               ?? "",
        page,
        pageSize,
      });
      setData(resp.data);
      setCounts(resp.counts);
      setMeta(resp.meta);
    } catch {
      setError(true);
    } finally {
      setLoading(false);
    }
  }, [searchParams, activeTab, page, pageSize]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // ── URL helpers ───────────────────────────

  function pushUrl(overrides: Record<string, string | number>) {
    const current: Record<string, string | number> = {};
    searchParams.forEach((v, k) => { current[k] = v; });
    const merged = { ...current, ...overrides };
    startTransition(() => {
      router.push(`${pathname}${buildSearch(merged)}`);
    });
  }

  // ── Handler tab ───────────────────────────

  function handleTabChange(value: string) {
    pushUrl({ status: value, page: 1 });
    // Reset filter lokal ke nilai dari URL (filter saat ini)
    setPendingFilter({
      q:                searchParams.get("q")                ?? "",
      letterTypeId:     searchParams.get("letterTypeId")     ?? "",
      confidentialityId: searchParams.get("confidentialityId") ?? "",
      from:             searchParams.get("from")             ?? "",
      to:               searchParams.get("to")               ?? "",
    });
  }

  // ── Handler filter ────────────────────────

  function handleApply() {
    pushUrl({
      ...pendingFilter,
      status: activeTab,
      page: 1,
    });
  }

  function handleReset() {
    setPendingFilter(EMPTY_FILTER);
    pushUrl({ status: activeTab, page: 1, q: "", letterTypeId: "", confidentialityId: "", from: "", to: "" });
  }

  // ── Pagination ────────────────────────────

  function handlePageChange(p: number) {
    pushUrl({ page: p });
  }

  function handlePageSizeChange(size: number) {
    pushUrl({ pageSize: size, page: 1 });
  }

  // ── Apakah ada filter aktif ───────────────

  const isFiltered =
    !!(searchParams.get("q") || searchParams.get("letterTypeId") ||
       searchParams.get("confidentialityId") || searchParams.get("from") || searchParams.get("to"));

  // ─── Render ────────────────────────────────

  return (
    <>
      {/* ── Header ── */}
      <div className="mb-5">
        {/* Breadcrumb */}
        <nav className="flex items-center gap-1 text-xs text-text-muted mb-2">
          <Link href="/" className="hover:text-text transition-colors">Beranda</Link>
          <ChevronRight size={12} />
          <span>Penciptaan Surat</span>
          <ChevronRight size={12} />
          <span className="text-text font-medium">Daftar Surat</span>
        </nav>

        {/* Judul + CTA */}
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-text">Daftar Surat</h1>
            <p className="text-text-muted text-sm mt-0.5">
              Kelola, pantau proses review &amp; persetujuan, serta tindak lanjut surat yang Anda susun.
            </p>
          </div>

          <Link
            href="/surat/baru"
            id="btn-buat-surat-baru"
            className="shrink-0 inline-flex items-center gap-1.5 h-10 px-4 rounded-lg text-sm font-semibold bg-primary text-white hover:bg-primary/90 transition-colors shadow-sm"
          >
            <Plus size={15} />
            Buat Surat Baru
          </Link>
        </div>
      </div>

      {/* ── Kartu ringkasan ── */}
      <SummaryCards counts={counts} loading={loading && data.length === 0} />

      {/* ── Tab status ── */}
      <StatusTabs
        counts={counts}
        activeTab={activeTab}
        onTabChange={handleTabChange}
      />

      {/* ── Filter ── */}
      <FilterBar
        values={pendingFilter}
        onChange={setPendingFilter}
        onApply={handleApply}
        onReset={handleReset}
      />

      {/* ── Tabel ── */}
      <LettersTable
        items={data}
        loading={loading}
        error={error}
        isFiltered={isFiltered || !!activeTab}
        onRetry={fetchData}
      />

      {/* ── Pagination ── */}
      {!loading && !error && meta.total > 0 && (
        <Pagination
          page={meta.page}
          pageSize={meta.pageSize}
          total={meta.total}
          onPageChange={handlePageChange}
          onPageSizeChange={handlePageSizeChange}
        />
      )}
    </>
  );
}
