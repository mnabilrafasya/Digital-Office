import type { AuditLogEntry } from "@/lib/api/audit";

export const ENTITY_LABELS: Record<string, string> = {
  LETTER: "Surat", TEMPLATE: "Template surat", COMPANY: "Perusahaan", UNIT: "Unit kerja",
  POSITION: "Jabatan", USER: "Pengguna", LOOKUP: "Referensi persuratan", AUDIT: "Unduhan riwayat",
};
export const ACTION_LABELS: Record<string, string> = {
  CREATE: "Membuat draf surat", SAVE: "Menyimpan naskah", SUBMIT: "Mengajukan surat",
  READ: "Membaca surat", APPROVE: "Menyetujui tahap pemeriksaan", RETURN: "Mengembalikan untuk revisi",
  SIGN: "Menyetujui surat", CANCEL: "Membatalkan surat", REASSIGN: "Mengganti penanggung jawab",
  ATTACHMENT_ADD: "Menambahkan lampiran", ATTACHMENT_DELETE: "Menghapus lampiran",
  FINAL_PDF: "Menyiapkan PDF resmi", DELIVER: "Mengirim surat ke penerima", INBOX_READ: "Membaca surat masuk",
  DISPOSITION: "Mendisposisikan surat", REDIRECT: "Mengalihkan surat", DOWNLOAD: "Mengunduh dokumen",
  VIEW: "Membuka dokumen", TEMPLATE_CREATE: "Menambahkan template", TEMPLATE_UPDATE: "Mengubah template",
  TEMPLATE_STATUS: "Mengubah status template", TEMPLATE_DELETE: "Menghapus template", AUDIT_EXPORT: "Mengunduh riwayat aktivitas",
  COMPANY_CREATE: "Menambahkan perusahaan", COMPANY_UPDATE: "Mengubah perusahaan",
  COMPANY_STATUS: "Mengubah status perusahaan", COMPANY_DELETE: "Menghapus perusahaan", COMPANY_SETTINGS: "Mengubah pengaturan perusahaan",
  UNIT_CREATE: "Menambahkan unit kerja", UNIT_UPDATE: "Mengubah unit kerja", UNIT_STATUS: "Mengubah status unit", UNIT_DELETE: "Menghapus unit kerja",
  POSITION_CREATE: "Menambahkan jabatan", POSITION_UPDATE: "Mengubah jabatan", POSITION_STATUS: "Mengubah status jabatan", POSITION_DELETE: "Menghapus jabatan",
  USER_CREATE: "Menambahkan pengguna", USER_UPDATE: "Mengubah pengguna", USER_STATUS: "Mengubah status pengguna", USER_DELETE: "Menghapus pengguna",
  USER_PASSWORD_RESET: "Mereset kata sandi pengguna", LOOKUP_CREATE: "Menambahkan referensi persuratan",
  LOOKUP_UPDATE: "Mengubah referensi persuratan", LOOKUP_STATUS: "Mengubah status referensi", LOOKUP_DELETE: "Menghapus referensi persuratan",
  UPLOAD_SIGNED: "Mengunggah surat bertanda tangan", INCOMING_RECEIVE: "Mencatat surat masuk", RECEIVE: "Menerima surat masuk",
};
export const activityLabel = (action: string) => ACTION_LABELS[action] ?? "Aktivitas lainnya";
export const entityLabel = (type: string) => ENTITY_LABELS[type] ?? "Data aplikasi";
export function dateTime(value: string) {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "—";
  return new Intl.DateTimeFormat("id-ID", {timeZone:"Asia/Jakarta",day:"numeric",month:"short",year:"numeric",hour:"2-digit",minute:"2-digit"}).format(date);
}
function jakartaDate(date: Date) {
  const parts = new Intl.DateTimeFormat("en", {timeZone:"Asia/Jakarta",year:"numeric",month:"2-digit",day:"2-digit"}).formatToParts(date);
  const value = (type: string) => parts.find(part=>part.type===type)!.value;
  return `${value("year")}-${value("month")}-${value("day")}`;
}
export function initialDates(now = new Date()) {
  return {from:jakartaDate(new Date(now.getTime()-6*86400000)),to:jakartaDate(now)};
}
export function waitingTime(hours: number) {
  if (!Number.isFinite(hours) || hours < 1) return "Kurang dari 1 jam";
  if (hours < 24) return `${Math.floor(hours)} jam`;
  const days=Math.floor(hours/24),rest=Math.floor(hours%24);
  return `${days} hari${rest ? ` ${rest} jam` : ""}`;
}
function record(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string,unknown> : {};
}
function text(value: unknown) { return typeof value === "string" && value.trim() ? value.trim() : undefined; }
export function activitySubject(log: AuditLogEntry) {
  if (log.letter) return log.letter.masked ? "Surat dengan akses dibatasi" : log.letter.title || log.letter.number || "Surat tanpa perihal";
  const snapshot=record(log.snapshot),after=record(snapshot.after),before=record(snapshot.before);
  return text(after.name) || text(before.name) || text(snapshot.name) || text(snapshot.code) || entityLabel(log.entityType);
}
export function readableChanges(log: AuditLogEntry): {label:string;before:string;after:string}[] {
  if(log.letter?.masked) return [];
  const snapshot=record(log.snapshot),before=record(snapshot.before),after=record(snapshot.after);
  const fields:Record<string,string>={name:"Nama",city:"Kota",isActive:"Status",letterNumberFormat:"Format nomor surat",watermarkText:"Watermark"};
  const result=[];
  for(const [key,label] of Object.entries(fields)) {
    if(!(key in before) || !(key in after) || before[key]===after[key]) continue;
    const display=(value:unknown)=>key==="isActive" && typeof value==="boolean" ? value ? "Aktif" : "Nonaktif" : text(value) ?? "Tidak diisi";
    result.push({label,before:display(before[key]),after:display(after[key])});
  }
  return result;
}
