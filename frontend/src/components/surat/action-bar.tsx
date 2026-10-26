"use client";

// ─────────────────────────────────────────────
// components/surat/action-bar.tsx
// Action bar sticky: Batal, Simpan Draft, Submit (PDF ada di toolbar editor)
// ─────────────────────────────────────────────

import { Info, Save, Send, X, Loader2 } from "lucide-react";
import { useFormContext, useFormState } from "react-hook-form";
import type { BuatSuratFormValues } from "@/lib/schemas";
import { Button } from "@/components/ui/button";

interface Props {
  onBatal: () => void;
  onSimpan: () => void;
  isSaving?: boolean;
}

export function ActionBar({ onBatal, onSimpan, isSaving = false }: Props) {
  const { control } = useFormContext<BuatSuratFormValues>();
  const { isValid } = useFormState({ control });

  return (
    <>
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

            <Button
              type="button"
              variant="outline"
              size="lg"
              onClick={onSimpan}
              disabled={isSaving}
              className="h-10 px-4 text-primary border-primary/30"
            >
              {isSaving ? (
                <>
                  <Loader2 size={14} className="animate-spin" /> Menyimpan...
                </>
              ) : (
                <>
                  <Save size={14} /> Simpan Draft
                </>
              )}
            </Button>

            <Button
              type="submit"
              form="buat-surat-form"
              size="lg"
              className="h-10 px-5 font-semibold cursor-pointer"
            >
              <Send size={14} /> Submit
            </Button>
          </div>
        </div>
      </div>
    </>
  );
}