// ─────────────────────────────────────────────
// lib/mock-data.ts
// Semua data FIKTIF — tidak mengandung nama/NIK asli
// Nama organisasi: PT Pupuk Sriwidjaja Palembang
// Unit fokus: Direktori Teknologi Informasi
// Tahun konsisten: 2026
// ─────────────────────────────────────────────

import type {
  Employee,
  OrgUnit,
  LetterType,
  LetterTemplate,
  Classification,
} from "./types";

// ─── Organisasi ─────────────────────────────

export const COMPANY_NAME = "PT Pupuk Sriwidjaja Palembang";
export const COMPANY_ID = "pupuk-sriwidjaja";

// ─── Unit Organisasi ─────────────────────────

export const ORG_UNITS: OrgUnit[] = [
  { id: "unit-ti",          name: "Direktori Teknologi Informasi",             companyId: COMPANY_ID },
  { id: "unit-ti-dev",      name: "Sub-Direktorat Pengembangan Sistem",        companyId: COMPANY_ID },
  { id: "unit-ti-infra",    name: "Sub-Direktorat Infrastruktur & Jaringan",   companyId: COMPANY_ID },
  { id: "unit-ti-sec",      name: "Sub-Direktorat Keamanan Informasi",         companyId: COMPANY_ID },
  { id: "unit-ti-gov",      name: "Sub-Direktorat Tata Kelola TI",             companyId: COMPANY_ID },
  { id: "unit-sdm",         name: "Direktori Sumber Daya Manusia",             companyId: COMPANY_ID },
  { id: "unit-keuangan",    name: "Direktori Keuangan & Akuntansi",            companyId: COMPANY_ID },
  { id: "unit-operasional", name: "Direktori Operasional & Produksi",          companyId: COMPANY_ID },
  { id: "unit-hukum",       name: "Direktori Hukum & Kepatuhan",               companyId: COMPANY_ID },
  { id: "unit-pengadaan",   name: "Direktori Pengadaan & Logistik",            companyId: COMPANY_ID },
  { id: "unit-komersial",   name: "Direktori Komersial & Pemasaran",           companyId: COMPANY_ID },
  { id: "unit-direksi",     name: "Sekretariat Direksi",                       companyId: COMPANY_ID },
];

// ─── Pegawai (drafter, reviewer, approver harus orang berbeda) ───

export const EMPLOYEES: Employee[] = [
  // Drafter (user login)
  {
    id: "emp-001",
    nik: "TI-2024-001",
    name: "Andi Kurniawan",
    jabatan: "Staf Pengembang Sistem",
    unitId: "unit-ti-dev",
    unitName: "Sub-Direktorat Pengembangan Sistem",
  },
  // Reviewer 1
  {
    id: "emp-002",
    nik: "TI-2022-015",
    name: "Budi Santoso",
    jabatan: "Koordinator Pengembangan Sistem",
    unitId: "unit-ti-dev",
    unitName: "Sub-Direktorat Pengembangan Sistem",
  },
  // Reviewer 2
  {
    id: "emp-003",
    nik: "TI-2021-008",
    name: "Citra Dewi Rahayu",
    jabatan: "Analis Tata Kelola TI",
    unitId: "unit-ti-gov",
    unitName: "Sub-Direktorat Tata Kelola TI",
  },
  // Approver
  {
    id: "emp-004",
    nik: "TI-2018-002",
    name: "Darmawan Hidayat",
    jabatan: "Manajer Teknologi Informasi",
    unitId: "unit-ti",
    unitName: "Direktori Teknologi Informasi",
  },
  // Pegawai lain untuk picker
  {
    id: "emp-005",
    nik: "TI-2023-011",
    name: "Eka Fitrianti",
    jabatan: "Staf Infrastruktur Jaringan",
    unitId: "unit-ti-infra",
    unitName: "Sub-Direktorat Infrastruktur & Jaringan",
  },
  {
    id: "emp-006",
    nik: "TI-2020-007",
    name: "Fajar Nugroho",
    jabatan: "Analis Keamanan Informasi",
    unitId: "unit-ti-sec",
    unitName: "Sub-Direktorat Keamanan Informasi",
  },
  {
    id: "emp-007",
    nik: "TI-2019-005",
    name: "Gita Purnama",
    jabatan: "Koordinator Infrastruktur",
    unitId: "unit-ti-infra",
    unitName: "Sub-Direktorat Infrastruktur & Jaringan",
  },
  {
    id: "emp-008",
    nik: "TI-2022-019",
    name: "Hendra Wijaya",
    jabatan: "Staf Tata Kelola TI",
    unitId: "unit-ti-gov",
    unitName: "Sub-Direktorat Tata Kelola TI",
  },
  {
    id: "emp-009",
    nik: "TI-2017-003",
    name: "Irma Sari Utami",
    jabatan: "Asisten Manajer Keamanan Informasi",
    unitId: "unit-ti-sec",
    unitName: "Sub-Direktorat Keamanan Informasi",
  },
  {
    id: "emp-010",
    nik: "TI-2023-022",
    name: "Joko Prasetyo",
    jabatan: "Staf Pengembang Sistem",
    unitId: "unit-ti-dev",
    unitName: "Sub-Direktorat Pengembangan Sistem",
  },
];

// User login = drafter
export const CURRENT_USER = EMPLOYEES[0]; // Andi Kurniawan

// ─── Jenis Surat ─────────────────────────────

export const LETTER_TYPES: LetterType[] = [
  { id: "lt-01", name: "Surat Antar Unit Kerja" },
  { id: "lt-02", name: "Surat Keluar Korporat" },
  { id: "lt-03", name: "Nota Dinas" },
  { id: "lt-04", name: "Surat Keputusan" },
  { id: "lt-05", name: "Surat Pemberitahuan" },
  { id: "lt-06", name: "Surat Permohonan" },
  { id: "lt-07", name: "Surat Undangan" },
];

// ─── Template Surat ──────────────────────────

export const LETTER_TEMPLATES: LetterTemplate[] = [
  { id: "tmpl-01", name: "Template Standar Korporat",            letterTypeId: "lt-01" },
  { id: "tmpl-02", name: "Template Nota Dinas Sederhana",        letterTypeId: "lt-03" },
  { id: "tmpl-03", name: "Template Surat Keputusan Direktur",    letterTypeId: "lt-04" },
  { id: "tmpl-04", name: "Template Pemberitahuan Umum",          letterTypeId: "lt-05" },
  { id: "tmpl-05", name: "Template Permohonan ke Eksternal",     letterTypeId: "lt-06" },
];

// ─── Klasifikasi Arsip ───────────────────────

export const CLASSIFICATIONS: Classification[] = [
  { id: "cls-001", code: "TI.01", name: "Kebijakan & Tata Kelola Teknologi Informasi" },
  { id: "cls-002", code: "TI.02", name: "Pengembangan & Implementasi Sistem" },
  { id: "cls-003", code: "TI.03", name: "Infrastruktur & Jaringan Komputer" },
  { id: "cls-004", code: "TI.04", name: "Keamanan Sistem Informasi" },
  { id: "cls-005", code: "TI.05", name: "Pengelolaan Data & Basis Data" },
  { id: "cls-006", code: "SDM.01", name: "Rekrutmen & Pengembangan SDM" },
  { id: "cls-007", code: "KU.01", name: "Anggaran & Pelaporan Keuangan" },
  { id: "cls-008", code: "OP.01", name: "Operasional & Produksi" },
  { id: "cls-009", code: "HK.01", name: "Hukum & Kepatuhan Perusahaan" },
  { id: "cls-010", code: "PG.01", name: "Pengadaan Barang & Jasa" },
];

// ─── Sifat helper labels ─────────────────────

export const SIFAT_HELP: Record<string, string> = {
  BIASA:
    "Sifat 'Biasa' dapat diakses oleh seluruh staf pengadministrasi unit terkait.",
  TERBATAS:
    "Sifat 'Terbatas' hanya dapat diakses oleh pejabat dan staf yang berwenang di unit terkait.",
  RAHASIA:
    "Sifat 'Rahasia' hanya dapat diakses oleh pejabat tertentu yang mendapat otorisasi khusus.",
};
