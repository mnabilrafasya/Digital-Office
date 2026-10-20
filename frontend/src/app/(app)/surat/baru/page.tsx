"use client";

// ─────────────────────────────────────────────
// app/(app)/surat/baru/page.tsx
// FE-03: Halaman Buat Surat Baru — orkestrator.
// Tampilan tiap bagian ada di components/surat/*.
// ─────────────────────────────────────────────

import { useState } from "react";
import { useRouter } from "next/navigation";
import { zodResolver } from "@hookform/resolvers/zod";
import { FormProvider, useForm, useWatch } from "react-hook-form";

import { BUAT_SURAT_DEFAULTS, buatSuratSchema, type BuatSuratFormValues } from "@/lib/schemas";
import type { LetterAttachment } from "@/lib/types";
import { useAutosave } from "@/hooks/use-autosave";

import { ActionBar } from "@/components/surat/action-bar";
import { BuatSuratHeader } from "@/components/surat/buat-surat-header";
import { ConfirmDialogs, type DialogType } from "@/components/surat/confirm-dialogs";
import { DetailSection } from "@/components/surat/detail-section";
import { SectionCard } from "@/components/surat/form-primitives";
import { EditorPlaceholder } from "@/components/surat/editor-placeholder";
import { LampiranSection } from "@/components/surat/lampiran-section";

/** Wrapper: mengganti `key` me-reset seluruh state (dipakai tombol "Buat Surat Lagi"). */
export default function BuatSuratPage() {
  const [formKey, setFormKey] = useState(0);
  return <BuatSuratFormInner key={formKey} onNew={() => setFormKey((k) => k + 1)} />;
}

function BuatSuratFormInner({ onNew }: { onNew: () => void }) {
  const router = useRouter();
  const methods = useForm<BuatSuratFormValues>({
    resolver: zodResolver(buatSuratSchema),
    mode: "onChange",
    defaultValues: BUAT_SURAT_DEFAULTS,
  });
  const { control, reset, trigger } = methods;

  const [attachments, setAttachments] = useState<LetterAttachment[]>([]);
  const [dialog, setDialog] = useState<DialogType>(null);
  const [loadingDraft, setLoadingDraft] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitFailed, setSubmitFailed] = useState(false);

  const values = useWatch({ control });

  // ─── Autosave ────────────────────────────────
  const { isDirty, isSaving, lastSaved, saveFailed, saveNow } = useAutosave({
    signature: JSON.stringify([values, attachments]),
    enabled: dialog !== "sukses",
    save: async () => {
      // TODO: ganti dengan API call saveDraft() saat backend siap
      await new Promise((r) => setTimeout(r, 600));
    },
  });

  // ─── Handlers ────────────────────────────────

  async function handleLoadDraft() {
    setLoadingDraft(true);
    try {
      // TODO: ganti dengan getLastDraft() saat backend siap
      await new Promise((r) => setTimeout(r, 300));
      // Catatan: endpoint GET /letters/draft-terakhir belum ada di backend.
      // Mereset ke default bersih agar tidak mencampur ID mock usang dengan ID database.
      reset(BUAT_SURAT_DEFAULTS);
      await trigger();
    } finally {
      setLoadingDraft(false);
    }
  }

  async function handleSimpan() {
    try {
      await saveNow();
      setDialog(null);
    } catch {
      // saveFailed sudah tampil di dalam ConfirmDialogs
    }
  }

  async function handleSubmit() {
    setSubmitFailed(false);
    setIsSubmitting(true);
    try {
      // TODO: ganti dengan submitLetter() saat backend siap
      await new Promise((r) => setTimeout(r, 900));
      setDialog("sukses");
    } catch {
      setSubmitFailed(true);
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
          onSubmit={handleSubmit}
          onLeave={() => router.push("/surat/daftar")}
          onNew={onNew}
          onGoToList={() => router.push("/surat/daftar")}
        />

      </div>
    </FormProvider>
  );
}
