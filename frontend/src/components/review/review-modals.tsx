"use client";

// ─────────────────────────────────────────────
// components/review/review-modals.tsx
// Action bar + modal: Setujui/Sahkan, Kembalikan ke Drafter, Unggah Scan TTD
// Digunakan oleh Reviewer DAN Approver
// ─────────────────────────────────────────────

import { useState, useRef } from "react";
import { useRouter } from "next/navigation";
import {
  CheckCircle2,
  ShieldCheck,
  RotateCcw,
  X,
  Loader2,
  Printer,
  UploadCloud,
  FileText,
} from "lucide-react";
import type { LetterDetail } from "@/lib/api/letter-detail";
import { reviewAction, formatBytes } from "@/lib/api/letter-detail";
import { officialPdfUrl, uploadSignedPdf } from "@/lib/api/letter-document";
import { cn } from "@/lib/utils";

type ModalType = "approve" | "return" | "upload_signed" | null;

interface ReviewModalsProps {
  letter: LetterDetail;
  /** dipanggil setelah aksi berhasil agar halaman reload data terbaru */
  onDone: () => void;
}

export function ReviewModals({ letter, onDone }: ReviewModalsProps) {
  const router = useRouter();

  const [modal, setModal]  = useState<ModalType>(null);
  const [note, setNote]    = useState("");
  const [signedFile, setSignedFile] = useState<File | null>(null);
  const signedFileInputRef = useRef<HTMLInputElement>(null);
  const [loading, setLoad] = useState(false);
  const [err, setErr]      = useState<string | null>(null);

  const isExternalOutgoing =
    letter.direction === "OUTGOING" ||
    letter.letterType?.code === "SURAT_EKSTERNAL";

  // ── Info assignee saat ini ──
  const activeStep = letter.steps.find((s) => s.state === "ACTIVE");
  const approverSteps = letter.steps.filter((s) => s.role === "APPROVER");
  const isApprover =
    activeStep?.role === "APPROVER" ||
    letter.status === "IN_APPROVAL" ||
    letter.status === "APPROVED";
  const isFinalApprover =
    isApprover &&
    (!activeStep ||
      approverSteps.length <= 1 ||
      activeStep.order >= approverSteps.length);
  const currentLabel = isApprover
    ? approverSteps.length > 1 && activeStep
      ? `Approver ${activeStep.order}`
      : "Approver"
    : `Reviewer ${activeStep?.order ?? ""}`;

  // ── Teks konteks aksi (semua pengembalian kembali ke Drafter dan diulang dari Reviewer 1) ──
  const returnTarget = "Drafter";

  const nextPendingStep = letter.steps.find((s) => s.state === "PENDING");
  const approveTarget = isApprover
    ? isFinalApprover
      ? isExternalOutgoing
        ? "Nomor surat diterbitkan otomatis dan tampilan naskah berubah menjadi PDF siap cetak untuk ditandatangani basah (status Menunggu Tanda Tangan)."
        : "Nomor surat diterbitkan otomatis dan surat dikirim ke inbox penerima."
      : nextPendingStep?.role === "APPROVER"
        ? `Surat diteruskan ke Approver ${nextPendingStep.order} (${nextPendingStep.user.name}) untuk persetujuan tahap berikutnya.`
        : "Surat diteruskan ke Approver berikutnya."
    : nextPendingStep?.role === "APPROVER"
      ? approverSteps.length > 1
        ? `Surat diteruskan ke Approver 1 (${nextPendingStep.user.name}) untuk tahap persetujuan.`
        : "Surat diteruskan ke Approver untuk ditandatangani."
      : nextPendingStep
        ? `Surat diteruskan ke Reviewer ${nextPendingStep.order}.`
        : "Surat diteruskan ke tahap berikutnya.";

  // ── Submit aksi ke backend ──
  async function submit(action: "APPROVE" | "RETURN") {
    if (action === "RETURN" && note.trim().length < 10) {
      setErr("Catatan minimal 10 karakter.");
      return;
    }

    setLoad(true);
    setErr(null);

    const res = await reviewAction(letter.id, action, note.trim() || undefined);

    setLoad(false);

    if (!res.ok) {
      setErr(res.error ?? "Terjadi kesalahan. Coba lagi.");
      return;
    }

    setModal(null);
    setNote("");

    if (action === "APPROVE") {
      onDone();
    } else {
      router.push("/surat/daftar?returned=1");
    }
  }

  async function handleUploadSignedSubmit() {
    if (!signedFile) {
      setErr("Pilih file PDF hasil scan bertanda tangan basah terlebih dahulu.");
      return;
    }
    setLoad(true);
    setErr(null);
    try {
      await uploadSignedPdf(letter.id, signedFile);
      setModal(null);
      setSignedFile(null);
      onDone();
    } catch (e) {
      setErr(
        e instanceof Error
          ? e.message
          : "Gagal mengunggah scan PDF bertanda tangan basah."
      );
    } finally {
      setLoad(false);
    }
  }

  function openModal(type: ModalType) {
    setModal(type);
    setNote("");
    setSignedFile(null);
    setErr(null);
  }

  const allowed = (letter.allowedActions ?? []).map((a) => a.toUpperCase());
  const hasApprove = allowed.includes("APPROVE");
  const hasReturn  = allowed.includes("RETURN");
  const hasUploadSigned = allowed.includes("UPLOAD_SIGNED");

  if (!hasApprove && !hasReturn && !hasUploadSigned) return null;

  return (
    <>
      {/* ── Action Bar ── */}
      <div className="fixed bottom-0 left-56 right-0 z-20 bg-surface border-t border-border shadow-lg px-6 py-3 flex items-center gap-3 justify-between">
        {/* Info peran aktif */}
        <div className="flex items-center gap-2 text-sm text-text-muted min-w-0">
          <span className="w-7 h-7 rounded-full bg-primary flex items-center justify-center text-white text-xs font-bold shrink-0">
            {currentLabel.charAt(0)}
          </span>
          <span className="truncate">
            Anda meninjau sebagai:{" "}
            <strong className="text-text">
              {currentLabel}
              {activeStep ? ` — ${activeStep.user.name}` : ""}
            </strong>
          </span>
        </div>

        {/* Tombol aksi */}
        <div className="flex items-center gap-2 shrink-0">
          {hasReturn && (
            <button
              type="button"
              onClick={() => openModal("return")}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium border border-border text-text hover:bg-panel transition-colors"
            >
              <RotateCcw size={15} />
              Kembalikan ke Drafter
            </button>
          )}
          {hasApprove && (
            <button
              type="button"
              onClick={() => openModal("approve")}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium bg-primary text-white hover:bg-primary-dark transition-colors"
            >
              {isFinalApprover ? <ShieldCheck size={15} /> : <CheckCircle2 size={15} />}
              {isFinalApprover
                ? isExternalOutgoing
                  ? "Setujui & Siapkan PDF Cetak"
                  : "Tanda Tangani & Sahkan"
                : isApprover
                  ? "Setujui & Teruskan"
                  : "Setujui Naskah"}
            </button>
          )}
          {hasUploadSigned && (
            <>
              <a
                href={officialPdfUrl(letter.id, true)}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium border border-border text-text hover:bg-panel transition-colors"
              >
                <Printer size={15} className="text-primary" />
                Cetak / Unduh PDF
              </a>
              <button
                type="button"
                onClick={() => openModal("upload_signed")}
                className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium bg-primary text-white hover:bg-primary-dark transition-colors"
              >
                <UploadCloud size={15} />
                Unggah Scan PDF &amp; Simpan
              </button>
            </>
          )}
        </div>
      </div>

      {/* ── Modal Overlay ── */}
      {modal && (
        <div
          className="fixed inset-0 z-40 flex items-center justify-center bg-black/40 backdrop-blur-sm"
          onClick={(e) => {
            if (e.target === e.currentTarget) setModal(null);
          }}
        >
          <div className="bg-surface rounded-xl shadow-xl w-full max-w-md mx-4 overflow-hidden">
            {/* Header */}
            <div className="flex items-center justify-between px-5 py-4 border-b border-border">
              <div className="flex items-center gap-2">
                {modal === "approve" && isFinalApprover && (
                  <ShieldCheck size={18} className="text-primary" />
                )}
                {modal === "approve" && !isFinalApprover && (
                  <CheckCircle2 size={18} className="text-primary" />
                )}
                {modal === "return" && (
                  <RotateCcw size={18} className="text-status-warn-text" />
                )}
                {modal === "upload_signed" && (
                  <UploadCloud size={18} className="text-primary" />
                )}
                <h2 className="text-base font-semibold text-text">
                  {modal === "approve" &&
                    isFinalApprover &&
                    (isExternalOutgoing
                      ? "Setujui & Terbitkan PDF Siap Cetak"
                      : "Konfirmasi Penandatanganan")}
                  {modal === "approve" && !isFinalApprover && "Konfirmasi Persetujuan"}
                  {modal === "return" && "Kembalikan ke Drafter untuk Revisi"}
                  {modal === "upload_signed" &&
                    "Unggah Scan PDF Bertanda Tangan Basah"}
                </h2>
              </div>
              <button
                type="button"
                onClick={() => setModal(null)}
                className="text-text-muted hover:text-text"
              >
                <X size={18} />
              </button>
            </div>

            {/* Body */}
            <div className="px-5 py-4 space-y-3">
              {modal === "approve" && (
                <div className="text-sm text-text-muted space-y-1">
                  {isFinalApprover ? (
                    <p className="text-xs bg-panel border border-border rounded-lg px-3 py-2 text-text-muted leading-relaxed">
                      Setelah ditandatangani:{" "}
                      <span className="font-medium text-text">{approveTarget}</span>
                    </p>
                  ) : (
                    <p>
                      Dengan menyetujui:{" "}
                      <span className="font-medium text-text">{approveTarget}</span>
                    </p>
                  )}
                </div>
              )}

              {modal === "return" && (
                <p className="text-sm text-text-muted">
                  Naskah akan dikembalikan ke{" "}
                  <strong className="text-text">{returnTarget}</strong> untuk
                  diperbaiki. Setelah diajukan ulang, proses pemeriksaan akan
                  dimulai kembali dari <strong className="text-text">Reviewer 1</strong>.
                </p>
              )}

              {modal === "return" && (
                <div>
                  <label className="text-xs font-semibold text-text-muted block mb-1">
                    Catatan revisi{" "}
                    <span className="text-status-danger-text">*</span>
                  </label>
                  <textarea
                    rows={3}
                    placeholder="Minimal 10 karakter..."
                    value={note}
                    onChange={(e) => {
                      setNote(e.target.value);
                      setErr(null);
                    }}
                    className={cn(
                      "w-full text-sm border rounded-lg px-3 py-2 outline-none resize-none bg-surface placeholder-text-muted/60",
                      "focus:border-primary focus:ring-2 focus:ring-primary/15 transition-colors",
                      err ? "border-status-danger-text" : "border-border"
                    )}
                  />
                  {err && (
                    <p className="text-xs text-status-danger-text mt-1">{err}</p>
                  )}
                </div>
              )}

              {modal === "approve" && err && (
                <p className="text-xs text-status-danger-text">{err}</p>
              )}

              {modal === "upload_signed" && (
                <div className="space-y-3">
                  <p className="text-xs text-text-muted leading-relaxed">
                    Pastikan dokumen PDF surat eksternal (No.{" "}
                    <strong className="font-mono text-text">
                      {letter.letterNumber ?? "—"}
                    </strong>
                    ) telah dicetak dan ditandatangani basah. Unggah hasil scan
                    PDF-nya di bawah ini untuk menyelesaikan surat dan
                    mengarsipkannya ke halaman{" "}
                    <strong className="text-text">Surat Eksternal</strong>.
                  </p>

                  <input
                    ref={signedFileInputRef}
                    type="file"
                    accept="application/pdf,.pdf"
                    className="hidden"
                    onChange={(e) => {
                      const f = e.target.files?.[0];
                      if (f) {
                        setSignedFile(f);
                        setErr(null);
                      }
                      e.target.value = "";
                    }}
                  />

                  {!signedFile ? (
                    <button
                      type="button"
                      onClick={() => signedFileInputRef.current?.click()}
                      className="w-full border-2 border-dashed border-border hover:border-primary/50 rounded-xl p-6 flex flex-col items-center justify-center gap-2 text-center transition-colors"
                    >
                      <UploadCloud size={22} className="text-primary" />
                      <span className="text-xs font-semibold text-text">
                        Klik untuk memilih file scan PDF (.pdf)
                      </span>
                      <span className="text-[11px] text-text-muted">
                        Maksimal 25 MB
                      </span>
                    </button>
                  ) : (
                    <div className="flex items-center justify-between gap-2 p-3 rounded-xl bg-panel border border-border">
                      <div className="flex items-center gap-2.5 min-w-0">
                        <FileText size={18} className="text-red-500 shrink-0" />
                        <div className="min-w-0">
                          <p className="text-xs font-semibold text-text truncate">
                            {signedFile.name}
                          </p>
                          <p className="text-[11px] text-text-muted">
                            {formatBytes(signedFile.size)}
                          </p>
                        </div>
                      </div>
                      <button
                        type="button"
                        onClick={() => signedFileInputRef.current?.click()}
                        className="text-xs text-primary font-semibold hover:underline shrink-0"
                      >
                        Ganti File
                      </button>
                    </div>
                  )}

                  {err && (
                    <p className="text-xs text-status-danger-text">{err}</p>
                  )}
                </div>
              )}
            </div>

            {/* Footer */}
            <div className="px-5 py-4 border-t border-border flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setModal(null)}
                className="px-4 py-2 text-sm rounded-lg border border-border text-text hover:bg-panel transition-colors"
              >
                Batal
              </button>
              <button
                type="button"
                onClick={() => {
                  if (modal === "upload_signed") {
                    void handleUploadSignedSubmit();
                    return;
                  }
                  void submit(modal === "approve" ? "APPROVE" : "RETURN");
                }}
                disabled={loading}
                className={cn(
                  "flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-lg transition-colors",
                  (modal === "approve" || modal === "upload_signed") &&
                    "bg-primary text-white hover:bg-primary-dark",
                  modal === "return"  && "bg-status-warn-text text-white hover:opacity-90",
                  loading && "opacity-60 cursor-not-allowed"
                )}
              >
                {loading && <Loader2 size={14} className="animate-spin" />}
                {modal === "approve" &&
                  (isFinalApprover
                    ? isExternalOutgoing
                      ? "Ya, Setujui & Terbitkan PDF"
                      : "Ya, Tanda Tangani"
                    : isApprover
                      ? "Ya, Setujui & Teruskan"
                      : "Ya, Setujui")}
                {modal === "return"  && "Kembalikan"}
                {modal === "upload_signed" && "Simpan & Selesaikan Surat"}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
