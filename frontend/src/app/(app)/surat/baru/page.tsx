"use client";

// ─────────────────────────────────────────────
// app/(app)/surat/baru/page.tsx
// FE-03: Halaman Buat Surat Baru — terintegrasi backend riil.
// Tampilan tiap bagian ada di components/surat/*.
// ─────────────────────────────────────────────

import { Suspense, useEffect, useState, useRef } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { zodResolver } from "@hookform/resolvers/zod";
import { FormProvider, useForm, useWatch } from "react-hook-form";

import { BUAT_SURAT_DEFAULTS, buatSuratSchema, type BuatSuratFormValues } from "@/lib/schemas";
import type { LetterAttachment, StepEntry } from "@/lib/types";
import { useAutosave } from "@/hooks/use-autosave";
import { generateId } from "@/lib/utils";

import {
  saveDraft,
  submitLetter,
  getDraftLetter,
} from "@/lib/api/letter-draft";
import { listLetters } from "@/lib/api/letters";
import type { RecipientUnit } from "@/lib/api/letter-detail";

import { ActionBar } from "@/components/surat/action-bar";
import { BuatSuratHeader } from "@/components/surat/buat-surat-header";
import { ConfirmDialogs, type DialogType } from "@/components/surat/confirm-dialogs";
import { DetailSection } from "@/components/surat/detail-section";
import { SectionCard } from "@/components/surat/form-primitives";
import { EditorPlaceholder } from "@/components/surat/editor-placeholder";
import { LampiranSection } from "@/components/surat/lampiran-section";
import { Loader2 } from "lucide-react";

export default function BuatSuratPage() {
  const [formKey, setFormKey] = useState(0);

  return (
    <Suspense
      fallback={
        <div className="min-h-screen flex items-center justify-center text-text-muted gap-2 text-sm">
          <Loader2 size={16} className="animate-spin text-primary" />
          Memuat form Buat Surat...
        </div>
      }
    >
      <BuatSuratFormInner key={formKey} onNew={() => setFormKey((k) => k + 1)} />
    </Suspense>
  );
}

function BuatSuratFormInner({ onNew }: { onNew: () => void }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const draftQueryId = searchParams.get("draft");

  const methods = useForm<BuatSuratFormValues>({
    resolver: zodResolver(buatSuratSchema),
    mode: "onChange",
    defaultValues: BUAT_SURAT_DEFAULTS,
  });
  const { control, reset, trigger } = methods;

  const [draftId, setDraftId] = useState<string | null>(draftQueryId || null);
  const [draftEtag, setDraftEtag] = useState<string | null>(null);

  const [attachments, setAttachments] = useState<LetterAttachment[]>([]);
  const [dialog, setDialog] = useState<DialogType>(null);
  const [loadingDraft, setLoadingDraft] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitFailed, setSubmitFailed] = useState(false);
  const [submitErrorMessage, setSubmitErrorMessage] = useState<string | null>(null);

  const values = useWatch({ control });

  // Refs untuk mencegah race condition pada autosave & simpan berurutan
  const draftIdRef = useRef<string | null>(draftId);
  draftIdRef.current = draftId;
  const draftEtagRef = useRef<string | null>(draftEtag);
  draftEtagRef.current = draftEtag;

  // ─── Memuat draf jika ada parameter ?draft=UUID ───
  useEffect(() => {
    if (!draftQueryId) return;

    let isMounted = true;
    setLoadingDraft(true);

    getDraftLetter(draftQueryId)
      .then(({ detail, etag }) => {
        if (!isMounted) return;

        setDraftId(detail.id);
        setDraftEtag(etag);

        // Ekstrak reviewers & approvers
        const reviewers: StepEntry[] = (detail.steps ?? [])
          .filter((s) => s.role === "REVIEWER")
          .map((s) => ({
            id: generateId(),
            employeeId: String(s.user.id),
            employeeName: s.user.name,
            jabatan: s.user.positionName,
            unitName: s.user.unitName,
            kodeAlias: "R",
          }));

        const approvers: StepEntry[] = (detail.steps ?? [])
          .filter((s) => s.role === "APPROVER")
          .map((s) => ({
            id: generateId(),
            employeeId: String(s.user.id),
            employeeName: s.user.name,
            jabatan: s.user.positionName,
            unitName: s.user.unitName,
            kodeAlias: "A",
          }));

        const toUnits = (detail.recipients?.to ?? [])
          .filter((r): r is RecipientUnit => r.type === "UNIT")
          .map((r) => ({ id: String(r.unit.id), name: r.unit.name }));

        const ccUnits = (detail.recipients?.cc ?? [])
          .filter((r): r is RecipientUnit => r.type === "UNIT")
          .map((r) => ({ id: String(r.unit.id), name: r.unit.name }));

        reset({
          letterTypeId: detail.letterType?.id ? String(detail.letterType.id) : "",
          templateId: detail.template?.id ? String(detail.template.id) : undefined,
          classificationId: detail.classification?.id ? String(detail.classification.id) : "",
          kecepatan: (detail.urgency?.code as "BIASA" | "SEGERA" | "SANGAT_SEGERA") || "BIASA",
          sifat: (detail.confidentiality?.code as "BIASA" | "TERBATAS" | "RAHASIA") || "BIASA",
          title: detail.title ?? "",
          specialNote: detail.specialNote ?? "",
          summary: detail.summary ?? "",
          reviewers,
          approvers,
          toUnits,
          ccUnits,
        });
      })
      .catch((e: unknown) => {
        console.error("Gagal memuat draf dari URL:", e);
      })
      .finally(() => {
        if (isMounted) setLoadingDraft(false);
      });

    return () => {
      isMounted = false;
    };
  }, [draftQueryId, reset]);

  // ─── Autosave riil ke backend ──────────────────
  const { isDirty, isSaving, lastSaved, saveFailed, saveNow } = useAutosave({
    signature: JSON.stringify([values, attachments]),
    enabled: dialog !== "sukses" && !loadingDraft,
    save: async () => {
      // Autosave hanya berjalan jika ada minimal 1 field terisi
      const hasAnyField = Boolean(
        values.title ||
        values.letterTypeId ||
        values.classificationId ||
        (values.reviewers && values.reviewers.length > 0) ||
        (values.toUnits && values.toUnits.length > 0)
      );
      if (!hasAnyField) return;

      const res = await saveDraft(values, draftIdRef.current, draftEtagRef.current);
      setDraftId(res.id);
      setDraftEtag(res.etag);
    },
  });

  // ─── Handlers ────────────────────────────────

  async function handleLoadDraft() {
    setLoadingDraft(true);
    try {
      // Ambil draft terakhir milik user dari backend
      const res = await listLetters({ scope: "mine", status: "DRAFT", page: 1, pageSize: 1 });
      if (res.data.length === 0) {
        alert("Belum ada draf tersimpan yang ditemukan di database.");
        return;
      }

      const latestDraftId = res.data[0].id;
      const { detail, etag } = await getDraftLetter(latestDraftId);

      setDraftId(detail.id);
      setDraftEtag(etag);

      const reviewers: StepEntry[] = (detail.steps ?? [])
        .filter((s) => s.role === "REVIEWER")
        .map((s) => ({
          id: generateId(),
          employeeId: String(s.user.id),
          employeeName: s.user.name,
          jabatan: s.user.positionName,
          unitName: s.user.unitName,
          kodeAlias: "R",
        }));

      const approvers: StepEntry[] = (detail.steps ?? [])
        .filter((s) => s.role === "APPROVER")
        .map((s) => ({
          id: generateId(),
          employeeId: String(s.user.id),
          employeeName: s.user.name,
          jabatan: s.user.positionName,
          unitName: s.user.unitName,
          kodeAlias: "A",
        }));

      const toUnits = (detail.recipients?.to ?? [])
        .filter((r): r is RecipientUnit => r.type === "UNIT")
        .map((r) => ({ id: String(r.unit.id), name: r.unit.name }));

      const ccUnits = (detail.recipients?.cc ?? [])
        .filter((r): r is RecipientUnit => r.type === "UNIT")
        .map((r) => ({ id: String(r.unit.id), name: r.unit.name }));

      reset({
        letterTypeId: detail.letterType?.id ? String(detail.letterType.id) : "",
        templateId: detail.template?.id ? String(detail.template.id) : undefined,
        classificationId: detail.classification?.id ? String(detail.classification.id) : "",
        kecepatan: (detail.urgency?.code as "BIASA" | "SEGERA" | "SANGAT_SEGERA") || "BIASA",
        sifat: (detail.confidentiality?.code as "BIASA" | "TERBATAS" | "RAHASIA") || "BIASA",
        title: detail.title ?? "",
        specialNote: detail.specialNote ?? "",
        summary: detail.summary ?? "",
        reviewers,
        approvers,
        toUnits,
        ccUnits,
      });

      await trigger();
    } catch (err: unknown) {
      console.error("Gagal memuat draf terakhir:", err);
      alert("Gagal memuat draf terakhir.");
    } finally {
      setLoadingDraft(false);
    }
  }

  async function handleSimpan() {
    try {
      const res = await saveDraft(values, draftIdRef.current, draftEtagRef.current);
      setDraftId(res.id);
      setDraftEtag(res.etag);
      setDialog(null);
    } catch (e: unknown) {
      console.error("Gagal simpan draf:", e);
      // saveFailed akan ditampilkan di dalam dialog
    }
  }

  async function handleSubmit() {
    setSubmitFailed(false);
    setSubmitErrorMessage(null);
    setIsSubmitting(true);

    try {
      // 1. Simpan perubahan draf terlebih dahulu ke database SQL Server
      const saved = await saveDraft(values, draftIdRef.current, draftEtagRef.current);
      setDraftId(saved.id);
      setDraftEtag(saved.etag);

      // 2. Eksekusi pengiriman surat ke reviewer pertama (POST /api/v1/letters/{id}/submit)
      await submitLetter(saved.id);

      // 3. Tampilkan modal dialog sukses
      setDialog("sukses");
    } catch (err: unknown) {
      console.error("Gagal mengirim surat:", err);
      setSubmitFailed(true);
      setSubmitErrorMessage(
        err instanceof Error
          ? err.message
          : "Gagal mengirim surat. Periksa kelengkapan surat dan coba lagi."
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  function handleBatal() {
    if (isDirty) setDialog("batal");
    else router.push("/surat/daftar");
  }

  // ─── Render ──────────────────────────────────

  return (
    <FormProvider {...methods}>
      <div className="min-h-screen pb-28">
        <BuatSuratHeader
          isSaving={isSaving}
          isDirty={isDirty}
          saveFailed={saveFailed}
          lastSaved={lastSaved}
          loadingDraft={loadingDraft}
          onLoadDraft={handleLoadDraft}
        />

        <div className="space-y-5">
          <SectionCard number={1} title="Detail Surat">
            <DetailSection onValidSubmit={() => setDialog("submit")} />
          </SectionCard>

          <SectionCard number={2} title="Lampiran">
            <LampiranSection value={attachments} onChange={setAttachments} />
          </SectionCard>

          <SectionCard number={3} title="Isi Surat & Tata Naskah Digital">
            <EditorPlaceholder />
          </SectionCard>
        </div>

        <ActionBar onBatal={handleBatal} onSimpan={() => setDialog("simpan")} />

        <ConfirmDialogs
          dialog={dialog}
          onClose={() => setDialog(null)}
          isSaving={isSaving}
          saveFailed={saveFailed}
          onSimpan={handleSimpan}
          isSubmitting={isSubmitting}
          submitFailed={submitFailed}
          submitError={submitErrorMessage}
          onSubmit={handleSubmit}
          onLeave={() => router.push("/surat/daftar")}
          onNew={onNew}
          onGoToList={() => router.push("/surat/daftar")}
        />
      </div>
    </FormProvider>
  );
}
