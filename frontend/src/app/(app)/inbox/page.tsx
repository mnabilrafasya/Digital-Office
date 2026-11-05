"use client";

// ─────────────────────────────────────────────
// app/(app)/inbox/page.tsx
// Halaman Kotak Masuk (Inbox) bergaya Gmail:
// - Omnibox search + filter lanjutan (Asal Surat, Sifat, Kecepatan, Berbintang)
// - Tab Kategori dengan penghitung real-time (Utama, Belum Dibaca, Sudah Dibaca, Didisposisikan, Dialihkan)
// - Daftar surat padat & nyaman dengan aksi cepat saat hover, seleksi checkbox, dan penanda bintang
// - Panel Baca Samping (Split Reading Pane) interaktif
// ─────────────────────────────────────────────

import { useState, useEffect, useCallback, useMemo, Suspense } from "react";
import { useRouter, useSearchParams, usePathname } from "next/navigation";
import {
  Inbox,
  Loader2,
  AlertCircle,
  MailOpen,
  Keyboard,
} from "lucide-react";
import {
  getInboxList,
  type InboxItem,
  type InboxCounts,
  type InboxItemDetail,
} from "@/lib/api/inbox";
import {
  InboxToolbar,
  DEFAULT_CLIENT_FILTERS,
  type TabStatus,
  type DensityMode,
  type InboxClientFilters,
} from "@/components/inbox/inbox-toolbar";
import { InboxMessageRow } from "@/components/inbox/inbox-message-row";
import { InboxReadingPane } from "@/components/inbox/inbox-reading-pane";
import { DisposisiModal } from "@/components/inbox/disposisi-modal";
import { RedirectModal } from "@/components/inbox/redirect-modal";
import { cn } from "@/lib/utils";

const PAGE_SIZE = 20;
const STARRED_STORAGE_KEY = "digital-office:inbox-starred-ids";

function loadStarredIds(): Set<string> {
  if (typeof window === "undefined") return new Set();
  try {
    const raw = window.localStorage.getItem(STARRED_STORAGE_KEY);
    if (!raw) return new Set();
    const parsed = JSON.parse(raw) as unknown;
    return Array.isArray(parsed) ? new Set(parsed.map(String)) : new Set();
  } catch {
    return new Set();
  }
}

function saveStarredIds(ids: Set<string>) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(
      STARRED_STORAGE_KEY,
      JSON.stringify(Array.from(ids))
    );
  } catch {
    // Abaikan bila storage penuh
  }
}

function InboxView() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const [activeTab, setActiveTab] = useState<TabStatus>(
    (searchParams.get("status") as TabStatus) ?? ""
  );
  const [query, setQuery] = useState(searchParams.get("q") ?? "");
  const [debouncedQuery, setDebouncedQuery] = useState(query);
  const [page, setPage] = useState(() => {
    const p = Number(searchParams.get("page") ?? "1");
    return Number.isFinite(p) && p > 0 ? p : 1;
  });

  const [items, setItems] = useState<InboxItem[]>([]);
  const [counts, setCounts] = useState<InboxCounts>({
    UNREAD: 0,
    READ: 0,
    DISPOSED: 0,
    REDIRECTED: 0,
  });
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [reloadTick, setReloadTick] = useState(0);

  // State antarmuka ala Gmail
  const [splitView, setSplitView] = useState(true);
  const [density, setDensity] = useState<DensityMode>("comfortable");
  const [showFilterPanel, setShowFilterPanel] = useState(false);
  const [clientFilters, setClientFilters] = useState<InboxClientFilters>(
    DEFAULT_CLIENT_FILTERS
  );
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set());
  const [starredIds, setStarredIds] = useState<Set<string>>(() =>
    loadStarredIds()
  );
  const [activeItemId, setActiveItemId] = useState<string | null>(null);
  const [modalState, setModalState] = useState<{
    itemId: string;
    type: "disposisi" | "redirect";
  } | null>(null);

  // Debounce input pencarian (250ms) agar pencarian halus
  useEffect(() => {
    const t = setTimeout(() => {
      setDebouncedQuery(query.trim());
    }, 250);
    return () => clearTimeout(t);
  }, [query]);

  // Sinkronkan URL query string
  useEffect(() => {
    const params = new URLSearchParams();
    if (activeTab) params.set("status", activeTab);
    if (debouncedQuery) params.set("q", debouncedQuery);
    if (page > 1) params.set("page", String(page));
    const qs = params.toString();
    const targetUrl = qs ? `${pathname}?${qs}` : pathname;
    window.history.replaceState(null, "", targetUrl);
  }, [activeTab, debouncedQuery, page, pathname]);

  // Fetch daftar inbox dari backend
  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(() => {
      if (cancelled) return;
      setLoading(true);
      setError(null);

      getInboxList({
        status: activeTab || undefined,
        q: debouncedQuery || undefined,
        page,
        pageSize: PAGE_SIZE,
      })
        .then((res) => {
          if (cancelled) return;
          setItems(res.data);
          setCounts(res.counts);
          setTotal(res.meta.total);
        })
        .catch((e: unknown) => {
          if (!cancelled) {
            setError(
              e instanceof Error ? e.message : "Gagal memuat kotak masuk."
            );
          }
        })
        .finally(() => {
          if (!cancelled) setLoading(false);
        });
    }, 0);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [activeTab, debouncedQuery, page, reloadTick]);

  // Hitung jumlah putaran per surat (letter.id) dan gabungkan baris surat yang sama
  // seperti thread percakapan Gmail agar surat yang dikembalikan/didisposisi ulang
  // tidak tampil ganda bertumpuk di daftar kotak masuk.
  const turnCountByLetterId = useMemo(() => {
    const map = new Map<string, number>();
    for (const it of items) {
      map.set(it.letter.id, (map.get(it.letter.id) ?? 0) + 1);
    }
    return map;
  }, [items]);

  // Penyaringan tambahan sisi klien (Asal Surat, Sifat, Kecepatan, Berbintang) + Threading per letter.id
  const visibleItems = useMemo(() => {
    const seenLetters = new Set<string>();
    return items.filter((item) => {
      // Hanya tampilkan putaran terbaru untuk surat (letter.id) yang sama
      if (seenLetters.has(item.letter.id)) {
        return false;
      }
      if (clientFilters.starredOnly && !starredIds.has(item.id)) {
        return false;
      }
      if (clientFilters.source !== "ALL") {
        if (
          clientFilters.source === "DIRECT_TO" &&
          !(
            item.source.type === "DIRECT" && item.source.recipientType === "TO"
          )
        ) {
          return false;
        }
        if (
          clientFilters.source === "DIRECT_CC" &&
          !(
            item.source.type === "DIRECT" && item.source.recipientType === "CC"
          )
        ) {
          return false;
        }
        if (
          clientFilters.source === "DISPOSITION" &&
          !(
            item.source.type === "FORWARD" &&
            item.source.forwardType === "DISPOSITION"
          )
        ) {
          return false;
        }
        if (
          clientFilters.source === "REDIRECT" &&
          !(
            item.source.type === "FORWARD" &&
            item.source.forwardType === "REDIRECT"
          )
        ) {
          return false;
        }
      }
      if (
        clientFilters.confidentiality !== "ALL" &&
        item.letter.confidentiality?.code !== clientFilters.confidentiality
      ) {
        return false;
      }
      if (
        clientFilters.urgency !== "ALL" &&
        item.letter.urgency?.code !== clientFilters.urgency
      ) {
        return false;
      }
      seenLetters.add(item.letter.id);
      return true;
    });
  }, [items, clientFilters, starredIds]);

  const activeIndex = useMemo(
    () => visibleItems.findIndex((it) => it.id === activeItemId),
    [visibleItems, activeItemId]
  );

  const toggleStar = useCallback((id: string) => {
    setStarredIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      saveStarredIds(next);
      return next;
    });
  }, []);

  const toggleSelect = useCallback((id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const allVisibleSelected =
    visibleItems.length > 0 &&
    visibleItems.every((it) => selectedIds.has(it.id));

  const handleToggleSelectAll = useCallback(() => {
    if (allVisibleSelected) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(visibleItems.map((it) => it.id)));
    }
  }, [allVisibleSelected, visibleItems]);

  // Saat surat dibuka di Panel Baca, perbarui status baris UNREAD -> READ secara reaktif
  const handleItemLoaded = useCallback((detail: InboxItemDetail) => {
    setItems((prev) => {
      const target = prev.find((it) => it.id === detail.id);
      if (!target || target.status === detail.status) return prev;
      const wasUnread = target.status === "UNREAD" && detail.status === "READ";
      if (wasUnread) {
        setCounts((c) => ({
          ...c,
          UNREAD: Math.max(0, c.UNREAD - 1),
          READ: c.READ + 1,
        }));
      }
      return prev.map((it) =>
        it.id === detail.id
          ? { ...it, status: detail.status, readAt: detail.readAt }
          : it
      );
    });
  }, []);

  // Pintasan keyboard ala Gmail: j (berikutnya), k (sebelumnya), o/Enter (buka), u/Escape (tutup panel), s (bintang)
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const target = e.target as HTMLElement | null;
      if (
        target?.tagName === "INPUT" ||
        target?.tagName === "TEXTAREA" ||
        target?.tagName === "SELECT" ||
        target?.isContentEditable ||
        modalState !== null
      ) {
        return;
      }

      if (e.key === "j" || e.key === "ArrowDown") {
        if (visibleItems.length === 0) return;
        e.preventDefault();
        const nextIdx =
          activeIndex < 0
            ? 0
            : Math.min(visibleItems.length - 1, activeIndex + 1);
        setActiveItemId(visibleItems[nextIdx].id);
      } else if (e.key === "k" || e.key === "ArrowUp") {
        if (visibleItems.length === 0) return;
        e.preventDefault();
        const prevIdx = activeIndex <= 0 ? 0 : activeIndex - 1;
        setActiveItemId(visibleItems[prevIdx].id);
      } else if (e.key === "u" || e.key === "Escape") {
        if (activeItemId) {
          e.preventDefault();
          setActiveItemId(null);
        }
      } else if (e.key === "s" && activeItemId) {
        e.preventDefault();
        toggleStar(activeItemId);
      } else if (e.key === "o" && activeItemId && !splitView) {
        e.preventDefault();
        router.push(`/inbox/${activeItemId}`);
      }
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [
    visibleItems,
    activeIndex,
    activeItemId,
    splitView,
    modalState,
    router,
    toggleStar,
  ]);

  function handleTabChange(tab: TabStatus) {
    setActiveTab(tab);
    setPage(1);
    setSelectedIds(new Set());
  }

  function handleRowClick(item: InboxItem) {
    if (splitView) {
      setActiveItemId(item.id);
    } else {
      router.push(`/inbox/${item.id}`);
    }
  }

  const firstSelectedId = useMemo(
    () => Array.from(selectedIds)[0] ?? null,
    [selectedIds]
  );

  return (
    <div className="space-y-3">
      {/* ── Header Ringkas ── */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg bg-primary/10 text-primary flex items-center justify-center">
            <Inbox size={18} />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-lg font-bold text-text leading-tight">
                Kotak Masuk (Inbox)
              </h1>
              {counts.UNREAD > 0 && (
                <span className="bg-status-danger-bg text-status-danger-text text-xs font-bold px-2 py-0.5 rounded-full">
                  {counts.UNREAD} belum dibaca
                </span>
              )}
            </div>
            <p className="text-xs text-text-muted">
              Surat resmi terkirim, tembusan (CC), disposisi, dan pengalihan masuk
            </p>
          </div>
        </div>
      </div>

      {/* ── Kontainer Utama Bergaya Gmail ── */}
      <div className="border border-border rounded-2xl bg-surface shadow-xs overflow-hidden flex flex-col">
        {/* Toolbar Pencarian, Filter, & Tab Kategori */}
        <InboxToolbar
          query={query}
          onQueryChange={(val) => {
            setQuery(val);
            setPage(1);
          }}
          activeTab={activeTab}
          onTabChange={handleTabChange}
          counts={counts}
          clientFilters={clientFilters}
          onClientFiltersChange={setClientFilters}
          showFilterPanel={showFilterPanel}
          onToggleFilterPanel={() => setShowFilterPanel((v) => !v)}
          loading={loading}
          onRefresh={() => setReloadTick((t) => t + 1)}
          page={page}
          pageSize={PAGE_SIZE}
          total={total}
          onPageChange={setPage}
          splitView={splitView}
          onToggleSplitView={() => setSplitView((v) => !v)}
          density={density}
          onToggleDensity={() =>
            setDensity((d) => (d === "comfortable" ? "compact" : "comfortable"))
          }
          selectedCount={selectedIds.size}
          allVisibleSelected={allVisibleSelected}
          onToggleSelectAll={handleToggleSelectAll}
          onClearSelection={() => setSelectedIds(new Set())}
          onOpenSelectedFirst={
            firstSelectedId
              ? () => {
                  if (splitView) setActiveItemId(firstSelectedId);
                  else router.push(`/inbox/${firstSelectedId}`);
                }
              : undefined
          }
          onDisposeSelectedFirst={
            firstSelectedId
              ? () =>
                  setModalState({
                    itemId: firstSelectedId,
                    type: "disposisi",
                  })
              : undefined
          }
        />

        {/* Workspace: Daftar Pesan + Panel Baca Samping (jika splitView aktif) */}
        <div
          className={cn(
            "grid min-h-[68vh] w-full min-w-0 max-w-full overflow-hidden",
            splitView && activeItemId
              ? "grid-cols-1 lg:grid-cols-[minmax(320px,42%)_minmax(0,1fr)] xl:grid-cols-[minmax(360px,45%)_minmax(0,1fr)] divide-y lg:divide-y-0 lg:divide-x divide-border"
              : "grid-cols-1"
          )}
        >
          {/* ─ Kolom Kiri: Daftar Pesan ─ */}
          <div className="flex flex-col min-w-0">
            {loading ? (
              <div className="flex-1 flex flex-col items-center justify-center py-20 gap-2.5 text-text-muted">
                <Loader2 size={22} className="animate-spin text-primary" />
                <span className="text-sm font-medium">
                  Memuat surat masuk...
                </span>
              </div>
            ) : error ? (
              <div className="flex-1 flex flex-col items-center justify-center py-20 gap-3 text-center px-4">
                <AlertCircle size={30} className="text-status-danger-text" />
                <p className="text-sm text-text-muted">{error}</p>
                <button
                  type="button"
                  onClick={() => setReloadTick((t) => t + 1)}
                  className="px-3.5 py-1.5 rounded-lg bg-primary text-white text-xs font-semibold hover:bg-primary-dark transition-colors"
                >
                  Coba Lagi
                </button>
              </div>
            ) : visibleItems.length === 0 ? (
              <div className="flex-1 flex flex-col items-center justify-center py-20 gap-2.5 text-center px-4">
                <div className="w-12 h-12 rounded-full bg-panel flex items-center justify-center text-text-muted">
                  <MailOpen size={24} />
                </div>
                <p className="text-sm font-semibold text-text">
                  Tidak ada surat yang cocok
                </p>
                <p className="text-xs text-text-muted max-w-xs">
                  {query ||
                  clientFilters.source !== "ALL" ||
                  clientFilters.confidentiality !== "ALL" ||
                  clientFilters.urgency !== "ALL" ||
                  clientFilters.starredOnly
                    ? "Coba ubah kata kunci pencarian atau atur ulang filter lanjutan."
                    : "Kotak masuk Anda pada kategori ini masih kosong."}
                </p>
              </div>
            ) : (
              <div className="flex-1 overflow-y-auto">
                {visibleItems.map((item) => (
                  <InboxMessageRow
                    key={item.id}
                    item={item}
                    turnCount={turnCountByLetterId.get(item.letter.id) ?? 1}
                    selected={selectedIds.has(item.id)}
                    active={activeItemId === item.id}
                    starred={starredIds.has(item.id)}
                    density={density}
                    onToggleSelect={() => toggleSelect(item.id)}
                    onToggleStar={() => toggleStar(item.id)}
                    onClickRow={() => handleRowClick(item)}
                    onOpenFullPage={() => router.push(`/inbox/${item.id}`)}
                    onQuickDispose={() =>
                      setModalState({ itemId: item.id, type: "disposisi" })
                    }
                    onQuickRedirect={() =>
                      setModalState({ itemId: item.id, type: "redirect" })
                    }
                  />
                ))}
              </div>
            )}

            {/* Footer Baris Daftar */}
            <div className="px-4 py-2.5 bg-panel/30 border-t border-border flex items-center justify-between text-xs text-text-muted mt-auto">
              <span>
                Menampilkan{" "}
                <strong className="text-text">{visibleItems.length}</strong>{" "}
                surat di halaman ini (Total:{" "}
                <strong className="text-text">{total}</strong>)
              </span>
              {splitView && !activeItemId && visibleItems.length > 0 && (
                <span className="hidden sm:inline text-[11px]">
                  Klik surat untuk membuka Panel Baca Samping
                </span>
              )}
            </div>
          </div>

          {/* ─ Kolom Kanan: Panel Baca Samping ala Gmail ─ */}
          {splitView && activeItemId && (
            <div className="min-w-0 max-w-full overflow-hidden h-full">
              <InboxReadingPane
                itemId={activeItemId}
                currentIndex={activeIndex}
                totalVisible={visibleItems.length}
                onPrev={() => {
                  if (activeIndex > 0) {
                    setActiveItemId(visibleItems[activeIndex - 1].id);
                  }
                }}
                onNext={() => {
                  if (
                    activeIndex >= 0 &&
                    activeIndex < visibleItems.length - 1
                  ) {
                    setActiveItemId(visibleItems[activeIndex + 1].id);
                  }
                }}
                onClose={() => setActiveItemId(null)}
                onOpenModal={(itemId, type) => setModalState({ itemId, type })}
                onItemLoaded={handleItemLoaded}
                refreshTrigger={reloadTick}
              />
            </div>
          )}
        </div>
      </div>

      {/* ── Modal Disposisi & Pengalihan ── */}
      {modalState?.type === "disposisi" && (
        <DisposisiModal
          itemId={modalState.itemId}
          onClose={() => setModalState(null)}
          onSuccess={() => {
            setModalState(null);
            setReloadTick((t) => t + 1);
          }}
        />
      )}
      {modalState?.type === "redirect" && (
        <RedirectModal
          itemId={modalState.itemId}
          onClose={() => setModalState(null)}
          onSuccess={() => {
            setModalState(null);
            setReloadTick((t) => t + 1);
          }}
        />
      )}
    </div>
  );
}

// ─── Page (bungkus Suspense untuk useSearchParams) ─

export default function InboxPage() {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center min-h-[60vh] gap-2 text-text-muted">
          <Loader2 size={18} className="animate-spin text-primary" />
          <span className="text-sm">Memuat kotak masuk...</span>
        </div>
      }
    >
      <InboxView />
    </Suspense>
  );
}
