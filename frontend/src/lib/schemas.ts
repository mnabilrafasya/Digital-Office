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

  letterTypeCode: z
    .string()
    .optional()
    .nullable(),

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

  toUnits: z.array(unitEntrySchema),

  ccUnits: z.array(unitEntrySchema).optional().nullable(),

  externalRecipient: z
    .object({
      name: z.string(),
      address: z.string().optional().nullable(),
      attention: z.string().optional().nullable(),
    })
    .optional()
    .nullable(),

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
}).superRefine((data, ctx) => {
  const revIds = (data.reviewers ?? []).map((r) => r.employeeId);
  if (new Set(revIds).size !== revIds.length) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: "Reviewer yang sama tidak boleh dipilih lebih dari sekali.",
      path: ["reviewers"],
    });
  }

  const appIds = (data.approvers ?? []).map((a) => a.employeeId);
  if (new Set(appIds).size !== appIds.length) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: "Approver yang sama tidak boleh dipilih lebih dari sekali.",
      path: ["approvers"],
    });
  }

  const isExternalOutgoing =
    (data.letterTypeCode ?? "").toUpperCase() === "SURAT_EKSTERNAL";

  if (isExternalOutgoing) {
    const extName = (data.externalRecipient?.name ?? "").trim();
    if (!extName) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Nama perusahaan / instansi tujuan eksternal wajib diisi.",
        path: ["externalRecipient", "name"],
      });
    } else if (extName.length > 200) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Nama penerima eksternal maksimal 200 karakter.",
        path: ["externalRecipient", "name"],
      });
    }

    const extAddress = (data.externalRecipient?.address ?? "").trim();
    if (extAddress.length > 500) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Alamat penerima eksternal maksimal 500 karakter.",
        path: ["externalRecipient", "address"],
      });
    }

    const extAttention = (data.externalRecipient?.attention ?? "").trim();
    if (extAttention.length > 200) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Tujuan UP penerima eksternal maksimal 200 karakter.",
        path: ["externalRecipient", "attention"],
      });
    }
  } else {
    if ((data.toUnits ?? []).length < 1) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Minimal 1 tujuan surat wajib dipilih.",
        path: ["toUnits"],
      });
    }
  }
});

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
  letterTypeCode: "",
  kecepatan: "BIASA",
  sifat: "BIASA",
  title: "",
  specialNote: "",
  summary: "",
  reviewers: [],
  approvers: [],
  toUnits: [],
  ccUnits: [],
  externalRecipient: {
    name: "",
    address: "",
    attention: "",
  },
};
