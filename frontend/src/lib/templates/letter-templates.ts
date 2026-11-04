// lib/templates/letter-templates.ts
// Registri template tata naskah dinas (.docx).
// - File .docx template fisik disimpan di `frontend/public/templates/*.docx` (selama endpoint
//   MinIO `/api/v1/templates` di backend belum tersedia). Anda dapat mengganti/mengedit file
//   .docx di folder `frontend/public/templates/` langsung menggunakan Microsoft Word.
// - Saat tombol "Sinkronkan Form" diklik, placeholder di dalam dokumen .docx yang dimuat
//   diperbarui secara langsung di editor dengan data form terbaru (Perihal, Sifat, Penerima, Approver).

import type { RichEdit } from "devexpress-richedit";

export interface LetterTemplateContext {
  title?: string;
  sifat?: "BIASA" | "TERBATAS" | "RAHASIA" | string;
  kecepatan?: "BIASA" | "SEGERA" | "SANGAT_SEGERA" | string;
  classificationCode?: string;
  classificationName?: string;
  recipients?: Array<{ name: string; position?: string }>;
  approver?: { name: string; position?: string; employeeNumber?: string };
  drafter?: { name: string; position?: string };
  date?: string;
  letterNumberPlaceholder?: string;
  attachmentsCount?: number;
}

export interface LetterTemplateDefinition {
  id: string;
  name: string;
  description: string;
  /** Path ke file .docx statis di folder frontend/public/templates/ (misal "/templates/surat-dinas-standar.docx") */
  fileUrl: string;
  letterTypeKeywords: string[];
}

export const LETTER_TEMPLATE_LIST: LetterTemplateDefinition[] = [
  {
    id: "tmpl-dinas-standar",
    name: "Surat Dinas Resmi Standar",
    description:
      "Format surat resmi dengan Kop PT Pupuk Sriwidjaja Palembang, Nomor ###NoSurat###, Yth, TTD & Tembusan.",
    fileUrl: "/templates/surat-dinas-standar.docx",
    letterTypeKeywords: ["surat", "antar", "keluar", "standar", "korporat"],
  },
  {
    id: "tmpl-nota-dinas",
    name: "Nota Dinas Internal",
    description:
      "Format korespondensi internal (Yth, Dari, Tanggal, Perihal, Pembahasan, TTD Elektronik).",
    fileUrl: "/templates/nota-dinas.docx",
    letterTypeKeywords: ["nota", "dinas", "internal"],
  },
  {
    id: "tmpl-undangan",
    name: "Surat Undangan Rapat",
    description:
      "Format undangan kegiatan/rapat koordinasi lengkap dengan tabel Hari, Waktu, Tempat, Agenda.",
    fileUrl: "/templates/undangan-rapat.docx",
    letterTypeKeywords: ["undangan", "rapat", "acara"],
  },
  {
    id: "tmpl-tugas",
    name: "Surat Perintah Tugas (SPT)",
    description:
      "Format penugasan kedinasan resmi dengan tabel daftar pegawai dan pejabat penandatangan.",
    fileUrl: "/templates/surat-tugas.docx",
    letterTypeKeywords: ["tugas", "perintah", "spt"],
  },
  {
    id: "tmpl-kosong",
    name: "Dokumen Kosong (Blank)",
    description: "Mulai dari halaman kosong A4 standar Word.",
    fileUrl: "/templates/dokumen-kosong.docx",
    letterTypeKeywords: [],
  },
];

export function getTemplateById(id?: string): LetterTemplateDefinition {
  const found = LETTER_TEMPLATE_LIST.find((t) => t.id === id);
  return found || LETTER_TEMPLATE_LIST[0];
}

/**
 * Mengambil Blob .docx template fisik dari `frontend/public/templates/*.docx`.
 */
export async function resolveTemplateDocxBlob(
  templateId?: string
): Promise<{ blob: Blob; template: LetterTemplateDefinition }> {
  const tpl = getTemplateById(templateId);
  const res = await fetch(tpl.fileUrl, { cache: "no-store" });
  if (!res.ok) {
    throw new Error(
      `Gagal memuat file template "${tpl.name}" dari ${tpl.fileUrl} (HTTP ${res.status})`
    );
  }
  const blob = await res.blob();
  return { blob, template: tpl };
}

/**
 * Mengganti placeholder teks di dalam dokumen DevExpress RichEdit yang sedang terbuka
 * sambil mempertahankan format karakter (tebal, garis bawah, ukuran font) dari file .docx asli.
 */
function replacePlaceholderInSubDoc(
  subDoc: RichEdit["document"]["subDocuments"]["main"],
  search: string,
  replacement?: string
) {
  const cleanVal = replacement?.trim();
  if (!search || !cleanVal || search === cleanVal) return;

  let text = subDoc.getText();
  let idx = text.lastIndexOf(search);
  let guard = 0;

  while (idx !== -1 && guard < 20) {
    guard++;
    const interval = { start: idx, length: search.length };
    const charProps = subDoc.getCharacterProperties(interval);
    subDoc.deleteText(interval);
    const inserted = subDoc.insertText(idx, cleanVal);
    subDoc.setCharacterProperties(inserted, charProps);
    text = subDoc.getText();
    idx = idx > 0 ? text.lastIndexOf(search, idx - 1) : -1;
  }
}

/**
 * Menyinkronkan data form (Perihal, Penerima, Pejabat Penandatangan, Nomor Surat)
 * ke dalam naskah .docx yang sedang dimuat di DevExpress RichEdit.
 */
export function applyTemplateContextToEditor(
  editor: RichEdit,
  ctx: LetterTemplateContext
) {
  const subDoc = editor.document?.subDocuments?.main;
  if (!subDoc) return;

  editor.beginUpdate();
  try {
    if (ctx.title?.trim()) {
      replacePlaceholderInSubDoc(subDoc, "[Perihal / Hal Surat]", ctx.title);
      replacePlaceholderInSubDoc(
        subDoc,
        "[Perihal / Hal Nota Dinas]",
        ctx.title
      );
      replacePlaceholderInSubDoc(
        subDoc,
        "Undangan Rapat Koordinasi",
        ctx.title
      );
      replacePlaceholderInSubDoc(
        subDoc,
        "Penugasan Implementasi Sistem",
        ctx.title
      );
    }

    if (ctx.recipients && ctx.recipients.length > 0) {
      const recipientText = ctx.recipients.map((r) => r.name).join(", ");
      replacePlaceholderInSubDoc(
        subDoc,
        "Pimpinan Unit Kerja Terkait",
        recipientText
      );
      replacePlaceholderInSubDoc(
        subDoc,
        "Pimpinan Unit Terkait",
        recipientText
      );
      replacePlaceholderInSubDoc(
        subDoc,
        "Bapak/Ibu Pimpinan Unit Kerja",
        recipientText
      );
    }

    if (ctx.approver?.name) {
      replacePlaceholderInSubDoc(
        subDoc,
        "[Nama Pejabat Penandatangan]",
        ctx.approver.name
      );
    }

    if (ctx.approver?.position) {
      replacePlaceholderInSubDoc(
        subDoc,
        "[Jabatan Pejabat Penandatangan]",
        ctx.approver.position
      );
    }

    if (ctx.approver?.employeeNumber) {
      replacePlaceholderInSubDoc(
        subDoc,
        "NIP. ..............................",
        `NIP. ${ctx.approver.employeeNumber}`
      );
    }

    if (ctx.drafter?.position) {
      replacePlaceholderInSubDoc(
        subDoc,
        "Staf Pengembang Sistem / Unit TI",
        ctx.drafter.position
      );
    }

    if (ctx.letterNumberPlaceholder) {
      replacePlaceholderInSubDoc(
        subDoc,
        "###NoSurat###",
        ctx.letterNumberPlaceholder
      );
    }
  } finally {
    editor.endUpdate();
  }
}
