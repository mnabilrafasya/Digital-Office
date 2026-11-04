"use client";

// ─────────────────────────────────────────────
// components/audit/audit-monitoring-view.tsx
// Tampilan terpadu Pantauan Surat (Monitoring) & Jejak Audit (Audit Log + Ekspor CSV)
// Khusus untuk pemegang permission AUDIT_VIEW
// ─────────────────────────────────────────────

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  Shield,
  Activity,
  Download,
  Search,
  Eye,
  AlertTriangle,
  Loader2,
  X,
  ExternalLink,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import {
  listAuditLogs,
  getAuditLogDetail,
  downloadAuditLogsCsv,
  listMonitoredLetters,
  type AuditLogEntry,
  type MonitoredLetter,
} from "@/lib/api/audit";
import { StatusBadge } from "@/components/daftar-surat/status-badge";
import type { LetterStatus } from "@/lib/types";
import { cn } from "@/lib/utils";

const AUDIT_ACTIONS = [
  "CREATE",
  "SAVE",
  "ATTACHMENT_ADD",
  "ATTACHMENT_DELETE",
  "SUBMIT",
  "APPROVE",
  "RETURN",
  "SIGN",
  "FINAL_PDF",
  "DELIVER",
  "INBOX_READ",
  "DISPOSITION",
  "REDIRECT",
  "DOWNLOAD",
  "VIEW",
  "TEMPLATE_CREATE",
  "TEMPLATE_UPDATE",
  "AUDIT_EXPORT",
];

function defaultDateRange() {
  const now = new Date();
  const to = now.toISOString().slice(0, 10);
  const past = new Date(now.getTime() - 6 * 24 * 60 * 60 * 1000);
  const from = past.toISOString().slice(0, 10);
  return { from, to };
}

function formatDateTime(iso: string) {
  return new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(iso));
}

export function AuditMonitoringView({
  defaultTab = "monitoring",
}: {
  defaultTab?: "monitoring" | "audit";
}) {
  const [tab, setTab] = useState<"monitoring" | "audit">(defaultTab);

  // State Pantauan Surat
  const [monStatus, setMonStatus] = useState("");
  const [minHours, setMinHours] = useState("");
  const [monPage, setMonPage] = useState(1);
  const [monItems, setMonItems] = useState<MonitoredLetter[]>([]);
  const [monTotal, setMonTotal] = useState(0);
  const [monLoading, setMonLoading] = useState(true);
  const [monError, setMonError] = useState<string | null>(null);

  // State Audit Log
  const initialDates = defaultDateRange();
  const [from, setFrom] = useState(initialDates.from);
  const [to, setTo] = useState(initialDates.to);
  const [entityType, setEntityType] = useState("");
  const [action, setAction] = useState("");
  const [search, setSearch] = useState("");
  const [auditPage, setAuditPage] = useState(1);
  const [auditItems, setAuditItems] = useState<AuditLogEntry[]>([]);
  const [auditTotal, setAuditTotal] = useState(0);
  const [auditLoading, setAuditLoading] = useState(true);
  const [auditError, setAuditError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);

  // State Modal Detail Audit Log
  const [selectedLog, setSelectedLog] = useState<AuditLogEntry | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  // Fetch Pantauan Surat
  useEffect(() => {
    if (tab !== "monitoring") return;
    let cancelled = false;
    listMonitoredLetters({
      status: monStatus || undefined,
      minWaitingHours: minHours || undefined,
      page: monPage,
      pageSize: 15,
    })
      .then((res) => {
        if (cancelled) return;
        setMonItems(res.data);
        setMonTotal(res.meta.total);
        setMonError(null);
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setMonError(
            e instanceof Error ? e.message : "Gagal memuat pantauan surat."
          );
        }
      })
      .finally(() => {
        if (!cancelled) setMonLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [tab, monStatus, minHours, monPage]);

  // Fetch Audit Log
  useEffect(() => {
    if (tab !== "audit") return;
    let cancelled = false;
    listAuditLogs({
      from: from || undefined,
      to: to || undefined,
      entityType: entityType || undefined,
      action: action || undefined,
      q: search || undefined,
      page: auditPage,
      pageSize: 15,
    })
      .then((res) => {
        if (cancelled) return;
        setAuditItems(res.data);
        setAuditTotal(res.meta.total);
        setAuditError(null);
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setAuditError(
            e instanceof Error ? e.message : "Gagal memuat daftar audit log."
          );
        }
      })
      .finally(() => {
        if (!cancelled) setAuditLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [tab, from, to, entityType, action, search, auditPage]);

  async function handleOpenDetail(logId: string) {
    setDetailLoading(true);
    try {
      const detail = await getAuditLogDetail(logId);
      if (detail) setSelectedLog(detail);
    } catch {
      // Abaikan bila gagal
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleExportCsv() {
    if (!from || !to) {
      alert("Rentang tanggal Dari dan Sampai wajib diisi untuk ekspor CSV.");
      return;
    }
    setExporting(true);
    try {
      await downloadAuditLogsCsv({
        from,
        to,
        entityType: entityType || undefined,
        action: action || undefined,
        q: search || undefined,
      });
    } catch (e) {
      alert(e instanceof Error ? e.message : "Gagal mengekspor CSV.");
    } finally {
      setExporting(false);
    }
  }

  return (
    <div className="space-y-5">
      {/* Header & Tab Switcher */}
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-xl font-bold text-text">
            Pantauan Persuratan & Audit Log
          </h1>
          <p className="text-xs text-text-muted mt-0.5">
            Pantau surat yang sedang diproses/tertahan serta telusuri jejak audit perusahaan secara real-time.
          </p>
        </div>

        <div className="inline-flex rounded-xl border border-border bg-panel p-1 gap-1">
          <button
            type="button"
            onClick={() => setTab("monitoring")}
            className={cn(
              "inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold transition-colors",
              tab === "monitoring"
                ? "bg-primary text-white shadow-xs"
                : "text-text-muted hover:text-text"
            )}
          >
            <Activity size={13} /> Pantauan Surat Berjalan
          </button>
          <button
            type="button"
            onClick={() => setTab("audit")}
            className={cn(
              "inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold transition-colors",
              tab === "audit"
                ? "bg-primary text-white shadow-xs"
                : "text-text-muted hover:text-text"
            )}
          >
            <Shield size={13} /> Jejak Audit Log
          </button>
        </div>
      </div>

      {/* TAB 1: PANTAUAN SURAT */}
      {tab === "monitoring" && (
        <div className="space-y-4">
          {/* Filter Bar */}
          <div className="p-4 rounded-xl border border-border bg-surface flex items-center gap-3 flex-wrap">
            <div>
              <label className="block text-[11px] font-semibold text-text-muted mb-1">
                Status Surat
              </label>
              <select
                value={monStatus}
                onChange={(e) => {
                  setMonLoading(true);
                  setMonStatus(e.target.value);
                  setMonPage(1);
                }}
                className="px-3 py-1.5 rounded-lg border border-border bg-surface text-xs text-text"
              >
                <option value="">Semua Status Berjalan</option>
                <option value="IN_REVIEW">Menunggu Review</option>
                <option value="IN_APPROVAL">Menunggu Approval</option>
                <option value="RETURNED_TO_DRAFTER">Dikembalikan ke Drafter</option>
                <option value="RETURNED_TO_REVIEWER">Dikembalikan ke Reviewer</option>
              </select>
            </div>

            <div>
              <label className="block text-[11px] font-semibold text-text-muted mb-1">
                Lama Menunggu
              </label>
              <select
                value={minHours}
                onChange={(e) => {
                  setMonLoading(true);
                  setMinHours(e.target.value);
                  setMonPage(1);
                }}
                className="px-3 py-1.5 rounded-lg border border-border bg-surface text-xs text-text"
              >
                <option value="">Semua Durasi</option>
                <option value="24">&ge; 24 Jam</option>
                <option value="48">&ge; 48 Jam (Tertahan / Stuck)</option>
              </select>
            </div>

            <div className="ml-auto text-xs text-text-muted">
              Total:{" "}
              <strong className="text-text">{monTotal}</strong> surat sedang diproses
            </div>
          </div>

          {/* Tabel Pantauan */}
          {monLoading ? (
            <div className="py-16 flex items-center justify-center gap-2 text-text-muted text-sm">
              <Loader2 size={16} className="animate-spin text-primary" />
              Memuat pantauan surat...
            </div>
          ) : monError ? (
            <div className="p-6 rounded-xl border border-status-danger-text/30 bg-status-danger-bg text-status-danger-text text-xs">
              {monError}
            </div>
          ) : monItems.length === 0 ? (
            <div className="p-10 rounded-xl border border-border bg-surface text-center text-xs text-text-muted">
              Tidak ada surat yang sedang tertahan/diproses sesuai filter.
            </div>
          ) : (
            <div className="border border-border rounded-xl bg-surface overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr className="border-b border-border bg-panel/40 text-text-muted uppercase text-[11px]">
                    <th className="px-4 py-3 font-semibold">Perihal & Unit</th>
                    <th className="px-4 py-3 font-semibold">Pembuat (Drafter)</th>
                    <th className="px-4 py-3 font-semibold">Pemegang Saat Ini</th>
                    <th className="px-4 py-3 font-semibold">Status</th>
                    <th className="px-4 py-3 font-semibold">Durasi Tunggu</th>
                    <th className="px-4 py-3 font-semibold text-right">Aksi</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {monItems.map((m) => (
                    <tr key={m.letterId} className="hover:bg-panel/20">
                      <td className="px-4 py-3 max-w-xs">
                        <p className="font-semibold text-text truncate">
                          {m.titleMasked
                            ? "(Disamarkan — Surat Terbatas/Rahasia)"
                            : m.title ?? "(Tanpa perihal)"}
                        </p>
                        <p className="text-text-muted truncate">{m.senderUnit}</p>
                      </td>
                      <td className="px-4 py-3">
                        <p className="font-medium text-text">{m.drafter.name}</p>
                        <p className="text-text-muted">{m.drafter.positionName}</p>
                      </td>
                      <td className="px-4 py-3">
                        {m.holder ? (
                          <>
                            <p className="font-medium text-text">
                              {m.holder.name}
                              {m.step != null ? ` (Tahap ${m.step})` : ""}
                            </p>
                            <p className="text-text-muted">
                              {m.holder.positionName}
                            </p>
                          </>
                        ) : (
                          <span className="text-text-muted">—</span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <StatusBadge status={m.status as LetterStatus} />
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-1.5">
                          <span
                            className={cn(
                              "font-semibold",
                              m.stuck ? "text-status-danger-text" : "text-text"
                            )}
                          >
                            {m.waitingHours} jam
                          </span>
                          {m.stuck && (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-status-danger-bg text-status-danger-text text-[10px] font-semibold">
                              <AlertTriangle size={10} /> Tertahan
                            </span>
                          )}
                        </div>
                        <p className="text-[11px] text-text-muted">
                          Sejak {formatDateTime(m.since)}
                        </p>
                      </td>
                      <td className="px-4 py-3 text-right">
                        <Link
                          href={`/surat/${m.letterId}`}
                          className="inline-flex items-center gap-1 text-primary font-semibold hover:underline"
                        >
                          Buka <ExternalLink size={12} />
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>

              {/* Pagination Pantauan */}
              <div className="px-4 py-2.5 border-t border-border flex items-center justify-between text-xs text-text-muted">
                <span>Halaman {monPage}</span>
                <div className="flex items-center gap-1.5">
                  <button
                    type="button"
                    disabled={monPage <= 1}
                    onClick={() => {
                      setMonLoading(true);
                      setMonPage((p) => Math.max(1, p - 1));
                    }}
                    className="p-1.5 rounded border border-border disabled:opacity-40 hover:bg-panel"
                  >
                    <ChevronLeft size={13} />
                  </button>
                  <button
                    type="button"
                    disabled={monPage * 15 >= monTotal}
                    onClick={() => {
                      setMonLoading(true);
                      setMonPage((p) => p + 1);
                    }}
                    className="p-1.5 rounded border border-border disabled:opacity-40 hover:bg-panel"
                  >
                    <ChevronRight size={13} />
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* TAB 2: AUDIT LOG */}
      {tab === "audit" && (
        <div className="space-y-4">
          {/* Filter & Ekspor CSV */}
          <div className="p-4 rounded-xl border border-border bg-surface flex items-end gap-3 flex-wrap">
            <div>
              <label className="block text-[11px] font-semibold text-text-muted mb-1">
                Dari Tanggal
              </label>
              <input
                type="date"
                value={from}
                onChange={(e) => {
                  setAuditLoading(true);
                  setFrom(e.target.value);
                  setAuditPage(1);
                }}
                className="px-2.5 py-1.5 rounded-lg border border-border bg-surface text-xs text-text"
              />
            </div>

            <div>
              <label className="block text-[11px] font-semibold text-text-muted mb-1">
                Sampai Tanggal
              </label>
              <input
                type="date"
                value={to}
                onChange={(e) => {
                  setAuditLoading(true);
                  setTo(e.target.value);
                  setAuditPage(1);
                }}
                className="px-2.5 py-1.5 rounded-lg border border-border bg-surface text-xs text-text"
              />
            </div>

            <div>
              <label className="block text-[11px] font-semibold text-text-muted mb-1">
                Entitas
              </label>
              <select
                value={entityType}
                onChange={(e) => {
                  setAuditLoading(true);
                  setEntityType(e.target.value);
                  setAuditPage(1);
                }}
                className="px-2.5 py-1.5 rounded-lg border border-border bg-surface text-xs text-text"
              >
                <option value="">Semua Entitas</option>
                <option value="LETTER">LETTER (Surat)</option>
                <option value="TEMPLATE">TEMPLATE (Template)</option>
                <option value="AUDIT">AUDIT (Ekspor Audit)</option>
              </select>
            </div>

            <div>
              <label className="block text-[11px] font-semibold text-text-muted mb-1">
                Aksi
              </label>
              <select
                value={action}
                onChange={(e) => {
                  setAuditLoading(true);
                  setAction(e.target.value);
                  setAuditPage(1);
                }}
                className="px-2.5 py-1.5 rounded-lg border border-border bg-surface text-xs text-text"
              >
                <option value="">Semua Aksi</option>
                {AUDIT_ACTIONS.map((a) => (
                  <option key={a} value={a}>
                    {a}
                  </option>
                ))}
              </select>
            </div>

            <div className="flex-1 min-w-44">
              <label className="block text-[11px] font-semibold text-text-muted mb-1">
                Cari Pelaku / Nomor / Perihal
              </label>
              <div className="relative">
                <Search
                  size={13}
                  className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted"
                />
                <input
                  type="text"
                  value={search}
                  onChange={(e) => {
                    setAuditLoading(true);
                    setSearch(e.target.value);
                    setAuditPage(1);
                  }}
                  placeholder="Ketik kata kunci..."
                  className="w-full pl-8 pr-3 py-1.5 rounded-lg border border-border bg-surface text-xs text-text"
                />
              </div>
            </div>

            <button
              type="button"
              onClick={handleExportCsv}
              disabled={exporting}
              className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-primary text-white text-xs font-semibold hover:bg-primary-dark transition-colors disabled:opacity-60"
            >
              {exporting ? (
                <Loader2 size={13} className="animate-spin" />
              ) : (
                <Download size={13} />
              )}
              Ekspor CSV (Maks. 31 Hari)
            </button>
          </div>

          {/* Tabel Audit Log */}
          {auditLoading ? (
            <div className="py-16 flex items-center justify-center gap-2 text-text-muted text-sm">
              <Loader2 size={16} className="animate-spin text-primary" />
              Memuat jejak audit...
            </div>
          ) : auditError ? (
            <div className="p-6 rounded-xl border border-status-danger-text/30 bg-status-danger-bg text-status-danger-text text-xs">
              {auditError}
            </div>
          ) : auditItems.length === 0 ? (
            <div className="p-10 rounded-xl border border-border bg-surface text-center text-xs text-text-muted">
              Tidak ada catatan audit log pada rentang & filter yang dipilih.
            </div>
          ) : (
            <div className="border border-border rounded-xl bg-surface overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr className="border-b border-border bg-panel/40 text-text-muted uppercase text-[11px]">
                    <th className="px-4 py-3 font-semibold">Waktu</th>
                    <th className="px-4 py-3 font-semibold">Aksi & Entitas</th>
                    <th className="px-4 py-3 font-semibold">Pelaku</th>
                    <th className="px-4 py-3 font-semibold">Surat Terkait</th>
                    <th className="px-4 py-3 font-semibold">Alamat IP</th>
                    <th className="px-4 py-3 font-semibold text-right">
                      Snapshot
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {auditItems.map((log) => (
                    <tr key={log.id} className="hover:bg-panel/20">
                      <td className="px-4 py-3 whitespace-nowrap text-text-muted">
                        {formatDateTime(log.at)}
                      </td>
                      <td className="px-4 py-3">
                        <span className="font-mono font-bold text-primary">
                          {log.action}
                        </span>
                        <span className="ml-1.5 text-[10px] px-1.5 py-0.5 rounded bg-panel border border-border text-text-muted">
                          {log.entityType}
                        </span>
                      </td>
                      <td className="px-4 py-3 font-medium text-text">
                        {log.actor.name}
                      </td>
                      <td className="px-4 py-3 max-w-xs">
                        {log.letter ? (
                          <div>
                            {log.letter.number && (
                              <p className="font-mono text-[11px] text-primary">
                                {log.letter.number}
                              </p>
                            )}
                            <p className="text-text truncate">
                              {log.letter.masked
                                ? "(Disamarkan)"
                                : log.letter.title ?? "(Tanpa perihal)"}
                            </p>
                          </div>
                        ) : (
                          <span className="text-text-muted">—</span>
                        )}
                      </td>
                      <td className="px-4 py-3 font-mono text-text-muted">
                        {log.ipAddress ?? "—"}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <button
                          type="button"
                          onClick={() => handleOpenDetail(log.id)}
                          disabled={detailLoading}
                          className="inline-flex items-center gap-1 text-primary font-semibold hover:underline"
                        >
                          <Eye size={12} /> Detail
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>

              {/* Pagination Audit Log */}
              <div className="px-4 py-2.5 border-t border-border flex items-center justify-between text-xs text-text-muted">
                <span>
                  Total <strong className="text-text">{auditTotal}</strong> baris
                  audit · Halaman {auditPage}
                </span>
                <div className="flex items-center gap-1.5">
                  <button
                    type="button"
                    disabled={auditPage <= 1}
                    onClick={() => {
                      setAuditLoading(true);
                      setAuditPage((p) => Math.max(1, p - 1));
                    }}
                    className="p-1.5 rounded border border-border disabled:opacity-40 hover:bg-panel"
                  >
                    <ChevronLeft size={13} />
                  </button>
                  <button
                    type="button"
                    disabled={auditPage * 15 >= auditTotal}
                    onClick={() => {
                      setAuditLoading(true);
                      setAuditPage((p) => p + 1);
                    }}
                    className="p-1.5 rounded border border-border disabled:opacity-40 hover:bg-panel"
                  >
                    <ChevronRight size={13} />
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Modal Detail Audit Log Snapshot */}
      {selectedLog && (
        <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
          <div className="bg-surface border border-border rounded-xl shadow-xl w-full max-w-lg overflow-hidden">
            <div className="px-5 py-3.5 border-b border-border flex items-center justify-between bg-panel/40">
              <h3 className="text-sm font-bold text-text">
                Detail Audit Log #{selectedLog.id}
              </h3>
              <button
                type="button"
                onClick={() => setSelectedLog(null)}
                className="text-text-muted hover:text-text p-1"
              >
                <X size={16} />
              </button>
            </div>
            <div className="p-5 space-y-3 text-xs">
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <span className="text-text-muted block">Waktu</span>
                  <strong className="text-text">
                    {formatDateTime(selectedLog.at)}
                  </strong>
                </div>
                <div>
                  <span className="text-text-muted block">Aksi & Entitas</span>
                  <strong className="font-mono text-primary">
                    {selectedLog.action} ({selectedLog.entityType})
                  </strong>
                </div>
                <div>
                  <span className="text-text-muted block">Pelaku</span>
                  <strong className="text-text">{selectedLog.actor.name}</strong>
                </div>
                <div>
                  <span className="text-text-muted block">IP & Request ID</span>
                  <strong className="font-mono text-text">
                    {selectedLog.ipAddress ?? "—"} /{" "}
                    {selectedLog.requestId ?? "—"}
                  </strong>
                </div>
              </div>

              <div>
                <span className="text-text-muted block mb-1 font-semibold">
                  Snapshot JSON
                </span>
                <pre className="p-3 rounded-lg bg-panel border border-border font-mono text-[11px] text-text overflow-x-auto max-h-60">
                  {selectedLog.snapshot
                    ? JSON.stringify(selectedLog.snapshot, null, 2)
                    : "(Tidak ada data snapshot tambahan)"}
                </pre>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

