"use client";

// ─────────────────────────────────────────────
// components/review/approver-panel.tsx
// Panel & Aksi Khusus Approver (Tanda Tangan Digital, QR Code, Disposisi, dan Submit Approval)
// ─────────────────────────────────────────────

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  ShieldCheck,
  QrCode,
  RotateCcw,
  CheckCircle2,
  FileText,
  UserCheck,
  Send,
  Loader2,
  AlertTriangle,
  X,
  Share2,
  BadgeCheck,
} from "lucide-react";
import type { LetterDetail } from "@/lib/api/letter-detail";
import { reviewAction } from "@/lib/api/letter-detail";
import { cn } from "@/lib/utils";

interface ApproverPanelProps {
  letter: LetterDetail;
  onDone: () => void;
}

export function ApproverPanel({ letter, onDone }: ApproverPanelProps) {
  const router = useRouter();

  const [modal, setModal] = useState<"approve" | "return" | null>(null);
  const [returnNote, setReturnNote] = useState("");
  const [disposisiPreset, setDisposisiPreset] = useState("Tindak lanjuti segera");
  const [disposisiNote, setDisposisiNote] = useState("");
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  // Ambil info approver dari steps
  const activeStep = letter.steps.find((s) => s.state === "ACTIVE");
  const approverName = activeStep?.user.name || "Pejabat Penandatangan";
  const approverJabatan = activeStep?.user.positionName || "Manajer Unit Kerja";
  const approverNip = (activeStep?.user as any)?.employeeNumber || "198005122005011003";

  // Tanggal pengesahan
  const todayFormatted = new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "long",
    year: "numeric",
  }).format(new Date());

  // Simulasi nomor surat otomatis yang akan terbit
  const generatedLetterNumber =
    letter.letterNumber ||
    `015/${letter.classification?.code || "TI.01"}/PSP/${getRomanMonth(new Date().getMonth() + 1)}/${new Date().getFullYear()}`;

  // Daftar opsi instruksi disposisi cepat
  const disposisiPresets = [
    "Tindak lanjuti segera",
    "Pelajari dan laporkan hasilnya",
    "Koordinasikan dengan unit terkait",
    "Untuk diketahui & arsip",
    "Wakili / hadiri kegiatan",
  ];

  // Submit persetujuan atau pengembalian
  async function handleSubmit(action: "APPROVE" | "RETURN") {
    if (action === "RETURN" && returnNote.trim().length < 10) {
      setErrorMsg("Catatan pengembalian wajib diisi minimal 10 karakter.");
      return;
    }

    setLoading(true);
    setErrorMsg(null);

    // Gabungkan catatan disposisi bila ada saat approve
    const finalNote =
      action === "APPROVE"
        ? disposisiNote.trim()
          ? `[Disposisi Pimpinan: ${disposisiPreset}] ${disposisiNote.trim()}`
          : `[Disposisi Pimpinan: ${disposisiPreset}]`
        : returnNote.trim();

    try {
      const res = await reviewAction(letter.id, action, finalNote);

      if (!res.ok) {
        setErrorMsg(res.error || "Gagal memproses persetujuan. Coba lagi.");
        setLoading(false);
        return;
      }

      setModal(null);
      if (action === "APPROVE") {
        router.push("/surat/daftar?approved=1");
      } else {
        router.push("/surat/daftar?returned=1");
      }
    } catch (e) {
      setErrorMsg((e as Error).message || "Terjadi kesalahan koneksi.");
      setLoading(false);
    }
  }

  return (
    <div className="space-y-4">
      {/* ── Banner Utama Approver ── */}
      <div className="border border-emerald-200 bg-linear-to-r from-emerald-50 via-teal-50 to-emerald-50 rounded-xl p-4 text-emerald-900 shadow-sm">
        <div className="flex items-start justify-between gap-3 flex-wrap">
          <div className="flex items-start gap-3">
            <div className="w-10 h-10 rounded-xl bg-emerald-600 text-white flex items-center justify-center shrink-0 shadow-sm">
              <ShieldCheck size={22} />
            </div>
            <div>
              <h2 className="text-base font-bold text-emerald-950 flex items-center gap-1.5">
                Tahap Pengesahan Akhir (Approver)
                <BadgeCheck size={18} className="text-emerald-600 inline shrink-0" />
              </h2>
              <p className="text-xs text-emerald-800 mt-0.5">
                Naskah ini telah melewati seluruh tahap pemeriksaan Reviewer dan menunggu pengesahan tanda tangan elektronik resmi dari Anda.
              </p>
            </div>
          </div>

          <div className="bg-white/80 border border-emerald-300/60 rounded-lg px-3 py-1.5 text-xs">
            <span className="text-emerald-700 font-medium">Nomor Surat Otomatis:</span>{" "}
            <strong className="text-emerald-950 font-mono font-bold">{generatedLetterNumber}</strong>
          </div>
        </div>
      </div>

      {/* ── Kartu Spesimen Tanda Tangan Digital & QR Code ── */}
      <div className="border border-border rounded-xl bg-surface p-5 shadow-sm">
        <div className="flex items-center justify-between border-b border-border pb-3 mb-4">
          <div className="flex items-center gap-2">
            <UserCheck size={18} className="text-primary" />
            <h3 className="text-sm font-bold text-text">
              Kelengkapan Tanda Tangan Elektronik &amp; QR Code
            </h3>
          </div>
          <span className="text-[11px] px-2.5 py-0.5 rounded-full bg-emerald-100 text-emerald-800 font-semibold border border-emerald-200">
            Tersertifikasi BSrE / Digital Office
          </span>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Spesimen Tanda Tangan */}
          <div className="border border-dashed border-border rounded-xl p-4 bg-panel/30 flex flex-col justify-between">
            <div className="text-xs font-semibold text-text-muted uppercase tracking-wider mb-2">
              Pejabat Penandatangan
            </div>
            <div className="space-y-1 my-2">
              <div className="text-sm font-bold text-text">{approverName}</div>
              <div className="text-xs text-text-muted">{approverJabatan}</div>
              <div className="text-xs font-mono text-text-muted">NIP. {approverNip}</div>
            </div>
            <div className="pt-3 border-t border-border/60 text-[11px] text-text-muted flex items-center gap-1.5">
              <CheckCircle2 size={13} className="text-emerald-600 shrink-0" />
              <span>Tanggal Pengesahan: <strong>{todayFormatted}</strong></span>
            </div>
          </div>

          {/* QR Code Verifikasi */}
          <div className="border border-border rounded-xl p-4 bg-panel/30 flex items-center gap-4">
            <div className="w-24 h-24 bg-white border border-border rounded-lg p-1.5 flex flex-col items-center justify-center shrink-0 shadow-sm relative group">
              {/* Simulasi Grafis QR Code Elegan */}
              <div className="w-full h-full bg-neutral-900 rounded flex flex-col items-center justify-center text-white p-1">
                <QrCode size={48} className="text-white" />
                <span className="text-[8px] font-mono tracking-tighter opacity-80 mt-0.5">VERIFIED</span>
              </div>
            </div>
            <div className="min-w-0 space-y-1">
              <div className="text-xs font-bold text-text flex items-center gap-1">
                QR Code Validasi Dokumen
              </div>
              <p className="text-[11px] text-text-muted leading-relaxed">
                QR Code disematkan pada lembar naskah dan dapat dipindai oleh penerima untuk memverifikasi keaslian surat melalui portal e-Office PT Pusri.
              </p>
              <div className="text-[10px] text-text-muted font-mono truncate">
                ID Dokumen: {letter.id}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* ── Form Instruksi Disposisi / Catatan Pimpinan (Opsional) ── */}
      <div className="border border-border rounded-xl bg-surface p-5 shadow-sm">
        <div className="flex items-center gap-2 mb-3">
          <Share2 size={17} className="text-primary" />
          <h3 className="text-sm font-bold text-text">
            Instruksi Disposisi / Arahan Pimpinan (Opsional)
          </h3>
        </div>
        <p className="text-xs text-text-muted mb-3">
          Instruksi ini akan disertakan secara otomatis saat surat terkirim ke unit penerima.
        </p>

        {/* Pilihan Cepat / Preset Disposisi */}
        <div className="flex flex-wrap gap-1.5 mb-3">
          {disposisiPresets.map((preset) => (
            <button
              key={preset}
              type="button"
              onClick={() => setDisposisiPreset(preset)}
              className={cn(
                "px-2.5 py-1 rounded-lg text-xs font-medium border transition-colors",
                disposisiPreset === preset
                  ? "bg-primary/10 border-primary text-primary font-semibold"
                  : "bg-surface border-border text-text-muted hover:text-text hover:bg-bg"
              )}
            >
              {preset}
            </button>
          ))}
        </div>

        {/* Textarea Catatan Tambahan */}
        <textarea
          rows={2}
          placeholder="Tuliskan arahan tambahan untuk unit penerima bila diperlukan..."
          value={disposisiNote}
          onChange={(e) => setDisposisiNote(e.target.value)}
          className="w-full text-xs border border-border rounded-lg p-2.5 outline-none resize-none bg-surface focus:border-primary focus:ring-1 focus:ring-primary/20 placeholder-text-muted/60"
        />
      </div>

      {/* ── Sticky Action Bar Khusus Approver ── */}
      <div className="fixed bottom-0 left-56 right-0 z-30 bg-surface border-t border-border shadow-xl px-6 py-3 flex items-center justify-between gap-4">
        <div className="flex items-center gap-2.5 min-w-0 text-sm">
          <span className="w-8 h-8 rounded-full bg-emerald-600 text-white flex items-center justify-center font-bold text-xs shrink-0 shadow-sm">
            A
          </span>
          <div className="truncate">
            <span className="text-xs text-text-muted block">Akses Otoritas:</span>
            <strong className="text-text font-semibold">{approverJabatan} — {approverName}</strong>
          </div>
        </div>

        <div className="flex items-center gap-2.5 shrink-0">
          {/* Tombol Kembalikan ke Reviewer */}
          <button
            type="button"
            onClick={() => {
              setModal("return");
              setErrorMsg(null);
            }}
            className="flex items-center gap-1.5 px-4 py-2.5 rounded-lg text-xs font-semibold border border-status-warn-text/40 text-status-warn-text hover:bg-status-warn-bg transition-colors shadow-sm"
          >
            <RotateCcw size={14} />
            Kembalikan ke Reviewer Terakhir
          </button>

          {/* Tombol Sahkan & Tanda Tangani */}
          <button
            type="button"
            onClick={() => {
              setModal("approve");
              setErrorMsg(null);
            }}
            className="flex items-center gap-2 px-5 py-2.5 rounded-lg text-xs font-bold bg-emerald-600 hover:bg-emerald-700 text-white shadow-md hover:shadow-lg transition-all"
          >
            <ShieldCheck size={16} />
            Tanda Tangani &amp; Sahkan Surat
          </button>
        </div>
      </div>

      {/* ── Modal Dialog Konfirmasi Approval ── */}
      {modal === "approve" && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
          <div className="bg-surface rounded-2xl shadow-2xl w-full max-w-lg border border-border overflow-hidden animate-scale-up">
            <div className="flex items-center justify-between px-6 py-4 border-b border-border bg-emerald-50/60">
              <div className="flex items-center gap-2">
                <ShieldCheck size={20} className="text-emerald-600" />
                <h3 className="text-base font-bold text-text">Konfirmasi Tanda Tangan &amp; Pengesahan</h3>
              </div>
              <button
                type="button"
                onClick={() => setModal(null)}
                className="text-text-muted hover:text-text p-1 rounded-lg"
              >
                <X size={18} />
              </button>
            </div>

            <div className="p-6 space-y-4">
              <div className="bg-panel/40 border border-border rounded-xl p-4 space-y-2 text-xs">
                <div className="flex justify-between">
                  <span className="text-text-muted">Nomor Surat Resmi:</span>
                  <strong className="font-mono text-primary font-bold">{generatedLetterNumber}</strong>
                </div>
                <div className="flex justify-between">
                  <span className="text-text-muted">Tanggal Terbit:</span>
                  <span className="font-medium text-text">{todayFormatted}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-text-muted">Pejabat Pengesah:</span>
                  <span className="font-semibold text-text">{approverName}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-text-muted">Instruksi Disposisi:</span>
                  <span className="font-medium text-emerald-700">{disposisiPreset}</span>
                </div>
              </div>

              <div className="text-xs text-text-muted space-y-1.5 leading-relaxed">
                <p>
                  Dengan menekan tombol <strong>&quot;Sahkan &amp; Terbitkan Sekarang&quot;</strong>:
                </p>
                <ul className="list-disc list-inside space-y-1 pl-1 text-[11.5px]">
                  <li>Tanda Tangan Elektronik &amp; QR Code verifikasi akan disematkan permanen pada naskah.</li>
                  <li>Nomor surat resmi akan diterbitkan ke dalam sistem.</li>
                  <li>Surat akan langsung didistribusikan ke Inbox pimpinan unit penerima.</li>
                </ul>
              </div>

              {errorMsg && (
                <div className="p-3 rounded-lg bg-status-danger-bg text-status-danger-text text-xs border border-status-danger-text/20">
                  {errorMsg}
                </div>
              )}
            </div>

            <div className="px-6 py-4 border-t border-border bg-panel/30 flex justify-end gap-2.5">
              <button
                type="button"
                onClick={() => setModal(null)}
                disabled={loading}
                className="px-4 py-2 text-xs font-semibold rounded-lg border border-border text-text hover:bg-panel transition-colors"
              >
                Batal
              </button>
              <button
                type="button"
                onClick={() => handleSubmit("APPROVE")}
                disabled={loading}
                className="flex items-center gap-1.5 px-5 py-2 text-xs font-bold rounded-lg bg-emerald-600 text-white hover:bg-emerald-700 transition-colors shadow-sm disabled:opacity-50"
              >
                {loading ? (
                  <>
                    <Loader2 size={14} className="animate-spin" />
                    Memproses Pengesahan...
                  </>
                ) : (
                  <>
                    <Send size={14} />
                    Sahkan &amp; Terbitkan Sekarang
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── Modal Dialog Kembalikan ke Reviewer ── */}
      {modal === "return" && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
          <div className="bg-surface rounded-2xl shadow-2xl w-full max-w-md border border-border overflow-hidden">
            <div className="flex items-center justify-between px-6 py-4 border-b border-border bg-status-warn-bg">
              <div className="flex items-center gap-2 text-status-warn-text">
                <RotateCcw size={18} />
                <h3 className="text-base font-bold">Kembalikan ke Reviewer Terakhir</h3>
              </div>
              <button
                type="button"
                onClick={() => setModal(null)}
                className="text-text-muted hover:text-text p-1 rounded-lg"
              >
                <X size={18} />
              </button>
            </div>

            <div className="p-6 space-y-3">
              <p className="text-xs text-text-muted leading-relaxed">
                Naskah akan dikembalikan ke <strong>Reviewer terakhir</strong> beserta catatan perbaikan Anda sebelum diajukan kembali ke meja persetujuan.
              </p>

              <div>
                <label className="text-xs font-semibold text-text block mb-1">
                  Catatan Pengembalian / Revisi <span className="text-status-danger-text">*</span>
                </label>
                <textarea
                  rows={4}
                  placeholder="Tuliskan alasan pengembalian dan poin naskah yang perlu diperbaiki (minimal 10 karakter)..."
                  value={returnNote}
                  onChange={(e) => {
                    setReturnNote(e.target.value);
                    setErrorMsg(null);
                  }}
                  className="w-full text-xs border border-border rounded-lg p-3 outline-none resize-none bg-surface focus:border-status-warn-text focus:ring-1 focus:ring-status-warn-text/20 placeholder-text-muted/60"
                />
              </div>

              {errorMsg && (
                <div className="p-2.5 rounded-lg bg-status-danger-bg text-status-danger-text text-xs border border-status-danger-text/20">
                  {errorMsg}
                </div>
              )}
            </div>

            <div className="px-6 py-4 border-t border-border bg-panel/30 flex justify-end gap-2.5">
              <button
                type="button"
                onClick={() => setModal(null)}
                disabled={loading}
                className="px-4 py-2 text-xs font-semibold rounded-lg border border-border text-text hover:bg-panel transition-colors"
              >
                Batal
              </button>
              <button
                type="button"
                onClick={() => handleSubmit("RETURN")}
                disabled={loading}
                className="flex items-center gap-1.5 px-4 py-2 text-xs font-semibold rounded-lg bg-status-warn-text text-white hover:opacity-90 transition-opacity disabled:opacity-50"
              >
                {loading ? (
                  <>
                    <Loader2 size={14} className="animate-spin" />
                    Mengirim...
                  </>
                ) : (
                  <>
                    <RotateCcw size={14} />
                    Kembalikan Naskah
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// Helper bulan romawi untuk nomor surat
function getRomanMonth(month: number): string {
  const romans = ["I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"];
  return romans[month - 1] || "X";
}
