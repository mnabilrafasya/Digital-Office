"use client";

// ─────────────────────────────────────────────
// components/inbox/inbox-reading-pane.tsx
// Panel Baca Samping (Split Reading Pane) bergaya Gmail untuk Kotak Masuk
// ─────────────────────────────────────────────

import { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import {
  X,
  ChevronLeft,
  ChevronRight,
  ExternalLink,
  Share2,
  ArrowRight,
  Download,
  Loader2,
  AlertCircle,
  MailOpen,
  CheckCircle2,
  FileText,
  Paperclip,
  Eye,
  QrCode,
} from "lucide-react";
import {
  getInboxItem,
  getInboxList,
  inboxSourceLabel,
  inboxStatusLabel,
  type InboxItemDetail,
} from "@/lib/api/inbox";
import {
  getLetter,
  getLetterHistory,
  formatBytes,
  type LetterDetail,
  type LetterHistoryEntry,
} from "@/lib/api/letter-detail";
import {
  getLetterTracking,
  type DistributionNode,
} from "@/lib/api/audit";
import {
  getLetterAttachmentDownloadUrl,
  officialPdfUrl,
  openLetterAttachmentInline,
} from "@/lib/api/letter-document";
import { InboxForwardingTrail } from "./inbox-forwarding-trail";
import { ReviewDocumentViewer } from "@/components/review/document-viewer";
import { LetterTrackingPanel } from "@/components/review/letter-tracking-panel";
import { cn } from "@/lib/utils";

interface InboxReadingPaneProps {
  itemId: string | null;
  currentIndex: number;
  totalVisible: number;
  onPrev: () => void;
  onNext: () => void;
  onClose: () => void;
  onOpenModal: (itemId: string, type: "disposisi" | "redirect") => void;
  onItemLoaded?: (detail: InboxItemDetail) => void;
  refreshTrigger?: number;
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

export function InboxReadingPane({
  itemId,
  currentIndex,
  totalVisible,
  onPrev,
  onNext,
  onClose,
  onOpenModal,
  onItemLoaded,
  refreshTrigger = 0,
}: InboxReadingPaneProps) {
  const [item, setItem] = useState<InboxItemDetail | null>(null);
  const [letter, setLetter] = useState<LetterDetail | null>(null);
  const [history, setHistory] = useState<LetterHistoryEntry[]>([]);
  const [distribution, setDistribution] = useState<DistributionNode[]>([]);
  const [siblingItems, setSiblingItems] = useState<InboxItemDetail[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pdfReady, setPdfReady] = useState(false);
  const [copyFor, setCopyFor] = useState<string | null>(null);
  const [showDocViewer, setShowDocViewer] = useState(true);

  const handlePdfReadyChange = useCallback(
    (ready: boolean, owner: string | null) => {
      setPdfReady(ready);
      setCopyFor(owner);
    },
    []
  );

  useEffect(() => {
    if (!itemId) return;
    let cancelled = false;

    const timer = setTimeout(() => {
      if (cancelled) return;
      setLoading(true);
      setError(null);
      setLetter(null);
      setHistory([]);
      setDistribution([]);
      setSiblingItems([]);
      setPdfReady(false);
      setCopyFor(null);

      getInboxItem(itemId)
        .then((inboxDetail) => {
          if (cancelled) return;
          setItem(inboxDetail);
          onItemLoaded?.(inboxDetail);
          getLetterHistory(inboxDetail.letter.id)
            .then((items) => {
              if (!cancelled) setHistory(items);
            })
            .catch(() => {});
          getLetterTracking(inboxDetail.letter.id)
            .then((tr) => {
              if (!cancelled && tr) setDistribution(tr.distribution ?? []);
            })
            .catch(() => {});
          // Muat semua item inbox untuk surat yang sama (bila surat pernah masuk >1 putaran)
          getInboxList({ pageSize: 50 })
            .then((res) => {
              if (cancelled) return;
              const sameLetterIds = res.data
                .filter((it) => it.letter.id === inboxDetail.letter.id)
                .map((it) => it.id);
              if (sameLetterIds.length > 0) {
                Promise.all(
                  sameLetterIds.map((sid) =>
                    sid === inboxDetail.id
                      ? Promise.resolve(inboxDetail)
                      : getInboxItem(sid).catch(() => null)
                  )
                ).then((details) => {
                  if (!cancelled) {
                    setSiblingItems(
                      details.filter((d): d is InboxItemDetail => d !== null)
                    );
                  }
                });
              }
            })
            .catch(() => {});
          return getLetter(inboxDetail.letter.id).then((ld) => {
            if (!cancelled && ld) setLetter(ld);
          });
        })
        .catch((e: unknown) => {
          if (!cancelled) {
            setError(
              e instanceof Error ? e.message : "Gagal memuat isi surat masuk."
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
  }, [itemId, refreshTrigger, onItemLoaded]);

  if (!itemId) {
    return (
      <div className="h-full min-h-120 flex flex-col items-center justify-center p-8 text-center bg-panel/20 text-text-muted">
        <div className="w-12 h-12 rounded-2xl bg-surface border border-border flex items-center justify-center mb-3 shadow-2xs">
          <MailOpen size={22} className="text-primary/70" />
        </div>
        <p className="text-sm font-semibold text-text">
          Belum ada surat yang dipilih
        </p>
        <p className="text-xs text-text-muted mt-1 max-w-xs">
          Klik salah satu baris surat di sebelah kiri (atau tekan tombol{" "}
          <kbd className="px-1.5 py-0.5 rounded bg-surface border border-border font-mono text-[10px]">
            j
          </kbd>{" "}
          /{" "}
          <kbd className="px-1.5 py-0.5 rounded bg-surface border border-border font-mono text-[10px]">
            k
          </kbd>
          ) untuk membaca naskah resmi dan melakukan disposisi.
        </p>
      </div>
    );
  }

  if (loading && (!item || item.id !== itemId)) {
    return (
      <div className="h-full min-h-120 flex flex-col items-center justify-center gap-2 text-text-muted bg-surface p-8">
        <Loader2 size={20} className="animate-spin text-primary" />
        <span className="text-xs font-medium">Memuat pratinjau surat...</span>
      </div>
    );
  }

  if (error || !item) {
    return (
      <div className="h-full min-h-120 flex flex-col items-center justify-center gap-3 p-8 text-center bg-surface">
        <AlertCircle size={28} className="text-status-danger-text" />
        <p className="text-xs text-text-muted">
          {error ?? "Surat tidak ditemukan."}
        </p>
      </div>
    );
  }

  const canDispose = item.allowedActions.includes("DISPOSE");
  const canRedirect = item.allowedActions.includes("REDIRECT");
  const isExternalPdf =
    item.letter.direction === "OUTGOING" ||
    item.letter.direction === "INCOMING";

  return (
    <div className="@container flex flex-col h-full w-full min-w-0 max-w-full overflow-hidden bg-surface">
      {/* ── Top Action Bar Panel Baca ala Gmail ── */}
      <div className="px-3 py-2 border-b border-border bg-panel/30 flex items-center justify-between gap-1.5 flex-wrap @2xl:flex-nowrap shrink-0">
        {/* Kiri: Tombol Aksi Cepat (Disposisi, Alihkan, Unduh PDF, Buka Halaman Penuh) */}
        <div className="flex items-center gap-1.5 flex-wrap @2xl:flex-nowrap min-w-0">
          {canDispose && (
            <button
              type="button"
              onClick={() => onOpenModal(item.id, "disposisi")}
              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-semibold bg-primary text-white hover:bg-primary-dark transition-colors shadow-2xs whitespace-nowrap shrink-0"
            >
              <Share2 size={13} /> Disposisi
            </button>
          )}
          {canRedirect && (
            <button
              type="button"
              onClick={() => onOpenModal(item.id, "redirect")}
              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium border border-border bg-surface text-text hover:bg-panel transition-colors whitespace-nowrap shrink-0"
            >
              <ArrowRight size={13} /> Alihkan
            </button>
          )}
          {!isExternalPdf && (
            <a
              href={pdfReady ? officialPdfUrl(item.letter.id) : undefined}
              aria-disabled={!pdfReady}
              title="Unduh PDF Resmi"
              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium border border-border bg-surface text-text hover:bg-panel transition-colors aria-disabled:opacity-50 aria-disabled:pointer-events-none whitespace-nowrap shrink-0"
            >
              {pdfReady ? (
                <>
                  <Download size={13} className="text-primary shrink-0" />
                  <span className="hidden @xl:inline">PDF Resmi</span>
                  <span className="@xl:hidden">PDF</span>
                </>
              ) : (
                <>
                  <Loader2 size={13} className="animate-spin text-primary shrink-0" />
                  <span className="hidden @xl:inline">Menyiapkan PDF...</span>
                  <span className="@xl:hidden">PDF...</span>
                </>
              )}
            </a>
          )}
          <Link
            href={`/inbox/${item.id}`}
            title="Buka di halaman detail penuh"
            className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium border border-border bg-surface text-text-muted hover:text-text hover:bg-panel transition-colors whitespace-nowrap shrink-0"
          >
            <ExternalLink size={13} className="shrink-0" />
            <span className="hidden @xl:inline">Layar Penuh</span>
          </Link>
        </div>

        {/* Kanan: Navigasi Pesan < > & Tombol Tutup Panel */}
        <div className="flex items-center gap-0.5 ml-auto text-xs text-text-muted shrink-0">
          {currentIndex >= 0 && totalVisible > 0 && (
            <>
              <span className="px-1.5 font-medium">
                {currentIndex + 1} dari {totalVisible}
              </span>
              <button
                type="button"
                disabled={currentIndex <= 0}
                onClick={onPrev}
                title="Surat sebelumnya (k)"
                className="w-7 h-7 rounded-full flex items-center justify-center hover:bg-panel disabled:opacity-35 disabled:pointer-events-none transition-colors"
              >
                <ChevronLeft size={15} />
              </button>
              <button
                type="button"
                disabled={currentIndex >= totalVisible - 1}
                onClick={onNext}
                title="Surat berikutnya (j)"
                className="w-7 h-7 rounded-full flex items-center justify-center hover:bg-panel disabled:opacity-35 disabled:pointer-events-none transition-colors"
              >
                <ChevronRight size={15} />
              </button>
              <span className="h-4 w-px bg-border mx-1" />
            </>
          )}
          <button
            type="button"
            onClick={onClose}
            title="Tutup panel baca (u)"
            className="w-7 h-7 rounded-full flex items-center justify-center hover:bg-panel text-text-muted hover:text-text transition-colors"
          >
            <X size={15} />
          </button>
        </div>
      </div>

      {/* ── Isi Panel Baca (Scrollable) ── */}
      <div className="flex-1 min-h-0 min-w-0 max-w-full overflow-y-auto overflow-x-hidden p-4 sm:p-5 space-y-4">
        {/* Judul & Badge Baris Atas */}
        <div className="space-y-2">
          <div className="flex items-start justify-between gap-2">
            <h2 className="text-base sm:text-lg font-bold text-text leading-snug">
              {item.letter.title ?? "(Tanpa perihal)"}
            </h2>
            <span
              className={cn(
                "text-[10px] font-semibold px-2.5 py-0.5 rounded-full shrink-0",
                item.status === "DISPOSED"
                  ? "bg-status-info-bg text-status-info-text"
                  : item.status === "REDIRECTED"
                    ? "bg-status-warn-bg text-status-warn-text"
                    : "bg-panel text-text-muted border border-border"
              )}
            >
              {inboxStatusLabel(item.status)}
            </span>
          </div>

          <div className="flex flex-wrap items-center gap-1.5">
            {item.letter.letterNumber && (
              <span className="text-xs font-mono bg-panel border border-border px-2.5 py-0.5 rounded-full text-text font-medium">
                {item.letter.letterNumber}
              </span>
            )}
            {item.letter.letterType && (
              <span className="text-xs border border-border rounded-full px-2.5 py-0.5 text-text-muted">
                {item.letter.letterType.name}
              </span>
            )}
            {item.letter.confidentiality && (
              <span className="text-xs border border-border rounded-full px-2.5 py-0.5 text-text-muted">
                Sifat: {item.letter.confidentiality.name}
              </span>
            )}
            {item.letter.urgency && (
              <span className="text-xs border border-border rounded-full px-2.5 py-0.5 text-text-muted">
                Kecepatan: {item.letter.urgency.name}
              </span>
            )}
            {copyFor && (
              <span className="text-xs bg-primary/10 text-primary font-semibold px-2.5 py-0.5 rounded-full">
                Salinan untuk {copyFor}
              </span>
            )}
          </div>
        </div>

        {/* Kartu Pengirim & Asal Surat ala Header Email Gmail */}
        <div className="bg-panel/40 border border-border rounded-xl p-3.5 space-y-2.5">
          <div className="flex items-start justify-between gap-3 flex-wrap">
            <div className="flex items-start gap-2.5 min-w-0">
              <div className="w-8 h-8 rounded-full bg-primary/10 text-primary flex items-center justify-center shrink-0 font-bold text-xs">
                {item.letter.senderUnit.code?.slice(0, 2) || "DO"}
              </div>
              <div className="min-w-0">
                <p className="text-xs sm:text-sm font-bold text-text truncate">
                  {item.letter.direction === "INCOMING" &&
                  item.letter.external?.sender
                    ? item.letter.external.sender
                    : item.letter.senderUnit.name}
                </p>
                {item.letter.approver ? (
                  <p className="text-xs text-text-muted">
                    Penandatangan:{" "}
                    <strong className="text-text">
                      {item.letter.approver.name}
                    </strong>{" "}
                    ({item.letter.approver.positionName})
                  </p>
                ) : item.letter.external?.referenceNumber ? (
                  <p className="text-xs text-text-muted">
                    No. Surat Asal:{" "}
                    <strong className="text-text font-mono">
                      {item.letter.external.referenceNumber}
                    </strong>{" "}
                    &bull; Dicatat oleh {item.letter.senderUnit.name}
                  </p>
                ) : null}
                {item.letter.external?.recipient && (
                  <p className="text-xs text-text-muted mt-0.5">
                    Tujuan Eksternal:{" "}
                    <strong className="text-text">
                      {item.letter.external.recipient.name}
                    </strong>
                    {item.letter.external.recipient.attention
                      ? ` (U.p. ${item.letter.external.recipient.attention})`
                      : ""}
                  </p>
                )}
              </div>
            </div>
            <div className="text-right text-[11px] text-text-muted shrink-0">
              <p>{formatDateTime(item.receivedAt)}</p>
              <p className="font-medium text-primary">
                {inboxSourceLabel(item.source)}
              </p>
            </div>
          </div>

          {item.source.note && (
            <div className="px-3 py-2 rounded-lg bg-surface border border-border text-xs text-text">
              <span className="font-semibold text-text-muted">
                Catatan / Instruksi Penerusan:{" "}
              </span>
              <span className="italic">&ldquo;{item.source.note}&rdquo;</span>
            </div>
          )}

          {letter &&
            (letter.recipients.to.length > 0 ||
              letter.recipients.cc.length > 0 ||
              letter.classification ||
              letter.drafter ||
              letter.summary ||
              letter.specialNote) && (
              <div className="pt-2 border-t border-border/70 space-y-1.5 text-xs text-text-muted">
                <div className="flex flex-wrap gap-x-4 gap-y-1">
                  {letter.classification && (
                    <div>
                      <span className="font-semibold">Klasifikasi: </span>
                      <span className="font-mono text-text">
                        {letter.classification.code}
                      </span>{" "}
                      — {letter.classification.name}
                    </div>
                  )}
                  {letter.drafter && (
                    <div>
                      <span className="font-semibold">Pembuat: </span>
                      <span className="text-text">{letter.drafter.name}</span> (
                      {letter.drafter.positionName})
                    </div>
                  )}
                  {letter.recipients.to.length > 0 && (
                    <div>
                      <span className="font-semibold">Kepada: </span>
                      <span className="text-text">
                        {letter.recipients.to
                          .map((r) =>
                            r.type === "UNIT" ? r.unit.name : r.alias.label
                          )
                          .join(", ")}
                      </span>
                    </div>
                  )}
                  {letter.recipients.cc.length > 0 && (
                    <div>
                      <span className="font-semibold">Tembusan: </span>
                      <span>
                        {letter.recipients.cc
                          .map((r) =>
                            r.type === "UNIT" ? r.unit.name : r.alias.label
                          )
                          .join(", ")}
                      </span>
                    </div>
                  )}
                </div>
                {letter.summary && (
                  <p className="text-xs text-text-muted">
                    <span className="font-semibold text-text">Ringkasan: </span>
                    {letter.summary}
                  </p>
                )}
                {letter.specialNote && (
                  <p className="text-xs text-status-warn-text">
                    <span className="font-semibold">Keterangan Khusus: </span>
                    {letter.specialNote}
                  </p>
                )}
              </div>
            )}
        </div>

        {/* Lampiran Berkas Pendukung (jika ada) */}
        {letter && letter.attachments.length > 0 && (
          <div className="border border-border rounded-xl overflow-hidden">
            <div className="px-3.5 py-2 bg-panel/30 border-b border-border flex items-center justify-between">
              <span className="text-xs font-semibold text-text inline-flex items-center gap-1.5">
                <Paperclip size={13} className="text-primary" />
                Lampiran ({letter.attachments.length} berkas)
              </span>
              <span className="text-[11px] text-text-muted">
                Total{" "}
                {formatBytes(
                  letter.attachments.reduce((s, a) => s + a.size, 0)
                )}
              </span>
            </div>
            <div className="p-2.5 flex flex-wrap gap-2">
              {letter.attachments.map((att) => {
                const url = getLetterAttachmentDownloadUrl(
                  item.letter.id,
                  att.id
                );
                return (
                  <div
                    key={att.id}
                    className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg border border-border bg-surface hover:bg-panel/40 text-xs transition-colors"
                  >
                    <FileText size={13} className="text-primary shrink-0" />
                    <span className="font-medium text-text truncate max-w-45">
                      {att.fileName}
                    </span>
                    <span className="text-[10px] text-text-muted">
                      ({formatBytes(att.size)})
                    </span>
                    <button
                      type="button"
                      onClick={() => {
                        void openLetterAttachmentInline(
                          item.letter.id,
                          att.id,
                          att.contentType,
                          att.fileName
                        ).catch(() => {});
                      }}
                      title="Buka lampiran"
                      className="text-text-muted hover:text-primary"
                    >
                      <Eye size={13} />
                    </button>
                    <a
                      href={url}
                      download={att.fileName}
                      title="Unduh lampiran"
                      className="text-primary hover:underline"
                    >
                      <Download size={13} />
                    </a>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* Visualisasi Ringkas Alur Disposisi & Pengalihan */}
        <InboxForwardingTrail
          item={item}
          letter={letter}
          copyFor={copyFor}
          distribution={distribution}
          siblingItems={siblingItems}
          compact
        />

        {/* Pratinjau Naskah Resmi (DevExpress Read-Only Viewer) */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <button
              type="button"
              onClick={() => setShowDocViewer((v) => !v)}
              className="text-xs font-semibold text-primary hover:underline inline-flex items-center gap-1.5"
            >
              <FileText size={13} />
              {showDocViewer
                ? "Sembunyikan Pratinjau Naskah Resmi"
                : "Tampilkan Pratinjau Naskah Resmi"}
            </button>
            {letter?.approval && (
              <span className="inline-flex items-center gap-1 text-[11px] text-status-success-text font-medium">
                <CheckCircle2 size={12} />
                {letter.direction === "OUTGOING" ? (
                  "Bertanda Tangan Basah Resmi"
                ) : (
                  <>
                    <QrCode size={12} /> Tersertifikasi QR Resmi
                  </>
                )}
              </span>
            )}
          </div>

          {showDocViewer &&
            (letter ? (
              <ReviewDocumentViewer
                letter={letter}
                onPdfReadyChange={handlePdfReadyChange}
              />
            ) : (
              <div className="border border-border rounded-xl bg-panel/30 flex items-center justify-center gap-2 text-text-muted p-8 min-h-55">
                <Loader2 size={18} className="animate-spin text-primary" />
                <span className="text-xs">Memuat naskah resmi surat...</span>
              </div>
            ))}
        </div>

        {/* Panel Pelacakan Diagram Alur Persetujuan & Pohon Distribusi */}
        <LetterTrackingPanel
          letterId={item.letter.id}
          letter={letter ?? undefined}
          history={history}
        />
      </div>
    </div>
  );
}

