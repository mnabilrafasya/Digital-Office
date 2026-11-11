import { allChoices, type MasterRow } from "@/lib/api/master-data";

export interface Field {
  key: string;
  label: string;
  source?: string;
  type?:
    | "text"
    | "email"
    | "password"
    | "select"
    | "checkbox"
    | "file"
    | "textarea"
    | "number";
  required?: boolean;
  max?: number;
  pattern?: string;
  createOnly?: boolean;
  choices?: string;
  hint?: string;
  initial?: string;
  accept?: string;
  placeholder?: string;
}

export interface Resource {
  key: string;
  label: string;
  permission: string;
  path: string;
  columns: { key: string; label: string }[];
  fields: Field[];
  paged?: boolean;
  identity?: string;
}

export const passwordField: Field = {
  key: "password",
  label: "Password",
  type: "password",
  required: true,
  placeholder: "Minimal 8 karakter...",
  hint: "Minimal 8 karakter, maksimal 72 byte. Reset password akan mengakhiri sesi aktif user tersebut.",
};

export const userFields: Field[] = [
  {
    key: "employeeNumber",
    label: "NIK / NIP",
    required: true,
    max: 30,
    pattern: "[A-Za-z0-9._-]+",
    placeholder: "Contoh: 2024001",
    hint: "Nomor induk pegawai unik (huruf, angka, titik, garis bawah, atau strip).",
  },
  {
    key: "name",
    label: "Nama lengkap",
    required: true,
    max: 150,
    placeholder: "Contoh: Budi Santoso",
  },
  {
    key: "email",
    label: "Alamat email",
    type: "email",
    required: true,
    max: 150,
    placeholder: "nama@perusahaan.co.id",
  },
];

export const RESOURCES: Resource[] = [
  {
    key: "companies",
    label: "Perusahaan",
    permission: "COMPANY_MANAGE",
    path: "/admin/companies",
    paged: true,
    identity: "code",
    columns: [
      { key: "name", label: "Entitas Perusahaan" },
      { key: "letterNumberFormat", label: "Format Nomor & Watermark PDF" },
      { key: "activeUnits", label: "Ringkasan Data" },
    ],
    fields: [
      {
        key: "code",
        label: "Kode perusahaan",
        required: true,
        max: 20,
        pattern: "[A-Z0-9]{2,20}",
        createOnly: true,
        placeholder: "Contoh: PSP",
        hint: "2–20 huruf kapital atau angka. Kode bersifat permanen dan dipakai pada penomoran surat.",
      },
      {
        key: "name",
        label: "Nama resmi perusahaan",
        required: true,
        max: 150,
        placeholder: "Contoh: PT Pupuk Sriwidjaja Palembang",
      },
      {
        key: "city",
        label: "Kota kedudukan",
        max: 50,
        placeholder: "Contoh: Palembang",
      },
    ],
  },
  {
    key: "units",
    label: "Unit Kerja",
    permission: "ORG_MANAGE",
    path: "/admin/units",
    columns: [
      { key: "name", label: "Unit Kerja" },
      { key: "parentName", label: "Unit Induk" },
      { key: "headName", label: "Pimpinan Unit" },
      { key: "activeUsers", label: "Komposisi" },
    ],
    fields: [
      {
        key: "name",
        label: "Nama unit kerja",
        required: true,
        max: 150,
        placeholder: "Contoh: Divisi Teknologi Informasi",
      },
      {
        key: "unitType",
        label: "Jenis / tingkat unit",
        required: true,
        max: 30,
        pattern: "[A-Z0-9_ ]+",
        initial: "UNIT",
        placeholder: "DIREKTORAT / KOMPARTEMEN / DEPARTEMEN / UNIT",
        hint: "Gunakan huruf kapital, misalnya DIREKTORAT, DEPARTEMEN, DIVISI, atau UNIT.",
      },
      {
        key: "parentId",
        label: "Unit induk (opsional)",
        type: "select",
        choices: "units",
        hint: "Kosongkan bila unit ini berada pada tingkat paling atas.",
      },
    ],
  },
  {
    key: "positions",
    label: "Jabatan",
    permission: "ORG_MANAGE",
    path: "/admin/positions",
    paged: true,
    columns: [
      { key: "name", label: "Jabatan & Kewenangan" },
      { key: "unitName", label: "Unit Kerja" },
      { key: "level.name", label: "Level Jabatan" },
      { key: "holders", label: "Pemegang Jabatan Aktif" },
    ],
    fields: [
      {
        key: "name",
        label: "Nama jabatan",
        required: true,
        max: 150,
        placeholder: "Contoh: VP Teknologi Informasi",
      },
      {
        key: "unitId",
        label: "Unit kerja",
        type: "select",
        choices: "units",
        required: true,
      },
      {
        key: "levelId",
        source: "level.id",
        label: "Level jabatan",
        type: "select",
        choices: "levels",
        required: true,
        hint: "Level DIREKTUR, SVP, VP, dan AVP termasuk pejabat yang berwenang menyetujui surat & disposisi.",
      },
      {
        key: "isUnitHead",
        label: "Tetapkan sebagai jabatan pimpinan unit",
        type: "checkbox",
        hint: "Satu unit kerja hanya boleh memiliki satu jabatan pimpinan aktif.",
      },
    ],
  },
  {
    key: "users",
    label: "User",
    permission: "USER_MANAGE",
    path: "/admin/users",
    paged: true,
    columns: [
      { key: "name", label: "Pegawai & Identitas" },
      { key: "position.name", label: "Jabatan & Unit Kerja" },
      { key: "role.name", label: "Role Akses" },
    ],
    fields: [
      ...userFields,
      {
        key: "positionId",
        source: "position.id",
        label: "Jabatan",
        type: "select",
        choices: "positions",
        required: true,
        hint: "Unit kerja user otomatis mengikuti unit dari jabatan yang dipilih.",
      },
      {
        key: "roleId",
        source: "role.id",
        label: "Role aplikasi",
        type: "select",
        choices: "roles",
        required: true,
      },
      { ...passwordField, createOnly: true },
    ],
  },
];

export function rowId(resource: Resource, row: MasterRow) {
  return String(row[resource.identity ?? "id"]);
}

export function display(value: unknown): string {
  if (value === null || value === undefined || value === "") return "—";
  if (typeof value === "boolean") return value ? "Ya" : "Tidak";
  if (Array.isArray(value)) return value.join(", ") || "—";
  return String(value);
}

const CHOICE_PATHS: Record<string, string> = {
  units: "/admin/units?status=ACTIVE",
  positions: "/admin/positions?status=ACTIVE",
  levels: "/admin/position-levels",
  roles: "/admin/roles",
  letterTypes: "/lookups?category=LETTER_TYPE",
};

export function loadChoices(key: string, company?: string) {
  const path = company
    ? `/admin/companies/${encodeURIComponent(company)}/units`
    : CHOICE_PATHS[key];
  return allChoices(path, (row) =>
    key === "positions" ? `${row.name} — ${row.unitName}` : String(row.name),
  );
}

export function lookupResource(category: string, label: string): Resource {
  return {
    key: `lookup-${category}`,
    label,
    permission: "ORG_MANAGE",
    path: `/admin/lookups/${category}`,
    columns: [
      { key: "code", label: "Kode" },
      { key: "name", label: `Nama ${label}` },
      { key: "note", label: "Keterangan" },
      { key: "sortOrder", label: "Urutan" },
    ],
    fields: [
      {
        key: "code",
        label: "Kode referensi",
        required: true,
        max: 30,
        pattern: "[A-Z0-9][A-Z0-9_.-]{0,29}",
        createOnly: true,
        placeholder: category === "LETTER_TYPE" ? "Contoh: ND" : "Contoh: KP.01",
        hint: "Kode bersifat tetap setelah dibuat. Gunakan huruf kapital, angka, titik, garis bawah, atau tanda hubung.",
      },
      {
        key: "name",
        label: `Nama ${label.toLowerCase()}`,
        required: true,
        max: 100,
        placeholder:
          category === "LETTER_TYPE"
            ? "Contoh: Nota Dinas"
            : "Contoh: Kepegawaian Umum",
      },
      {
        key: "note",
        label: "Keterangan penggunaan",
        type: "textarea",
        max: 2000,
        placeholder: "Penjelasan singkat mengenai penggunaan kode ini...",
      },
      {
        key: "sortOrder",
        label: "Urutan tampilan",
        type: "number",
        initial: "0",
        required: true,
        hint: "Angka lebih kecil ditampilkan lebih awal pada daftar pilihan surat.",
      },
    ],
  };
}
