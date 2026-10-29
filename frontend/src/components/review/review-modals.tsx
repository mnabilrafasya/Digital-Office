"use client";

// ─────────────────────────────────────────────
// components/review/review-modals.tsx
// Action bar + modal: Setujui/Sahkan, Kembalikan, Batalkan
// Digunakan oleh Reviewer DAN Approver
// ─────────────────────────────────────────────

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  CheckCircle2,
  ShieldCheck,
  RotateCcw,
  XCircle,
  X,
  Loader2,
  AlertTriangle,
} from "lucide-react";
import type { LetterDetail } from "@/lib/api/letter-detail";
import { reviewAction } from "@/lib/api/letter-detail";
import { cn } from "@/lib/utils";

type ModalType = "approve" | "return" | "cancel" | null;

interface ReviewModalsProps {
  letter: LetterDetail;
  /** dipanggil setelah aksi berhasil agar halaman reload data terbaru */
  onDone: () => void;
}

export function ReviewModals({ letter, onDone }: ReviewModalsProps) {
  const router = useRouter();

  const [modal, setModal]  = useState<ModalType>(null);
  const [note, setNote]    = useState("");
  const [loading, setLoad] = useState(false);
  const [err, setErr]      = useState<string | null>(null);

  // ── Info assignee saat ini ──
  const activeStep = letter.steps.find((s) => s.state === "ACTIVE");
  const isApprover = activeStep?.role === "APPROVER" || letter.status === "IN_APPROVAL";
  const currentLabel = isApprover
    ? "Approver"
    : `Reviewer ${activeStep?.order ?? ""}`;

  // ── Teks konteks aksi ──
  const returnTarget = isApprover ? "Reviewer terakhir" : "Drafter";

  const nextPendingStep = letter.steps.find((s) => s.state === "PENDING");
  const approveTarget = isApprover
    ? "Nomor surat diterbitkan otomatis dan surat dikirim ke inbox penerima."
    : nextPendingStep?.role === "APPROVER"
    ? "Surat diteruskan ke Approver untuk ditandatangani."
    : nextPendingStep
    ? `Surat diteruskan ke Reviewer ${nextPendingStep.order}.`
    : "Surat diteruskan ke tahap berikutnya.";

  // ── Submit aksi ke backend ──
  async function submit(action: "APPROVE" | "RETURN" | "CANCEL") {
    if ((action === "RETURN" || action === "CANCEL") && note.trim().length < 10) {
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
      const nextStatus = res.data?.status;
      if (nextStatus === "SENT") {
        // Approver menyetujui → reload halaman ini untuk tampilkan QR & nomor surat
        onDone();
      } else {
        // Reviewer meneruskan ke tahap berikutnya → reload
        onDone();
      }
    } else if (action === "RETURN") {
      router.push("/surat/daftar?returned=1");
    } else {
      router.push("/surat/daftar?cancelled=1");
    }
  }

  function openModal(type: ModalType) {
    setModal(type);
    setNote("");
    setErr(null);
  }

  const allowed = (letter.allowedActions ?? []).map((a) => a.toUpperCase());
  const hasApprove = allowed.includes("APPROVE");
  const hasReturn  = allowed.includes("RETURN");
  const hasCancel  = allowed.includes("CANCEL");

  if (!hasApprove && !hasReturn && !hasCancel) return null;

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
          {hasCancel && (
            <button
              type="button"
              onClick={() => openModal("cancel")}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium border border-status-danger-text/30 text-status-danger-text hover:bg-status-danger-bg transition-colors"
            >
              <XCircle size={15} />
              Batalkan Naskah
            </button>
          )}
          {hasReturn && (
            <button
              type="button"
              onClick={() => openModal("return")}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium border border-border text-text hover:bg-panel transition-colors"
            >
              <RotateCcw size={15} />
              {isApprover ? "Kembalikan ke Reviewer" : "Kembalikan (Revisi)"}
            </button>
          )}
          {hasApprove && (
            <button
              type="button"
              onClick={() => openModal("approve")}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium bg-primary text-white hover:bg-primary-dark transition-colors"
            >
              {isApprover ? <ShieldCheck size={15} /> : <CheckCircle2 size={15} />}
              {isApprover ? "Tanda Tangani & Sahkan" : "Setujui Naskah"}
            </button>
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
                {modal === "approve" && isApprover && (
                  <ShieldCheck size={18} className="text-primary" />
                )}
                {modal === "approve" && !isApprover && (
                  <CheckCircle2 size={18} className="text-primary" />
                )}
                {modal === "return" && (
                  <RotateCcw size={18} className="text-status-warn-text" />
                )}
                {modal === "cancel" && (
                  <XCircle size={18} className="text-status-danger-text" />
                )}
                <h2 className="text-base font-semibold text-text">
                  {modal === "approve" && isApprover && "Konfirmasi Penandatanganan"}
                  {modal === "approve" && !isApprover && "Konfirmasi Persetujuan"}
                  {modal === "return" && "Kembalikan untuk Revisi"}
                  {modal === "cancel" && "Batalkan Naskah"}
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
                  {isApprover && (
                    <p className="text-xs bg-panel border border-border rounded-lg px-3 py-2 text-text-muted leading-relaxed">
                      Setelah ditandatangani:{" "}
                      <span className="font-medium text-text">{approveTarget}</span>
                    </p>
                  )}
                  {!isApprover && (
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
                  diperbaiki.
                </p>
              )}

              {modal === "cancel" && (
                <div className="flex items-start gap-2 text-sm bg-status-danger-bg text-status-danger-text px-3 py-2 rounded-lg">
                  <AlertTriangle size={15} className="shrink-0 mt-0.5" />
                  <span>
                    Naskah yang dibatalkan tidak dapat dikembalikan ke status
                    sebelumnya.
                  </span>
                </div>
              )}

              {(modal === "return" || modal === "cancel") && (
                <div>
                  <label className="text-xs font-semibold text-text-muted block mb-1">
                    {modal === "cancel" ? "Alasan pembatalan" : "Catatan revisi"}{" "}
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
                onClick={() =>
                  submit(
                    modal === "approve"
                      ? "APPROVE"
                      : modal === "return"
                      ? "RETURN"
                      : "CANCEL"
                  )
                }
                disabled={loading}
                className={cn(
                  "flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-lg transition-colors",
                  modal === "approve" && "bg-primary text-white hover:bg-primary-dark",
                  modal === "return"  && "bg-status-warn-text text-white hover:opacity-90",
                  modal === "cancel"  && "bg-status-danger-text text-white hover:opacity-90",
                  loading && "opacity-60 cursor-not-allowed"
                )}
              >
                {loading && <Loader2 size={14} className="animate-spin" />}
                {modal === "approve" && (isApprover ? "Ya, Tanda Tangani" : "Ya, Setujui")}
                {modal === "return"  && "Kembalikan"}
                {modal === "cancel"  && "Batalkan Naskah"}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
