import {
  Building2,
  Users,
  Network,
  BriefcaseBusiness,
  FileText,
  Tags,
  FolderArchive,
  ShieldCheck,
  Gauge,
  Stamp,
  type LucideIcon,
} from "lucide-react";

export interface MasterCategory {
  key: string;
  label: string;
  description: string;
  group: string;
  icon: LucideIcon;
  permission?: string;
  category?: string;
  highlights?: string[];
}

export const CATEGORIES: MasterCategory[] = [
  {
    key: "users",
    label: "User",
    description: "Akun pegawai, penempatan jabatan, role aplikasi, dan keamanan kata sandi.",
    group: "Organisasi",
    icon: Users,
    permission: "USER_MANAGE",
    highlights: ["Akun & NIK", "Role", "Reset Password"],
  },
  {
    key: "companies",
    label: "Perusahaan",
    description: "Identitas entitas perusahaan, kota kedudukan, dan pembuatan admin perusahaan.",
    group: "Organisasi",
    icon: Building2,
    permission: "COMPANY_MANAGE",
    highlights: ["Multi-entitas", "Admin Perusahaan"],
  },
  {
    key: "units",
    label: "Unit Kerja",
    description: "Struktur direktorat, departemen, divisi, serta hirarki unit induk.",
    group: "Organisasi",
    icon: Network,
    permission: "ORG_MANAGE",
    highlights: ["Hirarki Unit", "Pimpinan Unit"],
  },
  {
    key: "positions",
    label: "Jabatan",
    description: "Daftar jabatan, level pejabat penandatangan, pimpinan unit, dan pemegang aktif.",
    group: "Organisasi",
    icon: BriefcaseBusiness,
    permission: "ORG_MANAGE",
    highlights: ["Level Jabatan", "Pejabat & Pimpinan"],
  },
  {
    key: "templates",
    label: "Template Surat",
    description: "Berkas master .docx berpenanda otomatis untuk pembuatan naskah surat.",
    group: "Persuratan",
    icon: FileText,
    permission: "TEMPLATE_CREATE",
    highlights: ["File .docx", "Placeholder Surat"],
  },
  {
    key: "letter-types",
    label: "Jenis Surat",
    description: "Kategori naskah dinas (Nota Dinas, Surat Keluar, SK, dll.) pada formulir surat.",
    group: "Persuratan",
    icon: Tags,
    category: "LETTER_TYPE",
    highlights: ["Standar & Kustom", "Kode Jenis"],
  },
  {
    key: "classifications",
    label: "Klasifikasi Arsip",
    description: "Kode klasifikasi pokok dan sub-masalah untuk pemberkasan dan penomoran.",
    group: "Persuratan",
    icon: FolderArchive,
    category: "CLASSIFICATION",
    highlights: ["Kode Arsip", "Standar & Kustom"],
  },
  {
    key: "confidentiality",
    label: "Sifat Surat",
    description: "Tingkat kerahasiaan dokumen (Biasa, Terbatas, Rahasia).",
    group: "Persuratan",
    icon: ShieldCheck,
    category: "CONFIDENTIALITY",
    highlights: ["Kerahasiaan", "Referensi Sistem"],
  },
  {
    key: "urgency",
    label: "Kecepatan Surat",
    description: "Prioritas penyampaian dan kecepatan tindak lanjut (Biasa, Segera, Sangat Segera).",
    group: "Persuratan",
    icon: Gauge,
    category: "URGENCY",
    highlights: ["Prioritas", "Referensi Sistem"],
  },
  {
    key: "settings",
    label: "Pengaturan Perusahaan",
    description: "Format penomoran surat otomatis dan watermark PDF naskah resmi perusahaan.",
    group: "Pengaturan",
    icon: Stamp,
    permission: "COMPANY_SETTINGS",
    highlights: ["Watermark PDF", "Format Nomor Surat"],
  },
];

export function canOpen(category: MasterCategory, permissions: string[]) {
  return (
    !category.permission ||
    permissions.includes(category.permission) ||
    (category.key === "templates" && permissions.includes("TEMPLATE_MANAGE"))
  );
}

export function accessLabel(category: MasterCategory, permissions: string[]) {
  if (!canOpen(category, permissions)) {
    return "Akses terbatas";
  }
  const isEditableLookup =
    permissions.includes("ORG_MANAGE") &&
    (category.category === "LETTER_TYPE" || category.category === "CLASSIFICATION");

  if (category.category && !isEditableLookup) {
    return "Referensi";
  }
  return "Kelola data";
}
