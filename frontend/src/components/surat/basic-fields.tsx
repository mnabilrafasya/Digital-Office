"use client";

// ─────────────────────────────────────────────
// components/surat/basic-fields.tsx
// Jenis, Template, Klasifikasi, Kecepatan, Sifat, Perihal
// Dropdown diisi dari backend (lookups & /api/v1/templates)
// ─────────────────────────────────────────────

import { CheckCircle2, Info, Loader2 } from "lucide-react";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import { useEffect, useState } from "react";
import type { BuatSuratFormValues } from "@/lib/schemas";
import {
  fetchTemplates,
  getCachedTemplates,
  type LetterTemplateDefinition,
} from "@/lib/templates/letter-templates";
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
    fetchLookups(category)
      .then((data) => {
        if (!cancelled) {
          setItems(data);
          setError(null);
        }
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : "Gagal memuat data.");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
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
  const { control, formState: { errors }, setValue } = useFormContext<BuatSuratFormValues>();
  const letterTypeId = useWatch({ control, name: "letterTypeId" });
  const templateId = useWatch({ control, name: "templateId" });
  const sifat = useWatch({ control, name: "sifat" });
  const kecepatan = useWatch({ control, name: "kecepatan" });

  // Lookup dari backend
  const letterTypes = useLookups("LETTER_TYPE");
  const urgencies   = useLookups("URGENCY");
  const [retryLT, setRetryLT] = useState(0);
  const [retryUG, setRetryUG] = useState(0);

  // Daftar template naskah dari backend (/api/v1/templates)
  const [templates, setTemplates] = useState<LetterTemplateDefinition[]>(() =>
    getCachedTemplates()
  );

  useEffect(() => {
    let cancelled = false;
    fetchTemplates().then((items) => {
      if (!cancelled) {
        setTemplates(items);
      }
    });
    return () => {
      cancelled = true;
    };
  }, []);

  // Normalisasi ID template warisan (mis. "tmpl-dinas-standar") menjadi ID numerik backend saat daftar selesai dimuat
  useEffect(() => {
    if (!templateId) return;
    const exactMatch = templates.find((t) => t.id === templateId);
    if (exactMatch) return;
    const legacyMatch = templates.find((t) => t.legacyId === templateId);
    if (legacyMatch && legacyMatch.id !== templateId) {
      setValue("templateId", legacyMatch.id);
    }
  }, [templateId, templates, setValue]);

  // Saat urgencies selesai muat, set default ke BIASA jika belum dipilih
  useEffect(() => {
    if (urgencies.items.length > 0 && !kecepatan) {
      setValue("kecepatan", "BIASA", { shouldValidate: true });
    }
  }, [urgencies.items, kecepatan, setValue]);

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
          <Controller
            name="letterTypeId"
            control={control}
            render={({ field }) => (
              <select
                id="letterTypeId"
                value={field.value ?? ""}
                onChange={(e) => {
                  const val = e.target.value;
                  field.onChange(val);
                  const selectedLT = letterTypes.items.find((lt) => lt.id === val);
                  if (selectedLT) {
                    const ltName = selectedLT.name.toLowerCase();
                    const ltCode = selectedLT.code.toLowerCase();
                    // 1. Cari template khusus untuk jenis surat ini
                    // 2. Jika tidak ada, cari template umum (!t.letterType) yang cocok keyword-nya
                    // 3. Fallback ke template umum pertama (mis. Surat Dinas Resmi Standar)
                    const matched =
                      templates.find(
                        (t) =>
                          t.letterType?.id === val ||
                          t.letterType?.code?.toLowerCase() === ltCode
                      ) ??
                      templates.find(
                        (t) =>
                          !t.letterType &&
                          t.letterTypeKeywords.some(
                            (kw) => ltName.includes(kw) || ltCode.includes(kw)
                          )
                      ) ??
                      templates.find((t) => !t.letterType) ??
                      templates[0];
                    if (matched) {
                      setValue("templateId", matched.id, {
                        shouldValidate: true,
                        shouldDirty: true,
                      });
                    }
                  }
                }}
                className={inputClass(!!errors.letterTypeId)}
                key={retryLT}
              >
                <option value="">— Pilih jenis surat —</option>
                {letterTypes.items.map((lt) => (
                  <option key={lt.id} value={lt.id}>{lt.name}</option>
                ))}
              </select>
            )}
          />
        )}
        <FieldError message={errors.letterTypeId?.message} />
      </div>

      {/* Template Surat — daftar aktif dari GET /api/v1/templates (dikelola di Master Data) */}
      <div>
        <FieldLabel label="Template Surat" htmlFor="templateId" />
        <Controller
          name="templateId"
          control={control}
          render={({ field }) => (
            <select
              id="templateId"
              value={field.value ?? ""}
              onChange={(e) => {
                const nextTplId = e.target.value || undefined;
                field.onChange(nextTplId);
                // Jika Jenis Surat belum dipilih dan user memilih template khusus jenis tertentu,
                // otomatis sesuaikan Jenis Surat-nya agar tidak ditolak backend
                if (nextTplId && !letterTypeId) {
                  const chosen = templates.find((t) => t.id === nextTplId);
                  if (chosen?.letterType?.id) {
                    setValue("letterTypeId", chosen.letterType.id, {
                      shouldValidate: true,
                      shouldDirty: true,
                    });
                  }
                }
              }}
              className={inputClass()}
            >
              <option value="">— Pilih template (opsional) —</option>
              {templates.map((t) => (
                <option key={t.id} value={t.id}>{t.name}</option>
              ))}
            </select>
          )}
        />
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
                  value={field.value ?? "BIASA"}
                  onChange={(e) =>
                    field.onChange(e.target.value as BuatSuratFormValues["kecepatan"])
                  }
                  className={inputClass(!!errors.kecepatan)}
                  key={retryUG}
                >
                  <option value="">— Pilih kecepatan —</option>
                  {urgencies.items.map((u) => (
                    <option key={u.id} value={u.code}>
                      {u.name}
                    </option>
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
        <Controller
          name="title"
          control={control}
          render={({ field }) => {
            const val = field.value ?? "";
            return (
              <div className="relative">
                <input
                  id="title"
                  type="text"
                  maxLength={250}
                  placeholder="Tuliskan perihal surat secara singkat dan jelas..."
                  value={val}
                  onChange={(e) => field.onChange(e.target.value)}
                  onBlur={field.onBlur}
                  className={cn(inputClass(!!errors.title), "pr-24")}
                />
                <div className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5 pointer-events-none select-none">
                  <span
                    className={cn(
                      "text-xs font-mono",
                      val.length > 230 ? "text-status-danger-text" : "text-text-muted"
                    )}
                  >
                    {val.length} / 250
                  </span>
                  {val.trim().length > 0 && !errors.title && (
                    <CheckCircle2 size={14} className="text-green-500" />
                  )}
                </div>
              </div>
            );
          }}
        />
        <FieldError message={errors.title?.message} />
      </div>
    </>
  );
}
