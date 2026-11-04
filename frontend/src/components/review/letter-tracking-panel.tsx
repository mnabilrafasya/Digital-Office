"use client";

// ─────────────────────────────────────────────
// components/review/letter-tracking-panel.tsx
// Panel Lacak Posisi, Pohon Distribusi Penerima, Riwayat Versi Naskah, & Audit Surat
// Terhubung ke:
// - GET /api/v1/letters/{id}/tracking
// - GET /api/v1/letters/{id}/versions
// - GET /api/v1/letters/{id}/audit (khusus pemegang AUDIT_VIEW)
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
  type DocumentVersion,
} from "@/lib/api/letter-document";
import { formatBytes } from "@/lib/api/letter-detail";
import { cn } from "@/lib/utils";

function formatShortDate(iso: string) {
  return new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(iso));
}

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
  TO: "Tujuan Utama",
  CC: "Tembusan (CC)",
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
    <li className={cn("space-y-2", depth > 0 && "ml-5 pl-3 border-l border-border")}>
      <div className="p-2.5 rounded-lg bg-panel/40 border border-border/70 flex items-start justify-between gap-2">
        <div className="min-w-0 space-y-0.5">
          <div className="flex items-center gap-1.5 flex-wrap">
            {node.status === "UNREAD" ? (
              <Mail size={12} className="text-status-danger-text shrink-0" />
            ) : (
              <MailOpen size={12} className="text-status-success-text shrink-0" />
            )}
            <span className="text-xs font-semibold text-text">
              {node.recipient.name}
            </span>
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-surface border border-border text-text-muted">
              {VIA_LABEL[node.via] ?? node.via}
            </span>
          </div>
          <p className="text-[11px] text-text-muted">
            {node.recipient.positionName}
          </p>
          {node.from && (
            <p className="text-[11px] text-text-muted">
              Dari: <span className="text-text font-medium">{node.from.name}</span>
            </p>
          )}
          {node.note && (
            <p className="text-[11px] text-text-muted italic">
              &ldquo;{node.note}&rdquo;
            </p>
          )}
          <p className="text-[10px] text-text-muted/80">
            Diterima {formatShortDate(node.receivedAt)}
            {node.readAt ? ` · Dibaca ${formatShortDate(node.readAt)}` : ""}
          </p>
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
        <ul className="space-y-2">
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

export function LetterTrackingPanel({
  letterId,
  showVersions = false,
}: {
  letterId: string;
  showVersions?: boolean;
}) {
  const [tracking, setTracking] = useState<LetterTracking | null>(null);
  const [versions, setVersions] = useState<DocumentVersion[]>([]);
  const [auditLogs, setAuditLogs] = useState<AuditLogEntry[]>([]);

  useEffect(() => {
    let cancelled = false;
    getLetterTracking(letterId)
      .then((res) => {
        if (!cancelled && res) setTracking(res);
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
  }, [letterId, showVersions]);

  if (!tracking && versions.length === 0 && auditLogs.length === 0) {
    return null;
  }

  return (
    <div className="space-y-4">
      {/* 1. Posisi Surat & Pohon Distribusi */}
      {tracking && (
        <div className="border border-border rounded-xl bg-surface overflow-hidden">
          <div className="px-4 py-3 border-b border-border bg-panel/30 flex items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <MapPin size={15} className="text-primary" />
              <h3 className="text-sm font-semibold text-text">
                Lacak Posisi & Distribusi Surat
              </h3>
            </div>
            {tracking.position.stuck && (
              <span className="inline-flex items-center gap-1 text-[11px] font-semibold px-2.5 py-0.5 rounded-full bg-status-danger-bg text-status-danger-text">
                <AlertTriangle size={11} /> Tertahan &ge; 48 Jam
              </span>
            )}
          </div>

          <div className="p-4 space-y-4">
            {/* Banner posisi saat ini */}
            <div
              className={cn(
                "rounded-lg p-3 border flex items-start justify-between gap-3 flex-wrap",
                tracking.position.stuck
                  ? "bg-status-danger-bg/40 border-status-danger-text/30"
                  : "bg-panel/40 border-border"
              )}
            >
              <div className="space-y-0.5">
                <p className="text-xs font-semibold text-text">
                  {tracking.position.label}
                </p>
                {tracking.position.holder && (
                  <p className="text-xs text-text-muted">
                    Pemegang saat ini:{" "}
                    <strong className="text-text">
                      {tracking.position.holder.name}
                    </strong>{" "}
                    ({tracking.position.holder.positionName})
                  </p>
                )}
                <p className="text-[11px] text-text-muted">
                  Sejak {formatShortDate(tracking.position.since)}
                  {tracking.position.waitingHours != null
                    ? ` · Menunggu ${tracking.position.waitingHours} jam`
                    : ""}
                </p>
              </div>
              <div className="flex items-center gap-1 text-xs text-text-muted">
                {tracking.status === "SENT" ? (
                  <CheckCircle2 size={15} className="text-status-success-text" />
                ) : (
                  <Clock size={15} className="text-primary" />
                )}
              </div>
            </div>

            {/* Pohon distribusi penerima (muncul setelah SENT) */}
            {tracking.distribution.length > 0 && (
              <div className="space-y-2">
                <div className="flex items-center gap-1.5 text-xs font-semibold text-text">
                  <Share2 size={13} className="text-primary" />
                  <span>
                    Status Baca & Pohon Distribusi Penerima (
                    {tracking.distribution.length})
                  </span>
                </div>
                <ul className="space-y-2">
                  {tracking.distribution.map((node) => (
                    <DistributionTreeItem
                      key={node.inboxItemId}
                      node={node}
                    />
                  ))}
                </ul>
              </div>
            )}
          </div>
        </div>
      )}

      {/* 2. Riwayat Versi Naskah (.docx) */}
      {showVersions && versions.length > 0 && (
        <div className="border border-border rounded-xl bg-surface overflow-hidden">
          <div className="px-4 py-3 border-b border-border bg-panel/30 flex items-center gap-2">
            <History size={15} className="text-primary" />
            <h3 className="text-sm font-semibold text-text">
              Riwayat Versi Naskah ({versions.length} Versi)
            </h3>
          </div>
          <ul className="divide-y divide-border">
            {versions.map((v) => (
              <li
                key={v.versionNumber}
                className="px-4 py-2.5 flex items-center justify-between gap-3 text-xs"
              >
                <div className="min-w-0">
                  <p className="font-semibold text-text">
                    Versi {v.versionNumber}{" "}
                    <span className="font-normal text-text-muted">
                      ({formatBytes(v.fileSize)})
                    </span>
                  </p>
                  <p className="text-text-muted truncate">
                    Disimpan oleh {v.createdBy.name} ({v.createdBy.positionName}) ·{" "}
                    {formatShortDate(v.createdAt)}
                  </p>
                </div>
                <a
                  href={`/api/v1/letters/${letterId}/document?version=${v.versionNumber}`}
                  download
                  className="inline-flex items-center gap-1 text-primary font-medium hover:underline shrink-0"
                >
                  <Download size={12} /> Unduh .docx
                </a>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* 3. Jejak Audit Surat (khusus user dengan permission AUDIT_VIEW) */}
      {showVersions && auditLogs.length > 0 && (
        <div className="border border-border rounded-xl bg-surface overflow-hidden">
          <div className="px-4 py-3 border-b border-border bg-panel/30 flex items-center gap-2">
            <Shield size={15} className="text-primary" />
            <h3 className="text-sm font-semibold text-text">
              Jejak Audit Surat ({auditLogs.length} Kejadian)
            </h3>
          </div>
          <ul className="divide-y divide-border max-h-64 overflow-y-auto">
            {auditLogs.map((log) => (
              <li
                key={log.id}
                className="px-4 py-2.5 flex items-center justify-between gap-2 text-xs"
              >
                <div className="min-w-0">
                  <span className="font-mono font-semibold text-primary">
                    {log.action}
                  </span>{" "}
                  <span className="text-text">oleh {log.actor.name}</span>
                  {log.ipAddress && (
                    <span className="text-text-muted">
                      {" "}
                      · IP {log.ipAddress}
                    </span>
                  )}
                </div>
                <span className="text-text-muted shrink-0">
                  {formatShortDate(log.at)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
