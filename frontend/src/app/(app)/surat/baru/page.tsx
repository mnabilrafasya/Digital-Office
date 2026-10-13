"use client";

// ─────────────────────────────────────────────
// app/(app)/surat/baru/page.tsx
// FE-03: Halaman Buat Surat Baru
// Section 1: Detail Surat
// Section 2: Lampiran
// Section 3: EditorPlaceholder
// Action bar sticky + Dialog konfirmasi + Autosave
// ─────────────────────────────────────────────

import { useCallback, useEffect, useRef, useState } from "react";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Save,
  Send,
  X,
  CheckCircle2,
  ChevronRight,
  Info,
  Clock,
  RotateCcw,
  Search,
  FileDown,
} from "lucide-react";

import { buatSuratSchema, type BuatSuratFormValues } from "@/lib/schemas";
import {
  LETTER_TYPES,
  LETTER_TEMPLATES,
  CLASSIFICATIONS,
  SIFAT_HELP,
} from "@/lib/mock-data";
import type { LetterAttachment, StepEntry } from "@/lib/types";
import { cn, formatTimeWIB } from "@/lib/utils";

import { StepPicker } from "@/components/surat/step-picker";
import { UnitPicker } from "@/components/surat/unit-picker";
import { LampiranSection } from "@/components/surat/lampiran-section";
import { EditorPlaceholder } from "@/components/surat/editor-placeholder";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

// ─── Types lokal ──────────────────────────────

type DialogType = "simpan" | "submit" | "batal" | null;

// ─── Komponen Field Label ─────────────────────

function FieldLabel({
  label,
  required,
  htmlFor,
}: {
  label: string;
  required?: boolean;
  htmlFor?: string;
}) {
  return (
    <label
      htmlFor={htmlFor}
      className="block text-sm font-semibold text-text mb-1.5"
    >
      {label}
      {required && <span className="text-status-danger-text ml-0.5">*</span>}
    </label>
  );
}

// ─── Section Card ─────────────────────────────

function SectionCard({
  number,
  title,
  children,
}: {
  number: number;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="bg-surface rounded-2xl border border-border shadow-sm overflow-hidden">
      {/* Header section */}
      <div className="flex items-center gap-3 px-6 py-4 border-b border-border bg-panel/60">
        <div className="w-7 h-7 rounded-full bg-primary flex items-center justify-center shrink-0">
          <span className="text-white text-xs font-bold">{number}</span>
        </div>
        <h2 className="text-base font-bold text-text">{title}</h2>
      </div>
      <div className="px-6 py-5">{children}</div>
    </div>
  );
}

// ─── Komponen utama ───────────────────────────

export default function BuatSuratPage() {
  const [attachments, setAttachments] = useState<LetterAttachment[]>([]);
  const [dialog, setDialog] = useState<DialogType>(null);
  const [isDirty, setIsDirty] = useState(false);
  const [lastSaved, setLastSaved] = useState<Date | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Kombobox klasifikasi
  const [classQuery, setClassQuery] = useState("");
  const [classOpen, setClassOpen] = useState(false);
  const classRef = useRef<HTMLDivElement>(null);

  const {
    register,
    control,
    watch,
    handleSubmit,
    setValue,
    getValues,
    formState: { errors, isValid },
  } = useForm<BuatSuratFormValues>({
    resolver: zodResolver(buatSuratSchema),
    mode: "onChange",
    defaultValues: {
      letterTypeId: "lt-01",
      sifat: "BIASA",
      reviewers: [],
      approvers: [],
      toUnits: [],
      ccUnits: [],
      specialNote: "",
      summary: "",
    },
  });

  const watchedValues = watch();
  const titleValue = watchedValues.title ?? "";
  const sifatValue = watchedValues.sifat ?? "BIASA";
  const letterTypeId = watchedValues.letterTypeId ?? "";
  const classificationId = watchedValues.classificationId ?? "";

  // Template filter by letter type
  const filteredTemplates = LETTER_TEMPLATES.filter(
    (t) => t.letterTypeId === letterTypeId,
  );

  // Klasifikasi filter
  const filteredClass = CLASSIFICATIONS.filter(
    (c) =>
      classQuery === "" ||
      c.code.toLowerCase().includes(classQuery.toLowerCase()) ||
      c.name.toLowerCase().includes(classQuery.toLowerCase()),
  );

  const selectedClass = CLASSIFICATIONS.find((c) => c.id === classificationId);

  // Close klasifikasi dropdown on outside click
  useEffect(() => {
    function handler(e: MouseEvent) {
      if (classRef.current && !classRef.current.contains(e.target as Node)) {
        setClassOpen(false);
      }
    }
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  // Autosave debounce (1.5 detik)
  const autosaveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    if (!isDirty) return;
    if (autosaveTimer.current) clearTimeout(autosaveTimer.current);
    autosaveTimer.current = setTimeout(() => {
      // Simulasi autosave
      setLastSaved(new Date());
      setIsDirty(false);
    }, 1500);
    return () => {
      if (autosaveTimer.current) clearTimeout(autosaveTimer.current);
    };
  }, [watchedValues, isDirty]);

  // Mark dirty on any change
  useEffect(() => {
    const sub = watch(() => setIsDirty(true));
    return () => sub.unsubscribe();
  }, [watch]);

  // ─── Handlers ────────────────────────────────

  function handleLoadDraft() {
    // Mock: isi form dengan draf fiktif
    setValue("letterTypeId", "lt-01");
    setValue("templateId", "tmpl-01");
    setValue("classificationId", "cls-002");
    setClassQuery("");
    setValue("kecepatan", "SEGERA");
    setValue("sifat", "BIASA");
    setValue("title", "Pembaruan Sistem Informasi Kepegawaian Triwulan IV 2026");
    setValue("reviewers", [
      {
        id: "draft-r1",
        employeeId: "emp-002",
        employeeName: "Budi Santoso",
        jabatan: "Koordinator Pengembangan Sistem",
        unitName: "Sub-Direktorat Pengembangan Sistem",
      },
    ]);
    setValue("approvers", [
      {
        id: "draft-a1",
        employeeId: "emp-004",
        employeeName: "Darmawan Hidayat",
        jabatan: "Manajer Teknologi Informasi",
        unitName: "Direktori Teknologi Informasi",
      },
    ]);
    setValue("toUnits", [
      { id: "unit-sdm", name: "Direktori Sumber Daya Manusia" },
    ]);
    setLastSaved(new Date(Date.now() - 300_000)); // 5 menit lalu
  }

  function doSaveDraft() {
    setIsSaving(true);
    setTimeout(() => {
      setLastSaved(new Date());
      setIsDirty(false);
      setIsSaving(false);
      setDialog(null);
    }, 800);
  }

  function doSubmit() {
    setIsSubmitting(true);
    setTimeout(() => {
      setIsSubmitting(false);
      setDialog(null);
      // TODO: redirect ke daftar surat atau detail
      alert("Surat berhasil dikirim ke Reviewer Tahap 1. Status: Menunggu Review.");
    }, 1000);
  }

  function handleBatal() {
    if (isDirty) {
      setDialog("batal");
    } else {
      window.history.back();
    }
  }

  // ─── Reviewer / Approver IDs untuk warning ───

  const reviewerIds = (watchedValues.reviewers ?? []).map((r: StepEntry) => r.employeeId);
  const approverIds = (watchedValues.approvers ?? []).map((a: StepEntry) => a.employeeId);
  const toUnitIds = (watchedValues.toUnits ?? []).map((u: { id: string }) => u.id);

  // ─── Render ──────────────────────────────────

  return (
    <TooltipProvider>
      <div className="min-h-screen pb-28">

        {/* ── Breadcrumb & Header halaman ──────── */}
        <div className="mb-6">
          {/* Breadcrumb */}
          <nav className="flex items-center gap-1 text-xs text-text-muted mb-2">
            <span>Beranda</span>
            <ChevronRight size={12} />
            <span>Penciptaan Surat</span>
            <ChevronRight size={12} />
            <span className="text-text font-medium">Buat Surat Baru</span>
          </nav>

          <div className="flex items-start justify-between gap-4">
            <div>
              <h1 className="text-2xl font-bold text-text">Buat Surat Baru</h1>
              <p className="text-text-muted text-sm mt-0.5">
                Isi detail surat, pilih reviewer & approver, lalu lampirkan berkas pendukung.
              </p>
            </div>

            {/* Autosave indicator + Muat Draf Terakhir */}
            <div className="flex items-center gap-3 shrink-0 mt-1">
              {/* Indikator simpan */}
              <div className="flex items-center gap-1.5 text-xs text-text-muted">
                {isSaving ? (
                  <>
                    <Clock size={12} className="animate-pulse text-status-warn-text" />
                    <span className="text-status-warn-text">Menyimpan...</span>
                  </>
                ) : lastSaved ? (
                  <>
                    <span className="w-2 h-2 rounded-full bg-green-500 shrink-0" />
                    <span>Tersimpan Otomatis {formatTimeWIB(lastSaved)}</span>
                  </>
                ) : isDirty ? (
                  <>
                    <span className="w-2 h-2 rounded-full bg-status-warn-text animate-pulse shrink-0" />
                    <span className="text-status-warn-text">Belum tersimpan</span>
                  </>
                ) : null}
              </div>

              {/* Muat Draf */}
              <button
                type="button"
                onClick={handleLoadDraft}
                className="flex items-center gap-1.5 text-xs text-text-muted border border-border px-3 py-1.5 rounded-lg hover:bg-panel hover:text-text transition-colors"
              >
                <RotateCcw size={12} />
                Muat Draf Terakhir
              </button>
            </div>
          </div>
        </div>

        {/* ── Sections ────────────────────────── */}
        <div className="space-y-5">

          {/* ═══════════════════════════════════
              SECTION 1: DETAIL SURAT
          ═══════════════════════════════════ */}
          <SectionCard number={1} title="Detail Surat">
            <form id="buat-surat-form" onSubmit={handleSubmit(() => setDialog("submit"))}>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-5">

                {/* Jenis Surat */}
                <div>
                  <FieldLabel label="Jenis Surat" required htmlFor="letterTypeId" />
                  <select
                    id="letterTypeId"
                    {...register("letterTypeId")}
                    className={cn(
                      "w-full px-3 py-2.5 text-sm border rounded-lg bg-surface outline-none transition-colors",
                      "focus:border-primary focus:ring-2 focus:ring-primary/15",
                      errors.letterTypeId ? "border-status-danger-text" : "border-border-input",
                    )}
                  >
                    {LETTER_TYPES.map((lt) => (
                      <option key={lt.id} value={lt.id}>{lt.name}</option>
                    ))}
                  </select>
                  {errors.letterTypeId && (
                    <p className="text-status-danger-text text-xs mt-1">{errors.letterTypeId.message}</p>
                  )}
                </div>

                {/* Template Surat */}
                <div>
                  <FieldLabel label="Template Surat" htmlFor="templateId" />
                  <select
                    id="templateId"
                    {...register("templateId")}
                    className="w-full px-3 py-2.5 text-sm border border-border-input rounded-lg bg-surface outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 transition-colors"
                  >
                    <option value="">— Pilih template (opsional) —</option>
                    {filteredTemplates.map((t) => (
                      <option key={t.id} value={t.id}>{t.name}</option>
                    ))}
                  </select>
                </div>

                {/* Klasifikasi Arsip — Combobox */}
                <div ref={classRef} className="md:col-span-2">
                  <FieldLabel label="Klasifikasi Arsip" required htmlFor="classificationSearch" />
                  <div className="relative">
                    <div
                      className={cn(
                        "w-full flex items-center gap-2 px-3 py-2.5 text-sm border rounded-lg bg-surface cursor-pointer transition-colors",
                        classOpen
                          ? "border-primary ring-2 ring-primary/15"
                          : errors.classificationId
                          ? "border-status-danger-text"
                          : "border-border-input",
                      )}
                      onClick={() => setClassOpen((v) => !v)}
                    >
                      <Search size={14} className="text-text-muted shrink-0" />
                      {selectedClass ? (
                        <span className="flex-1 text-text">
                          <span className="font-mono text-text-muted text-xs mr-2">{selectedClass.code}</span>
                          {selectedClass.name}
                        </span>
                      ) : (
                        <span className="flex-1 text-text-muted">Cari berdasarkan kode atau nama klasifikasi...</span>
                      )}
                      {selectedClass && (
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            setValue("classificationId", "");
                            setClassQuery("");
                          }}
                          className="text-text-muted hover:text-text"
                        >
                          <X size={13} />
                        </button>
                      )}
                    </div>

                    {classOpen && (
                      <div className="absolute top-full left-0 right-0 mt-1 z-20 bg-surface border border-border rounded-lg shadow-lg overflow-hidden">
                        <div className="px-3 py-2 border-b border-border">
                          <input
                            id="classificationSearch"
                            autoFocus
                            type="text"
                            placeholder="Ketik kode atau nama..."
                            value={classQuery}
                            onChange={(e) => setClassQuery(e.target.value)}
                            className="w-full text-sm outline-none bg-transparent text-text placeholder-text-muted"
                          />
                        </div>
                        <ul className="max-h-52 overflow-y-auto divide-y divide-border">
                          {filteredClass.length === 0 ? (
                            <li className="px-4 py-6 text-center text-text-muted text-sm">Tidak ditemukan.</li>
                          ) : (
                            filteredClass.map((c) => (
                              <li key={c.id}>
                                <button
                                  type="button"
                                  onClick={() => {
                                    setValue("classificationId", c.id, { shouldValidate: true });
                                    setClassOpen(false);
                                    setClassQuery("");
                                  }}
                                  className={cn(
                                    "w-full text-left px-4 py-2.5 hover:bg-panel transition-colors",
                                    classificationId === c.id && "bg-primary/5",
                                  )}
                                >
                                  <span className="font-mono text-text-muted text-xs mr-2">{c.code}</span>
                                  <span className="text-sm text-text">{c.name}</span>
                                </button>
                              </li>
                            ))
                          )}
                        </ul>
                      </div>
                    )}
                  </div>
                  {errors.classificationId && (
                    <p className="text-status-danger-text text-xs mt-1">{errors.classificationId.message}</p>
                  )}
                </div>

                {/* Kecepatan Tanggapan */}
                <div>
                  <FieldLabel label="Kecepatan Tanggapan" required htmlFor="kecepatan" />
                  <select
                    id="kecepatan"
                    {...register("kecepatan")}
                    className={cn(
                      "w-full px-3 py-2.5 text-sm border rounded-lg bg-surface outline-none transition-colors",
                      "focus:border-primary focus:ring-2 focus:ring-primary/15",
                      errors.kecepatan ? "border-status-danger-text" : "border-border-input",
                    )}
                  >
                    <option value="">— Pilih kecepatan —</option>
                    <option value="SEGERA">Segera (&lt; 24 Jam Kerja)</option>
                    <option value="SANGAT_SEGERA">Sangat Segera</option>
                  </select>
                  {errors.kecepatan && (
                    <p className="text-status-danger-text text-xs mt-1">{errors.kecepatan.message}</p>
                  )}
                </div>

                {/* Sifat Kerahasiaan */}
                <div>
                  <FieldLabel label="Sifat Kerahasiaan" required />
                  <Controller
                    name="sifat"
                    control={control}
                    render={({ field }) => (
                      <div className="space-y-2">
                        {/* Radio buttons */}
                        <div className="flex gap-3">
                          {(["BIASA", "TERBATAS", "RAHASIA"] as const).map((val) => (
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
                        {/* Teks bantu */}
                        {sifatValue && SIFAT_HELP[sifatValue] && (
                          <p className="text-xs text-text-muted flex items-start gap-1.5">
                            <Info size={12} className="mt-0.5 shrink-0" />
                            {SIFAT_HELP[sifatValue]}
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
                      className={cn(
                        "w-full px-3 py-2.5 pr-24 text-sm border rounded-lg bg-surface outline-none transition-colors",
                        "focus:border-primary focus:ring-2 focus:ring-primary/15",
                        errors.title ? "border-status-danger-text" : "border-border-input",
                      )}
                    />
                    {/* Penghitung + centang */}
                    <div className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5">
                      <span
                        className={cn(
                          "text-xs",
                          titleValue.length > 230 ? "text-status-danger-text" : "text-text-muted",
                        )}
                      >
                        {titleValue.length} / 250
                      </span>
                      {titleValue.length > 0 && !errors.title && (
                        <CheckCircle2 size={14} className="text-green-500" />
                      )}
                    </div>
                  </div>
                  {errors.title && (
                    <p className="text-status-danger-text text-xs mt-1">{errors.title.message}</p>
                  )}
                </div>

                {/* Reviewer */}
                <div className="md:col-span-2">
                  <FieldLabel label="Reviewer" required />
                  <Controller
                    name="reviewers"
                    control={control}
                    render={({ field }) => (
                      <StepPicker
                        role="REVIEWER"
                        value={field.value}
                        onChange={field.onChange}
                        error={errors.reviewers?.message}
                        otherRoleIds={approverIds}
                      />
                    )}
                  />
                </div>

                {/* Approver */}
                <div className="md:col-span-2">
                  <FieldLabel label="Approver" required />
                  <Controller
                    name="approvers"
                    control={control}
                    render={({ field }) => (
                      <StepPicker
                        role="APPROVER"
                        value={field.value}
                        onChange={field.onChange}
                        error={errors.approvers?.message}
                        otherRoleIds={reviewerIds}
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
                        excludeIds={(watchedValues.ccUnits ?? []).map((u: { id: string }) => u.id)}
                      />
                    )}
                  />
                </div>

                {/* Tembusan / CC */}
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
                        excludeIds={toUnitIds}
                      />
                    )}
                  />
                </div>

                {/* Keterangan Khusus */}
                <div className="md:col-span-2">
                  <FieldLabel label="Keterangan Khusus" htmlFor="specialNote" />
                  <input
                    id="specialNote"
                    type="text"
                    placeholder="Keterangan tambahan yang perlu diperhatikan (opsional)..."
                    {...register("specialNote")}
                    className="w-full px-3 py-2.5 text-sm border border-border-input rounded-lg bg-surface outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 transition-colors"
                  />
                </div>

                {/* Isi Ringkas Arsip */}
                <div className="md:col-span-2">
                  <FieldLabel label="Isi Ringkas Arsip" htmlFor="summary" />
                  <textarea
                    id="summary"
                    rows={3}
                    placeholder="Ringkasan isi surat untuk keperluan arsip (opsional)..."
                    {...register("summary")}
                    className="w-full px-3 py-2.5 text-sm border border-border-input rounded-lg bg-surface outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 transition-colors resize-none"
                  />
                </div>

              </div>
            </form>
          </SectionCard>

          {/* ═══════════════════════════════════
              SECTION 2: LAMPIRAN
          ═══════════════════════════════════ */}
          <SectionCard number={2} title="Lampiran">
            <LampiranSection
              value={attachments}
              onChange={setAttachments}
            />
          </SectionCard>

          {/* ═══════════════════════════════════
              SECTION 3: ISI SURAT (Placeholder)
          ═══════════════════════════════════ */}
          <SectionCard number={3} title="Isi Surat & Tata Naskah Digital">
            <EditorPlaceholder />
          </SectionCard>

        </div>

        {/* ═══════════════════════════════════
            ACTION BAR STICKY
        ═══════════════════════════════════ */}
        <div className="fixed bottom-0 left-56 right-0 bg-surface border-t border-border shadow-[0_-4px_24px_rgba(0,0,0,0.06)] z-20">
          <div className="max-w-6xl mx-auto px-6 py-3 flex items-center justify-between gap-3">
            {/* Info nomor surat */}
            <div className="text-xs text-text-muted flex items-center gap-1.5">
              <Info size={12} />
              <span>Nomor & tanggal surat dibuat otomatis saat disetujui.</span>
            </div>

            {/* Tombol */}
            <div className="flex items-center gap-2 shrink-0">
              {/* Batal */}
              <button
                type="button"
                id="action-batal"
                onClick={handleBatal}
                className="flex items-center gap-1.5 px-4 py-2.5 text-sm text-text-muted border border-border rounded-lg hover:bg-panel hover:text-text transition-colors"
              >
                <X size={14} />
                Batal
              </button>

              {/* Simpan Draft */}
              <button
                type="button"
                id="action-simpan-draft"
                onClick={() => setDialog("simpan")}
                className="flex items-center gap-1.5 px-4 py-2.5 text-sm text-primary border border-primary/30 rounded-lg hover:bg-primary/5 transition-colors font-medium"
              >
                <Save size={14} />
                Simpan Draft
              </button>

              {/* Pratinjau PDF (nonaktif) */}
              <Tooltip>
                <TooltipTrigger>
                  <button
                    type="button"
                    id="action-pratinjau-pdf"
                    disabled
                    className="flex items-center gap-1.5 px-4 py-2.5 text-sm text-text-muted border border-border rounded-lg opacity-50 cursor-not-allowed"
                  >
                    <FileDown size={14} />
                    Pratinjau Cetak PDF
                  </button>
                </TooltipTrigger>
                <TooltipContent side="top">
                  <p>Tersedia setelah editor surat siap</p>
                </TooltipContent>
              </Tooltip>

              {/* Submit */}
              <button
                type="submit"
                form="buat-surat-form"
                id="action-submit"
                disabled={!isValid}
                className={cn(
                  "flex items-center gap-1.5 px-5 py-2.5 text-sm rounded-lg font-semibold transition-colors",
                  isValid
                    ? "bg-primary text-white hover:bg-primary-dark"
                    : "bg-border text-text-muted cursor-not-allowed",
                )}
              >
                <Send size={14} />
                Submit
              </button>
            </div>
          </div>
        </div>

        {/* ═══════════════════════════════════
            DIALOGS
        ═══════════════════════════════════ */}

        {/* Dialog Simpan Draft */}
        <Dialog open={dialog === "simpan"} onOpenChange={(o) => !o && setDialog(null)}>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle>Simpan Draft Surat?</DialogTitle>
              <DialogDescription>
                Draft akan disimpan sebagai versi baru. Versi sebelumnya tidak akan ditimpa.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <button
                type="button"
                onClick={() => setDialog(null)}
                className="px-4 py-2 text-sm border border-border rounded-lg text-text hover:bg-panel transition-colors"
              >
                Batal
              </button>
              <button
                type="button"
                onClick={doSaveDraft}
                disabled={isSaving}
                className="px-4 py-2 text-sm bg-primary text-white rounded-lg hover:bg-primary-dark transition-colors font-medium disabled:opacity-60"
              >
                {isSaving ? "Menyimpan..." : "Simpan"}
              </button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Dialog Submit */}
        <Dialog open={dialog === "submit"} onOpenChange={(o) => !o && setDialog(null)}>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle>Kirim Surat ke Reviewer?</DialogTitle>
              <DialogDescription>
                Surat akan dikirim ke <strong>Reviewer Tahap 1</strong> untuk ditinjau.
                Status surat akan berubah menjadi &ldquo;Menunggu Review&rdquo;.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <button
                type="button"
                onClick={() => setDialog(null)}
                className="px-4 py-2 text-sm border border-border rounded-lg text-text hover:bg-panel transition-colors"
              >
                Batal
              </button>
              <button
                type="button"
                onClick={doSubmit}
                disabled={isSubmitting}
                className="px-4 py-2 text-sm bg-primary text-white rounded-lg hover:bg-primary-dark transition-colors font-medium disabled:opacity-60"
              >
                {isSubmitting ? "Mengirim..." : "Kirim ke Reviewer"}
              </button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Dialog Batal / Perubahan Belum Tersimpan */}
        <Dialog open={dialog === "batal"} onOpenChange={(o) => !o && setDialog(null)}>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle>Tinggalkan Halaman?</DialogTitle>
              <DialogDescription>
                Ada perubahan yang belum tersimpan. Jika Anda pergi sekarang, perubahan ini akan hilang.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <button
                type="button"
                onClick={() => setDialog(null)}
                className="px-4 py-2 text-sm border border-border rounded-lg text-text hover:bg-panel transition-colors"
              >
                Tetap di Halaman
              </button>
              <button
                type="button"
                onClick={() => window.history.back()}
                className="px-4 py-2 text-sm bg-status-danger-text text-white rounded-lg hover:opacity-90 transition-colors font-medium"
              >
                Tinggalkan
              </button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

      </div>
    </TooltipProvider>
  );
}
