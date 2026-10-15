"use client";

// ─────────────────────────────────────────────
// components/surat/basic-fields.tsx
// Jenis, Template, Klasifikasi, Kecepatan, Sifat, Perihal
// ─────────────────────────────────────────────

import { CheckCircle2, Info } from "lucide-react";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import type { BuatSuratFormValues } from "@/lib/schemas";
import { LETTER_TEMPLATES, LETTER_TYPES, SIFAT_HELP } from "@/lib/mock-data";
import { cn } from "@/lib/utils";
import { ClassificationCombobox } from "./classification-combobox";
import { FieldError, FieldLabel, inputClass } from "./form-primitives";

const SIFAT_OPTIONS = ["BIASA", "TERBATAS", "RAHASIA"] as const;

export function BasicFields() {
  const { register, control, formState: { errors } } = useFormContext<BuatSuratFormValues>();
  const [letterTypeId, title = "", sifat] = useWatch({ control, name: ["letterTypeId", "title", "sifat"] });

  const templates = LETTER_TEMPLATES.filter((t) => t.letterTypeId === letterTypeId);

  return (
    <>
      {/* Jenis Surat */}
      <div>
        <FieldLabel label="Jenis Surat" required htmlFor="letterTypeId" />
        <select id="letterTypeId" {...register("letterTypeId")} className={inputClass(!!errors.letterTypeId)}>
          {LETTER_TYPES.map((lt) => (
            <option key={lt.id} value={lt.id}>{lt.name}</option>
          ))}
        </select>
        <FieldError message={errors.letterTypeId?.message} />
      </div>

      {/* Template Surat */}
      <div>
        <FieldLabel label="Template Surat" htmlFor="templateId" />
        <select id="templateId" {...register("templateId")} className={inputClass()}>
          <option value="">— Pilih template (opsional) —</option>
          {templates.map((t) => (
            <option key={t.id} value={t.id}>{t.name}</option>
          ))}
        </select>
      </div>

      {/* Klasifikasi Arsip */}
      <div className="md:col-span-2">
        <FieldLabel label="Klasifikasi Arsip" required htmlFor="classificationSearch" />
        <Controller
          name="classificationId"
          control={control}
          render={({ field }) => (
            <ClassificationCombobox
              value={field.value}
              onChange={(id) => field.onChange(id)}
              hasError={!!errors.classificationId}
            />
          )}
        />
        <FieldError message={errors.classificationId?.message} />
      </div>

      {/* Kecepatan Tanggapan */}
      <div>
        <FieldLabel label="Kecepatan Tanggapan" required htmlFor="kecepatan" />
        <select id="kecepatan" {...register("kecepatan")} className={inputClass(!!errors.kecepatan)}>
          <option value="">— Pilih kecepatan —</option>
          <option value="SEGERA">Segera (&lt; 24 Jam Kerja)</option>
          <option value="SANGAT_SEGERA">Sangat Segera</option>
        </select>
        <FieldError message={errors.kecepatan?.message} />
      </div>

      {/* Sifat Kerahasiaan */}
      <div>
        <FieldLabel label="Sifat Kerahasiaan" required />
        <Controller
          name="sifat"
          control={control}
          render={({ field }) => (
            <div className="space-y-2">
              <div className="flex gap-3">
                {SIFAT_OPTIONS.map((val) => (
                  <label
                    key={val}
                    className={cn(
                      "flex items-center gap-2 px-3 py-2 rounded-lg border cursor-pointer text-sm transition-colors select-none",
                      field.value === val
                        ? "border-primary bg-primary/5 text-primary font-medium"
                        : "border-border text-text hover:border-primary/40 hover:bg-panel/50",
                    )}
                  >
                    <input
                      type="radio"
                      value={val}
                      checked={field.value === val}
                      onChange={() => field.onChange(val)}
                      className="accent-primary"
                    />
                    {val.charAt(0) + val.slice(1).toLowerCase()}
                  </label>
                ))}
              </div>
              {sifat && SIFAT_HELP[sifat] && (
                <p className="text-xs text-text-muted flex items-start gap-1.5">
                  <Info size={12} className="mt-0.5 shrink-0" />
                  {SIFAT_HELP[sifat]}
                </p>
              )}
            </div>
          )}
        />
      </div>

      {/* Perihal / Judul */}
      <div className="md:col-span-2">
        <FieldLabel label="Perihal / Judul Surat" required htmlFor="title" />
        <div className="relative">
          <input
            id="title"
            type="text"
            maxLength={250}
            placeholder="Tuliskan perihal surat secara singkat dan jelas..."
            {...register("title")}
            className={cn(inputClass(!!errors.title), "pr-24")}
          />
          <div className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5">
            <span className={cn("text-xs", title.length > 230 ? "text-status-danger-text" : "text-text-muted")}>
              {title.length} / 250
            </span>
            {title.trim().length > 0 && !errors.title && (
              <CheckCircle2 size={14} className="text-green-500" />
            )}
          </div>
        </div>
        <FieldError message={errors.title?.message} />
      </div>
    </>
  );
}
