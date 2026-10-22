// ─────────────────────────────────────────────
// lib/schemas.ts
// Zod schema untuk form Buat Surat (react-hook-form)
// Validasi mengikuti PROJECT_CONTEXT bagian 7
// ─────────────────────────────────────────────

import { z } from "zod";

// ─── StepEntry schema ────────────────────────

const stepEntrySchema = z.object({
  id: z.string(),
  employeeId: z.string().min(1, { message: "ID pegawai wajib ada." }),
  employeeName: z.string(),
  jabatan: z.string(),
  unitName: z.string(),
  kodeAlias: z.string().optional().nullable(),
  kode_alias: z.string().optional().nullable(),
});

// ─── UnitEntry schema ────────────────────────

const unitEntrySchema = z.object({
  id: z.string().min(1, { message: "ID unit wajib ada." }),
  name: z.string(),
});

// ─── Form schema utama ───────────────────────

export const buatSuratSchema = z.object({
  // Section 1: Detail Surat
  letterTypeId: z
    .string()
    .min(1, { message: "Jenis surat wajib dipilih." }),

  templateId: z
    .string()
    .optional()
    .nullable(),

  classificationId: z
    .string()
    .min(1, { message: "Klasifikasi arsip wajib dipilih." }),

  kecepatan: z
    .enum(["BIASA", "SEGERA", "SANGAT_SEGERA"], {
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
    .min(1, { message: "Minimal 1 reviewer wajib ditambahkan." })
    .max(10, { message: "Maksimal 10 reviewer diperbolehkan." }),

  approvers: z
    .array(stepEntrySchema)
    .min(1, { message: "Minimal 1 approver (pejabat penandatangan) wajib dipilih." })
    .max(10, { message: "Maksimal 10 approver diperbolehkan." }),

  toUnits: z
    .array(unitEntrySchema)
    .min(1, { message: "Minimal 1 tujuan surat wajib dipilih." }),

  ccUnits: z.array(unitEntrySchema).optional().nullable(),

  specialNote: z
    .string()
    .max(500, { message: "Keterangan khusus maksimal 500 karakter." })
    .optional()
    .nullable(),

  summary: z
    .string()
    .max(1000, { message: "Isi ringkas arsip maksimal 1000 karakter." })
    .optional()
    .nullable(),
})
  .refine(
    (data) => {
      const reviewerEmpIds = new Set(data.reviewers?.map((r) => r.employeeId) ?? []);
      return !(data.approvers ?? []).some((a) => reviewerEmpIds.has(a.employeeId));
    },
    {
      message: "Approver tidak boleh sekaligus menjadi reviewer. Silakan hapus atau ganti salah satunya.",
      path: ["approvers"],
    }
  )
  .refine(
    (data) => {
      const ids = (data.reviewers ?? []).map((r) => r.employeeId);
      return new Set(ids).size === ids.length;
    },
    {
      message: "Reviewer yang sama tidak boleh dipilih lebih dari sekali.",
      path: ["reviewers"],
    }
  );

export type BuatSuratFormValues = z.infer<typeof buatSuratSchema>;

// ─── Form login ──────────────────────────────

export const loginSchema = z.object({
  nik: z.string().trim().min(1, { message: "NIP / NIK wajib diisi." }).max(30, { message: "NIP / NIK maksimal 30 karakter." }),
  password: z.string().min(1, { message: "Kata sandi wajib diisi." }),
  remember: z.boolean(),
});

/** Nilai awal form Buat Surat.
 * letterTypeId dikosongkan: ID dari backend berbeda dengan mock,
 * akan diisi otomatis saat lookup LETTER_TYPE selesai dimuat. */
export const BUAT_SURAT_DEFAULTS: Partial<BuatSuratFormValues> = {
  letterTypeId: "",
  kecepatan: "BIASA",
  sifat: "BIASA",
  title: "",
  specialNote: "",
  summary: "",
  reviewers: [],
  approvers: [],
  toUnits: [],
  ccUnits: [],
};
