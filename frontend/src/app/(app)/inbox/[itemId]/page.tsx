"use client";

// ─────────────────────────────────────────────
// app/(app)/inbox/[itemId]/page.tsx
// Halaman detail satu item inbox (dengan bilah navigasi pesan < > bergaya Gmail)
// ─────────────────────────────────────────────

import { useState, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import {
  ChevronRight,
  ChevronLeft,
  Loader2,
  AlertCircle,
  ArrowLeft,
  Download,
  QrCode,
  Share2,
  ArrowRight,
  Building2,
  CheckCircle2,
  MailOpen,
  ExternalLink,
  FileText,
  File,
  FileSpreadsheet,
  Eye,
} from "lucide-react";
import {
  getInboxItem,
  getInboxList,
  inboxSourceLabel,
  type InboxItemDetail,
  type ForwardRecord,
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
import { DisposisiModal } from "@/components/inbox/disposisi-modal";
import { RedirectModal } from "@/components/inbox/redirect-modal";
import { ReviewDocumentViewer } from "@/components/review/document-viewer";
import { LetterTrackingPanel } from "@/components/review/letter-tracking-panel";
import { cn } from "@/lib/utils";

// ─── Helper ──────────────────────────────────

function formatDateTime(iso: string) {
  return new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "long",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    timeZoneName: "short",
  }).format(new Date(iso));
}

function fileIcon(contentType: string) {
  if (contentType.includes("pdf"))
    return <File size={16} className="text-red-500 shrink-0" />;
  if (contentType.includes("spreadsheet") || contentType.includes("excel"))
    return <FileSpreadsheet size={16} className="text-emerald-600 shrink-0" />;
  return <FileText size={16} className="text-blue-500 shrink-0" />;
}

function MetaRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <dt className="text-[11px] font-semibold uppercase tracking-wide text-text-muted mb-0.5">
        {label}
      </dt>
      <dd className="text-sm text-text">{children}</dd>
    </div>
  );
}

function ForwardItem({ f }: { f: ForwardRecord }) {
  return (
    <li className="px-4 py-3 flex items-start gap-3">
      <div className="w-7 h-7 rounded-full bg-panel border border-border flex items-center justify-center shrink-0 mt-0.5">
        {f.type === "DISPOSITION" ? (
          <Share2 size={12} className="text-status-info-text" />
        ) : (
          <ArrowRight size={12} className="text-status-warn-text" />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-xs font-semibold text-text">
          {f.type === "DISPOSITION" ? "Disposisi" : "Dialihkan"} ke{" "}
          {f.targetType === "UNIT" ? (
            <span className="inline-flex items-center gap-1">
              <Building2 size={10} className="shrink-0" />
              {f.targetUnit?.name ?? "—"}
            </span>
          ) : (
            f.targetUser?.name ?? "—"
          )}
        </p>
        <p className="text-xs text-text-muted mt-0.5">
          Diterima oleh:{" "}
          <strong className="text-text">{f.recipient.name}</strong>
          {" · "}
          {f.recipient.positionName}
        </p>
        {f.note && (
          <p className="text-xs text-text-muted italic mt-0.5">
            &ldquo;{f.note}&rdquo;
          </p>
        )}
        <p className="text-[11px] text-text-muted/70 mt-0.5">
          {formatDateTime(f.createdAt)}
        </p>
      </div>
    </li>
  );
}

// ─── Page ────────────────────────────────────

export default function InboxDetailPage() {
  const { itemId } = useParams<{ itemId: string }>();
  const router = useRouter();

  const [item, setItem] = useState<InboxItemDetail | null>(null);
  const [letter, setLetter] = useState<LetterDetail | null>(null);
  const [neighborIds, setNeighborIds] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [modal, setModal] = useState<"disposisi" | "redirect" | null>(null);
  const [pdfReady, setPdfReady] = useState(false);
  const [copyFor, setCopyFor] = useState<string | null>(null);

  const handlePdfReadyChange = useCallback(
    (ready: boolean, owner: string | null) => {
      setPdfReady(ready);
      setCopyFor(owner);
    },
    []
  );

  const load = useCallback(async () => {
    try {
      const inboxItem = await getInboxItem(itemId);
      setItem(inboxItem);
      setError(null);
      // Muat detail surat (untuk pratinjau naskah resmi, QR, penerima lengkap, dan lampiran)
      getLetter(inboxItem.letter.id)
        .then((ld) => {
          if (ld) setLetter(ld);
        })
        .catch(() => {});
    } catch (e) {
      setError((e as Error).message ?? "Gagal memuat item inbox.");
    } finally {
      setLoading(false);
    }
  }, [itemId]);

  useEffect(() => {
    let cancelled = false;
    getInboxItem(itemId)
      .then((inboxItem) => {
        if (cancelled) return;
        setItem(inboxItem);
        setError(null);
        getLetter(inboxItem.letter.id)
          .then((ld) => {
            if (!cancelled && ld) setLetter(ld);
          })
          .catch(() => {});
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setError(
            e instanceof Error ? e.message : "Gagal memuat item inbox."
          );
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    // Ambil urutan ID inbox untuk navigasi Pesan Sebelumnya / Berikutnya ala Gmail
    getInboxList({ pageSize: 50 })
      .then((res) => {
        if (!cancelled) {
          setNeighborIds(res.data.map((it) => it.id));
        }
      })
      .catch(() => {});

    return () => {
      cancelled = true;
    };
  }, [itemId]);

  const currentIdx = neighborIds.indexOf(itemId);
  const prevItemId = currentIdx > 0 ? neighborIds[currentIdx - 1] : null;
  const nextItemId =
    currentIdx >= 0 && currentIdx < neighborIds.length - 1
      ? neighborIds[currentIdx + 1]
      : null;

  // ── Loading ──
  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh] gap-2 text-text-muted">
        <Loader2 size={18} className="animate-spin text-primary" />
        <span className="text-sm">Memuat surat masuk...</span>
      </div>
    );
  }

  // ── Error ──
  if (error || !item) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-3 text-center">
        <AlertCircle size={32} className="text-status-danger-text" />
        <p className="text-sm text-text-muted">
          {error ?? "Item tidak ditemukan."}
        </p>
        <button
          type="button"
          onClick={() => router.push("/inbox")}
          className="text-sm text-primary hover:underline flex items-center gap-1"
        >
          <ArrowLeft size={13} /> Kembali ke Inbox
        </button>
      </div>
    );
  }

  const canDispose = item.allowedActions.includes("DISPOSE");
  const canRedirect = item.allowedActions.includes("REDIRECT");
  const hasActions = canDispose || canRedirect;

  return (
    <div className={cn("space-y-4", hasActions && "pb-20")}>
      {/* ── Top Action & Navigation Bar ala Gmail ── */}
      <div className="bg-surface border border-border rounded-xl px-4 py-2.5 flex items-center justify-between gap-3 flex-wrap shadow-2xs">
        <div className="flex items-center gap-2 flex-wrap">
          <Link
            href="/inbox"
            className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-semibold text-text-muted hover:text-text hover:bg-panel transition-colors"
            title="Kembali ke Kotak Masuk"
          >
            <ArrowLeft size={14} /> Kembali ke Inbox
          </Link>

          <span className="h-4 w-px bg-border" />

          {canDispose && (
            <button
              type="button"
              onClick={() => setModal("disposisi")}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold bg-primary text-white hover:bg-primary-dark transition-colors"
            >
              <Share2 size={13} /> Disposisi
            </button>
          )}
          {canRedirect && (
            <button
              type="button"
              onClick={() => setModal("redirect")}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-border text-text hover:bg-panel transition-colors"
            >
              <ArrowRight size={13} /> Alihkan
            </button>
          )}
        </div>

        {/* Navigasi Pesan < > ala Gmail */}
        {currentIdx >= 0 && neighborIds.length > 0 && (
          <div className="flex items-center gap-1 text-xs text-text-muted ml-auto">
            <span className="px-1.5 font-medium">
              {currentIdx + 1} dari {neighborIds.length}
            </span>
            <button
              type="button"
              disabled={!prevItemId}
              onClick={() => prevItemId && router.push(`/inbox/${prevItemId}`)}
              title="Surat sebelumnya"
              className="w-7 h-7 rounded-full flex items-center justify-center hover:bg-panel disabled:opacity-35 disabled:pointer-events-none transition-colors"
            >
              <ChevronLeft size={15} />
            </button>
            <button
              type="button"
              disabled={!nextItemId}
              onClick={() => nextItemId && router.push(`/inbox/${nextItemId}`)}
              title="Surat berikutnya"
              className="w-7 h-7 rounded-full flex items-center justify-center hover:bg-panel disabled:opacity-35 disabled:pointer-events-none transition-colors"
            >
              <ChevronRight size={15} />
            </button>
          </div>
        )}
      </div>

      {/* ── Breadcrumb ── */}
      <nav className="flex items-center gap-1 text-xs text-text-muted flex-wrap">
        <Link href="/inbox" className="hover:text-text transition-colors">
          Inbox
        </Link>
        <ChevronRight size={12} />
        <span className="text-text font-medium truncate max-w-xs">
          {item.letter.title ?? "(Tanpa perihal)"}
        </span>
      </nav>

      {/* ── Banner sumber surat ── */}
      <div className="bg-panel/50 border border-border rounded-xl p-4 flex items-start gap-3">
        <div className="w-9 h-9 rounded-full bg-primary/10 text-primary flex items-center justify-center shrink-0">
          {item.source.type === "DIRECT" ? (
            <MailOpen size={16} />
          ) : item.source.forwardType === "DISPOSITION" ? (
            <Share2 size={16} />
          ) : (
            <ArrowRight size={16} />
          )}
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-semibold text-text">
            {inboxSourceLabel(item.source)}
          </p>
          <p className="text-xs text-text-muted">
            Diterima: {formatDateTime(item.receivedAt)}
          </p>
          {item.source.note && (
            <p className="text-xs text-text-muted mt-1 italic">
              Instruksi: &ldquo;{item.source.note}&rdquo;
            </p>
          )}
        </div>
        <span
          className={cn(
            "text-[10px] font-semibold px-2.5 py-1 rounded-full shrink-0",
            item.status === "UNREAD"
              ? "bg-status-danger-bg text-status-danger-text"
              : item.status === "DISPOSED"
                ? "bg-status-info-bg text-status-info-text"
                : item.status === "REDIRECTED"
                  ? "bg-status-warn-bg text-status-warn-text"
                  : "bg-panel text-text-muted border border-border"
          )}
        >
          {item.status === "UNREAD"
            ? "Belum Dibaca"
            : item.status === "READ"
              ? "Sudah Dibaca"
              : item.status === "DISPOSED"
                ? "Didisposisikan"
                : "Dialihkan"}
        </span>
      </div>

      {/* ── Judul & chip ── */}
      <div className="space-y-2">
        <h1 className="text-xl font-bold text-text leading-snug">
          {item.letter.title ?? "(Tanpa perihal)"}
        </h1>
        <div className="flex flex-wrap gap-2">
          {item.letter.letterNumber && (
            <span className="text-xs font-mono bg-panel border border-border px-2.5 py-1 rounded-full text-text">
              {item.letter.letterNumber}
            </span>
          )}
          {item.letter.letterType && (
            <span className="text-xs border border-border rounded-full px-2.5 py-1 text-text-muted">
              {item.letter.letterType.name}
            </span>
          )}
          {item.letter.confidentiality && (
            <span className="text-xs border border-border rounded-full px-2.5 py-1 text-text-muted">
              {item.letter.confidentiality.name}
            </span>
          )}
          {item.letter.urgency && (
            <span className="text-xs border border-border rounded-full px-2.5 py-1 text-text-muted">
              {item.letter.urgency.name}
            </span>
          )}
        </div>
      </div>

      {/* ── Dua kolom ── */}
      <div className="grid grid-cols-1 lg:grid-cols-[1fr_280px] gap-5">
        {/* ─ Kolom Kiri ─ */}
        <div className="space-y-4">
          {/* Pratinjau Naskah Resmi & Penyiapan Salinan PDF Penerima */}
          {letter ? (
            <ReviewDocumentViewer
              letter={letter}
              onPdfReadyChange={handlePdfReadyChange}
            />
          ) : (
            <div className="border border-border rounded-xl bg-panel/30 flex items-center justify-center gap-2 text-text-muted p-8 min-h-[240px]">
              <Loader2 size={18} className="animate-spin text-primary" />
              <span className="text-xs">Memuat naskah resmi surat...</span>
            </div>
          )}

          {/* QR & Unduh PDF — hanya tersedia setelah surat SENT (approval exists) */}
          {letter?.approval && (
            <div className="border border-border rounded-xl overflow-hidden">
              <div className="px-4 py-3 border-b border-border bg-panel/30 flex items-center justify-between gap-2 flex-wrap">
                <div className="flex items-center gap-2">
                  <QrCode size={15} className="text-primary" />
                  <h3 className="text-sm font-semibold text-text">
                    QR Persetujuan & Dokumen Resmi
                  </h3>
                </div>
                {copyFor && (
                  <span className="text-[11px] font-medium px-2 py-0.5 rounded-full bg-primary/10 text-primary">
                    Salinan untuk {copyFor}
                  </span>
                )}
              </div>
              <div className="p-4 flex items-start gap-4 flex-wrap">
                <div className="w-24 h-24 border border-border rounded-lg bg-white p-1 shrink-0 flex items-center justify-center">
                  <img
                    src={letter.approval.qrUrl}
                    alt="QR Persetujuan"
                    className="w-full h-full object-contain"
                    onError={(e) => {
                      (e.currentTarget as HTMLImageElement).style.display =
                        "none";
                    }}
                  />
                </div>
                <div className="flex-1 min-w-0 space-y-2">
                  <div>
                    <p className="text-[11px] font-semibold uppercase tracking-wide text-text-muted mb-0.5">
                      Disetujui oleh
                    </p>
                    <p className="text-sm font-bold text-text">
                      {letter.approval.approvedBy.name}
                    </p>
                    <p className="text-xs text-text-muted">
                      {letter.approval.approvedBy.positionName}
                    </p>
                  </div>
                  <div className="flex items-center gap-1.5 text-xs text-text-muted">
                    <CheckCircle2
                      size={13}
                      className="text-status-success-text shrink-0"
                    />
                    <span>{formatDateTime(letter.approval.approvedAt)}</span>
                  </div>
                  <a
                    href={pdfReady ? officialPdfUrl(item.letter.id) : undefined}
                    aria-disabled={!pdfReady}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-lg bg-primary text-white hover:bg-primary-dark transition-colors aria-disabled:opacity-50 aria-disabled:pointer-events-none"
                  >
                    {pdfReady ? (
                      <>
                        <Download size={13} /> Unduh PDF Resmi
                      </>
                    ) : (
                      <>
                        <Loader2 size={13} className="animate-spin" /> Menyiapkan PDF Resmi...
                      </>
                    )}
                  </a>
                </div>
              </div>
            </div>
          )}

          {/* Lampiran Berkas Pendukung */}
          {letter && letter.attachments.length > 0 && (
            <div className="border border-border rounded-xl overflow-hidden">
              <div className="px-4 py-3 border-b border-border bg-panel/30">
                <h3 className="text-sm font-semibold text-text">
                  Lampiran Berkas Pendukung ({letter.attachments.length} Berkas)
                </h3>
                <p className="text-xs text-text-muted">
                  Total{" "}
                  {formatBytes(
                    letter.attachments.reduce((s, a) => s + a.size, 0)
                  )}
                </p>
              </div>
              <ul className="divide-y divide-border">
                {letter.attachments.map((att) => {
                  const downloadUrl = getLetterAttachmentDownloadUrl(
                    item.letter.id,
                    att.id
                  );
                  return (
                    <li
                      key={att.id}
                      className="flex items-center gap-3 px-4 py-3"
                    >
                      {fileIcon(att.contentType)}
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-text truncate">
                          {att.fileName}
                        </p>
                        <p className="text-xs text-text-muted">
                          {formatBytes(att.size)}
                        </p>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <a
                          href={downloadUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-xs text-text-muted hover:text-primary flex items-center gap-1 transition-colors"
                        >
                          <Eye size={13} /> Buka
                        </a>
                        <a
                          href={downloadUrl}
                          download={att.fileName}
                          className="text-xs text-primary font-medium hover:underline flex items-center gap-1"
                        >
                          <Download size={13} /> Unduh
                        </a>
                      </div>
                    </li>
                  );
                })}
              </ul>
            </div>
          )}

          {/* Riwayat penerusan */}
          {item.forwards.length > 0 && (
            <div className="border border-border rounded-xl overflow-hidden">
              <div className="px-4 py-3 border-b border-border bg-panel/30">
                <h3 className="text-sm font-semibold text-text">
                  Riwayat Penerusan ({item.forwards.length})
                </h3>
              </div>
              <ul className="divide-y divide-border">
                {item.forwards.map((f) => (
                  <ForwardItem key={f.id} f={f} />
                ))}
              </ul>
            </div>
          )}

          {/* Belum ada penerusan */}
          {item.forwards.length === 0 && !hasActions && (
            <div className="border border-dashed border-border rounded-xl px-4 py-6 text-center text-xs text-text-muted">
              Belum ada penerusan dari item ini.
            </div>
          )}

          {/* Pelacakan Cabang Distribusi Penerima */}
          <LetterTrackingPanel letterId={item.letter.id} />
        </div>

        {/* ─ Kolom Kanan ─ */}
        <div className="space-y-4">
          {/* Info surat */}
          <div className="border border-border rounded-xl overflow-hidden">
            <div className="px-4 py-3 border-b border-border bg-panel/30">
              <h3 className="text-sm font-semibold text-text">Info Surat</h3>
            </div>
            <dl className="px-4 py-3 space-y-3">
              <MetaRow label="Unit Pengirim">
                {item.letter.senderUnit.name}
              </MetaRow>
              <MetaRow label="Penandatangan">
                {item.letter.approver.name}
                <span className="block text-xs text-text-muted">
                  {item.letter.approver.positionName}
                </span>
              </MetaRow>
              <MetaRow label="Tanggal Surat">
                {formatDateTime(item.letter.letterDate)}
              </MetaRow>
              {letter && letter.recipients.to.length > 0 && (
                <MetaRow label="Tujuan">
                  <div className="flex flex-wrap gap-1 mt-0.5">
                    {letter.recipients.to.map((r, i) => (
                      <span
                        key={i}
                        className="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-primary/10 text-primary font-medium"
                      >
                        {r.type === "UNIT" ? r.unit.name : r.alias.label}
                      </span>
                    ))}
                  </div>
                </MetaRow>
              )}
              {letter && letter.recipients.cc.length > 0 && (
                <MetaRow label="Tembusan (CC)">
                  <div className="flex flex-wrap gap-1 mt-0.5">
                    {letter.recipients.cc.map((r, i) => (
                      <span
                        key={i}
                        className="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-panel text-text-muted border border-border font-medium"
                      >
                        {r.type === "UNIT" ? r.unit.name : r.alias.label}
                      </span>
                    ))}
                  </div>
                </MetaRow>
              )}
            </dl>
          </div>

          {/* Tautan ke halaman detail surat lengkap */}
          <Link
            href={`/surat/${item.letter.id}`}
            className="flex items-center justify-between gap-2 w-full px-4 py-3 border border-border rounded-xl hover:bg-panel transition-colors text-sm text-text font-medium"
          >
            <span className="flex items-center gap-2">
              <ExternalLink size={14} className="text-text-muted shrink-0" />
              Buka Detail Surat Lengkap
            </span>
            <ChevronRight size={14} className="text-text-muted shrink-0" />
          </Link>
        </div>
      </div>

      {/* ── Sticky Action Bar ── */}
      {hasActions && (
        <div className="fixed bottom-0 left-56 right-0 z-20 bg-surface border-t border-border shadow-lg px-6 py-3 flex items-center justify-end gap-2">
          {canRedirect && (
            <button
              type="button"
              onClick={() => setModal("redirect")}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium border border-border text-text hover:bg-panel transition-colors"
            >
              <ArrowRight size={15} /> Alihkan
            </button>
          )}
          {canDispose && (
            <button
              type="button"
              onClick={() => setModal("disposisi")}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium bg-primary text-white hover:bg-primary-dark transition-colors"
            >
              <Share2 size={15} /> Disposisi
            </button>
          )}
        </div>
      )}

      {/* ── Modals ── */}
      {modal === "disposisi" && (
        <DisposisiModal
          itemId={itemId}
          onClose={() => setModal(null)}
          onSuccess={() => {
            setModal(null);
            void load();
          }}
        />
      )}
      {modal === "redirect" && (
        <RedirectModal
          itemId={itemId}
          onClose={() => setModal(null)}
          onSuccess={() => {
            setModal(null);
            void load();
          }}
        />
      )}
    </div>
  );
}
