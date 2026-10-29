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

  return (
    <div className="border border-border rounded-xl bg-surface p-5 space-y-4">
      <div className="flex items-center gap-2">
        <ShieldCheck size={18} className="text-primary shrink-0" />
        <h3 className="text-sm font-bold text-text">Tahap Pengesahan</h3>
      </div>
      <p className="text-xs text-text-muted leading-relaxed">
        Naskah ini telah melewati seluruh tahap pemeriksaan Reviewer dan menunggu pengesahan dari Anda.
        Periksa isi naskah, lalu pilih aksi di bawah.
      </p>
      {activeStep && (
        <div className="flex items-start gap-3 bg-panel/50 border border-border rounded-lg p-3">
          <UserCheck size={16} className="text-primary shrink-0 mt-0.5" />
          <div>
            <p className="text-xs font-semibold text-text-muted uppercase tracking-wide mb-0.5">Pejabat Penandatangan</p>
            <p className="text-sm font-bold text-text">{activeStep.user.name}</p>
            <p className="text-xs text-text-muted">{activeStep.user.positionName}</p>
          </div>
        </div>
      )}
      <p className="text-[11px] text-text-muted italic">
        Nomor surat, tanggal, dan QR persetujuan dibuat otomatis oleh sistem setelah Anda menyetujui.
      </p>
    </div>
  );
}
