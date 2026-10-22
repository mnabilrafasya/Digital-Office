"use client";

// ─────────────────────────────────────────────
// components/surat/recipient-fields.tsx
// Reviewer, Approver, Tujuan Surat (TO), Tembusan (CC)
// ─────────────────────────────────────────────

import { Controller, useFormContext, useWatch } from "react-hook-form";
import type { BuatSuratFormValues } from "@/lib/schemas";
import { FieldLabel } from "./form-primitives";
import { StepPicker } from "./step-picker";
import { UnitPicker } from "./unit-picker";

export function RecipientFields() {
  const { control, formState: { errors } } = useFormContext<BuatSuratFormValues>();
  const [reviewers = [], approvers = [], toUnits = [], ccUnits = []] = useWatch({
    control,
    name: ["reviewers", "approvers", "toUnits", "ccUnits"],
  });

  return (
    <>
      {/* Reviewer */}
      <div className="md:col-span-2">
        <div className="flex items-center justify-between mb-1">
          <FieldLabel label="Reviewer" required />
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
              otherRoleIds={approvers.map((a) => a.employeeId)}
            />
          )}
        />
      </div>

      {/* Approver */}
      <div className="md:col-span-2">
        <div className="flex items-center justify-between mb-1">
          <FieldLabel label="Approver" required />
          <span className="text-xs text-text-muted">Maks. 10 orang</span>
        </div>
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
              otherRoleIds={reviewers.map((r) => r.employeeId)}
            />
          )}
        />
      </div>

      {/* Tujuan Surat */}
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
              excludeIds={ccUnits.map((u) => u.id)}
            />
          )}
        />
      </div>

      {/* Tembusan (CC) */}
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
              excludeIds={toUnits.map((u) => u.id)}
            />
          )}
        />
      </div>
    </>
  );
}
