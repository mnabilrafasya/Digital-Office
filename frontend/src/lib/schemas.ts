// ─────────────────────────────────────────────
// lib/schemas.ts
// Zod schema untuk form Buat Surat (react-hook-form)
// Validasi mengikuti PROJECT_CONTEXT bagian 7
// ─────────────────────────────────────────────

import { z } from "zod";

// ─── StepEntry schema ────────────────────────

const stepEntrySchema = z.object({
  id: z.string(),
  employeeId: z.string().min(1),
  employeeName: z.string().min(1),
  jabatan: z.string(),
  unitName: z.string(),
});

// ─── UnitEntry schema ────────────────────────

const unitEntrySchema = z.object({
  id: z.string(),
  name: z.string().min(1),
});

// ─── Form schema utama ───────────────────────

export const buatSuratSchema = z.object({
  // Section 1: Detail Surat
  letterTypeId: z
    .string()
    .min(1, { message: "Jenis surat wajib dipilih." }),

  templateId: z
    .string()
    .optional(),

  classificationId: z
    .string()
    .min(1, { message: "Klasifikasi arsip wajib dipilih." }),

  kecepatan: z
    .enum(["SEGERA", "SANGAT_SEGERA"], {
      message: "Kecepatan tanggapan wajib dipilih.",
    }),

  sifat: z.enum(["BIASA", "TERBATAS", "RAHASIA"], {
    message: "Sifat kerahasiaan wajib dipilih.",
  }),

  title: z
    .string()
    .min(1, { message: "Perihal surat tidak boleh kosong." })
    .max(250, { message: "Perihal maksimal 250 karakter." }),

  reviewers: z
    .array(stepEntrySchema)
    .min(1, { message: "Minimal 1 reviewer wajib ditambahkan." }),

  approvers: z
    .array(stepEntrySchema)
    .min(1, { message: "Minimal 1 approver wajib ditambahkan." }),

  toUnits: z
    .array(unitEntrySchema)
    .min(1, { message: "Minimal 1 tujuan surat wajib dipilih." }),

  ccUnits: z.array(unitEntrySchema).optional(),

  specialNote: z
    .string()
    .max(500, { message: "Keterangan khusus maksimal 500 karakter." })
    .optional(),

  summary: z
    .string()
    .max(1000, { message: "Isi ringkas arsip maksimal 1000 karakter." })
    .optional(),
});

export type BuatSuratFormValues = z.infer<typeof buatSuratSchema>;

// ─── Form login ──────────────────────────────

export const loginSchema = z.object({
  nik: z.string().trim().min(1, { message: "NIP / NIK wajib diisi." }).max(30, { message: "NIP / NIK maksimal 30 karakter." }),
  password: z.string().min(1, { message: "Kata sandi wajib diisi." }),
  remember: z.boolean(),
});

/** Nilai awal form Buat Surat */
export const BUAT_SURAT_DEFAULTS: Partial<BuatSuratFormValues> = {
  letterTypeId: "lt-01",
  sifat: "BIASA",
  title: "",
  specialNote: "",
  summary: "",
  reviewers: [],
  approvers: [],
  toUnits: [],
  ccUnits: [],
};
