"use client";

// ─────────────────────────────────────────────
// app/(app)/surat/baru/page.tsx
// FE-03: Halaman Buat Surat Baru — terintegrasi backend riil & MinIO.
// Tampilan tiap bagian ada di components/surat/*.
// ─────────────────────────────────────────────

import { Suspense, useEffect, useState, useRef } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { zodResolver } from "@hookform/resolvers/zod";
import { FormProvider, useForm, useWatch } from "react-hook-form";

import {
  BUAT_SURAT_DEFAULTS,
  buatSuratSchema,
  type BuatSuratFormValues,
} from "@/lib/schemas";
import type { LetterAttachment } from "@/lib/types";

import {
  saveDraft,
  submitLetter,
  getDraftLetter,
  detailToFormValues,
  DraftSaveError,
} from "@/lib/api/letter-draft";
import { isEditableByDrafter } from "@/lib/api/letter-detail";
import { listLetters } from "@/lib/api/letters";

import { ActionBar } from "@/components/surat/action-bar";
import { BuatSuratHeader } from "@/components/surat/buat-surat-header";
import {
  ConfirmDialogs,
  type DialogType,
} from "@/components/surat/confirm-dialogs";
import { DetailSection } from "@/components/surat/detail-section";
import { SectionCard } from "@/components/surat/form-primitives";
import {
  LetterEditorSection,
  type LetterEditorHandle,
} from "@/components/surat/letter-editor";
import { uploadLetterDocument } from "@/lib/api/letter-document";
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
      <BuatSuratFormInner
        key={formKey}
        onNew={() => {
          if (typeof window !== "undefined") {
            window.history.replaceState(null, "", "/surat/baru");
          }
          setFormKey((k) => k + 1);
        }}
      />
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
  const [serverVersionNumber, setServerVersionNumber] = useState<number>(0);
  const editorHandleRef = useRef<LetterEditorHandle | null>(null);

  const [attachments, setAttachments] = useState<LetterAttachment[]>([]);
  const [savedAttachmentsSignature, setSavedAttachmentsSignature] =
    useState<string>("[]");
  const [dialog, setDialog] = useState<DialogType>(null);
  const [loadingDraft, setLoadingDraft] = useState(Boolean(draftQueryId));
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitFailed, setSubmitFailed] = useState(false);
  const [submitErrorMessage, setSubmitErrorMessage] = useState<string | null>(
    null
  );

  const [isSavingDraft, setIsSavingDraft] = useState(false);
  const [saveFailed, setSaveFailed] = useState(false);
  const [lastSaved, setLastSaved] = useState<Date | null>(null);

  const values = useWatch({ control });

  // Refs untuk mencegah race condition pada penyimpanan
  const draftIdRef = useRef<string | null>(draftId);
  const draftEtagRef = useRef<string | null>(draftEtag);
  const creatingDraftPromiseRef = useRef<Promise<string> | null>(null);
  useEffect(() => {
    draftIdRef.current = draftId;
  }, [draftId]);
  useEffect(() => {
    draftEtagRef.current = draftEtag;
  }, [draftEtag]);
  const loadedDraftIdRef = useRef<string | null>(null);

  // Status perubahan yang belum disimpan (dirty state)
  const isDirty =
    methods.formState.isDirty ||
    JSON.stringify(attachments) !== savedAttachmentsSignature;

  // ─── Memuat draf jika ada parameter ?draft=UUID ───
  useEffect(() => {
    if (!draftQueryId) return;
    if (loadedDraftIdRef.current === draftQueryId) return;

    let cancelled = false;

    getDraftLetter(draftQueryId)
      .then(({ detail, etag }) => {
        if (cancelled) return;

        // Jika surat sudah tidak bisa diedit drafter (mis. IN_REVIEW, IN_APPROVAL, SENT, CANCELLED),
        // alihkan ke halaman detail/review surat (/surat/<id>)
        if (!isEditableByDrafter(detail.status, detail.allowedActions ?? [])) {
          router.replace(`/surat/${detail.id}`);
          return;
        }

        setDraftId(detail.id);
        setDraftEtag(etag);
        setServerVersionNumber(detail.currentVersion ?? 0);
        draftIdRef.current = detail.id;
        draftEtagRef.current = etag;
        loadedDraftIdRef.current = detail.id;

        const { formValues, attachments: loadedAttachments } =
          detailToFormValues(detail);

        setAttachments(loadedAttachments);
        setSavedAttachmentsSignature(JSON.stringify(loadedAttachments));

        reset(formValues);
        setLastSaved(new Date());
        setSaveFailed(false);
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          console.error("Gagal memuat draf dari URL:", e);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoadingDraft(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [draftQueryId, reset, router]);

  // ─── Pastikan draf sudah dibuat di server sebelum mengunggah lampiran ke MinIO ───
  async function ensureDraftId(): Promise<string> {
    if (draftIdRef.current) {
      return draftIdRef.current;
    }
    if (creatingDraftPromiseRef.current) {
      return creatingDraftPromiseRef.current;
    }

    const createPromise = saveDraft(
      values,
      draftIdRef.current,
      draftEtagRef.current
    )
      .then((res) => {
        setDraftId(res.id);
        setDraftEtag(res.etag);
        draftIdRef.current = res.id;
        draftEtagRef.current = res.etag;
        loadedDraftIdRef.current = res.id;
        setLastSaved(new Date());
        if (!draftQueryId && res.id) {
          window.history.replaceState(null, "", `/surat/baru?draft=${res.id}`);
        }
        return res.id;
      })
      .finally(() => {
        creatingDraftPromiseRef.current = null;
      });

    creatingDraftPromiseRef.current = createPromise;
    return createPromise;
  }

  // ─── Sinkronisasi ulang state bila terjadi 412 (VERSION_CONFLICT) atau 409 (INVALID_STATE) ───
  function handleDraftConflictOrStateError(err: unknown) {
    if (!(err instanceof DraftSaveError)) return false;

    if (err.status === 409) {
      const targetId = err.latestDetail?.id ?? draftIdRef.current;
      alert(err.message);
      if (targetId) {
        router.replace(`/surat/${targetId}`);
      }
      return true;
    }

    if (err.status === 412 && err.latestDetail) {
      const detail = err.latestDetail;
      if (!isEditableByDrafter(detail.status, detail.allowedActions ?? [])) {
        alert(err.message);
        router.replace(`/surat/${detail.id}`);
        return true;
      }
      const { formValues, attachments: latestAttachments } =
        detailToFormValues(detail);
      setDraftEtag(err.latestEtag ?? null);
      draftEtagRef.current = err.latestEtag ?? null;
      setServerVersionNumber(detail.currentVersion ?? 0);
      setAttachments(latestAttachments);
      setSavedAttachmentsSignature(JSON.stringify(latestAttachments));
      reset(formValues);
      alert(err.message);
      return true;
    }

    return false;
  }

  // ─── Handlers ────────────────────────────────

  async function handleLoadDraft() {
    setLoadingDraft(true);
    try {
      // Ambil draft terakhir milik user dari backend
      const res = await listLetters({
        scope: "mine",
        status: "DRAFT",
        page: 1,
        pageSize: 1,
      });
      if (res.data.length === 0) {
        alert("Belum ada draf tersimpan yang ditemukan di database.");
        return;
      }

      const latestDraftId = res.data[0].id;
      const { detail, etag } = await getDraftLetter(latestDraftId);

      setDraftId(detail.id);
      setDraftEtag(etag);
      setServerVersionNumber(detail.currentVersion ?? 0);
      draftIdRef.current = detail.id;
      draftEtagRef.current = etag;
      loadedDraftIdRef.current = detail.id;

      const { formValues, attachments: loadedAttachments } =
        detailToFormValues(detail);

      setAttachments(loadedAttachments);
      setSavedAttachmentsSignature(JSON.stringify(loadedAttachments));

      reset(formValues);
      setLastSaved(new Date());
      setSaveFailed(false);
      window.history.replaceState(null, "", `/surat/baru?draft=${detail.id}`);
      await trigger();
    } catch (err: unknown) {
      console.error("Gagal memuat draf terakhir:", err);
      alert("Gagal memuat draf terakhir.");
    } finally {
      setLoadingDraft(false);
    }
  }

  async function handleSimpan() {
    setIsSavingDraft(true);
    setSaveFailed(false);
    try {
      const res = await saveDraft(
        values,
        draftIdRef.current,
        draftEtagRef.current
      );
      let latestEtag = res.etag;
      if (editorHandleRef.current) {
        const exported = await editorHandleRef.current.exportDocxFile();
        if (exported) {
          const uploaded = await uploadLetterDocument(
            res.id,
            exported.file,
            exported.fileName
          );
          if (uploaded.etag) latestEtag = uploaded.etag;
          setServerVersionNumber(uploaded.version.versionNumber);
        }
      }
      setDraftId(res.id);
      setDraftEtag(latestEtag);
      draftIdRef.current = res.id;
      draftEtagRef.current = latestEtag;
      loadedDraftIdRef.current = res.id;
      setLastSaved(new Date());
      setSavedAttachmentsSignature(JSON.stringify(attachments));
      reset(values as BuatSuratFormValues);
      if (!draftQueryId && res.id) {
        window.history.replaceState(null, "", `/surat/baru?draft=${res.id}`);
      }
      setDialog("draf_sukses");
    } catch (err: unknown) {
      console.error("Gagal simpan draf:", err);
      setSaveFailed(true);
      if (!handleDraftConflictOrStateError(err)) {
        alert(err instanceof Error ? err.message : "Gagal menyimpan draf.");
      }
    } finally {
      setIsSavingDraft(false);
    }
  }

  const handleDirectSaveDraft = handleSimpan;

  async function handleSubmit() {
    setSubmitFailed(false);
    setSubmitErrorMessage(null);
    setIsSubmitting(true);

    try {
      // 1. Simpan perubahan draf & unggah versi terbaru naskah .docx ke server
      const saved = await saveDraft(
        values,
        draftIdRef.current,
        draftEtagRef.current
      );
      let latestEtag = saved.etag;
      if (editorHandleRef.current) {
        const exported = await editorHandleRef.current.exportDocxFile();
        if (exported) {
          const uploaded = await uploadLetterDocument(
            saved.id,
            exported.file,
            exported.fileName
          );
          if (uploaded.etag) latestEtag = uploaded.etag;
          setServerVersionNumber(uploaded.version.versionNumber);
        }
      }
      setDraftId(saved.id);
      setDraftEtag(latestEtag);
      draftIdRef.current = saved.id;
      draftEtagRef.current = latestEtag;
      loadedDraftIdRef.current = saved.id;
      setLastSaved(new Date());
      setSavedAttachmentsSignature(JSON.stringify(attachments));
      reset(values as BuatSuratFormValues);

      // 2. Eksekusi pengiriman surat ke reviewer pertama (POST /api/v1/letters/{id}/submit)
      await submitLetter(saved.id);

      // 3. Tampilkan modal dialog sukses
      setDialog("sukses");
    } catch (err: unknown) {
      console.error("Gagal mengirim surat:", err);
      setSubmitFailed(true);
      if (!handleDraftConflictOrStateError(err)) {
        setSubmitErrorMessage(
          err instanceof Error
            ? err.message
            : "Gagal mengirim surat. Periksa kelengkapan surat dan coba lagi."
        );
      }
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
          isSaving={isSavingDraft}
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
            <LampiranSection
              value={attachments}
              onChange={(next) => {
                setAttachments(next);
                setSavedAttachmentsSignature(JSON.stringify(next));
              }}
              letterId={draftId}
              ensureLetterId={ensureDraftId}
              onEtagChange={(newEtag) => {
                setDraftEtag(newEtag);
                draftEtagRef.current = newEtag;
              }}
            />
          </SectionCard>

          <SectionCard number={3} title="Isi Surat & Tata Naskah Digital">
            <LetterEditorSection
              letterId={draftId}
              serverVersionNumber={serverVersionNumber}
              editorHandleRef={editorHandleRef}
              onQuickSaveDraft={handleDirectSaveDraft}
            />
          </SectionCard>
        </div>

        <ActionBar
          onBatal={handleBatal}
          onSimpan={handleDirectSaveDraft}
          isSaving={isSavingDraft}
        />

        <ConfirmDialogs
          dialog={dialog}
          onClose={() => setDialog(null)}
          isSaving={isSavingDraft}
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
