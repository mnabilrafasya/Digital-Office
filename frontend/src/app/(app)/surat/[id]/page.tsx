"use client";

// app/(app)/surat/[id]/page.tsx
// Halaman detail / review surat — real backend

import { useEffect, useState, useCallback, useRef } from "react";
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
  QrCode,
  CheckCircle2,
  Printer,
  ExternalLink,
  UploadCloud,
  Building2,
  Inbox,
  UserCheck,
  Info,
} from "lucide-react";
import {
  getLetter,
  formatBytes,
  isEditableByDrafter,
  getLetterHistory,
} from "@/lib/api/letter-detail";
import type { LetterDetail, LetterHistoryEntry } from "@/lib/api/letter-detail";
import {
  getLetterAttachmentDownloadUrl,
  officialPdfUrl,
  openLetterAttachmentInline,
  uploadSignedPdf,
} from "@/lib/api/letter-document";
import { StatusBadge } from "@/components/daftar-surat/status-badge";
import type { LetterStatus } from "@/lib/types";
import { CompactAuditTrail } from "@/components/review/compact-audit-trail";
import { ReviewModals } from "@/components/review/review-modals";
import { ReviewDocumentViewer } from "@/components/review/document-viewer";
import { ApproverPanel } from "@/components/review/approver-panel";
import { LetterTrackingPanel } from "@/components/review/letter-tracking-panel";
import { cn } from "@/lib/utils";

// ─── Helper kecil ────────────────────────────

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
  if (contentType.includes("spreadsheet") || contentType.includes("excel"))
    return <FileSpreadsheet size={16} className="text-emerald-600 shrink-0" />;
  return <FileText size={16} className="text-blue-500 shrink-0" />;
}

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

// ─── Page ────────────────────────────────────

export default function SuratDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();

  const [letter, setLetter] = useState<LetterDetail | null>(null);
  const [history, setHistory] = useState<LetterHistoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pdfReady, setPdfReady] = useState(false);
  const [copyFor, setCopyFor] = useState<string | null>(null);
  const [signedScanFile, setSignedScanFile] = useState<File | null>(null);
  const [uploadingSigned, setUploadingSigned] = useState(false);
  const [uploadSignedError, setUploadSignedError] = useState<string | null>(
    null
  );
  const signedScanInputRef = useRef<HTMLInputElement>(null);

  const handlePdfReadyChange = useCallback(
    (ready: boolean, owner: string | null) => {
      setPdfReady(ready);
      setCopyFor(owner);
    },
    []
  );

  const load = useCallback(async () => {
    try {
      const data = await getLetter(id);
      if (!data) {
        setError("Surat tidak ditemukan atau Anda tidak memiliki akses.");
        return;
      }
      setLetter(data);
      setError(null);
      // Muat riwayat secara paralel, gagal tidak fatal
      getLetterHistory(id)
        .then(setHistory)
        .catch(() => {});
    } catch (e) {
      setError((e as Error).message ?? "Gagal memuat surat.");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    let cancelled = false;
    getLetter(id)
      .then((data) => {
        if (cancelled) return;
        if (!data) {
          setError("Surat tidak ditemukan atau Anda tidak memiliki akses.");
          return;
        }
        setLetter(data);
        setError(null);
        getLetterHistory(id)
          .then((items) => {
            if (!cancelled) setHistory(items);
          })
          .catch(() => {});
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setError(
            e instanceof Error ? e.message : "Gagal memuat surat."
          );
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

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
        <p className="text-sm text-text-muted">
          {error ?? "Surat tidak ditemukan."}
        </p>
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

  if (isEditableByDrafter(letter.status, letter.allowedActions)) {
    return (
      <div className="flex items-center justify-center min-h-[60vh] gap-2 text-text-muted">
        <Loader2 size={18} className="animate-spin text-primary" />
        <span className="text-sm">Membuka form edit surat...</span>
      </div>
    );
  }

  const status = letter.status as LetterStatus;
  const isReturned =
    status === "RETURNED_TO_DRAFTER" || status === "RETURNED_TO_REVIEWER";
  const hasActions = (letter.allowedActions ?? []).some((a) =>
    ["APPROVE", "RETURN", "UPLOAD_SIGNED"].includes(a.toUpperCase())
  );
  const canUploadSigned = (letter.allowedActions ?? [])
    .map((a) => a.toUpperCase())
    .includes("UPLOAD_SIGNED");
  const isApprover =
    letter.steps.some((s) => s.state === "ACTIVE" && s.role === "APPROVER") ||
    status === "IN_APPROVAL";
  const isExternalOutgoing =
    letter.direction === "OUTGOING" ||
    letter.letterType?.code === "SURAT_EKSTERNAL";
  const isExternalIncoming =
    letter.direction === "INCOMING" ||
    letter.status === "RECEIVED" ||
    letter.letterType?.code === "SURAT_MASUK_EKSTERNAL";

  async function handleSaveSignedScan() {
    if (!signedScanFile || !letter) return;
    setUploadingSigned(true);
    setUploadSignedError(null);
    try {
      await uploadSignedPdf(letter.id, signedScanFile);
      setSignedScanFile(null);
      await load();
    } catch (e) {
      setUploadSignedError(
        e instanceof Error
          ? e.message
          : "Gagal menyimpan dokumen scan bertanda tangan basah."
      );
    } finally {
      setUploadingSigned(false);
    }
  }

  return (
    <div className={cn("space-y-4", hasActions && "pb-20")}>
      {/* ── Breadcrumb ── */}
      <nav className="flex items-center gap-1 text-xs text-text-muted flex-wrap">
        <Link href="/dashboard" className="hover:text-text transition-colors">
          Beranda
        </Link>
        <ChevronRight size={12} />
        <Link
          href="/surat/daftar"
          className="hover:text-text transition-colors"
        >
          Daftar Surat
        </Link>
        <ChevronRight size={12} />
        <span className="text-text font-medium">
          {hasActions ? "Review Surat" : "Detail Surat"}
        </span>
        {letter.letterNumber && (
          <>
            <ChevronRight size={12} />
            <span className="text-primary font-semibold">
              {letter.letterNumber}
            </span>
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
        </span>
      </div>

      {/* ── Judul ── */}
      <h1 className="text-xl font-bold text-text leading-snug">
        {letter.title ?? "(Tanpa perihal)"}
      </h1>

      {/* ── Banner: Pengerjaan Atas Nama Delegasi ── */}
      {letter.onBehalfOf && (
        <div className="flex items-start gap-2.5 bg-primary/10 border border-primary/25 text-primary rounded-xl px-4 py-3 text-sm">
          <UserCheck size={16} className="shrink-0 mt-0.5" />
          <div className="space-y-0.5">
            <p className="font-semibold text-xs uppercase tracking-wide">
              Pengerjaan Surat Atas Nama Rekan Kerja
            </p>
            <p className="text-xs text-text">
              Anda memeriksa dan memproses surat ini atas nama{" "}
              <strong>{letter.onBehalfOf.name}</strong>{" "}
              ({letter.onBehalfOf.positionName}) berdasarkan jadwal delegasi yang sedang aktif.
            </p>
          </div>
        </div>
      )}

      {/* ── Info: Tugas Sedang Didelegasikan ── */}
      {letter.delegatedTo && (
        <div className="flex items-center gap-2.5 bg-panel border border-border text-text-muted rounded-xl px-4 py-2.5 text-xs">
          <Info size={14} className="shrink-0 text-primary" />
          <span>
            Tugas penerima saat ini sedang didelegasikan ke{" "}
            <strong className="text-text">{letter.delegatedTo.delegate.name}</strong>{" "}
            ({letter.delegatedTo.delegate.positionName}) sampai{" "}
            <strong className="text-text">{letter.delegatedTo.until}</strong>.
          </span>
        </div>
      )}

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
      {isReturned &&
        status === "RETURNED_TO_DRAFTER" &&
        letter.lastReturnNote && (
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
      <div className="grid grid-cols-1 lg:grid-cols-[minmax(0,1fr)_280px] gap-5">
        {/* ─ Kolom Kiri ─ */}
        <div className="min-w-0 space-y-4">
          <ReviewDocumentViewer
            letter={letter}
            onPdfReadyChange={handlePdfReadyChange}
          />

          {/* Panel info approver (IN_APPROVAL) */}
          {isApprover && hasActions && <ApproverPanel letter={letter} />}

          {/* Kartu Alur Cetak PDF -> Tanda Tangan Basah -> Unggah Scan PDF (Status APPROVED pada Surat Eksternal Keluar) */}
          {status === "APPROVED" && (
            <div className="border-2 border-status-warn-text/30 rounded-xl bg-surface overflow-hidden shadow-xs">
              <div className="px-4 py-3 border-b border-status-warn-text/20 bg-status-warn-bg/50 flex items-center justify-between gap-2 flex-wrap">
                <div className="flex items-center gap-2">
                  <Printer size={16} className="text-status-warn-text" />
                  <h3 className="text-sm font-bold text-text">
                    Tahap Tanda Tangan Basah &amp; Unggah Dokumen PDF
                  </h3>
                </div>
                {letter.letterNumber && (
                  <span className="text-xs font-mono font-semibold px-2.5 py-0.5 rounded-full bg-surface border border-border text-text">
                    No: {letter.letterNumber}
                  </span>
                )}
              </div>

              <div className="p-4 space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-3 text-xs">
                  <div className="p-3 rounded-lg bg-panel/50 border border-border space-y-1">
                    <p className="font-bold text-text">1. Cetak Surat (PDF)</p>
                    <p className="text-text-muted leading-relaxed">
                      Unduh atau cetak dokumen PDF ber-watermark di atas yang telah diberi nomor surat resmi.
                    </p>
                  </div>
                  <div className="p-3 rounded-lg bg-panel/50 border border-border space-y-1">
                    <p className="font-bold text-text">2. Tanda Tangan Basah</p>
                    <p className="text-text-muted leading-relaxed">
                      Pejabat penandatangan ({letter.approval?.approvedBy.name ?? "Approver"}) menandatangani basah dokumen cetak lalu di-scan menjadi PDF.
                    </p>
                  </div>
                  <div className="p-3 rounded-lg bg-panel/50 border border-border space-y-1">
                    <p className="font-bold text-text">3. Unggah Scan &amp; Simpan</p>
                    <p className="text-text-muted leading-relaxed">
                      Unggah dokumen scan PDF di bawah lalu klik Simpan agar surat masuk ke halaman Surat Eksternal.
                    </p>
                  </div>
                </div>

                {/* Bar tombol Buka / Unduh PDF */}
                <div className="flex items-center gap-2 flex-wrap">
                  <a
                    href={pdfReady ? officialPdfUrl(letter.id, true) : undefined}
                    target="_blank"
                    rel="noopener noreferrer"
                    aria-disabled={!pdfReady}
                    className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg bg-primary text-white text-xs font-semibold hover:bg-primary-dark transition-colors aria-disabled:opacity-50 aria-disabled:pointer-events-none"
                  >
                    <ExternalLink size={14} /> Buka PDF
                  </a>
                  <a
                    href={pdfReady ? officialPdfUrl(letter.id) : undefined}
                    aria-disabled={!pdfReady}
                    className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg border border-border bg-surface text-xs font-semibold text-text hover:bg-panel transition-colors aria-disabled:opacity-50 aria-disabled:pointer-events-none"
                  >
                    <Download size={14} /> Unduh PDF Siap Cetak
                  </a>
                </div>

                {/* Area Unggah Dokumen Scan PDF Bertanda Tangan Basah */}
                {canUploadSigned ? (
                  <div className="pt-3 border-t border-border space-y-3">
                    <p className="text-xs font-semibold text-text">
                      Unggah Dokumen Scan PDF Bertanda Tangan Basah:
                    </p>
                    <input
                      ref={signedScanInputRef}
                      type="file"
                      accept="application/pdf,.pdf"
                      className="hidden"
                      onChange={(e) => {
                        const f = e.target.files?.[0];
                        if (f) {
                          setSignedScanFile(f);
                          setUploadSignedError(null);
                        }
                        e.target.value = "";
                      }}
                    />

                    {!signedScanFile ? (
                      <div
                        onClick={() => signedScanInputRef.current?.click()}
                        className="border-2 border-dashed border-primary/40 hover:border-primary bg-primary/[0.02] hover:bg-primary/[0.05] rounded-xl p-5 text-center cursor-pointer transition-colors flex flex-col items-center justify-center gap-1.5"
                      >
                        <UploadCloud size={22} className="text-primary" />
                        <p className="text-xs font-semibold text-text">
                          Klik untuk memilih file scan PDF (.pdf) yang sudah ditandatangani basah
                        </p>
                        <p className="text-[11px] text-text-muted">
                          Wajib format PDF &bull; Maksimal 25 MB
                        </p>
                      </div>
                    ) : (
                      <div className="flex items-center justify-between gap-3 flex-wrap p-3 rounded-xl bg-panel border border-border">
                        <div className="flex items-center gap-2.5 min-w-0">
                          <File size={18} className="text-red-500 shrink-0" />
                          <div className="min-w-0">
                            <p className="text-xs font-semibold text-text truncate">
                              {signedScanFile.name}
                            </p>
                            <p className="text-[11px] text-text-muted">
                              {formatBytes(signedScanFile.size)} &bull; Siap disimpan
                            </p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2 shrink-0">
                          <button
                            type="button"
                            onClick={() => signedScanInputRef.current?.click()}
                            disabled={uploadingSigned}
                            className="px-3 py-1.5 rounded-lg border border-border bg-surface text-xs font-medium text-text hover:bg-panel transition-colors"
                          >
                            Ganti File
                          </button>
                          <button
                            type="button"
                            onClick={() => void handleSaveSignedScan()}
                            disabled={uploadingSigned}
                            className="inline-flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-primary text-white text-xs font-semibold hover:bg-primary-dark disabled:opacity-60 transition-colors"
                          >
                            {uploadingSigned ? (
                              <>
                                <Loader2 size={13} className="animate-spin" />
                                Menyimpan...
                              </>
                            ) : (
                              <>
                                <CheckCircle2 size={13} />
                                Simpan &amp; Kirim ke Surat Eksternal
                              </>
                            )}
                          </button>
                        </div>
                      </div>
                    )}

                    {uploadSignedError && (
                      <p className="text-xs text-status-danger-text">
                        {uploadSignedError}
                      </p>
                    )}
                  </div>
                ) : (
                  <p className="text-xs text-text-muted italic pt-2 border-t border-border">
                    Menunggu pejabat penandatangan ({letter.approval?.approvedBy.name ?? "Approver terakhir"}) mengunggah scan PDF yang telah ditandatangani basah.
                  </p>
                )}
              </div>
            </div>
          )}

          {/* Kartu Pengesahan — setelah SENT */}
          {status === "SENT" && letter.approval && (
            <div className="border border-border rounded-xl bg-surface overflow-hidden">
              <div className="px-4 py-3 border-b border-border bg-panel/30 flex items-center justify-between gap-2 flex-wrap">
                <div className="flex items-center gap-2">
                  {isExternalOutgoing ? (
                    <CheckCircle2 size={16} className="text-status-success-text" />
                  ) : (
                    <QrCode size={16} className="text-primary" />
                  )}
                  <h3 className="text-sm font-semibold text-text">
                    {isExternalOutgoing
                      ? "Pengesahan Surat Eksternal Keluar (Tanda Tangan Basah)"
                      : "Pengesahan & QR Persetujuan"}
                  </h3>
                </div>
                {copyFor && (
                  <span className="text-[11px] font-medium px-2 py-0.5 rounded-full bg-primary/10 text-primary">
                    Salinan untuk {copyFor}
                  </span>
                )}
              </div>
              <div className="p-4 flex items-start gap-4 flex-wrap">
                {/* QR image dari backend (hanya untuk surat internal) */}
                {!isExternalOutgoing && letter.approval.qrUrl && (
                  <div className="w-28 h-28 border border-border rounded-lg bg-white p-1 shrink-0 flex items-center justify-center">
                    {/* eslint-disable-next-line @next/next/no-img-element */}
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
                )}
                <div className="flex-1 min-w-0 space-y-2">
                  <div>
                    <p className="text-[11px] font-semibold uppercase tracking-wide text-text-muted mb-0.5">
                      Disetujui &amp; Ditandatangani oleh
                    </p>
                    <p className="text-sm font-bold text-text">
                      {letter.approval.approvedBy.name}
                      {letter.approval.onBehalfOf && (
                        <span className="font-normal text-xs text-text-muted ml-1.5">
                          a.n. {letter.approval.onBehalfOf.name} ({letter.approval.onBehalfOf.positionName})
                        </span>
                      )}
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
                    <span>
                      Tanggal:{" "}
                      <strong className="text-text">
                        {formatDateTime(letter.approval.approvedAt)}
                      </strong>
                    </span>
                  </div>
                  {letter.letterNumber && (
                    <div className="text-xs text-text-muted">
                      Nomor:{" "}
                      <strong className="text-text font-mono">
                        {letter.letterNumber}
                      </strong>
                    </div>
                  )}
                  <div className="flex items-center gap-2 flex-wrap pt-1">
                    <a
                      href={pdfReady ? officialPdfUrl(letter.id) : undefined}
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
                    {isExternalOutgoing && (
                      <Link
                        href="/inbox?category=EXTERNAL"
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-lg border border-border bg-surface text-text hover:bg-panel transition-colors"
                      >
                        <Inbox size={13} className="text-primary" /> Buka Halaman Surat Eksternal
                      </Link>
                    )}
                  </div>
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
                  Total{" "}
                  {formatBytes(
                    letter.attachments.reduce((s, a) => s + a.size, 0)
                  )}
                </p>
              </div>
              <ul className="divide-y divide-border">
                {letter.attachments.map((att) => {
                  const downloadUrl = getLetterAttachmentDownloadUrl(
                    letter.id,
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
                        <button
                          type="button"
                          onClick={() => {
                            void openLetterAttachmentInline(
                              letter.id,
                              att.id,
                              att.contentType,
                              att.fileName
                            ).catch(() => {});
                          }}
                          className="text-xs text-text-muted hover:text-primary flex items-center gap-1 transition-colors"
                        >
                          <Eye size={13} /> Buka
                        </button>
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

          {letter.attachments.length === 0 && (
            <div className="border border-border rounded-xl px-4 py-6 text-center text-xs text-text-muted">
              Tidak ada berkas lampiran.
            </div>
          )}

          {/* Lacak Posisi, Diagram Alur, Pohon Distribusi, Riwayat Versi Naskah (.docx), & Audit Surat */}
          <LetterTrackingPanel
            letterId={letter.id}
            letter={letter}
            history={history}
            showVersions
          />
        </div>

        {/* ─ Kolom Kanan ─ */}
        <div className="space-y-4">
          {/* Metadata */}
          <div className="border border-border rounded-xl overflow-hidden">
            <div className="px-4 py-3 border-b border-border bg-panel/30">
              <h3 className="text-sm font-semibold text-text">
                Metadata & Ringkasan
              </h3>
            </div>
            <dl className="px-4 py-3 space-y-3">
              {letter.letterType && (
                <MetaRow label="Jenis Surat">{letter.letterType.name}</MetaRow>
              )}
              {letter.external?.sender && (
                <MetaRow label="Pengirim Eksternal">
                  <span className="font-semibold text-text">
                    {letter.external.sender}
                  </span>
                </MetaRow>
              )}
              {letter.external?.referenceNumber && (
                <MetaRow label="Nomor Surat Asal">
                  <span className="font-mono text-xs">
                    {letter.external.referenceNumber}
                  </span>
                </MetaRow>
              )}
              {letter.external?.letterDate && (
                <MetaRow label="Tanggal Surat Asal">
                  {letter.external.letterDate}
                </MetaRow>
              )}
              {letter.classification && (
                <MetaRow label="Klasifikasi">
                  <span className="font-mono text-xs">
                    {letter.classification.code}
                  </span>{" "}
                  — {letter.classification.name}
                </MetaRow>
              )}
              <MetaRow
                label={
                  isExternalIncoming ? "Dicatat Oleh" : "Pejabat Pembuat"
                }
              >
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
              {letter.external?.recipient && (
                <MetaRow label="Tujuan Eksternal">
                  <div className="mt-0.5 p-2.5 rounded-lg bg-primary/5 border border-primary/20 space-y-1">
                    <p className="text-xs font-bold text-text flex items-center gap-1.5">
                      <Building2 size={13} className="text-primary shrink-0" />
                      <span>{letter.external.recipient.name}</span>
                    </p>
                    {letter.external.recipient.attention && (
                      <p className="text-xs text-text-muted">
                        U.p. {letter.external.recipient.attention}
                      </p>
                    )}
                    {letter.external.recipient.address && (
                      <p className="text-xs text-text-muted">
                        {letter.external.recipient.address}
                      </p>
                    )}
                  </div>
                </MetaRow>
              )}
              {letter.recipients.to.length > 0 && (
                <MetaRow label="Tujuan / Penerima">
                  <div className="flex flex-wrap gap-1.5 mt-0.5">
                    {letter.recipients.to.map((r, i) => (
                      <span
                        key={i}
                        className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs bg-primary/10 text-primary font-medium"
                      >
                        <span>
                          {r.type === "UNIT" ? r.unit.name : r.alias.label}
                        </span>
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
                      <span
                        key={i}
                        className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs bg-panel text-text-muted border border-border font-medium"
                      >
                        <span>
                          {r.type === "UNIT" ? r.unit.name : r.alias.label}
                        </span>
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

          {/* Alur & Riwayat Ringkas + Modal Visualisasi Diagram */}
          <CompactAuditTrail letter={letter} history={history} />
        </div>
      </div>

      {/* ── Action Bar + Modals (Reviewer & Approver) ── */}
      {hasActions && <ReviewModals letter={letter} onDone={load} />}
    </div>
  );
}
