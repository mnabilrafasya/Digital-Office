"use client";

// ─────────────────────────────────────────────
// components/surat/recipient-fields.tsx
// Reviewer, Approver, Tujuan Surat (TO), Tembusan (CC)
// ─────────────────────────────────────────────

import { Building2, Info } from "lucide-react";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import type { BuatSuratFormValues } from "@/lib/schemas";
import { FieldError, FieldLabel, inputClass } from "./form-primitives";
import { StepPicker } from "./step-picker";
import { UnitPicker } from "./unit-picker";

export function RecipientFields() {
  const {
    control,
    formState: { errors },
  } = useFormContext<BuatSuratFormValues>();
  const [
    reviewers = [],
    approvers = [],
    toUnits = [],
    ccUnits = [],
    letterTypeCode = "",
  ] = useWatch({
    control,
    name: ["reviewers", "approvers", "toUnits", "ccUnits", "letterTypeCode"],
  });

  const isExternalOutgoing =
    (letterTypeCode ?? "").toUpperCase() === "SURAT_EKSTERNAL";

  return (
    <>
      {/* Reviewer */}
      <div className="md:col-span-2">
        <div className="flex items-center justify-between mb-1">
          <FieldLabel label="Reviewer (Pemeriksa / Pemaraf)" required />
          <span className="text-xs text-text-muted">Maks. 10 orang</span>
        </div>
        <Controller
          name="reviewers"
          control={control}
          render={({ field }) => (
            <StepPicker
              kodeAlias="R"
              role="REVIEWER"
              value={field.value}
              onChange={field.onChange}
              error={errors.reviewers?.message}
              otherRoleIds={(approvers ?? []).map((a) => a.employeeId)}
            />
          )}
        />
      </div>

      {/* Approver */}
      <div className="md:col-span-2">
        <div className="flex items-center justify-between mb-1">
          <FieldLabel label="Approver (Pejabat Penyetuju & Penandatangan)" required />
          <span className="text-xs text-text-muted">Maks. 10 orang (berurutan)</span>
        </div>
        <p className="text-xs text-text-muted mb-2">
          {isExternalOutgoing ? (
            <>
              Jika lebih dari 1 approver, persetujuan berjalan berurutan (A1 s/d A{(approvers ?? []).length || "N"}). Setelah{" "}
              <strong className="font-medium text-text">Approver terakhir</strong> menyetujui, nomor surat terbit otomatis dan naskah menjadi PDF siap cetak untuk ditandatangani basah lalu diunggah kembali hasil scan-nya.
            </>
          ) : (
            <>
              Jika lebih dari 1 approver, persetujuan berjalan berurutan (A1 s/d A{(approvers ?? []).length || "N"}) dan{" "}
              <strong className="font-medium text-text">Approver terakhir</strong> menjadi pejabat penandatangan yang mengesahkan nomor & QR surat.
            </>
          )}
        </p>
        <Controller
          name="approvers"
          control={control}
          render={({ field }) => (
            <StepPicker
              kodeAlias="A"
              role="APPROVER"
              value={field.value}
              onChange={field.onChange}
              error={errors.approvers?.message}
              otherRoleIds={(reviewers ?? []).map((r) => r.employeeId)}
            />
          )}
        />
      </div>

      {isExternalOutgoing ? (
        /* Tujuan Surat Eksternal (Nama Perusahaan/Instansi, UP, Alamat) */
        <div className="md:col-span-2 border border-primary/25 bg-primary/[0.03] rounded-xl p-4 space-y-4">
          <div className="flex items-start gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-primary/10 text-primary flex items-center justify-center shrink-0 mt-0.5">
              <Building2 size={16} />
            </div>
            <div>
              <h4 className="text-sm font-bold text-text">
                Tujuan Surat Eksternal (Instansi / Perusahaan Luar)
              </h4>
              <p className="text-xs text-text-muted mt-0.5">
                Surat Eksternal Keluar ditujukan ke pihak luar perusahaan dan menggunakan tanda tangan basah (tanpa QR internal).
              </p>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <FieldLabel
                label="Nama Perusahaan / Instansi Tujuan"
                required
                htmlFor="externalRecipientName"
              />
              <Controller
                name="externalRecipient.name"
                control={control}
                render={({ field }) => (
                  <input
                    id="externalRecipientName"
                    type="text"
                    maxLength={200}
                    placeholder="Contoh: PT Pupuk Indonesia (Persero) / Kementerian BUMN..."
                    value={field.value ?? ""}
                    onChange={(e) => field.onChange(e.target.value)}
                    onBlur={field.onBlur}
                    className={inputClass(!!errors.externalRecipient?.name)}
                  />
                )}
              />
              <FieldError message={errors.externalRecipient?.name?.message} />
            </div>

            <div>
              <FieldLabel
                label="U.p. / Untuk Perhatian (Jabatan / Pejabat Tujuan)"
                htmlFor="externalRecipientAttention"
              />
              <Controller
                name="externalRecipient.attention"
                control={control}
                render={({ field }) => (
                  <input
                    id="externalRecipientAttention"
                    type="text"
                    maxLength={200}
                    placeholder="Contoh: Yth. Direktur Utama / Kepala Divisi (opsional)..."
                    value={field.value ?? ""}
                    onChange={(e) => field.onChange(e.target.value)}
                    onBlur={field.onBlur}
                    className={inputClass(!!errors.externalRecipient?.attention)}
                  />
                )}
              />
              <FieldError
                message={errors.externalRecipient?.attention?.message}
              />
            </div>

            <div className="md:col-span-2">
              <FieldLabel
                label="Alamat Lengkap Tujuan Eksternal"
                htmlFor="externalRecipientAddress"
              />
              <Controller
                name="externalRecipient.address"
                control={control}
                render={({ field }) => (
                  <textarea
                    id="externalRecipientAddress"
                    rows={2}
                    maxLength={500}
                    placeholder="Contoh: Jl. Taman Anggrek Kemanggisan Jaya, Jakarta Barat (opsional)..."
                    value={field.value ?? ""}
                    onChange={(e) => field.onChange(e.target.value)}
                    onBlur={field.onBlur}
                    className={`${inputClass(!!errors.externalRecipient?.address)} resize-none`}
                  />
                )}
              />
              <FieldError message={errors.externalRecipient?.address?.message} />
            </div>
          </div>

          <div className="flex items-start gap-2 text-xs text-text-muted bg-surface border border-border rounded-lg px-3 py-2">
            <Info size={14} className="text-primary shrink-0 mt-0.5" />
            <span>
              Alur Surat Eksternal Keluar: <strong>Draft &rarr; Review &rarr; Approval &rarr; Cetak PDF &amp; Tanda Tangan Basah &rarr; Unggah Scan PDF Resmi</strong>. Setelah scan diunggah oleh Approver terakhir, surat otomatis terarsip di halaman <strong>Kotak Masuk (Kategori Eksternal)</strong>.
            </span>
          </div>
        </div>
      ) : (
        <>
          {/* Tujuan Surat Internal */}
          <div className="md:col-span-2">
            <FieldLabel label="Tujuan Surat (Kepada)" required />
            <p className="text-xs text-text-muted mb-2">
              Pilih unit tujuan. Pimpinan unit akan dicari otomatis oleh sistem.
            </p>
            <Controller
              name="toUnits"
              control={control}
              render={({ field }) => (
                <UnitPicker
                  label="+ Tambah Instansi / Unit"
                  value={field.value}
                  onChange={field.onChange}
                  error={errors.toUnits?.message}
                  excludeIds={(ccUnits ?? []).map((u) => u.id)}
                />
              )}
            />
          </div>

          {/* Tembusan (CC) Internal */}
          <div className="md:col-span-2">
            <FieldLabel label="Tembusan (CC)" />
            <p className="text-xs text-text-muted mb-2">
              Opsional. Unit di Tembusan tidak boleh sama dengan Tujuan Surat.
            </p>
            <Controller
              name="ccUnits"
              control={control}
              render={({ field }) => (
                <UnitPicker
                  label="Pilih Tembusan"
                  value={field.value ?? []}
                  onChange={field.onChange}
                  excludeIds={(toUnits ?? []).map((u) => u.id)}
                />
              )}
            />
          </div>
        </>
      )}
    </>
  );
}
