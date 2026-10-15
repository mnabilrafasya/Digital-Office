"use client";

// ─────────────────────────────────────────────
// components/surat/detail-section.tsx
// Section 1: Detail Surat (form id "buat-surat-form")
// ─────────────────────────────────────────────

import { useFormContext } from "react-hook-form";
import type { BuatSuratFormValues } from "@/lib/schemas";
import { BasicFields } from "./basic-fields";
import { FieldError, FieldLabel, inputClass } from "./form-primitives";
import { RecipientFields } from "./recipient-fields";

export function DetailSection({ onValidSubmit }: { onValidSubmit: () => void }) {
  const { register, handleSubmit, formState: { errors } } = useFormContext<BuatSuratFormValues>();

  return (
    <form id="buat-surat-form" onSubmit={handleSubmit(onValidSubmit)}>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
        <BasicFields />
        <RecipientFields />

        <div className="md:col-span-2">
          <FieldLabel label="Keterangan Khusus" htmlFor="specialNote" />
          <input
            id="specialNote"
            type="text"
            placeholder="Keterangan tambahan yang perlu diperhatikan (opsional)..."
            {...register("specialNote")}
            className={inputClass(!!errors.specialNote)}
          />
          <FieldError message={errors.specialNote?.message} />
        </div>

        <div className="md:col-span-2">
          <FieldLabel label="Isi Ringkas Arsip" htmlFor="summary" />
          <textarea
            id="summary"
            rows={3}
            placeholder="Ringkasan isi surat untuk keperluan arsip (opsional)..."
            {...register("summary")}
            className={`${inputClass(!!errors.summary)} resize-none`}
          />
          <FieldError message={errors.summary?.message} />
        </div>
      </div>
    </form>
  );
}
