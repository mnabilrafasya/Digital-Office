"use client";

// app/(app)/inbox/page.tsx
// Halaman daftar Inbox

import { useState, useEffect, useCallback, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import {
  Inbox, Loader2, AlertCircle, Mail, MailOpen,
  Share2, ArrowRight, Search, RefreshCw, ChevronRight,
} from "lucide-react";
import {
  getInboxList, inboxStatusLabel, inboxSourceLabel,
} from "@/lib/api/inbox";
import type { InboxItem, InboxCounts } from "@/lib/api/inbox";
import { cn } from "@/lib/utils";

// ─── Tipe ────────────────────────────────────

type TabStatus = "" | "UNREAD" | "READ" | "DISPOSED" | "REDIRECTED";

const TABS: { value: TabStatus; label: string }[] = [
  { value: "",           label: "Semua" },
  { value: "UNREAD",    label: "Belum Dibaca" },
  { value: "READ",      label: "Sudah Dibaca" },
  { value: "DISPOSED",  label: "Didisposisikan" },
  { value: "REDIRECTED",label: "Dialihkan" },
];

// ─── Sub-components ──────────────────────────

function StatusPill({ status }: { status: InboxItem["status"] }) {
  const styles: Record<InboxItem["status"], string> = {
    UNREAD:     "bg-status-danger-bg text-status-danger-text",
    READ:       "bg-panel text-text-muted border border-border",
    DISPOSED:   "bg-status-info-bg text-status-info-text",
    REDIRECTED: "bg-status-warn-bg text-status-warn-text",
  };
  return (
    <span className={cn("text-[10px] font-semibold px-2 py-0.5 rounded-full", styles[status])}>
      {inboxStatusLabel(status)}
    </span>
  );
}

function SourceIcon({ source }: { source: InboxItem["source"] }) {
  if (source.type === "DIRECT") return <Mail size={15} className="text-primary shrink-0" />;
  if (source.forwardType === "DISPOSITION") return <Share2 size={15} className="text-status-info-text shrink-0" />;
  return <ArrowRight size={15} className="text-status-warn-text shrink-0" />;
}

function formatDate(iso: string) {
  return new Intl.DateTimeFormat("id-ID", {
    day: "numeric", month: "short", year: "numeric",
  }).format(new Date(iso));
}

// ─── Inner view (butuh useSearchParams) ──────

function InboxView() {
  const router       = useRouter();
  const searchParams = useSearchParams();

  const [activeTab, setActiveTab] = useState<TabStatus>(
    (searchParams.get("status") as TabStatus) ?? ""
  );
  const [query,   setQuery]   = useState(searchParams.get("q") ?? "");
  const [items,   setItems]   = useState<InboxItem[]>([]);
  const [counts,  setCounts]  = useState<InboxCounts>({ UNREAD: 0, READ: 0, DISPOSED: 0, REDIRECTED: 0 });
  const [total,   setTotal]   = useState(0);
  const [page,    setPage]    = useState(1);
  const [loading, setLoading] = useState(true);
  const [error,   setError]   = useState<string | null>(null);

  const load = useCallback(async (tab: TabStatus, q: string, pg: number) => {
    setLoading(true);
    setError(null);
    try {
      const res = await getInboxList({
        status:   tab || undefined,
        q:        q || undefined,
        page:     pg,
        pageSize: 20,
      });
      setItems(res.data);
      setCounts(res.counts);
      setTotal(res.meta.total);
    } catch (e) {
      setError((e as Error).message ?? "Gagal memuat inbox.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(activeTab, query, page); }, [load, activeTab, query, page]);

  function handleTabChange(tab: TabStatus) {
    setActiveTab(tab);
    setPage(1);
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-2">
          <Inbox size={20} className="text-primary" />
          <h1 className="text-lg font-bold text-text">Kotak Masuk (Inbox)</h1>
          {counts.UNREAD > 0 && (
            <span className="bg-status-danger-bg text-status-danger-text text-xs font-bold px-2 py-0.5 rounded-full">
              {counts.UNREAD} belum dibaca
            </span>
          )}
        </div>
        <button
          type="button"
          onClick={() => load(activeTab, query, page)}
          className="flex items-center gap-1.5 text-xs text-text-muted hover:text-text border border-border rounded-lg px-3 py-1.5 transition-colors"
        >
          <RefreshCw size={13} /> Perbarui
        </button>
      </div>

      {/* Tab filter */}
      <div className="flex gap-1 border-b border-border overflow-x-auto">
        {TABS.map((tab) => {
          const count = tab.value ? counts[tab.value] : undefined;
          return (
            <button
              key={tab.value}
              type="button"
              onClick={() => handleTabChange(tab.value)}
              className={cn(
                "flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px whitespace-nowrap transition-colors",
                activeTab === tab.value
                  ? "border-primary text-primary"
                  : "border-transparent text-text-muted hover:text-text"
              )}
            >
              {tab.label}
              {count != null && count > 0 && (
                <span className={cn(
                  "text-[10px] font-bold px-1.5 py-0.5 rounded-full",
                  activeTab === tab.value
                    ? "bg-primary/15 text-primary"
                    : "bg-panel text-text-muted"
                )}>
                  {count}
                </span>
              )}
            </button>
          );
        })}
      </div>

      {/* Search */}
      <div className="relative max-w-sm">
        <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
        <input
          value={query}
          onChange={(e) => { setQuery(e.target.value); setPage(1); }}
          placeholder="Cari perihal atau nomor surat..."
          className="w-full pl-8 pr-3 py-2 text-sm border border-border rounded-lg bg-surface outline-none focus:border-primary"
        />
      </div>

      {/* Content */}
      {loading ? (
        <div className="flex items-center justify-center py-16 gap-2 text-text-muted">
          <Loader2 size={18} className="animate-spin text-primary" />
          <span className="text-sm">Memuat inbox...</span>
        </div>
      ) : error ? (
        <div className="flex flex-col items-center justify-center py-16 gap-3 text-center">
          <AlertCircle size={28} className="text-status-danger-text" />
          <p className="text-sm text-text-muted">{error}</p>
          <button
            type="button"
            onClick={() => load(activeTab, query, page)}
            className="text-sm text-primary hover:underline"
          >
            Coba lagi
          </button>
        </div>
      ) : items.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 gap-2 text-center border border-dashed border-border rounded-xl">
          <MailOpen size={28} className="text-text-muted" />
          <p className="text-sm text-text-muted">Tidak ada surat masuk.</p>
        </div>
      ) : (
        <div className="border border-border rounded-xl overflow-hidden divide-y divide-border">
          {items.map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => router.push(`/inbox/${item.id}`)}
              className={cn(
                "w-full text-left flex items-start gap-3 px-4 py-4 hover:bg-panel/50 transition-colors",
                item.status === "UNREAD" && "bg-status-info-bg/20"
              )}
            >
              <div className="mt-0.5 shrink-0">
                <SourceIcon source={item.source} />
              </div>
              <div className="flex-1 min-w-0 space-y-1">
                <div className="flex items-center gap-2 flex-wrap">
                  <StatusPill status={item.status} />
                  {item.letter.letterType && (
                    <span className="text-[10px] text-text-muted border border-border rounded-full px-2 py-0.5">
                      {item.letter.letterType.name}
                    </span>
                  )}
                  <span className="text-[10px] text-text-muted">
                    {formatDate(item.receivedAt)}
                  </span>
                </div>
                <p className={cn(
                  "text-sm truncate",
                  item.status === "UNREAD" ? "font-bold text-text" : "font-medium text-text"
                )}>
                  {item.letter.title ?? "(Tanpa perihal)"}
                </p>
                <p className="text-xs text-text-muted">
                  {inboxSourceLabel(item.source)} · {item.letter.senderUnit.name}
                </p>
                {item.letter.letterNumber && (
                  <p className="text-xs font-mono text-text-muted">{item.letter.letterNumber}</p>
                )}
                {item.source.note && (
                  <p className="text-xs text-text-muted italic truncate">
                    &ldquo;{item.source.note}&rdquo;
                  </p>
                )}
              </div>
              <ChevronRight size={14} className="text-text-muted shrink-0 mt-1" />
            </button>
          ))}
        </div>
      )}

      {/* Paginasi */}
      {total > 20 && (
        <div className="flex items-center justify-between text-xs text-text-muted pt-2">
          <span>Menampilkan {items.length} dari {total} surat</span>
          <div className="flex gap-2">
            <button
              type="button"
              disabled={page === 1}
              onClick={() => setPage((p) => p - 1)}
              className="px-3 py-1.5 border border-border rounded-lg disabled:opacity-40 hover:bg-panel transition-colors"
            >
              Sebelumnya
            </button>
            <button
              type="button"
              disabled={page * 20 >= total}
              onClick={() => setPage((p) => p + 1)}
              className="px-3 py-1.5 border border-border rounded-lg disabled:opacity-40 hover:bg-panel transition-colors"
            >
              Berikutnya
            </button>
          </div>
        </div>
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
          <span className="text-sm">Memuat inbox...</span>
        </div>
      }
    >
      <InboxView />
    </Suspense>
  );
}
