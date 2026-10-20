"use client";

// ─────────────────────────────────────────────
// components/surat/basic-fields.tsx
// Jenis, Template, Klasifikasi, Kecepatan, Sifat, Perihal
// Dropdown diisi dari backend (lookups, template tetap mock)
// ─────────────────────────────────────────────

import { CheckCircle2, Info, Loader2 } from "lucide-react";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import { useEffect, useState } from "react";
import type { BuatSuratFormValues } from "@/lib/schemas";
import { LETTER_TEMPLATES } from "@/lib/mock-data";
import { fetchLookups, type LookupItem } from "@/lib/api/form";
import { cn } from "@/lib/utils";
import { ClassificationCombobox } from "./classification-combobox";
import { FieldError, FieldLabel, inputClass } from "./form-primitives";

// ─── Hook: ambil lookups dari backend ────────

function useLookups(category: Parameters<typeof fetchLookups>[0]) {
  const [items, setItems] = useState<LookupItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    fetchLookups(category)
      .then((data) => { if (!cancelled) setItems(data); })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : "Gagal memuat data.");
      })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [category]);

  return { items, loading, error };
}

// ─── SelectLoading — placeholder saat fetch ──

function SelectLoading({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-2 px-3 py-2.5 border border-border-input rounded-lg bg-surface text-sm text-text-muted">
      <Loader2 size={14} className="animate-spin shrink-0" />
      Memuat {label}...
    </div>
  );
}

function SelectError({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="flex items-center justify-between gap-2 px-3 py-2 border border-status-danger-text/40 rounded-lg bg-status-danger-bg text-xs text-status-danger-text">
      <span>{message}</span>
      <button type="button" onClick={onRetry} className="underline shrink-0">Coba lagi</button>
    </div>
  );
}

// ─── Component utama ─────────────────────────

export function BasicFields() {
  const { register, control, formState: { errors }, setValue } = useFormContext<BuatSuratFormValues>();
  const [letterTypeId, title = "", sifat] = useWatch({ control, name: ["letterTypeId", "title", "sifat"] });

  // Lookup dari backend
  const letterTypes = useLookups("LETTER_TYPE");
  const urgencies   = useLookups("URGENCY");
  const [retryLT, setRetryLT] = useState(0);
  const [retryUG, setRetryUG] = useState(0);

  // Saat letter types pertama kali muat, set default ke item pertama
  useEffect(() => {
    if (letterTypes.items.length > 0 && !letterTypeId) {
      setValue("letterTypeId", letterTypes.items[0]!.id);
    }
  }, [letterTypes.items, letterTypeId, setValue]);

  // Template masih mock — belum ada endpoint backend
  const templates = LETTER_TEMPLATES.filter((t) => t.letterTypeId === letterTypeId);

  const sifatOptions: LookupItem[] = [
    { id: "sifat-biasa",    code: "BIASA",    name: "Biasa",    note: "Surat dapat diakses seluruh staf unit terkait.", sortOrder: 1 },
    { id: "sifat-terbatas", code: "TERBATAS", name: "Terbatas", note: "Hanya pejabat dan staf berwenang yang dapat mengakses.", sortOrder: 2 },
    { id: "sifat-rahasia",  code: "RAHASIA",  name: "Rahasia",  note: "Hanya pejabat yang mendapat otorisasi khusus.", sortOrder: 3 },
  ];

  const selectedSifat = sifatOptions.find((s) => s.code === sifat);

  return (
    <>
      {/* Jenis Surat */}
      <div>
        <FieldLabel label="Jenis Surat" required htmlFor="letterTypeId" />
        {letterTypes.loading ? (
          <SelectLoading label="jenis surat" />
        ) : letterTypes.error ? (
          <SelectError message={letterTypes.error} onRetry={() => setRetryLT((n) => n + 1)} />
        ) : (
          <select
            id="letterTypeId"
            {...register("letterTypeId")}
            className={inputClass(!!errors.letterTypeId)}
            key={retryLT}
          >
            {letterTypes.items.map((lt) => (
              <option key={lt.id} value={lt.id}>{lt.name}</option>
            ))}
          </select>
        )}
        <FieldError message={errors.letterTypeId?.message} />
      </div>

      {/* Template Surat — tetap mock sampai backend punya endpoint */}
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

      {/* Kecepatan Tanggapan — dari backend URGENCY lookup */}
      <div>
        <FieldLabel label="Kecepatan Tanggapan" required htmlFor="kecepatan" />
        {urgencies.loading ? (
          <SelectLoading label="pilihan kecepatan" />
        ) : urgencies.error ? (
          <SelectError message={urgencies.error} onRetry={() => setRetryUG((n) => n + 1)} />
        ) : (
          <Controller
            name="kecepatan"
            control={control}
            render={({ field }) => (
              <div className="space-y-1">
                <select
                  id="kecepatan"
                  value={field.value ?? ""}
                  onChange={(e) => {
                    const item = urgencies.items.find((u) => u.id === e.target.value);
                    if (item) field.onChange(item.code as BuatSuratFormValues["kecepatan"]);
                  }}
                  className={inputClass(!!errors.kecepatan)}
                  key={retryUG}
                >
                  <option value="">— Pilih kecepatan —</option>
                  {urgencies.items.map((u) => (
                    <option key={u.id} value={u.id}>{u.name}</option>
                  ))}
                </select>
                {/* Teks bantu dari backend (field `note`) */}
                {field.value && (() => {
                  const item = urgencies.items.find((u) => u.code === field.value);
                  return item?.note ? (
                    <p className="text-xs text-text-muted flex items-start gap-1.5">
                      <Info size={12} className="mt-0.5 shrink-0" />
                      {item.note}
                    </p>
                  ) : null;
                })()}
              </div>
            )}
          />
        )}
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
              <div className="flex gap-3 flex-wrap">
                {sifatOptions.map((opt) => (
                  <label
                    key={opt.code}
                    className={cn(
                      "flex items-center gap-2 px-3 py-2 rounded-lg border cursor-pointer text-sm transition-colors select-none",
                      field.value === opt.code
                        ? "border-primary bg-primary/5 text-primary font-medium"
                        : "border-border text-text hover:border-primary/40 hover:bg-panel/50",
                    )}
                  >
                    <input
                      type="radio"
                      value={opt.code}
                      checked={field.value === opt.code}
                      onChange={() => field.onChange(opt.code)}
                      className="accent-primary"
                    />
                    {opt.name}
                  </label>
                ))}
              </div>
              {selectedSifat?.note && (
                <p className="text-xs text-text-muted flex items-start gap-1.5">
                  <Info size={12} className="mt-0.5 shrink-0" />
                  {selectedSifat.note}
                </p>
              )}
            </div>
          )}
        />
        <FieldError message={errors.sifat?.message} />
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
