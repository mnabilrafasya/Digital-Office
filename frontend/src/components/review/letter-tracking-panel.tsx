"use client";

// ─────────────────────────────────────────────
// components/review/letter-tracking-panel.tsx
// Panel Terpadu & Ringkas (Tabbed + Collapsible):
// Diagram Alur, Posisi & Pohon Distribusi Penerima, Versi Naskah (.docx), & Log Audit Sistem
// ─────────────────────────────────────────────

import { useEffect, useState } from "react";
import {
  MapPin,
  AlertTriangle,
  Share2,
  History,
  Download,
  Shield,
  CheckCircle2,
  Clock,
  MailOpen,
  Mail,
  GitCommitHorizontal,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import {
  getLetterTracking,
  getLetterAuditLogs,
  type LetterTracking,
  type DistributionNode,
  type AuditLogEntry,
} from "@/lib/api/audit";
import {
  getLetterVersions,
  officialPdfUrl,
  type DocumentVersion,
} from "@/lib/api/letter-document";
import { activityLabel } from "@/components/audit/audit-presenter";
import {
  formatBytes,
  type LetterDetail,
  type LetterHistoryEntry,
} from "@/lib/api/letter-detail";
import { LetterFlowDiagram } from "./letter-flow-diagram";
import {
  formatCompactTime,
  trackingTimelineToHistory,
} from "./audit-trail-utils";
import { cn } from "@/lib/utils";

const DIST_STATUS_BADGE: Record<
  DistributionNode["status"],
  { label: string; cls: string }
> = {
  UNREAD: {
    label: "Belum Dibaca",
    cls: "bg-status-danger-bg text-status-danger-text",
  },
  READ: {
    label: "Sudah Dibaca",
    cls: "bg-status-success-bg text-status-success-text",
  },
  DISPOSED: {
    label: "Didisposisikan",
    cls: "bg-status-info-bg text-status-info-text",
  },
  REDIRECTED: {
    label: "Dialihkan",
    cls: "bg-status-warn-bg text-status-warn-text",
  },
};

const VIA_LABEL: Record<DistributionNode["via"], string> = {
  TO: "Tujuan",
  CC: "Tembusan",
  DISPOSITION: "Disposisi",
  REDIRECT: "Pengalihan",
};

function DistributionTreeItem({
  node,
  depth = 0,
}: {
  node: DistributionNode;
  depth?: number;
}) {
  const badge = DIST_STATUS_BADGE[node.status] ?? {
    label: node.status,
    cls: "bg-panel text-text-muted",
  };

  return (
    <li
      className={cn(
        "space-y-1.5",
        depth > 0 && "ml-4 pl-3 border-l border-border"
      )}
    >
      <div className="px-3 py-2 rounded-lg bg-panel/35 border border-border/70 flex items-center justify-between gap-2 flex-wrap">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5 flex-wrap">
            {node.status === "UNREAD" ? (
              <Mail size={12} className="text-status-danger-text shrink-0" />
            ) : (
              <MailOpen
                size={12}
                className="text-status-success-text shrink-0"
              />
            )}
            <span className="text-xs font-semibold text-text">
              {node.recipient.name}
            </span>
            <span className="text-[10px] text-text-muted">
              ({node.recipient.positionName})
            </span>
            <span className="text-[10px] px-1.5 py-0.2 rounded bg-surface border border-border text-text-muted">
              {VIA_LABEL[node.via] ?? node.via}
            </span>
          </div>
          <div className="flex items-center gap-2 text-[10px] text-text-muted mt-0.5 flex-wrap">
            {node.from && <span>Dari: {node.from.name}</span>}
            <span>Diterima {formatCompactTime(node.receivedAt)}</span>
            {node.readAt && (
              <span>· Dibaca {formatCompactTime(node.readAt)}</span>
            )}
          </div>
          {node.note && (
            <p className="text-[11px] text-text-muted italic mt-0.5">
              &ldquo;{node.note}&rdquo;
            </p>
          )}
        </div>
        <span
          className={cn(
            "text-[10px] font-semibold px-2 py-0.5 rounded-full shrink-0",
            badge.cls
          )}
        >
          {badge.label}
        </span>
      </div>
      {node.forwards.length > 0 && (
        <ul className="space-y-1.5">
          {node.forwards.map((child) => (
            <DistributionTreeItem
              key={child.inboxItemId}
              node={child}
              depth={depth + 1}
            />
          ))}
        </ul>
      )}
    </li>
  );
}

type TrackingTab = "diagram" | "distribution" | "versions" | "audit";

export function LetterTrackingPanel({
  letterId,
  letter,
  history,
  showVersions = false,
}: {
  letterId: string;
  letter?: LetterDetail;
  history?: LetterHistoryEntry[];
  showVersions?: boolean;
}) {
  const [tracking, setTracking] = useState<LetterTracking | null>(null);
  const [versions, setVersions] = useState<DocumentVersion[]>([]);
  const [auditLogs, setAuditLogs] = useState<AuditLogEntry[]>([]);
  const [collapsed, setCollapsed] = useState(false);
  const [activeTab, setActiveTab] = useState<TrackingTab>("diagram");

  useEffect(() => {
    let cancelled = false;
    getLetterTracking(letterId)
      .then((res) => {
        if (!cancelled && res) {
          setTracking(res);
          if (!letter && res.distribution.length > 0) {
            setActiveTab("distribution");
          }
        }
      })
      .catch(() => {});

    if (showVersions) {
      getLetterVersions(letterId)
        .then((items) => {
          if (!cancelled) setVersions(items);
        })
        .catch(() => {});

      getLetterAuditLogs(letterId)
        .then((logs) => {
          if (!cancelled) setAuditLogs(logs);
        })
        .catch(() => {});
    }

    return () => {
      cancelled = true;
    };
  }, [letterId, showVersions, letter]);

  if (!tracking && !letter && versions.length === 0 && auditLogs.length === 0) {
    return null;
  }

  const effectiveHistory =
    history && history.length > 0
      ? history
      : tracking
        ? trackingTimelineToHistory(tracking.timeline)
        : [];

  const hasDiagram = Boolean(letter);
  const distCount = tracking?.distribution.length ?? 0;
  const effectiveStatus = tracking?.status ?? letter?.status ?? "";
  const isOfficialPdfActive =
    effectiveStatus === "SENT" ||
    effectiveStatus === "APPROVED" ||
    effectiveStatus === "RECEIVED";

  return (
    <div className="w-full min-w-0 max-w-full border border-border rounded-xl bg-surface overflow-hidden">
      {/* Header Ringkas: Posisi Saat Ini + Tombol Ciutkan */}
      <div className="px-4 py-2.5 bg-panel/30 border-b border-border flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-2 min-w-0 flex-wrap">
          <MapPin size={15} className="text-primary shrink-0" />
          <span className="text-xs font-semibold text-text">
            {tracking ? tracking.position.label : "Pelacakan & Visualisasi Alur"}
          </span>
          {tracking?.position.holder && (
            <span className="text-xs text-text-muted truncate">
              · Pemegang:{" "}
              <strong className="text-text">
                {tracking.position.holder.name}
              </strong>
              {tracking.position.waitingHours != null
                ? ` (${tracking.position.waitingHours}j)`
                : ""}
            </span>
          )}
          {tracking?.position.stuck && (
            <span className="inline-flex items-center gap-1 text-[10px] font-semibold px-2 py-0.5 rounded-full bg-status-danger-bg text-status-danger-text">
              <AlertTriangle size={10} /> Tertahan &ge; 48 Jam
            </span>
          )}
        </div>

        <div className="flex items-center gap-2 ml-auto">
          {tracking && (
            <span className="text-[11px] text-text-muted hidden sm:inline-flex items-center gap-1">
              {tracking.status === "SENT" || tracking.status === "RECEIVED" ? (
                <CheckCircle2 size={13} className="text-status-success-text" />
              ) : (
                <Clock size={13} className="text-primary" />
              )}
              Sejak {formatCompactTime(tracking.position.since)}
            </span>
          )}
          <button
            type="button"
            onClick={() => setCollapsed((v) => !v)}
            className="inline-flex items-center gap-1 px-2 py-1 rounded-md border border-border bg-surface text-[11px] font-medium text-text-muted hover:text-text transition-colors"
          >
            {collapsed ? (
              <>
                <span>Tampilkan</span>
                <ChevronDown size={12} />
              </>
            ) : (
              <>
                <span>Ciutkan</span>
                <ChevronUp size={12} />
              </>
            )}
          </button>
        </div>
      </div>

      {!collapsed && (
        <div className="p-4 space-y-3">
          {/* Navigasi Tab Ringkas */}
          <div className="flex items-center gap-1 border-b border-border pb-2 overflow-x-auto">
            {hasDiagram && (
              <button
                type="button"
                onClick={() => setActiveTab("diagram")}
                className={cn(
                  "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold transition-colors shrink-0",
                  activeTab === "diagram"
                    ? "bg-primary text-white"
                    : "text-text-muted hover:text-text hover:bg-panel"
                )}
              >
                <GitCommitHorizontal size={13} />
                <span>Diagram Alur</span>
              </button>
            )}

            {distCount > 0 && (
              <button
                type="button"
                onClick={() => setActiveTab("distribution")}
                className={cn(
                  "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold transition-colors shrink-0",
                  activeTab === "distribution"
                    ? "bg-primary text-white"
                    : "text-text-muted hover:text-text hover:bg-panel"
                )}
              >
                <Share2 size={12} />
                <span>Distribusi ({distCount})</span>
              </button>
            )}

            {showVersions && versions.length > 0 && (
              <button
                type="button"
                onClick={() => setActiveTab("versions")}
                className={cn(
                  "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold transition-colors shrink-0",
                  activeTab === "versions"
                    ? "bg-primary text-white"
                    : "text-text-muted hover:text-text hover:bg-panel"
                )}
              >
                <History size={12} />
                <span>Versi Naskah ({versions.length})</span>
              </button>
            )}

            {showVersions && auditLogs.length > 0 && (
              <button
                type="button"
                onClick={() => setActiveTab("audit")}
                className={cn(
                  "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold transition-colors shrink-0",
                  activeTab === "audit"
                    ? "bg-primary text-white"
                    : "text-text-muted hover:text-text hover:bg-panel"
                )}
              >
                <Shield size={12} />
                <span>Log Audit ({auditLogs.length})</span>
              </button>
            )}
          </div>

          {/* Konten Tab 1: Visualisasi Diagram Alur */}
          {activeTab === "diagram" && letter && (
            <LetterFlowDiagram
              letter={letter}
              history={effectiveHistory}
              compact
            />
          )}

          {/* Konten Tab 2: Pohon Distribusi Penerima */}
          {activeTab === "distribution" && tracking && (
            <ul className="space-y-1.5 max-h-64 overflow-y-auto pr-1">
              {tracking.distribution.map((node) => (
                <DistributionTreeItem key={node.inboxItemId} node={node} />
              ))}
            </ul>
          )}

          {/* Konten Tab 3: Riwayat Versi Naskah (.docx / .pdf) */}
          {activeTab === "versions" && versions.length > 0 && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-56 overflow-y-auto">
              {versions.map((v) => {
                const isFinalVersion =
                  v.type === "FINAL" ||
                  v.fileExtension.toLowerCase() === "pdf";
                const canDownloadVersion = isFinalVersion
                  ? isOfficialPdfActive
                  : !isOfficialPdfActive;
                const downloadHref = isFinalVersion
                  ? officialPdfUrl(letterId)
                  : `/api/v1/letters/${letterId}/document?version=${v.versionNumber}`;

                return (
                  <div
                    key={v.versionNumber}
                    className="px-3 py-2 rounded-lg border border-border bg-panel/25 flex items-center justify-between gap-2 text-xs"
                  >
                    <div className="min-w-0">
                      <p className="font-semibold text-text flex items-center gap-1.5 flex-wrap">
                        <span>Versi {v.versionNumber}</span>
                        <span
                          className={cn(
                            "text-[10px] px-1.5 py-0.2 rounded font-semibold",
                            isFinalVersion
                              ? "bg-status-success-bg text-status-success-text"
                              : "bg-panel text-text-muted border border-border"
                          )}
                        >
                          {isFinalVersion ? "PDF Resmi" : "Draf .docx"}
                        </span>
                        <span className="font-normal text-text-muted">
                          ({formatBytes(v.fileSize)})
                        </span>
                      </p>
                      <p className="text-[11px] text-text-muted truncate">
                        {v.createdBy.name} · {formatCompactTime(v.createdAt)}
                      </p>
                    </div>
                    {canDownloadVersion ? (
                      <a
                        href={downloadHref}
                        download={!isFinalVersion}
                        className="inline-flex items-center gap-1 px-2 py-1 rounded bg-surface border border-border text-primary font-semibold hover:bg-panel shrink-0 text-[11px]"
                      >
                        <Download size={11} />{" "}
                        {isFinalVersion ? "PDF" : ".docx"}
                      </a>
                    ) : (
                      <span className="text-[10px] text-text-muted shrink-0">
                        Arsip Draf
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
          )}

          {/* Konten Tab 4: Jejak Audit Sistem (AUDIT_VIEW) */}
          {activeTab === "audit" && auditLogs.length > 0 && (
            <ul className="divide-y divide-border max-h-52 overflow-y-auto border border-border rounded-lg">
              {auditLogs.map((log) => (
                <li
                  key={log.id}
                  className="px-3 py-2 flex items-center justify-between gap-2 text-xs hover:bg-panel/20"
                >
                  <div className="min-w-0 truncate">
                    <span
                      title={log.action}
                      className="font-semibold text-primary"
                    >
                      {activityLabel(log.action)}
                    </span>{" "}
                    <span className="text-text">
                      oleh {log.actor.name || "Sistem"}
                    </span>
                    {log.ipAddress && (
                      <span className="text-text-muted">
                        {" "}
                        · IP {log.ipAddress}
                      </span>
                    )}
                  </div>
                  <span className="text-[11px] text-text-muted shrink-0">
                    {formatCompactTime(log.at)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
