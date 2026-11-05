"use client";

// ─────────────────────────────────────────────
// components/inbox/inbox-message-row.tsx
// Baris surat masuk bergaya Gmail (checkbox, bintang, pengirim, subjek + cuplikan, aksi cepat saat hover)
// ─────────────────────────────────────────────

import {
  Star,
  Share2,
  ArrowRight,
  ExternalLink,
  Mail,
  Building2,
  ShieldAlert,
  Zap,
  CheckCircle2,
} from "lucide-react";
import { inboxSourceLabel, inboxStatusLabel } from "@/lib/api/inbox";
import type { InboxItem } from "@/lib/api/inbox";
import type { DensityMode } from "./inbox-toolbar";
import { cn } from "@/lib/utils";

interface InboxMessageRowProps {
  item: InboxItem;
  selected: boolean;
  active: boolean;
  starred: boolean;
  density: DensityMode;
  onToggleSelect: () => void;
  onToggleStar: () => void;
  onClickRow: () => void;
  onOpenFullPage: () => void;
  onQuickDispose: () => void;
  onQuickRedirect: () => void;
}

/** Format waktu cerdas ala Gmail: jam bila hari ini, tanggal+bulan bila tahun ini, lengkap bila tahun lalu */
function formatGmailTimestamp(iso: string): string {
  const date = new Date(iso);
  const now = new Date();
  const isSameDay =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate();

  if (isSameDay) {
    return new Intl.DateTimeFormat("id-ID", {
      hour: "2-digit",
      minute: "2-digit",
    }).format(date);
  }

  if (date.getFullYear() === now.getFullYear()) {
    return new Intl.DateTimeFormat("id-ID", {
      day: "numeric",
      month: "short",
    }).format(date);
  }

  return new Intl.DateTimeFormat("id-ID", {
    day: "2-digit",
    month: "2-digit",
    year: "2-digit",
  }).format(date);
}

export function InboxMessageRow({
  item,
  selected,
  active,
  starred,
  density,
  onToggleSelect,
  onToggleStar,
  onClickRow,
  onOpenFullPage,
  onQuickDispose,
  onQuickRedirect,
}: InboxMessageRowProps) {
  const isUnread = item.status === "UNREAD";
  const canForward = item.status === "UNREAD" || item.status === "READ";

  const senderPrimary =
    item.source.type === "FORWARD" && item.source.forwardedBy
      ? item.source.forwardedBy.name
      : item.letter.senderUnit.name;

  const snippetParts: string[] = [];
  if (item.source.note) {
    snippetParts.push(`Instruksi: "${item.source.note}"`);
  }
  if (item.letter.letterNumber) {
    snippetParts.push(item.letter.letterNumber);
  }
  if (item.letter.approver?.name) {
    snippetParts.push(`TTD: ${item.letter.approver.name}`);
  }
  const snippetText = snippetParts.join(" · ");

  const isSecret = item.letter.confidentiality?.code === "RAHASIA";
  const isRestricted = item.letter.confidentiality?.code === "TERBATAS";
  const isVeryUrgent = item.letter.urgency?.code === "SANGAT_SEGERA";
  const isUrgent = item.letter.urgency?.code === "SEGERA";

  return (
    <div
      role="row"
      tabIndex={0}
      onClick={onClickRow}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onClickRow();
        }
      }}
      className={cn(
        "group relative flex items-center gap-2.5 px-3 sm:px-4 border-b border-border cursor-pointer select-none transition-colors outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary/40",
        density === "compact" ? "py-2" : "py-3",
        active
          ? "bg-primary/10 hover:bg-primary/12"
          : selected
            ? "bg-status-info-bg/35 hover:bg-status-info-bg/50"
            : isUnread
              ? "bg-surface hover:bg-panel/60"
              : "bg-panel/25 hover:bg-panel/65"
      )}
    >
      {/* Indikator bar kiri untuk item belum dibaca atau aktif */}
      {(isUnread || active) && (
        <span
          className={cn(
            "absolute left-0 top-0 bottom-0 w-1",
            active ? "bg-primary" : "bg-status-info-text"
          )}
        />
      )}

      {/* 1. Checkbox & Bintang */}
      <div
        className="flex items-center gap-1.5 shrink-0"
        onClick={(e) => e.stopPropagation()}
      >
        <input
          type="checkbox"
          checked={selected}
          onChange={onToggleSelect}
          aria-label={`Pilih surat ${item.letter.title ?? ""}`}
          className="w-4 h-4 rounded border-border-input text-primary focus:ring-primary/30 cursor-pointer accent-primary"
        />
        <button
          type="button"
          onClick={onToggleStar}
          title={starred ? "Hapus bintang" : "Tandai berbintang"}
          aria-label={starred ? "Hapus bintang" : "Tandai berbintang"}
          className={cn(
            "p-1 rounded-full transition-colors",
            starred
              ? "text-status-warn-text hover:bg-status-warn-bg"
              : "text-text-muted/50 hover:text-text-muted hover:bg-panel"
          )}
        >
          <Star size={15} className={cn(starred && "fill-current")} />
        </button>
      </div>

      {/* 2. Kolom Pengirim / Asal Surat (Lebar tetap di desktop seperti Gmail) */}
      <div className="w-36 sm:w-48 lg:w-52 shrink-0 min-w-0 flex items-center gap-1.5 pr-2">
        {item.source.type === "DIRECT" ? (
          <Building2
            size={13}
            className={cn(
              "shrink-0",
              isUnread ? "text-primary" : "text-text-muted"
            )}
          />
        ) : item.source.forwardType === "DISPOSITION" ? (
          <Share2 size={13} className="text-status-info-text shrink-0" />
        ) : (
          <ArrowRight size={13} className="text-status-warn-text shrink-0" />
        )}
        <div className="min-w-0 flex-1">
          <p
            className={cn(
              "text-xs sm:text-sm truncate",
              isUnread
                ? "font-bold text-text"
                : "font-medium text-text-muted group-hover:text-text"
            )}
            title={`${senderPrimary} (${inboxSourceLabel(item.source)})`}
          >
            {senderPrimary}
          </p>
          {density === "comfortable" && (
            <p className="text-[11px] text-text-muted truncate leading-tight">
              {inboxSourceLabel(item.source)}
            </p>
          )}
        </div>
      </div>

      {/* 3. Kolom Tengah: Label Chip + Perihal (Title) + Cuplikan (Snippet) satu baris ala Gmail */}
      <div className="flex-1 min-w-0 flex items-center gap-1.5 overflow-hidden">
        {/* Badge Asal Ringkas bila mode compact */}
        {density === "compact" && (
          <span
            className={cn(
              "text-[10px] font-semibold px-1.5 py-0.5 rounded shrink-0",
              item.source.type === "DIRECT"
                ? item.source.recipientType === "CC"
                  ? "bg-panel text-text-muted border border-border"
                  : "bg-primary/10 text-primary"
                : item.source.forwardType === "DISPOSITION"
                  ? "bg-status-info-bg text-status-info-text"
                  : "bg-status-warn-bg text-status-warn-text"
            )}
          >
            {item.source.type === "DIRECT"
              ? item.source.recipientType === "CC"
                ? "CC"
                : "Tujuan"
              : item.source.forwardType === "DISPOSITION"
                ? "Disposisi"
                : "Alihan"}
          </span>
        )}

        {/* Badge Sifat (Rahasia / Terbatas) */}
        {(isSecret || isRestricted) && (
          <span
            className={cn(
              "inline-flex items-center gap-0.5 text-[10px] font-semibold px-1.5 py-0.5 rounded shrink-0",
              isSecret
                ? "bg-status-danger-bg text-status-danger-text"
                : "bg-status-warn-bg text-status-warn-text"
            )}
          >
            <ShieldAlert size={10} />
            {item.letter.confidentiality.name}
          </span>
        )}

        {/* Badge Urgensi (Sangat Segera / Segera) */}
        {(isVeryUrgent || isUrgent) && (
          <span
            className={cn(
              "inline-flex items-center gap-0.5 text-[10px] font-semibold px-1.5 py-0.5 rounded shrink-0",
              isVeryUrgent
                ? "bg-status-danger-bg text-status-danger-text"
                : "bg-status-warn-bg text-status-warn-text"
            )}
          >
            <Zap size={10} />
            {item.letter.urgency.name}
          </span>
        )}

        {/* Badge Jenis Surat */}
        {item.letter.letterType && (
          <span className="hidden md:inline-block text-[10px] font-medium px-1.5 py-0.5 rounded bg-panel text-text-muted border border-border shrink-0">
            {item.letter.letterType.name}
          </span>
        )}

        {/* Subjek + Cuplikan */}
        <div className="min-w-0 flex-1 truncate text-xs sm:text-sm">
          <span
            className={cn(
              isUnread ? "font-bold text-text" : "font-medium text-text"
            )}
          >
            {item.letter.title ?? "(Tanpa perihal)"}
          </span>
          {snippetText && (
            <span className="text-text-muted font-normal">
              {" "}
              — {snippetText}
            </span>
          )}
        </div>
      </div>

      {/* 4. Kolom Kanan: Status + Tanggal (Digantikan oleh Tombol Aksi Cepat saat Hover ala Gmail) */}
      <div className="shrink-0 flex items-center justify-end min-w-23 sm:min-w-33.75">
        {/* Tampilan Normal (saat tidak di-hover) */}
        <div className="flex items-center gap-2 group-hover:hidden">
          {(item.status === "DISPOSED" || item.status === "REDIRECTED") && (
            <span
              className={cn(
                "hidden sm:inline-flex items-center gap-1 text-[10px] font-semibold px-2 py-0.5 rounded-full",
                item.status === "DISPOSED"
                  ? "bg-status-info-bg text-status-info-text"
                  : "bg-status-warn-bg text-status-warn-text"
              )}
            >
              <CheckCircle2 size={10} />
              {inboxStatusLabel(item.status)}
            </span>
          )}
          <span
            className={cn(
              "text-xs whitespace-nowrap",
              isUnread ? "font-bold text-text" : "text-text-muted"
            )}
          >
            {formatGmailTimestamp(item.receivedAt)}
          </span>
        </div>

        {/* Tampilan Hover Quick Actions ala Gmail */}
        <div
          className="hidden group-hover:flex items-center gap-1 bg-surface/95 px-1.5 py-0.5 rounded-lg shadow-2xs border border-border"
          onClick={(e) => e.stopPropagation()}
        >
          {canForward && (
            <>
              <button
                type="button"
                onClick={onQuickDispose}
                title="Disposisikan surat ini"
                className="p-1.5 rounded-md text-text-muted hover:text-primary hover:bg-primary/10 transition-colors"
              >
                <Share2 size={14} />
              </button>
              <button
                type="button"
                onClick={onQuickRedirect}
                title="Alihkan surat ini"
                className="p-1.5 rounded-md text-text-muted hover:text-status-warn-text hover:bg-status-warn-bg transition-colors"
              >
                <ArrowRight size={14} />
              </button>
            </>
          )}
          <button
            type="button"
            onClick={onClickRow}
            title="Baca surat di panel"
            className="p-1.5 rounded-md text-text-muted hover:text-text hover:bg-panel transition-colors"
          >
            <Mail size={14} />
          </button>
          <button
            type="button"
            onClick={onOpenFullPage}
            title="Buka di halaman penuh (/inbox/[id])"
            className="p-1.5 rounded-md text-text-muted hover:text-primary hover:bg-panel transition-colors"
          >
            <ExternalLink size={14} />
          </button>
        </div>
      </div>
    </div>
  );
}

