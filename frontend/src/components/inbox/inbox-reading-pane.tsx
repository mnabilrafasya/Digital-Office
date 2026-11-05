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
  Building2,
  CheckCircle2,
  FileText,
  Paperclip,
  Eye,
  QrCode,
} from "lucide-react";
import {
  getInboxItem,
  inboxSourceLabel,
  inboxStatusLabel,
  type InboxItemDetail,
} from "@/lib/api/inbox";
import {
  getLetter,
  formatBytes,
  type LetterDetail,
} from "@/lib/api/letter-detail";
import {
  getLetterAttachmentDownloadUrl,
  officialPdfUrl,
} from "@/lib/api/letter-document";
import { ReviewDocumentViewer } from "@/components/review/document-viewer";
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
      setPdfReady(false);
      setCopyFor(null);

      getInboxItem(itemId)
        .then((inboxDetail) => {
          if (cancelled) return;
          setItem(inboxDetail);
          onItemLoaded?.(inboxDetail);
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
           belum ada surat yang dipilih
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

  return (
    <div className="flex flex-col h-full bg-surface">
      {/* ── Top Action Bar Panel Baca ala Gmail ── */}
      <div className="px-4 py-2.5 border-b border-border bg-panel/30 flex items-center justify-between gap-2 flex-wrap shrink-0">
        {/* Kiri: Tombol Aksi Cepat (Disposisi, Alihkan, Unduh PDF, Buka Halaman Penuh) */}
        <div className="flex items-center gap-1.5 flex-wrap">
          {canDispose && (
            <button
              type="button"
              onClick={() => onOpenModal(item.id, "disposisi")}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold bg-primary text-white hover:bg-primary-dark transition-colors shadow-2xs"
            >
              <Share2 size={13} /> Disposisi
            </button>
          )}
          {canRedirect && (
            <button
              type="button"
              onClick={() => onOpenModal(item.id, "redirect")}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-border bg-surface text-text hover:bg-panel transition-colors"
            >
              <ArrowRight size={13} /> Alihkan
            </button>
          )}
          <a
            href={pdfReady ? officialPdfUrl(item.letter.id) : undefined}
            aria-disabled={!pdfReady}
            title="Unduh PDF Resmi"
            className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium border border-border bg-surface text-text hover:bg-panel transition-colors aria-disabled:opacity-50 aria-disabled:pointer-events-none"
          >
            {pdfReady ? (
              <>
                <Download size={13} className="text-primary" />
                <span>PDF Resmi</span>
              </>
            ) : (
              <>
                <Loader2 size={13} className="animate-spin text-primary" />
                <span>Menyiapkan PDF...</span>
              </>
            )}
          </a>
          <Link
            href={`/inbox/${item.id}`}
            title="Buka di halaman detail penuh"
            className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium border border-border bg-surface text-text-muted hover:text-text hover:bg-panel transition-colors"
          >
            <ExternalLink size={13} />
            <span className="hidden sm:inline">Layar Penuh</span>
          </Link>
        </div>

        {/* Kanan: Navigasi Pesan < > & Tombol Tutup Panel */}
        <div className="flex items-center gap-1 ml-auto text-xs text-text-muted">
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
      <div className="flex-1 overflow-y-auto p-4 sm:p-5 space-y-4">
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
                  {item.letter.senderUnit.name}
                </p>
                <p className="text-xs text-text-muted">
                  Penandatangan:{" "}
                  <strong className="text-text">
                    {item.letter.approver.name}
                  </strong>{" "}
                  ({item.letter.approver.positionName})
                </p>
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
              letter.recipients.cc.length > 0) && (
              <div className="pt-2 border-t border-border/70 flex flex-wrap gap-x-4 gap-y-1 text-xs text-text-muted">
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
                    <a
                      href={url}
                      target="_blank"
                      rel="noopener noreferrer"
                      title="Buka lampiran"
                      className="text-text-muted hover:text-primary"
                    >
                      <Eye size={13} />
                    </a>
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

        {/* Riwayat Penerusan dari Item Ini (jika ada) */}
        {item.forwards.length > 0 && (
          <div className="border border-border rounded-xl overflow-hidden">
            <div className="px-3.5 py-2 bg-panel/30 border-b border-border">
              <h4 className="text-xs font-semibold text-text">
                Riwayat Penerusan dari Anda ({item.forwards.length})
              </h4>
            </div>
            <ul className="divide-y divide-border text-xs">
              {item.forwards.map((f) => (
                <li key={f.id} className="px-3.5 py-2.5 flex items-start gap-2.5">
                  {f.type === "DISPOSITION" ? (
                    <Share2
                      size={13}
                      className="text-status-info-text shrink-0 mt-0.5"
                    />
                  ) : (
                    <ArrowRight
                      size={13}
                      className="text-status-warn-text shrink-0 mt-0.5"
                    />
                  )}
                  <div className="min-w-0 flex-1">
                    <p className="font-semibold text-text">
                      {f.type === "DISPOSITION" ? "Disposisi" : "Dialihkan"} ke{" "}
                      {f.targetType === "UNIT" ? (
                        <span className="inline-flex items-center gap-1">
                          <Building2 size={11} />
                          {f.targetUnit?.name ?? "—"}
                        </span>
                      ) : (
                        f.targetUser?.name ?? "—"
                      )}
                    </p>
                    <p className="text-text-muted">
                      Diterima oleh: {f.recipient.name} (
                      {f.recipient.positionName})
                    </p>
                    {f.note && (
                      <p className="italic text-text-muted mt-0.5">
                        &ldquo;{f.note}&rdquo;
                      </p>
                    )}
                  </div>
                  <span className="text-[10px] text-text-muted shrink-0">
                    {formatDateTime(f.createdAt)}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}

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
                <QrCode size={12} /> Tersertifikasi QR Resmi
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
      </div>
    </div>
  );
}

