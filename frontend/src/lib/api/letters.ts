// ─────────────────────────────────────────────
// lib/api/letters.ts
// Mock listLetters — bentuk data meniru backend bagian 13
// Ganti isi fungsi ini saat endpoint /api/v1/letters siap
// ─────────────────────────────────────────────

import type {
  LetterListItem,
  ListLettersParams,
  ListLettersResponse,
  LetterStatus,
} from "@/lib/types";

// ─── Raw mock data (48 surat fiktif) ──────────

const RAW_LETTERS: LetterListItem[] = [
  // ── DRAFT (8) ──────────────────────────────
  {
    id: "srt-001",
    status: "DRAFT",
    letterNumber: null,
    title: "Permintaan Pengadaan Perangkat Server untuk Data Center",
    letterType: { id: "lt-01", code: "SAUT", name: "Surat Antar Unit Kerja" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Pengadaan & Logistik", total: 1 },
    createdAt: "2026-10-18T09:15:00+07:00",
    currentAssignee: null,
    lastReturnNote: null,
    allowedActions: ["EDIT"],
  },
  {
    id: "srt-002",
    status: "DRAFT",
    letterNumber: null,
    title: null,
    letterType: { id: "lt-03", code: "ND", name: "Nota Dinas" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Sekretariat Direksi", total: 1 },
    createdAt: "2026-10-17T14:30:00+07:00",
    currentAssignee: null,
    lastReturnNote: null,
    allowedActions: ["EDIT"],
  },
  {
    id: "srt-003",
    status: "DRAFT",
    letterNumber: null,
    title: "Laporan Kemajuan Proyek Digitalisasi Arsip Q3 2026",
    letterType: { id: "lt-03", code: "ND", name: "Nota Dinas" },
    confidentiality: { id: "cf-2", code: "TERBATAS", name: "Terbatas" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Teknologi Informasi", total: 1 },
    createdAt: "2026-10-16T10:00:00+07:00",
    currentAssignee: null,
    lastReturnNote: null,
    allowedActions: ["EDIT"],
  },
  {
    id: "srt-004",
    status: "DRAFT",
    letterNumber: null,
    title: "Permohonan Izin Akses VPN untuk Tim Pengembang Eksternal",
    letterType: { id: "lt-06", code: "SPER", name: "Surat Permohonan" },
    confidentiality: { id: "cf-2", code: "TERBATAS", name: "Terbatas" },
    urgency: { id: "ug-2", code: "SEGERA", name: "Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Sub-Direktorat Keamanan Informasi", total: 2 },
    createdAt: "2026-10-15T08:45:00+07:00",
    currentAssignee: null,
    lastReturnNote: null,
    allowedActions: ["EDIT"],
  },
  {
    id: "srt-005",
    status: "DRAFT",
    letterNumber: null,
    title: "Undangan Rapat Koordinasi Penerapan ERP Modul Keuangan",
    letterType: { id: "lt-07", code: "SUND", name: "Surat Undangan" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-2", code: "SEGERA", name: "Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Keuangan & Akuntansi", total: 3 },
    createdAt: "2026-10-14T13:20:00+07:00",
    currentAssignee: null,
    lastReturnNote: null,
    allowedActions: ["EDIT"],
  },
  {
    id: "srt-006",
    status: "DRAFT",
    letterNumber: null,
    title: "Pemberitahuan Jadwal Pemeliharaan Sistem Informasi Korporat",
    letterType: { id: "lt-05", code: "SPTB", name: "Surat Pemberitahuan" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-3", code: "SANGAT_SEGERA", name: "Sangat Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Seluruh VP (12)", total: 12 },
    createdAt: "2026-10-13T16:00:00+07:00",
    currentAssignee: null,
    lastReturnNote: null,
    allowedActions: ["EDIT"],
  },
  {
    id: "srt-007",
    status: "DRAFT",
    letterNumber: null,
    title: "Usulan Anggaran Pengembangan Aplikasi Mobile Internal 2027",
    letterType: { id: "lt-01", code: "SAUT", name: "Surat Antar Unit Kerja" },
    confidentiality: { id: "cf-2", code: "TERBATAS", name: "Terbatas" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Keuangan & Akuntansi", total: 1 },
    createdAt: "2026-10-12T09:30:00+07:00",
    currentAssignee: null,
    lastReturnNote: null,
    allowedActions: ["EDIT"],
  },
  {
    id: "srt-008",
    status: "DRAFT",
    letterNumber: null,
    title: "Permohonan Penambahan Kapasitas Penyimpanan Cloud Storage",
    letterType: { id: "lt-06", code: "SPER", name: "Surat Permohonan" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Sub-Direktorat Infrastruktur & Jaringan", total: 1 },
    createdAt: "2026-10-11T11:15:00+07:00",
    currentAssignee: null,
    lastReturnNote: null,
    allowedActions: ["EDIT"],
  },

  // ── IN_REVIEW (12) ──────────────────────────
  {
    id: "srt-009",
    status: "IN_REVIEW",
    letterNumber: null,
    title: "Kebijakan Penggunaan Perangkat Pribadi (BYOD) di Lingkungan Kerja",
    letterType: { id: "lt-04", code: "SK", name: "Surat Keputusan" },
    confidentiality: { id: "cf-2", code: "TERBATAS", name: "Terbatas" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Sumber Daya Manusia", total: 2 },
    createdAt: "2026-10-10T08:00:00+07:00",
    currentAssignee: { id: "emp-002", name: "Budi Santoso", role: "REVIEWER", order: 1 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-010",
    status: "IN_REVIEW",
    letterNumber: null,
    title: "Permohonan Dukungan Teknis Implementasi Sistem ERP Modul Produksi",
    letterType: { id: "lt-01", code: "SAUT", name: "Surat Antar Unit Kerja" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-2", code: "SEGERA", name: "Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Operasional & Produksi", total: 1 },
    createdAt: "2026-10-09T14:00:00+07:00",
    currentAssignee: { id: "emp-003", name: "Citra Dewi Rahayu", role: "REVIEWER", order: 2 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-011",
    status: "IN_REVIEW",
    letterNumber: null,
    title: "Nota Dinas Hasil Evaluasi Keamanan Jaringan Semester I 2026",
    letterType: { id: "lt-03", code: "ND", name: "Nota Dinas" },
    confidentiality: { id: "cf-3", code: "RAHASIA", name: "Rahasia" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Teknologi Informasi", total: 1 },
    createdAt: "2026-10-08T10:30:00+07:00",
    currentAssignee: { id: "emp-002", name: "Budi Santoso", role: "REVIEWER", order: 1 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-012",
    status: "IN_REVIEW",
    letterNumber: null,
    title: "Surat Pemberitahuan Pembaruan Kebijakan Keamanan Data Perusahaan",
    letterType: { id: "lt-05", code: "SPTB", name: "Surat Pemberitahuan" },
    confidentiality: { id: "cf-2", code: "TERBATAS", name: "Terbatas" },
    urgency: { id: "ug-2", code: "SEGERA", name: "Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Seluruh Manajer (8)", total: 8 },
    createdAt: "2026-10-07T09:00:00+07:00",
    currentAssignee: { id: "emp-003", name: "Citra Dewi Rahayu", role: "REVIEWER", order: 2 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-013",
    status: "IN_REVIEW",
    letterNumber: null,
    title: "Permohonan Penetapan Tim Proyek Migrasi Sistem Lama ke Platform Cloud",
    letterType: { id: "lt-01", code: "SAUT", name: "Surat Antar Unit Kerja" },
    confidentiality: { id: "cf-2", code: "TERBATAS", name: "Terbatas" },
    urgency: { id: "ug-3", code: "SANGAT_SEGERA", name: "Sangat Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Sekretariat Direksi", total: 2 },
    createdAt: "2026-10-06T13:00:00+07:00",
    currentAssignee: { id: "emp-002", name: "Budi Santoso", role: "REVIEWER", order: 1 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-014",
    status: "IN_REVIEW",
    letterNumber: null,
    title: "Usulan Pelatihan Sertifikasi Cloud Computing untuk Staf TI",
    letterType: { id: "lt-06", code: "SPER", name: "Surat Permohonan" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Sumber Daya Manusia", total: 1 },
    createdAt: "2026-10-05T10:00:00+07:00",
    currentAssignee: { id: "emp-003", name: "Citra Dewi Rahayu", role: "REVIEWER", order: 2 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-015",
    status: "IN_REVIEW",
    letterNumber: null,
    title: "Nota Dinas Koordinasi Pemindahan Ruang Server",
    letterType: { id: "lt-03", code: "ND", name: "Nota Dinas" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-2", code: "SEGERA", name: "Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Sub-Direktorat Infrastruktur & Jaringan", total: 1 },
    createdAt: "2026-10-04T14:30:00+07:00",
    currentAssignee: { id: "emp-002", name: "Budi Santoso", role: "REVIEWER", order: 1 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-016",
    status: "IN_REVIEW",
    letterNumber: null,
    title: "Permohonan Anggaran Lisensi Perangkat Lunak Desain UX/UI 2027",
    letterType: { id: "lt-06", code: "SPER", name: "Surat Permohonan" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Keuangan & Akuntansi", total: 1 },
    createdAt: "2026-10-03T09:15:00+07:00",
    currentAssignee: { id: "emp-002", name: "Budi Santoso", role: "REVIEWER", order: 1 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-017",
    status: "IN_REVIEW",
    letterNumber: null,
    title: "Undangan Workshop Pengembangan Kompetensi Manajemen Proyek IT",
    letterType: { id: "lt-07", code: "SUND", name: "Surat Undangan" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-2", code: "SEGERA", name: "Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Sumber Daya Manusia", total: 2 },
    createdAt: "2026-10-02T11:00:00+07:00",
    currentAssignee: { id: "emp-003", name: "Citra Dewi Rahayu", role: "REVIEWER", order: 2 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-018",
    status: "IN_REVIEW",
    letterNumber: null,
    title: "Surat Keterangan Penggunaan Sistem Informasi Arsip Digital",
    letterType: { id: "lt-05", code: "SPTB", name: "Surat Pemberitahuan" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Hukum & Kepatuhan", total: 1 },
    createdAt: "2026-10-01T08:30:00+07:00",
    currentAssignee: { id: "emp-002", name: "Budi Santoso", role: "REVIEWER", order: 1 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-019",
    status: "IN_REVIEW",
    letterNumber: null,
    title: "Permintaan Penunjukan Admin Sistem Informasi Unit Pengadaan",
    letterType: { id: "lt-01", code: "SAUT", name: "Surat Antar Unit Kerja" },
    confidentiality: { id: "cf-2", code: "TERBATAS", name: "Terbatas" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Pengadaan & Logistik", total: 1 },
    createdAt: "2026-09-30T14:00:00+07:00",
    currentAssignee: { id: "emp-003", name: "Citra Dewi Rahayu", role: "REVIEWER", order: 2 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-020",
    status: "IN_REVIEW",
    letterNumber: null,
    title: "Laporan Insiden Keamanan Siber dan Tindak Lanjut September 2026",
    letterType: { id: "lt-03", code: "ND", name: "Nota Dinas" },
    confidentiality: { id: "cf-3", code: "RAHASIA", name: "Rahasia" },
    urgency: { id: "ug-3", code: "SANGAT_SEGERA", name: "Sangat Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Teknologi Informasi", total: 1 },
    createdAt: "2026-09-29T10:00:00+07:00",
    currentAssignee: { id: "emp-002", name: "Budi Santoso", role: "REVIEWER", order: 1 },
    lastReturnNote: null,
    allowedActions: [],
  },

  // ── RETURNED_TO_DRAFTER (4) ─────────────────
  {
    id: "srt-021",
    status: "RETURNED_TO_DRAFTER",
    letterNumber: null,
    title: "Nota Dinas Perubahan Alur Pengajuan Anggaran Divisi TI",
    letterType: { id: "lt-03", code: "ND", name: "Nota Dinas" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-2", code: "SEGERA", name: "Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Keuangan & Akuntansi", total: 1 },
    createdAt: "2026-09-25T09:00:00+07:00",
    currentAssignee: null,
    lastReturnNote: "Mohon lampirkan analisis dampak anggaran dan perbandingan dengan prosedur lama. Bagian IV paragraf 2 perlu diperjelas.",
    allowedActions: ["EDIT"],
  },
  {
    id: "srt-022",
    status: "RETURNED_TO_DRAFTER",
    letterNumber: null,
    title: "Permohonan Pengecualian Kebijakan Penggunaan USB di Area Produksi",
    letterType: { id: "lt-06", code: "SPER", name: "Surat Permohonan" },
    confidentiality: { id: "cf-2", code: "TERBATAS", name: "Terbatas" },
    urgency: { id: "ug-3", code: "SANGAT_SEGERA", name: "Sangat Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Sub-Direktorat Keamanan Informasi", total: 1 },
    createdAt: "2026-09-22T13:00:00+07:00",
    currentAssignee: null,
    lastReturnNote: "Dasar hukum pengecualian belum dicantumkan. Sertakan pula rekomendasi kontrol kompensasi yang akan diterapkan.",
    allowedActions: ["EDIT"],
  },
  {
    id: "srt-023",
    status: "RETURNED_TO_DRAFTER",
    letterNumber: null,
    title: "Usulan Revisi SLA Layanan Helpdesk Teknologi Informasi",
    letterType: { id: "lt-01", code: "SAUT", name: "Surat Antar Unit Kerja" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Operasional & Produksi", total: 2 },
    createdAt: "2026-09-20T10:30:00+07:00",
    currentAssignee: null,
    lastReturnNote: "Tabel SLA pada halaman 3 menggunakan satuan yang berbeda dari standar korporat. Harap diseragamkan dan cantumkan referensi kebijakan.",
    allowedActions: ["EDIT"],
  },
  {
    id: "srt-024",
    status: "RETURNED_TO_DRAFTER",
    letterNumber: null,
    title: "Rencana Audit Keamanan Sistem Informasi Internal Q4 2026",
    letterType: { id: "lt-04", code: "SK", name: "Surat Keputusan" },
    confidentiality: { id: "cf-3", code: "RAHASIA", name: "Rahasia" },
    urgency: { id: "ug-2", code: "SEGERA", name: "Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Sub-Direktorat Keamanan Informasi", total: 1 },
    createdAt: "2026-09-18T08:00:00+07:00",
    currentAssignee: null,
    lastReturnNote: "Ruang lingkup audit perlu dipersempit. Hapus poin 7 dan 8 yang bukan wewenang unit ini, serta perbaiki metodologi di bagian III.",
    allowedActions: ["EDIT"],
  },

  // ── IN_APPROVAL (5) ─────────────────────────
  {
    id: "srt-025",
    status: "IN_APPROVAL",
    letterNumber: null,
    title: "Permohonan Pengadaan Firewall Generasi Berikutnya untuk Kantor Pusat",
    letterType: { id: "lt-06", code: "SPER", name: "Surat Permohonan" },
    confidentiality: { id: "cf-2", code: "TERBATAS", name: "Terbatas" },
    urgency: { id: "ug-2", code: "SEGERA", name: "Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Pengadaan & Logistik", total: 1 },
    createdAt: "2026-09-15T09:00:00+07:00",
    currentAssignee: { id: "emp-004", name: "Darmawan Hidayat", role: "APPROVER", order: 1 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-026",
    status: "IN_APPROVAL",
    letterNumber: null,
    title: "Kebijakan Pengelolaan Identitas dan Akses Digital Karyawan",
    letterType: { id: "lt-04", code: "SK", name: "Surat Keputusan" },
    confidentiality: { id: "cf-2", code: "TERBATAS", name: "Terbatas" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Sumber Daya Manusia", total: 2 },
    createdAt: "2026-09-12T14:00:00+07:00",
    currentAssignee: { id: "emp-004", name: "Darmawan Hidayat", role: "APPROVER", order: 1 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-027",
    status: "IN_APPROVAL",
    letterNumber: null,
    title: "Surat Rekomendasi Vendor Pengembang Sistem Informasi Terpadu",
    letterType: { id: "lt-02", code: "SKK", name: "Surat Keluar Korporat" },
    confidentiality: { id: "cf-3", code: "RAHASIA", name: "Rahasia" },
    urgency: { id: "ug-3", code: "SANGAT_SEGERA", name: "Sangat Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Sekretariat Direksi", total: 1 },
    createdAt: "2026-09-10T10:00:00+07:00",
    currentAssignee: { id: "emp-004", name: "Darmawan Hidayat", role: "APPROVER", order: 1 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-028",
    status: "IN_APPROVAL",
    letterNumber: null,
    title: "Nota Dinas Persetujuan Rancangan Teknis Sistem Pelaporan Terintegrasi",
    letterType: { id: "lt-03", code: "ND", name: "Nota Dinas" },
    confidentiality: { id: "cf-2", code: "TERBATAS", name: "Terbatas" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Teknologi Informasi", total: 1 },
    createdAt: "2026-09-08T09:30:00+07:00",
    currentAssignee: { id: "emp-004", name: "Darmawan Hidayat", role: "APPROVER", order: 1 },
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-029",
    status: "IN_APPROVAL",
    letterNumber: null,
    title: "Permohonan Perpanjangan Kontrak Maintenance Infrastruktur TI",
    letterType: { id: "lt-01", code: "SAUT", name: "Surat Antar Unit Kerja" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-2", code: "SEGERA", name: "Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Pengadaan & Logistik", total: 1 },
    createdAt: "2026-09-05T13:00:00+07:00",
    currentAssignee: { id: "emp-004", name: "Darmawan Hidayat", role: "APPROVER", order: 1 },
    lastReturnNote: null,
    allowedActions: [],
  },

  // ── SENT / Disetujui (18) ──────────────────
  ...Array.from({ length: 18 }, (_, i): LetterListItem => {
    const idx = i + 1;
    const month = String(Math.max(1, 10 - Math.floor(i / 3))).padStart(2, "0");
    const day   = String(Math.max(1, 28 - (i % 15))).padStart(2, "0");
    const num   = String(idx).padStart(3, "0");
    const types = [
      { id: "lt-01", code: "SAUT", name: "Surat Antar Unit Kerja" },
      { id: "lt-03", code: "ND",   name: "Nota Dinas" },
      { id: "lt-04", code: "SK",   name: "Surat Keputusan" },
      { id: "lt-06", code: "SPER", name: "Surat Permohonan" },
      { id: "lt-05", code: "SPTB", name: "Surat Pemberitahuan" },
      { id: "lt-02", code: "SKK",  name: "Surat Keluar Korporat" },
    ];
    const titles = [
      "Persetujuan Anggaran Pengembangan Sistem Informasi Manajemen Aset",
      "Penetapan Standar Keamanan Data untuk Aplikasi Internal",
      "Kebijakan Pengarsipan Dokumen Digital Unit TI",
      "Surat Keputusan Penunjukan Tim Pengembang Digital Office",
      "Persetujuan Kerja Sama Pengembangan Sistem dengan Vendor Lokal",
      "Penetapan Prosedur Backup dan Pemulihan Data Kritis",
      "Kebijakan Penggunaan Teknologi Kecerdasan Buatan di Perusahaan",
      "Persetujuan Rencana Transformasi Digital Tahap II",
      "Penetapan Standar API untuk Integrasi Sistem Antar Unit",
      "Persetujuan Anggaran Operasional TI Triwulan IV 2026",
      "Kebijakan Manajemen Perubahan Sistem Informasi Korporat",
      "Surat Keputusan Pembentukan Komite Keamanan Informasi",
      "Persetujuan Rancangan Arsitektur Sistem Pelaporan Digital",
      "Penetapan SOP Penanganan Insiden Keamanan Siber",
      "Persetujuan Penggunaan Platform Low-Code untuk Otomatisasi Proses",
      "Kebijakan Retensi Data dan Penghapusan Aset Informasi",
      "Surat Keputusan Penetapan Data Steward per Unit Kerja",
      "Persetujuan Rencana Migrasi Infrastruktur ke Private Cloud",
    ];
    const units = [
      "Direktori Keuangan & Akuntansi",
      "Direktori Sumber Daya Manusia",
      "Sekretariat Direksi",
      "Direktori Operasional & Produksi",
      "Sub-Direktorat Keamanan Informasi",
    ];
    return {
      id: `srt-sent-${num}`,
      status: "SENT",
      letterNumber: `${num}/TI/PSP/X/2026`,
      title: titles[i % titles.length]!,
      letterType: types[i % types.length]!,
      confidentiality: i % 3 === 0
        ? { id: "cf-2", code: "TERBATAS", name: "Terbatas" }
        : { id: "cf-1", code: "BIASA", name: "Biasa" },
      urgency: i % 4 === 0
        ? { id: "ug-2", code: "SEGERA", name: "Segera" }
        : { id: "ug-1", code: "BIASA", name: "Biasa" },
      senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
      recipientsSummary: {
        first: units[i % units.length]!,
        total: i % 3 === 0 ? 2 : 1,
      },
      createdAt: `2026-${month}-${day}T09:00:00+07:00`,
      currentAssignee: null,
      lastReturnNote: null,
      allowedActions: [],
    };
  }),

  // ── CANCELLED (3) ───────────────────────────
  {
    id: "srt-047",
    status: "CANCELLED",
    letterNumber: null,
    title: "Nota Dinas Peninjauan Ulang Kontrak Layanan Internet Dedicated",
    letterType: { id: "lt-03", code: "ND", name: "Nota Dinas" },
    confidentiality: { id: "cf-1", code: "BIASA", name: "Biasa" },
    urgency: { id: "ug-1", code: "BIASA", name: "Biasa" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Sub-Direktorat Infrastruktur & Jaringan", total: 1 },
    createdAt: "2026-09-01T10:00:00+07:00",
    currentAssignee: null,
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-048",
    status: "CANCELLED",
    letterNumber: null,
    title: "Permohonan Pembukaan Akses Remote Desktop untuk Tim Audit Eksternal",
    letterType: { id: "lt-06", code: "SPER", name: "Surat Permohonan" },
    confidentiality: { id: "cf-2", code: "TERBATAS", name: "Terbatas" },
    urgency: { id: "ug-2", code: "SEGERA", name: "Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Sub-Direktorat Keamanan Informasi", total: 1 },
    createdAt: "2026-08-28T14:00:00+07:00",
    currentAssignee: null,
    lastReturnNote: null,
    allowedActions: [],
  },
  {
    id: "srt-049",
    status: "CANCELLED",
    letterNumber: null,
    title: "Usulan Pemindahan Pusat Data ke Lokasi Baru Gedung B",
    letterType: { id: "lt-01", code: "SAUT", name: "Surat Antar Unit Kerja" },
    confidentiality: { id: "cf-3", code: "RAHASIA", name: "Rahasia" },
    urgency: { id: "ug-3", code: "SANGAT_SEGERA", name: "Sangat Segera" },
    senderUnit: { id: "unit-ti-dev", name: "Sub-Direktorat Pengembangan Sistem" },
    recipientsSummary: { first: "Direktori Operasional & Produksi", total: 2 },
    createdAt: "2026-08-25T09:00:00+07:00",
    currentAssignee: null,
    lastReturnNote: null,
    allowedActions: [],
  },
];

// ─── Simulasi delay jaringan ──────────────────

function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms));
}

// ─── listLetters — mock sesuai kontrak bagian 13 ──

export async function listLetters(
  params: ListLettersParams = {}
): Promise<ListLettersResponse> {
  await sleep(400); // simulasi latensi

  const {
    status,
    q = "",
    letterTypeId,
    confidentialityId,
    from,
    to,
    page = 1,
    pageSize = 10,
  } = params;

  // Hitung counts (sebelum filter status)
  const counts = {
    ALL: RAW_LETTERS.length,
    DRAFT: RAW_LETTERS.filter((l) => l.status === "DRAFT").length,
    IN_REVIEW: RAW_LETTERS.filter((l) => l.status === "IN_REVIEW").length,
    RETURNED:  RAW_LETTERS.filter(
      (l) => l.status === "RETURNED_TO_DRAFTER" || l.status === "RETURNED_TO_REVIEWER"
    ).length,
    IN_APPROVAL: RAW_LETTERS.filter((l) => l.status === "IN_APPROVAL").length,
    SENT: RAW_LETTERS.filter((l) => l.status === "SENT").length,
    CANCELLED: RAW_LETTERS.filter((l) => l.status === "CANCELLED").length,
  };

  // Filter
  let filtered = RAW_LETTERS.filter((l) => {
    // Status tab
    const statusStr = (status ?? "") as string;
    if (statusStr === "RETURNED_TO_DRAFTER") {
      if (l.status !== "RETURNED_TO_DRAFTER" && l.status !== "RETURNED_TO_REVIEWER") return false;
    } else if (statusStr && statusStr !== "") {
      if ((l.status as string) !== statusStr) return false;
    }

    // Pencarian
    if (q) {
      const lower = q.toLowerCase();
      const match =
        (l.title ?? "").toLowerCase().includes(lower) ||
        (l.letterNumber ?? "").toLowerCase().includes(lower) ||
        l.letterType.name.toLowerCase().includes(lower);
      if (!match) return false;
    }

    // Jenis Surat
    if (letterTypeId && l.letterType.id !== letterTypeId) return false;

    // Sifat
    if (confidentialityId && l.confidentiality.id !== confidentialityId) return false;

    // Rentang tanggal
    if (from && l.createdAt < from) return false;
    if (to   && l.createdAt > to + "T23:59:59+07:00") return false;

    return true;
  });

  // Urutkan terbaru dulu
  filtered.sort((a, b) => b.createdAt.localeCompare(a.createdAt));

  const total = filtered.length;
  const start = (page - 1) * pageSize;
  const data  = filtered.slice(start, start + pageSize);

  return { data, meta: { page, pageSize, total }, counts };
}
