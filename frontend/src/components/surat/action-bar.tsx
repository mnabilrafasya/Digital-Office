"use client";

// ─────────────────────────────────────────────
// components/surat/action-bar.tsx
// Action bar sticky: Batal, Simpan Draft, Pratinjau PDF (nonaktif), Submit
// ─────────────────────────────────────────────

import { FileDown, Info, Save, Send, X } from "lucide-react";
import { useFormContext, useFormState } from "react-hook-form";
import type { BuatSuratFormValues } from "@/lib/schemas";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

interface Props {
  onBatal: () => void;
  onSimpan: () => void;
}

export function ActionBar({ onBatal, onSimpan }: Props) {
  const { control } = useFormContext<BuatSuratFormValues>();
  const { isValid } = useFormState({ control });

  return (
    <TooltipProvider>
      <div className="fixed bottom-0 left-56 right-0 bg-surface border-t border-border shadow-[0_-4px_24px_rgba(0,0,0,0.06)] z-20">
        <div className="max-w-6xl mx-auto px-6 py-3 flex items-center justify-between gap-3">
          <div className="text-xs text-text-muted flex items-center gap-1.5">
            <Info size={12} />
            <span>Nomor &amp; tanggal surat dibuat otomatis saat disetujui.</span>
          </div>

          <div className="flex items-center gap-2 shrink-0">
            <Button type="button" variant="outline" size="lg" onClick={onBatal} className="h-10 px-4">
              <X size={14} /> Batal
            </Button>

            <Button type="button" variant="outline" size="lg" onClick={onSimpan} className="h-10 px-4 text-primary border-primary/30">
              <Save size={14} /> Simpan Draft
            </Button>

            <Tooltip>
              <TooltipTrigger>
                <Button type="button" variant="outline" size="lg" disabled className="h-10 px-4">
                  <FileDown size={14} /> Pratinjau Cetak PDF
                </Button>
              </TooltipTrigger>
              <TooltipContent side="top">
                <p>Tersedia setelah editor surat siap</p>
              </TooltipContent>
            </Tooltip>

            <Button type="submit" form="buat-surat-form" size="lg" disabled={!isValid} className="h-10 px-5 font-semibold">
              <Send size={14} /> Submit
            </Button>
          </div>
        </div>
      </div>
    </TooltipProvider>
  );
}
