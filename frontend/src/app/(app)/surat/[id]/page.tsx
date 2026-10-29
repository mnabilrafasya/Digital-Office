"use client";

// app/(app)/surat/[id]/page.tsx
// Halaman detail / review surat — real backend

import { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import {
  ChevronRight, FileText, File, FileSpreadsheet, Eye, Download,
  Loader2, AlertCircle, ArrowLeft, Pencil, QrCode, CheckCircle2,
  Clock, ArrowRight,
} from "lucide-react";
import {
  getLetter, formatBytes, isEditableByDrafter, getLetterHistory,
} from "@/lib/api/letter-detail";
import type { LetterDetail, LetterHistoryEntry } from "@/lib/api/letter-detail";
import { StatusBadge } from "@/components/daftar-surat/status-badge";
import type { LetterStatus } from "@/lib/types";
import { LetterSteps } from "@/components/review/letter-steps";
import { ReviewModals } from "@/components/review/review-modals";
import { ReviewDocumentViewer } from "@/components/review/document-viewer";
import { ApproverPanel } from "@/components/review/approver-panel";
import { cn } from "@/lib/utils";

// ─── Helper kecil ────────────────────────────

function ChipPill({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <span className={cn("inline-flex items-center gap-1 text-xs px-2.5 py-1 rounded-full font-medium", color ?? "bg-panel text-text-muted border border-border")}>
      <span className="text-text-muted/70">{label}:</span>
      <span className="text-text font-semibold">{value}</span>
    </span>
  );
}

function MetaRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <dt className="text-[11px] font-semibold uppercase tracking-wide text-text-muted mb-0.5">{label}</dt>
      <dd className="text-sm text-text">{children}</dd>
    </div>
  );
}

function fileIcon(contentType: string) {
  if (contentType.includes("pdf")) return <File size={16} className="text-red-500 shrink-0" />;
  if (contentType.includes("spreadsheet") || contentType.includes("excel"))
    return <FileSpreadsheet size={16} className="text-emerald-600 shrink-0" />;
  return <FileText size={16} className="text-blue-500 shrink-0" />;
}

function formatDateTime(iso: string) {
  return new Intl.DateTimeFormat("id-ID", {
    day: "numeric", month: "long", year: "numeric",
    hour: "2-digit", minute: "2-digit", timeZoneName: "short",
  }).format(new Date(iso));
}

const ACTION_LABEL: Record<string, string> = {
  SUBMIT: "Dikirim ke Reviewer",
  APPROVE: "Disetujui",
  RETURN: "Dikembalikan",
  READ: "Dibaca",
  SIGN: "Ditandatangani & Diterbitkan",
};

// ─── Page ────────────────────────────────────

export default function SuratDetailPage() {
  const { id }  = useParams<{ id: string }>();
  const router  = useRouter();

  const [letter,  setLetter]  = useState<LetterDetail | null>(null);
  const [history, setHistory] = useState<LetterHistoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error,   setError]   = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getLetter(id);
      if (!data) {
        setError("Surat tidak ditemukan atau Anda tidak memiliki akses.");
        return;
      }
      setLetter(data);
      // Muat riwayat secara paralel, gagal tidak fatal
      getLetterHistory(id).then(setHistory).catch(() => {});
    } catch (e) {
      setError((e as Error).message ?? "Gagal memuat surat.");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { load(); }, [load]);

  useEffect(() => {
    if (!letter) return;
    if (isEditableByDrafter(letter.status, letter.allowedActions)) {
      router.replace(`/surat/baru?draft=${id}`);
    }
  }, [letter, id, router]);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh] gap-2 text-text-muted">
        <Loader2 size={18} className="animate-spin text-primary" />
        <span className="text-sm">Memuat surat...</span>
      </div>
    );
  }

  if (error || !letter) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-3 text-center">
        <AlertCircle size={32} className="text-status-danger-text" />
        <p className="text-sm text-text-muted">{error ?? "Surat tidak ditemukan."}</p>
        <button type="button" onClick={() => router.push("/surat/daftar")}
          className="text-sm text-primary hover:underline flex items-center gap-1">
          <ArrowLeft size={13} /> Kembali ke Daftar Surat
        </button>
      </div>
    );
  }

  if (isEditableByDrafter(letter.status, letter.allowedActions)) {
    return (
      <div className="flex items-center justify-center min-h-[60vh] gap-2 text-text-muted">
        <Loader2 size={18} className="animate-spin text-primary" />
        <span className="text-sm">Membuka form edit surat...</span>
      </div>
    );
  }

  const status     = letter.status as LetterStatus;
  const isReturned = status === "RETURNED_TO_DRAFTER" || status === "RETURNED_TO_REVIEWER";
  const hasActions = (letter.allowedActions ?? []).some((a) => ["APPROVE", "RETURN", "CANCEL"].includes(a.toUpperCase()));
  const isApprover = letter.steps.some((s) => s.state === "ACTIVE" && s.role === "APPROVER") || status === "IN_APPROVAL";

  return (
    <div className={cn("space-y-4", hasActions && "pb-20")}>
      {/* ── Breadcrumb ── */}
      <nav className="flex items-center gap-1 text-xs text-text-muted flex-wrap">
        <Link href="/dashboard" className="hover:text-text transition-colors">Beranda</Link>
        <ChevronRight size={12} />
        <Link href="/surat/daftar" className="hover:text-text transition-colors">Daftar Surat</Link>
        <ChevronRight size={12} />
        <span className="text-text font-medium">{hasActions ? "Review Surat" : "Detail Surat"}</span>
        {letter.letterNumber && (
          <>
            <ChevronRight size={12} />
            <span className="text-primary font-semibold">{letter.letterNumber}</span>
          </>
        )}
      </nav>

      {/* ── Chips status ── */}
      <div className="flex items-center flex-wrap gap-2">
        {letter.confidentiality && (
          <ChipPill label="Sifat" value={letter.confidentiality.name}
            color={letter.confidentiality.code === "BIASA"
              ? "bg-status-info-bg text-status-info-text border border-status-info-text/20"
              : letter.confidentiality.code === "RAHASIA"
              ? "bg-status-danger-bg text-status-danger-text border border-status-danger-text/20"
              : "bg-status-warn-bg text-status-warn-text border border-status-warn-text/20"} />
        )}
        {letter.urgency && (
          <ChipPill label="Kecepatan" value={letter.urgency.name}
            color={letter.urgency.code === "SANGAT_SEGERA"
              ? "bg-status-danger-bg text-status-danger-text border border-status-danger-text/20"
              : letter.urgency.code === "SEGERA"
              ? "bg-status-warn-bg text-status-warn-text border border-status-warn-text/20"
              : "bg-panel text-text-muted border border-border"} />
        )}
        <span className="inline-flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full font-medium bg-panel border border-border text-text-muted">
          Status: <StatusBadge status={status} />
        </span>
      </div>

      {/* ── Judul ── */}
      <h1 className="text-xl font-bold text-text leading-snug">
        {letter.title ?? "(Tanpa perihal)"}
      </h1>

      {/* ── Banner: dikembalikan ke Reviewer ── */}
      {status === "RETURNED_TO_REVIEWER" && letter.lastReturnNote && (
        <div className="flex items-start gap-2.5 bg-status-warn-bg border border-status-warn-text/30 text-status-warn-text rounded-lg px-4 py-3 text-sm">
          <AlertCircle size={16} className="shrink-0 mt-0.5" />
          <div>
            <p className="font-semibold">Catatan dari Approver</p>
            <p className="mt-0.5 opacity-80">{letter.lastReturnNote}</p>
          </div>
        </div>
      )}

      {/* ── Banner: dikembalikan ke Drafter ── */}
      {isReturned && status === "RETURNED_TO_DRAFTER" && letter.lastReturnNote && (
        <div className="flex items-start gap-2.5 bg-status-danger-bg border border-status-danger-text/30 text-status-danger-text rounded-lg px-4 py-3 text-sm">
          <AlertCircle size={16} className="shrink-0 mt-0.5" />
          <div>
            <p className="font-semibold">Catatan Revisi untuk Drafter</p>
            <p className="mt-0.5 opacity-80">{letter.lastReturnNote}</p>
            <Link href={`/surat/baru?draft=${letter.id}`}
              className="inline-flex items-center gap-1 mt-2 text-xs font-semibold underline underline-offset-2">
              <Pencil size={11} /> Buka Form Revisi
            </Link>
          </div>
        </div>
      )}

      {/* ── Dua kolom ── */}
      <div className="grid grid-cols-1 lg:grid-cols-[1fr_280px] gap-5">
        {/* ─ Kolom Kiri ─ */}
        <div className="space-y-4">
          <ReviewDocumentViewer letter={letter} />

          {/* Panel info approver (IN_APPROVAL) */}
          {isApprover && hasActions && (
            <ApproverPanel letter={letter} />
          )}

          {/* Kartu Pengesahan & QR — hanya setelah SENT */}
          {status === "SENT" && letter.approval && (
            <div className="border border-border rounded-xl bg-surface overflow-hidden">
              <div className="px-4 py-3 border-b border-border bg-panel/30 flex items-center gap-2">
                <QrCode size={16} className="text-primary" />
                <h3 className="text-sm font-semibold text-text">Pengesahan & QR Persetujuan</h3>
              </div>
              <div className="p-4 flex items-start gap-4 flex-wrap">
                {/* QR image dari backend */}
                <div className="w-28 h-28 border border-border rounded-lg bg-white p-1 shrink-0 flex items-center justify-center">
                  <img
                    src={letter.approval.qrUrl}
                    alt="QR Persetujuan"
                    className="w-full h-full object-contain"
                    onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = "none"; }}
                  />
                </div>
                <div className="flex-1 min-w-0 space-y-2">
                  <div>
                    <p className="text-[11px] font-semibold uppercase tracking-wide text-text-muted mb-0.5">Disetujui oleh</p>
                    <p className="text-sm font-bold text-text">{letter.approval.approvedBy.name}</p>
                    <p className="text-xs text-text-muted">{letter.approval.approvedBy.positionName}</p>
                  </div>
                  <div className="flex items-center gap-1.5 text-xs text-text-muted">
                    <CheckCircle2 size={13} className="text-status-success-text shrink-0" />
                    <span>Tanggal: <strong className="text-text">{formatDateTime(letter.approval.approvedAt)}</strong></span>
                  </div>
                  {letter.letterNumber && (
                    <div className="text-xs text-text-muted">
                      Nomor: <strong className="text-text font-mono">{letter.letterNumber}</strong>
                    </div>
                  )}
                  <a
                    href={`/api/v1/letters/${letter.id}/document`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-lg bg-primary text-white hover:bg-primary-dark transition-colors"
                  >
                    <Download size={13} /> Unduh PDF Resmi
                  </a>
                </div>
              </div>
            </div>
          )}

          {/* Lampiran */}
          {letter.attachments.length > 0 && (
            <div className="border border-border rounded-xl overflow-hidden">
              <div className="px-4 py-3 border-b border-border bg-panel/30">
                <h3 className="text-sm font-semibold text-text">
                  Lampiran Berkas Pendukung ({letter.attachments.length} Berkas)
                </h3>
                <p className="text-xs text-text-muted">
                  Total {formatBytes(letter.attachments.reduce((s, a) => s + a.size, 0))}
                </p>
              </div>
              <ul className="divide-y divide-border">
                {letter.attachments.map((att) => (
                  <li key={att.id} className="flex items-center gap-3 px-4 py-3">
                    {fileIcon(att.contentType)}
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-text truncate">{att.fileName}</p>
                      <p className="text-xs text-text-muted">{formatBytes(att.size)}</p>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <button type="button" disabled className="text-xs text-text-muted flex items-center gap-1 opacity-60">
                        <Eye size={13} /> Pratinjau
                      </button>
                      <button type="button" disabled className="text-xs text-text-muted flex items-center gap-1 opacity-60">
                        <Download size={13} /> Unduh
                      </button>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {letter.attachments.length === 0 && (
            <div className="border border-border rounded-xl px-4 py-6 text-center text-xs text-text-muted">
              Tidak ada berkas lampiran.
            </div>
          )}
        </div>

        {/* ─ Kolom Kanan ─ */}
        <div className="space-y-4">
          {/* Metadata */}
          <div className="border border-border rounded-xl overflow-hidden">
            <div className="px-4 py-3 border-b border-border bg-panel/30">
              <h3 className="text-sm font-semibold text-text">Metadata & Ringkasan</h3>
            </div>
            <dl className="px-4 py-3 space-y-3">
              {letter.letterType && <MetaRow label="Jenis Surat">{letter.letterType.name}</MetaRow>}
              {letter.classification && (
                <MetaRow label="Klasifikasi">
                  <span className="font-mono text-xs">{letter.classification.code}</span>{" "}— {letter.classification.name}
                </MetaRow>
              )}
              <MetaRow label="Pejabat Pembuat">
                {letter.drafter.name}
                <span className="block text-xs text-text-muted">{letter.drafter.positionName}</span>
              </MetaRow>
              {letter.steps.filter((s) => s.role === "APPROVER").map((s) => (
                <MetaRow key={s.order} label="Penandatangan">
                  {s.user.name}
                  <span className="block text-xs text-text-muted">{s.user.positionName}</span>
                </MetaRow>
              ))}
              {letter.recipients.to.length > 0 && (
                <MetaRow label="Tujuan / Penerima">
                  <div className="flex flex-wrap gap-1.5 mt-0.5">
                    {letter.recipients.to.map((r, i) => (
                      <span key={i} className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs bg-primary/10 text-primary font-medium">
                        <span>{r.type === "UNIT" ? r.unit.name : r.alias.label}</span>
                        {r.type === "UNIT" && r.resolvedHead && (
                          <span className="font-normal opacity-75">
                            ({r.resolvedHead.name})
                          </span>
                        )}
                      </span>
                    ))}
                  </div>
                </MetaRow>
              )}
              {letter.recipients.cc.length > 0 && (
                <MetaRow label="Tembusan (CC)">
                  <div className="flex flex-wrap gap-1.5 mt-0.5">
                    {letter.recipients.cc.map((r, i) => (
                      <span key={i} className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs bg-panel text-text-muted border border-border font-medium">
                        <span>{r.type === "UNIT" ? r.unit.name : r.alias.label}</span>
                        {r.type === "UNIT" && r.resolvedHead && (
                          <span className="font-normal opacity-75">
                            ({r.resolvedHead.name})
                          </span>
                        )}
                      </span>
                    ))}
                  </div>
                </MetaRow>
              )}
              {letter.summary && <MetaRow label="Ringkasan"><p className="text-sm text-text-muted">{letter.summary}</p></MetaRow>}
              {letter.specialNote && (
                <MetaRow label="Keterangan Khusus">
                  <p className="text-sm text-status-warn-text">{letter.specialNote}</p>
                </MetaRow>
              )}
            </dl>
          </div>

          {/* Alur & Riwayat */}
          <div className="border border-border rounded-xl overflow-hidden">
            <div className="px-4 py-3 border-b border-border bg-panel/30">
              <h3 className="text-sm font-semibold text-text">Alur & Riwayat</h3>
            </div>
            <div className="px-4 py-4">
              <LetterSteps steps={letter.steps} />
            </div>
            {/* Riwayat aksi dari backend */}
            {history.length > 0 && (
              <div className="border-t border-border px-4 py-3 space-y-3">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-text-muted">Riwayat Aksi</p>
                <ol className="space-y-2.5">
                  {history.map((h, i) => (
                    <li key={i} className="flex items-start gap-2 text-xs">
                      <div className="w-5 h-5 rounded-full bg-panel border border-border flex items-center justify-center shrink-0 mt-0.5">
                        {h.action === "APPROVE" || h.action === "SIGN"
                          ? <CheckCircle2 size={10} className="text-status-success-text" />
                          : h.action === "RETURN"
                          ? <ArrowRight size={10} className="text-status-warn-text rotate-180" />
                          : <Clock size={10} className="text-text-muted" />}
                      </div>
                      <div className="min-w-0">
                        <span className="font-semibold text-text">{ACTION_LABEL[h.action] ?? h.action}</span>
                        <span className="text-text-muted"> oleh {h.actor.name}</span>
                        {h.target && <span className="text-text-muted"> → {h.target.name}</span>}
                        {h.note && <p className="text-text-muted mt-0.5 italic">&quot;{h.note}&quot;</p>}
                        <p className="text-text-muted/70 mt-0.5">
                          {new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }).format(new Date(h.createdAt))}
                        </p>
                      </div>
                    </li>
                  ))}
                </ol>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* ── Action Bar + Modals (Reviewer & Approver) ── */}
      {hasActions && (
        <ReviewModals letter={letter} onDone={load} />
      )}
    </div>
  );
}
