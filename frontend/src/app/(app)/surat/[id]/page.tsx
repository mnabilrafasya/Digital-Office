"use client";

// ─────────────────────────────────────────────
// app/(app)/surat/[id]/page.tsx
// Halaman detail / review surat — real backend
// ─────────────────────────────────────────────

import { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import {
  ChevronRight,
  FileText,
  File,
  FileSpreadsheet,
  Eye,
  Download,
  Loader2,
  AlertCircle,
  ArrowLeft,
  Pencil,
} from "lucide-react";
import {
  getLetter,
  formatBytes,
  isEditableByDrafter,
} from "@/lib/api/letter-detail";
import type { LetterDetail } from "@/lib/api/letter-detail";
import { StatusBadge } from "@/components/daftar-surat/status-badge";
import type { LetterStatus } from "@/lib/types";
import { LetterSteps } from "@/components/review/letter-steps";
import { ReviewModals } from "@/components/review/review-modals";
import { ReviewDocumentViewer } from "@/components/review/document-viewer";
import { ApproverPanel } from "@/components/review/approver-panel";
import { cn } from "@/lib/utils";

// ─── Helper ──────────────────────────────────

function ChipPill({
  label,
  value,
  color,
}: {
  label: string;
  value: string;
  color?: string;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 text-xs px-2.5 py-1 rounded-full font-medium",
        color ?? "bg-panel text-text-muted border border-border"
      )}
    >
      <span className="text-text-muted/70">{label}:</span>
      <span className="text-text font-semibold">{value}</span>
    </span>
  );
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

function fileIcon(contentType: string) {
  if (contentType.includes("pdf"))
    return <File size={16} className="text-red-500 shrink-0" />;
  if (
    contentType.includes("spreadsheet") ||
    contentType.includes("excel")
  )
    return <FileSpreadsheet size={16} className="text-emerald-600 shrink-0" />;
  return <FileText size={16} className="text-blue-500 shrink-0" />;
}

// ─── Page ────────────────────────────────────

export default function SuratDetailPage() {
  const { id }  = useParams<{ id: string }>();
  const router  = useRouter();

  const [letter,  setLetter]  = useState<LetterDetail | null>(null);
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
    } catch (e) {
      setError((e as Error).message ?? "Gagal memuat surat.");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { load(); }, [load]);

  // ── Redirect: drafter ke form edit ──
  useEffect(() => {
    if (!letter) return;
    if (isEditableByDrafter(letter.status, letter.allowedActions)) {
      router.replace(`/surat/baru?draft=${id}`);
    }
  }, [letter, id, router]);

  // ── Loading ──
  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh] gap-2 text-text-muted">
        <Loader2 size={18} className="animate-spin text-primary" />
        <span className="text-sm">Memuat surat...</span>
      </div>
    );
  }

  // ── Error ──
  if (error || !letter) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-3 text-center">
        <AlertCircle size={32} className="text-status-danger-text" />
        <p className="text-sm text-text-muted">{error ?? "Surat tidak ditemukan."}</p>
        <button
          type="button"
          onClick={() => router.push("/surat/daftar")}
          className="text-sm text-primary hover:underline flex items-center gap-1"
        >
          <ArrowLeft size={13} /> Kembali ke Daftar Surat
        </button>
      </div>
    );
  }

  // ── Jangan render kalau sedang redirect ke form edit ──
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
  const hasActions = letter.allowedActions.some((a) =>
    ["APPROVE", "RETURN", "CANCEL"].includes(a)
  );

  // Label peran reviewer/approver yang sedang aktif
  const activeRole = letter.currentAssignee
    ? letter.steps.find(
        (s) => s.state === "ACTIVE" && s.user.id === letter.currentAssignee?.id
      )
    : null;

  const isApprover =
    activeRole?.role === "APPROVER" || status === "IN_APPROVAL";

  return (
    <div className={cn("space-y-4", hasActions && "pb-20")}>
      {/* ── Breadcrumb ── */}
      <nav className="flex items-center gap-1 text-xs text-text-muted flex-wrap">
        <Link href="/dashboard" className="hover:text-text transition-colors">
          Beranda
        </Link>
        <ChevronRight size={12} />
        <Link href="/surat/daftar" className="hover:text-text transition-colors">
          Penciptaan Surat
        </Link>
        <ChevronRight size={12} />
        <Link href="/surat/daftar" className="hover:text-text transition-colors">
          Daftar Surat
        </Link>
        <ChevronRight size={12} />
        <span className="text-text font-medium">
          {hasActions ? "Review Surat" : "Detail Surat"}
        </span>
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
          <ChipPill
            label="Sifat"
            value={letter.confidentiality.name}
            color={
              letter.confidentiality.code === "BIASA"
                ? "bg-status-info-bg text-status-info-text border border-status-info-text/20"
                : letter.confidentiality.code === "RAHASIA"
                ? "bg-status-danger-bg text-status-danger-text border border-status-danger-text/20"
                : "bg-status-warn-bg text-status-warn-text border border-status-warn-text/20"
            }
          />
        )}
        {letter.urgency && (
          <ChipPill
            label="Kecepatan"
            value={letter.urgency.name}
            color={
              letter.urgency.code === "SANGAT_SEGERA"
                ? "bg-status-danger-bg text-status-danger-text border border-status-danger-text/20"
                : letter.urgency.code === "SEGERA"
                ? "bg-status-warn-bg text-status-warn-text border border-status-warn-text/20"
                : "bg-panel text-text-muted border border-border"
            }
          />
        )}
        <span className="inline-flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full font-medium bg-panel border border-border text-text-muted">
          Status: <StatusBadge status={status} />
          {activeRole && (
            <span className="font-medium text-text">
              ({activeRole.role === "APPROVER"
                ? "Approver"
                : `Reviewer ${activeRole.order}`}{" "}
              · {activeRole.user.name})
            </span>
          )}
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

      {/* ── Banner: dikembalikan ke Drafter (hanya baca — redirect seharusnya sudah terjadi) ── */}
      {isReturned && status === "RETURNED_TO_DRAFTER" && letter.lastReturnNote && (
        <div className="flex items-start gap-2.5 bg-status-danger-bg border border-status-danger-text/30 text-status-danger-text rounded-lg px-4 py-3 text-sm">
          <AlertCircle size={16} className="shrink-0 mt-0.5" />
          <div>
            <p className="font-semibold">Catatan Revisi untuk Drafter</p>
            <p className="mt-0.5 opacity-80">{letter.lastReturnNote}</p>
            <Link
              href={`/surat/baru?draft=${letter.id}`}
              className="inline-flex items-center gap-1 mt-2 text-xs font-semibold underline underline-offset-2"
            >
              <Pencil size={11} /> Buka Form Revisi
            </Link>
          </div>
        </div>
      )}

      {/* ── Dua kolom ── */}
      <div className="grid grid-cols-1 lg:grid-cols-[1fr_280px] gap-5">
        {/* ─ Kolom Kiri ─ */}
        <div className="space-y-4">
          {/* Pratinjau & Editor Dokumen DevExpress */}
          <ReviewDocumentViewer letter={letter} />

          {/* Panel Khusus Pengesahan Approver (Tanda Tangan Digital, QR Code, Disposisi) */}
          {isApprover && hasActions && (
            <ApproverPanel letter={letter} onDone={load} />
          )}

          {/* Lampiran */}
          {letter.attachments.length > 0 && (
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
                {letter.attachments.map((att) => (
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
                      <button
                        type="button"
                        disabled
                        className="text-xs text-text-muted flex items-center gap-1 opacity-60"
                      >
                        <Eye size={13} /> Pratinjau
                      </button>
                      <button
                        type="button"
                        disabled
                        className="text-xs text-text-muted flex items-center gap-1 opacity-60"
                      >
                        <Download size={13} /> Unduh
                      </button>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Tidak ada lampiran */}
          {letter.attachments.length === 0 && (
            <div className="border border-border rounded-xl px-4 py-6 text-center text-xs text-text-muted">
              Tidak ada berkas lampiran.
            </div>
          )}
        </div>

        {/* ─ Kolom Kanan ─ */}
        <div className="space-y-4">
          {/* Kartu Metadata */}
          <div className="border border-border rounded-xl overflow-hidden">
            <div className="px-4 py-3 border-b border-border bg-panel/30">
              <h3 className="text-sm font-semibold text-text">
                Metadata &amp; Ringkasan
              </h3>
            </div>
            <dl className="px-4 py-3 space-y-3">
              {letter.letterType && (
                <MetaRow label="Jenis Surat">{letter.letterType.name}</MetaRow>
              )}
              {letter.classification && (
                <MetaRow label="Klasifikasi">
                  <span className="font-mono text-xs">
                    {letter.classification.code}
                  </span>{" "}
                  — {letter.classification.name}
                </MetaRow>
              )}
              <MetaRow label="Pejabat Pembuat">
                {letter.drafter.name}
                <span className="block text-xs text-text-muted">
                  {letter.drafter.positionName}
                </span>
              </MetaRow>
              {letter.steps
                .filter((s) => s.role === "APPROVER")
                .map((s) => (
                  <MetaRow key={s.order} label="Penandatangan">
                    {s.user.name}
                    <span className="block text-xs text-text-muted">
                      {s.user.positionName}
                    </span>
                  </MetaRow>
                ))}
              {letter.recipients.to.length > 0 && (
                <MetaRow label="Tujuan / Penerima">
                  <div className="flex flex-wrap gap-1 mt-0.5">
                    {letter.recipients.to.map((r, i) => (
                      <span
                        key={i}
                        className="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-primary/10 text-primary font-medium"
                      >
                        {r.type === "UNIT"
                          ? r.unit.name
                          : r.alias.label}
                      </span>
                    ))}
                  </div>
                </MetaRow>
              )}
              {letter.recipients.cc.length > 0 && (
                <MetaRow label="Tembusan (CC)">
                  <div className="flex flex-wrap gap-1 mt-0.5">
                    {letter.recipients.cc.map((r, i) => (
                      <span
                        key={i}
                        className="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-panel text-text-muted border border-border font-medium"
                      >
                        {r.type === "UNIT"
                          ? r.unit.name
                          : r.alias.label}
                      </span>
                    ))}
                  </div>
                </MetaRow>
              )}
              {letter.summary && (
                <MetaRow label="Ringkasan">
                  <p className="text-sm text-text-muted">{letter.summary}</p>
                </MetaRow>
              )}
              {letter.specialNote && (
                <MetaRow label="Keterangan Khusus">
                  <p className="text-sm text-status-warn-text">
                    {letter.specialNote}
                  </p>
                </MetaRow>
              )}
            </dl>
          </div>

          {/* Kartu Alur & Riwayat */}
          <div className="border border-border rounded-xl overflow-hidden">
            <div className="px-4 py-3 border-b border-border bg-panel/30">
              <h3 className="text-sm font-semibold text-text">
                Alur &amp; Riwayat
              </h3>
            </div>
            <div className="px-4 py-4">
              <LetterSteps steps={letter.steps} />
            </div>
          </div>
        </div>
      </div>

      {/* ── Action Bar + Modals Reviewer (Hanya jika bukan Approver) ── */}
      {!isApprover && hasActions && (
        <ReviewModals letter={letter} onDone={load} />
      )}
    </div>
  );
}
