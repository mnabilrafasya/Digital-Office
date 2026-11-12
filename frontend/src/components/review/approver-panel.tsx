"use client";

// components/review/approver-panel.tsx
// Info panel khusus Approver (IN_APPROVAL) — action bar ada di ReviewModals

import { ShieldCheck, UserCheck } from "lucide-react";
import type { LetterDetail } from "@/lib/api/letter-detail";

interface Props {
  letter: LetterDetail;
}

export function ApproverPanel({ letter }: Props) {
  const activeStep = letter.steps.find((s) => s.state === "ACTIVE");
  const approverSteps = letter.steps.filter((s) => s.role === "APPROVER");
  const isFinalApprover =
    !activeStep ||
    approverSteps.length <= 1 ||
    activeStep.order >= approverSteps.length;
  const finalApprover = approverSteps[approverSteps.length - 1];
  const isExternalOutgoing =
    letter.direction === "OUTGOING" ||
    letter.letterType?.code === "SURAT_EKSTERNAL";

  return (
    <div className="border border-border rounded-xl bg-surface p-5 space-y-4">
      <div className="flex items-center gap-2">
        <ShieldCheck size={18} className="text-primary shrink-0" />
        <h3 className="text-sm font-bold text-text">
          {isFinalApprover
            ? isExternalOutgoing
              ? "Tahap Persetujuan Akhir Surat Eksternal (Cetak & TTD Basah)"
              : "Tahap Pengesahan & Tanda Tangan"
            : `Tahap Persetujuan Approver ${activeStep?.order ?? ""}/${approverSteps.length}`}
        </h3>
      </div>
      <p className="text-xs text-text-muted leading-relaxed">
        {isFinalApprover
          ? isExternalOutgoing
            ? "Naskah Surat Eksternal Keluar ini telah melewati seluruh tahap pemeriksaan. Setelah Anda menyetujui, nomor & tanggal surat diterbitkan otomatis dan tampilan naskah diubah menjadi PDF siap cetak untuk ditandatangani basah, kemudian diunggah kembali hasil scan-nya."
            : "Naskah ini telah melewati seluruh tahap pemeriksaan dan menunggu pengesahan akhir dari Anda. Periksa isi naskah, lalu pilih aksi di bawah."
          : "Naskah ini sedang dalam tahap persetujuan bertingkat. Setelah Anda menyetujui, surat akan diteruskan ke Approver berikutnya."}
      </p>
      {activeStep && (
        <div className="flex items-start gap-3 bg-panel/50 border border-border rounded-lg p-3">
          <UserCheck size={16} className="text-primary shrink-0 mt-0.5" />
          <div>
            <p className="text-xs font-semibold text-text-muted uppercase tracking-wide mb-0.5">
              {isFinalApprover
                ? "Pejabat Penandatangan (Approver Terakhir)"
                : `Approver Tahap ${activeStep.order} dari ${approverSteps.length}`}
            </p>
            <p className="text-sm font-bold text-text">{activeStep.user.name}</p>
            <p className="text-xs text-text-muted">{activeStep.user.positionName}</p>
          </div>
        </div>
      )}
      <p className="text-[11px] text-text-muted italic">
        {isFinalApprover
          ? isExternalOutgoing
            ? "Nomor & tanggal surat dibuat otomatis setelah Anda menyetujui, lalu surat berstatus Menunggu Tanda Tangan hingga scan PDF diunggah."
            : "Nomor surat, tanggal, dan QR persetujuan dibuat otomatis oleh sistem setelah Anda menyetujui."
          : `Nomor surat akan diterbitkan otomatis atas nama ${finalApprover?.user.name ?? "Approver terakhir"} setelah seluruh tahap Approver selesai.`}
      </p>
    </div>
  );
}
