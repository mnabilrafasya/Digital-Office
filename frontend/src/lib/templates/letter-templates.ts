// lib/templates/letter-templates.ts
// Registri & klien API template tata naskah dinas (.docx).
// - Mengambil daftar dan file .docx template dari backend & MinIO (`/api/v1/templates` dan `/api/v1/templates/{id}/file`).
// - Menyediakan fallback ke `frontend/public/templates/*.docx` bila diakses tanpa sesi login (misal `/dev/editor`).
// - Saat tombol "Sinkronkan Form" diklik, placeholder statis di dalam dokumen .docx yang dimuat
//   diperbarui secara langsung di editor dengan data form terbaru (Perihal, Penerima, Approver),
//   sementara penanda sistem `###*...*###` / `###...###` tetap dijaga utuh untuk diisi backend saat surat terbit.

import type { RichEdit } from "devexpress-richedit";

export interface SystemPlaceholderDefinition {
  /** Kode penanda sistem sesuai `backend/internal/surat/penanda.go` */
  code: string;
  /** Label singkat penanda */
  label: string;
  /** Penjelasan nilai yang diisikan oleh server saat surat disetujui (SENT) */
  description: string;
}

/**
 * Daftar 4 penanda sistem (`###*Nama*###`) yang diisi otomatis oleh backend (`internal/surat/penanda.go`).
 * Backend juga menerima bentuk tanpa bintang (`###Nama###`) dan tidak peka huruf besar/kecil.
 * Penanda lainnya (Perihal, Kepada, NamaPenandatangan, dsb.) ditulis oleh drafter di editor.
 */
export const SYSTEM_PLACEHOLDERS: SystemPlaceholderDefinition[] = [
  {
    code: "###*NoSurat*###",
    label: "Nomor Surat",
    description: "Nomor surat resmi (diisi otomatis saat disetujui Approver terakhir)",
  },
  {
    code: "###*TglSurat*###",
    label: "Tanggal Surat",
    description: "Tanggal surat disetujui (contoh: 5 November 2026)",
  },
  {
    code: "###*Sifat*###",
    label: "Sifat Surat",
    description: "Sifat kerahasiaan surat (Biasa / Terbatas / Rahasia)",
  },
  {
    code: "###*QRCode*###",
    label: "QR Code Persetujuan",
    description:
      "Posisi gambar QR pengesahan resmi berukuran 3 × 3 cm (wajib untuk surat internal, dikosongkan untuk surat eksternal)",
  },
];

export interface LetterTemplateContext {
  title?: string;
  sifat?: "BIASA" | "TERBATAS" | "RAHASIA" | string;
  kecepatan?: "BIASA" | "SEGERA" | "SANGAT_SEGERA" | string;
  classificationCode?: string;
  classificationName?: string;
  recipients?: Array<{ name: string; position?: string }>;
  ccRecipients?: Array<{ name: string }>;
  externalRecipient?: {
    name?: string;
    address?: string;
    attention?: string;
  };
  approver?: { name: string; position?: string; employeeNumber?: string };
  drafter?: { name: string; position?: string };
  date?: string;
  letterNumberPlaceholder?: string;
  attachmentsCount?: number;
}

export interface BackendTemplateItem {
  id: string;
  name: string;
  description: string | null;
  letterType: { id: string; code: string; name: string } | null;
  fileUrl: string;
  createdBy: { id: string; name: string };
  createdAt: string;
  updatedAt: string;
}

export interface LetterTemplateDefinition {
  id: string;
  /** ID statis warisan (mis. "tmpl-dinas-standar") untuk pencocokan mundur */
  legacyId?: string;
  name: string;
  description: string;
  /** URL unduhan file .docx (`/api/v1/templates/{id}/file` atau fallback `/templates/*.docx`) */
  fileUrl: string;
  /** Path ke file .docx statis di `frontend/public/templates/` sebagai cadangan */
  fallbackFileUrl?: string;
  letterType?: { id: string; code: string; name: string } | null;
  letterTypeKeywords: string[];
}

export const LETTER_TEMPLATE_LIST: LetterTemplateDefinition[] = [
  {
    id: "tmpl-dinas-standar",
    legacyId: "tmpl-dinas-standar",
    name: "Surat Dinas Resmi Standar",
    description:
      "Format surat resmi dengan kop PT Pupuk Sriwidjaja Palembang, nomor ###NoSurat###, Yth, tanda tangan, dan tembusan.",
    fileUrl: "/templates/surat-dinas-standar.docx",
    fallbackFileUrl: "/templates/surat-dinas-standar.docx",
    letterTypeKeywords: ["surat_dinas", "antar", "standar", "korporat"],
  },
  {
    id: "tmpl-nota-dinas",
    legacyId: "tmpl-nota-dinas",
    name: "Nota Dinas Internal",
    description:
      "Format korespondensi internal (Yth, Dari, Tanggal, Perihal, Pembahasan, tanda tangan elektronik).",
    fileUrl: "/templates/nota-dinas.docx",
    fallbackFileUrl: "/templates/nota-dinas.docx",
    letterTypeKeywords: ["nota", "dinas"],
  },
  {
    id: "tmpl-undangan",
    legacyId: "tmpl-undangan",
    name: "Surat Undangan Rapat",
    description:
      "Format undangan kegiatan atau rapat koordinasi dengan tabel hari, waktu, tempat, dan agenda.",
    fileUrl: "/templates/undangan-rapat.docx",
    fallbackFileUrl: "/templates/undangan-rapat.docx",
    letterTypeKeywords: ["undangan", "rapat", "acara"],
  },
  {
    id: "tmpl-tugas",
    legacyId: "tmpl-tugas",
    name: "Surat Perintah Tugas (SPT)",
    description:
      "Format penugasan kedinasan dengan tabel daftar pegawai dan pejabat penandatangan.",
    fileUrl: "/templates/surat-tugas.docx",
    fallbackFileUrl: "/templates/surat-tugas.docx",
    letterTypeKeywords: ["tugas", "perintah", "spt"],
  },
  {
    id: "tmpl-eksternal",
    legacyId: "tmpl-eksternal",
    name: "Surat Eksternal",
    description:
      "Format surat keluar kepada perusahaan atau instansi di luar perusahaan.",
    fileUrl: "/templates/surat-external.docx",
    fallbackFileUrl: "/templates/surat-external.docx",
    letterTypeKeywords: ["eksternal", "external", "surat_eksternal"],
  },
  {
    id: "tmpl-kosong",
    legacyId: "tmpl-kosong",
    name: "Dokumen Kosong (Blank)",
    description: "Mulai dari halaman kosong A4 standar Word.",
    fileUrl: "/templates/dokumen-kosong.docx",
    fallbackFileUrl: "/templates/dokumen-kosong.docx",
    letterTypeKeywords: [],
  },
];

let cachedTemplates: LetterTemplateDefinition[] | null = null;

/** Buang cache setelah admin mengubah status template. */
export function invalidateTemplateCache() { cachedTemplates = null; }
let inflightTemplatesPromise: Promise<LetterTemplateDefinition[]> | null = null;

function mapBackendTemplateToDefinition(
  item: BackendTemplateItem
): LetterTemplateDefinition {
  const normalizedName = item.name.trim().toLowerCase();
  const matchedStatic =
    LETTER_TEMPLATE_LIST.find(
      (s) => s.name.trim().toLowerCase() === normalizedName
    ) ??
    (item.letterType?.code
      ? LETTER_TEMPLATE_LIST.find((s) =>
          s.letterTypeKeywords.some(
            (kw) =>
              item.letterType!.code.toLowerCase().includes(kw) ||
              item.letterType!.name.toLowerCase().includes(kw)
          )
        )
      : undefined);

  const extraKeywords = item.letterType
    ? [
        item.letterType.code.toLowerCase(),
        ...item.letterType.name.toLowerCase().split(/\s+/),
      ]
    : [];

  return {
    id: String(item.id),
    legacyId: matchedStatic?.legacyId ?? matchedStatic?.id,
    name: item.name,
    description: item.description ?? matchedStatic?.description ?? "",
    fileUrl:
      item.fileUrl || `/api/v1/templates/${encodeURIComponent(item.id)}/file`,
    fallbackFileUrl: matchedStatic?.fallbackFileUrl ?? matchedStatic?.fileUrl,
    letterType: item.letterType,
    letterTypeKeywords: Array.from(
      new Set([...(matchedStatic?.letterTypeKeywords ?? []), ...extraKeywords])
    ),
  };
}

function sortTemplatesForDisplay(
  items: LetterTemplateDefinition[]
): LetterTemplateDefinition[] {
  const priorityOrder = [
    "surat dinas resmi standar",
    "nota dinas internal",
    "surat undangan rapat",
    "surat perintah tugas (spt)",
    "surat eksternal",
  ];
  return [...items].sort((a, b) => {
    const aLower = a.name.trim().toLowerCase();
    const bLower = b.name.trim().toLowerCase();
    const aIsBlank =
      aLower.includes("dokumen kosong") || a.legacyId === "tmpl-kosong";
    const bIsBlank =
      bLower.includes("dokumen kosong") || b.legacyId === "tmpl-kosong";
    if (aIsBlank && !bIsBlank) return 1;
    if (!aIsBlank && bIsBlank) return -1;

    const aIdx = priorityOrder.indexOf(aLower);
    const bIdx = priorityOrder.indexOf(bLower);
    if (aIdx !== -1 && bIdx !== -1) return aIdx - bIdx;
    if (aIdx !== -1) return -1;
    if (bIdx !== -1) return 1;
    return a.name.localeCompare(b.name, "id");
  });
}

export function getCachedTemplates(): LetterTemplateDefinition[] {
  return cachedTemplates ?? LETTER_TEMPLATE_LIST;
}

function isBlankTemplate(t: LetterTemplateDefinition): boolean {
  return (
    t.legacyId === "tmpl-kosong" ||
    t.name.trim().toLowerCase().includes("dokumen kosong")
  );
}

function isGenericStandardTemplate(t: LetterTemplateDefinition): boolean {
  return (
    t.legacyId === "tmpl-dinas-standar" ||
    t.name.trim().toLowerCase() === "surat dinas resmi standar"
  );
}

/** Memilih template terbaik yang paling sesuai untuk jenis surat terpilih. */
export function matchTemplateForLetterType(
  templates: LetterTemplateDefinition[],
  letterType: { id?: string | null; code?: string | null; name?: string | null }
): LetterTemplateDefinition | undefined {
  if (templates.length === 0) return undefined;
  const ltId = letterType.id ? String(letterType.id) : "";
  const ltCode = (letterType.code ?? "").trim().toLowerCase();
  const ltName = (letterType.name ?? "").trim().toLowerCase();

  const exactByLink = templates.find(
    (t) =>
      (ltId && t.letterType?.id != null && String(t.letterType.id) === ltId) ||
      (ltCode && t.letterType?.code?.trim().toLowerCase() === ltCode)
  );
  if (exactByLink) return exactByLink;

  const nonGeneric = templates.filter(
    (t) => !isBlankTemplate(t) && !isGenericStandardTemplate(t)
  );

  const bySpecificNameOrKeyword = nonGeneric.find((t) => {
    const tName = t.name.trim().toLowerCase();
    const matchesName =
      (ltName && tName.includes(ltName)) ||
      (ltName && tName.length >= 4 && ltName.includes(tName));
    const matchesKeyword = t.letterTypeKeywords.some(
      (kw) => (ltName && ltName.includes(kw)) || (ltCode && ltCode.includes(kw))
    );
    return matchesName || matchesKeyword;
  });
  if (bySpecificNameOrKeyword) return bySpecificNameOrKeyword;

  return (
    templates.find((t) => !t.letterType && !isBlankTemplate(t)) ??
    templates.find((t) => !isBlankTemplate(t)) ??
    templates[0]
  );
}

/**
 * Mengambil daftar template aktif dari `GET /api/v1/templates`.
 * Bila gagal (misal pada halaman dev tanpa sesi login), kembali ke `LETTER_TEMPLATE_LIST`.
 */
export async function fetchTemplates(
  letterTypeId?: string
): Promise<LetterTemplateDefinition[]> {
  const cleanTypeId = letterTypeId?.trim();
  if (!cleanTypeId && cachedTemplates) {
    return cachedTemplates;
  }
  if (!cleanTypeId && inflightTemplatesPromise) {
    return inflightTemplatesPromise;
  }

  const query = cleanTypeId
    ? `?letterTypeId=${encodeURIComponent(cleanTypeId)}`
    : "";

  const requestPromise = fetch(`/api/v1/templates${query}`, {
    cache: "no-store",
    credentials: "include",
  })
    .then(async (res) => {
      if (!res.ok) return cachedTemplates ?? LETTER_TEMPLATE_LIST;
      const payload = (await res.json()) as { data?: BackendTemplateItem[] };
      const items = payload.data;
      if (!Array.isArray(items) || items.length === 0) {
        return LETTER_TEMPLATE_LIST;
      }
      const mapped = sortTemplatesForDisplay(
        items.map(mapBackendTemplateToDefinition)
      );
      if (!cleanTypeId) {
        cachedTemplates = mapped;
      }
      return mapped;
    })
    .catch(() => cachedTemplates ?? LETTER_TEMPLATE_LIST)
    .finally(() => {
      if (!cleanTypeId) {
        inflightTemplatesPromise = null;
      }
    });

  if (!cleanTypeId) {
    inflightTemplatesPromise = requestPromise;
  }
  return requestPromise;
}

export interface SaveTemplateInput {
  name: string;
  description?: string;
  letterTypeId?: string;
  file?: File;
}

/**
 * Membuat template naskah baru (`POST /api/v1/templates`, `multipart/form-data`).
 */
export async function createTemplate(
  input: SaveTemplateInput & { file: File },
  companyCode?: string
): Promise<LetterTemplateDefinition> {
  const formData = new FormData();
  formData.append("name", input.name.trim());
  if (input.description?.trim()) {
    formData.append("description", input.description.trim());
  }
  if (input.letterTypeId?.trim()) {
    formData.append("letterTypeId", input.letterTypeId.trim());
  }
  formData.append("file", input.file, input.file.name);

  const res = await fetch("/api/v1/templates", {
    method: "POST",
    body: formData,
    credentials: "include",
    headers: companyCode ? { "X-Company-Code": companyCode } : undefined,
  });
  const payload = (await res.json().catch(() => null)) as {
    data?: BackendTemplateItem;
    error?: { message?: string };
  } | null;

  if (!res.ok || !payload?.data) {
    throw new Error(
      payload?.error?.message ?? `Gagal membuat template (HTTP ${res.status})`
    );
  }
  cachedTemplates = null;
  return mapBackendTemplateToDefinition(payload.data);
}

/**
 * Memperbarui template naskah (`PUT /api/v1/templates/{templateId}`, `multipart/form-data`).
 */
export async function updateTemplate(
  templateId: string,
  input: SaveTemplateInput,
  companyCode?: string
): Promise<LetterTemplateDefinition> {
  const formData = new FormData();
  formData.append("name", input.name.trim());
  if (input.description?.trim()) {
    formData.append("description", input.description.trim());
  }
  if (input.letterTypeId?.trim()) {
    formData.append("letterTypeId", input.letterTypeId.trim());
  }
  if (input.file) {
    formData.append("file", input.file, input.file.name);
  }

  const res = await fetch(
    `/api/v1/templates/${encodeURIComponent(templateId)}`,
    {
      method: "PUT",
      body: formData,
      credentials: "include",
      headers: companyCode ? { "X-Company-Code": companyCode } : undefined,
    }
  );
  const payload = (await res.json().catch(() => null)) as {
    data?: BackendTemplateItem;
    error?: { message?: string };
  } | null;

  if (!res.ok || !payload?.data) {
    throw new Error(
      payload?.error?.message ??
        `Gagal memperbarui template (HTTP ${res.status})`
    );
  }
  cachedTemplates = null;
  return mapBackendTemplateToDefinition(payload.data);
}

export function getTemplateById(
  id?: string,
  list: LetterTemplateDefinition[] = getCachedTemplates()
): LetterTemplateDefinition {
  const source = list.length > 0 ? list : LETTER_TEMPLATE_LIST;
  if (!id) return source[0];

  const clean = id.trim();
  const cleanLower = clean.toLowerCase();

  const found =
    source.find((t) => t.id === clean || t.legacyId === clean) ??
    source.find((t) => t.name.trim().toLowerCase() === cleanLower) ??
    source.find((t) => t.letterType?.code?.toLowerCase() === cleanLower);

  if (found) return found;

  if (/^\d+$/.test(clean)) {
    return {
      id: clean,
      name: `Template #${clean}`,
      description: "",
      fileUrl: `/api/v1/templates/${encodeURIComponent(clean)}/file`,
      fallbackFileUrl: LETTER_TEMPLATE_LIST[0].fileUrl,
      letterTypeKeywords: [],
    };
  }

  return source[0];
}

/**
 * Mengambil Blob .docx template dari MinIO (`GET /api/v1/templates/{id}/file`),
 * dengan fallback ke `frontend/public/templates/*.docx` bila API tidak dapat diakses.
 */
export async function resolveTemplateDocxBlob(
  templateId?: string
): Promise<{ blob: Blob; template: LetterTemplateDefinition }> {
  const templates = cachedTemplates ?? (await fetchTemplates());
  const tpl = getTemplateById(templateId, templates);

  try {
    const res = await fetch(tpl.fileUrl, {
      cache: "no-store",
      credentials: "include",
    });

    if (res.ok) {
      const blob = await res.blob();
      return { blob, template: tpl };
    }
  } catch (err) {
    console.warn(`[Template] Gagal mengambil template "${tpl.name}" dari ${tpl.fileUrl}:`, err);
  }

  // Coba file cadangan statis di `frontend/public/templates/*.docx` bila sesuai jenis template
  if (tpl.fallbackFileUrl && tpl.fallbackFileUrl !== tpl.fileUrl) {
    try {
      const fallbackRes = await fetch(tpl.fallbackFileUrl, {
        cache: "no-store",
      });
      if (fallbackRes.ok) {
        const blob = await fallbackRes.blob();
        return {
          blob,
          template: {
            ...tpl,
            fileUrl: tpl.fallbackFileUrl,
          },
        };
      }
    } catch {
      // Lanjut ke dokumen kosong di bawah
    }
  }

  // Fallback otomatis jika berkas template tidak ada di MinIO (mis. terhapus):
  // langsung beralih ke Dokumen Kosong (Blank) agar editor naskah tetap bisa terbuka
  const blankTpl =
    LETTER_TEMPLATE_LIST.find((t) => t.id === "tmpl-kosong") ??
    LETTER_TEMPLATE_LIST[LETTER_TEMPLATE_LIST.length - 1];

  console.warn(
    `[Template] Berkas "${tpl.name}" tidak ditemukan di server/MinIO. Otomatis beralih ke dokumen kosong.`
  );

  const blankRes = await fetch(blankTpl.fileUrl, { cache: "no-store" });
  if (blankRes.ok) {
    const blob = await blankRes.blob();
    return {
      blob,
      template: {
        ...tpl,
        fileUrl: blankTpl.fileUrl,
      },
    };
  }

  throw new Error(
    `Gagal memuat file template "${tpl.name}" dan dokumen kosong cadangan.`
  );
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
 * Menyinkronkan data form (Perihal, Penerima, Tembusan, Lampiran, Pejabat Penandatangan)
 * ke dalam naskah .docx yang sedang dimuat di DevExpress RichEdit.
 * Empat penanda sistem backend (`###*NoSurat*###`, `###*TglSurat*###`, `###*Sifat*###`, `###*QRCode*###`)
 * sengaja TIDAK ditimpa agar tetap diisi secara dinamis oleh backend (`penanda.go`) saat surat disetujui Approver.
 */
export function applyTemplateContextToEditor(
  editor: RichEdit | null | undefined,
  ctx: LetterTemplateContext
) {
  if (!editor || editor.isDisposed) return;

  let subDoc: RichEdit["document"]["subDocuments"]["main"] | undefined;
  try {
    subDoc = editor.document?.subDocuments?.main;
  } catch {
    return;
  }
  if (!subDoc) return;

  try {
    editor.beginUpdate();
    try {
      if (ctx.title?.trim()) {
        replacePlaceholderInSubDoc(subDoc, "###*Perihal*###", ctx.title);
        replacePlaceholderInSubDoc(subDoc, "###Perihal###", ctx.title);
        replacePlaceholderInSubDoc(subDoc, "###*NamaRapat*###", ctx.title);
        replacePlaceholderInSubDoc(subDoc, "###NamaRapat###", ctx.title);
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

      if (ctx.externalRecipient?.name?.trim()) {
        const extName = ctx.externalRecipient.name.trim();
        const extAttention = ctx.externalRecipient.attention?.trim();
        const extLabel = extAttention
          ? `${extName} (u.p. ${extAttention})`
          : extName;
        replacePlaceholderInSubDoc(
          subDoc,
          "Nama perusahaan yang dituju",
          extLabel
        );
        replacePlaceholderInSubDoc(subDoc, "###*Kepada*###", extLabel);
        replacePlaceholderInSubDoc(subDoc, "###Kepada###", extLabel);
      } else if (ctx.recipients && ctx.recipients.length > 0) {
        const recipientText = ctx.recipients.map((r) => r.name).join(", ");
        replacePlaceholderInSubDoc(subDoc, "###*Kepada*###", recipientText);
        replacePlaceholderInSubDoc(subDoc, "###Kepada###", recipientText);
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

      if (ctx.externalRecipient?.address?.trim()) {
        replacePlaceholderInSubDoc(
          subDoc,
          "Alamat Perusahaan yang dituju",
          ctx.externalRecipient.address.trim()
        );
      }

      if (ctx.ccRecipients && ctx.ccRecipients.length > 0) {
        const ccText = ctx.ccRecipients.map((r) => r.name).join(", ");
        replacePlaceholderInSubDoc(subDoc, "###*Tembusan*###", ccText);
        replacePlaceholderInSubDoc(subDoc, "###Tembusan###", ccText);
      }

      if (ctx.attachmentsCount != null) {
        const attText =
          ctx.attachmentsCount > 0 ? `${ctx.attachmentsCount} Berkas` : "-";
        replacePlaceholderInSubDoc(subDoc, "###*Lampiran*###", attText);
        replacePlaceholderInSubDoc(subDoc, "###Lampiran###", attText);
      }

      if (ctx.approver?.name) {
        replacePlaceholderInSubDoc(
          subDoc,
          "###*NamaPenandatangan*###",
          ctx.approver.name
        );
        replacePlaceholderInSubDoc(
          subDoc,
          "###NamaPenandatangan###",
          ctx.approver.name
        );
        replacePlaceholderInSubDoc(
          subDoc,
          "NamaPenandatangan",
          ctx.approver.name
        );
        replacePlaceholderInSubDoc(
          subDoc,
          "[Nama Pejabat Penandatangan]",
          ctx.approver.name
        );
      }

      if (ctx.approver?.position) {
        replacePlaceholderInSubDoc(
          subDoc,
          "###*JabatanPenandatangan*###",
          ctx.approver.position
        );
        replacePlaceholderInSubDoc(
          subDoc,
          "###JabatanPenandatangan###",
          ctx.approver.position
        );
        replacePlaceholderInSubDoc(
          subDoc,
          "JabatanPenandatangan",
          ctx.approver.position
        );
        replacePlaceholderInSubDoc(
          subDoc,
          "[Jabatan Pejabat Penandatangan]",
          ctx.approver.position
        );
      }

      if (ctx.approver?.employeeNumber) {
        replacePlaceholderInSubDoc(
          subDoc,
          "###*NoBadge*###",
          ctx.approver.employeeNumber
        );
        replacePlaceholderInSubDoc(
          subDoc,
          "###NoBadge###",
          ctx.approver.employeeNumber
        );
        replacePlaceholderInSubDoc(
          subDoc,
          "NIP. ..............................",
          `NIP. ${ctx.approver.employeeNumber}`
        );
      }

      if (ctx.drafter?.position) {
        replacePlaceholderInSubDoc(
          subDoc,
          "###*Dari*###",
          ctx.drafter.position
        );
        replacePlaceholderInSubDoc(subDoc, "###Dari###", ctx.drafter.position);
        replacePlaceholderInSubDoc(
          subDoc,
          "Staf Pengembang Sistem / Unit TI",
          ctx.drafter.position
        );
      }
    } finally {
      editor.endUpdate();
    }
  } catch {
    // Abaikan jika instance RichEdit sudah ditutup saat callback sinkronisasi berjalan
  }
}
