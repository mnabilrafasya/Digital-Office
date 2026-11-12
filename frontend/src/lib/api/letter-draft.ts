// ─────────────────────────────────────────────
// lib/api/letter-draft.ts
// Client API untuk Buat Surat & Draft:
// - POST /api/v1/letters (buat draf baru)
// - PUT /api/v1/letters/{id} (update draf dengan ETag)
// - POST /api/v1/letters/{id}/submit (kirim surat ke reviewer)
// - GET /api/v1/letters/{id} (ambil draf untuk diedit)
// ─────────────────────────────────────────────

import { fetchLookups } from "./form";
import { getCachedTemplates } from "@/lib/templates/letter-templates";
import type { BuatSuratFormValues } from "@/lib/schemas";
import type { LetterAttachment, StepEntry } from "@/lib/types";
import type { LetterDetail } from "./letter-detail";

const API = "/api/v1";

export interface SaveDraftResult {
  id: string;
  etag: string | null;
  detail: LetterDetail;
}

/** Error khusus untuk konflik versi (412) atau status tidak valid (409) saat simpan draf */
export class DraftSaveError extends Error {
  status: number;
  code?: string;
  latestDetail?: LetterDetail;
  latestEtag?: string | null;

  constructor(
    message: string,
    status: number,
    options?: {
      code?: string;
      latestDetail?: LetterDetail;
      latestEtag?: string | null;
    }
  ) {
    super(message);
    this.name = "DraftSaveError";
    this.status = status;
    this.code = options?.code;
    this.latestDetail = options?.latestDetail;
    this.latestEtag = options?.latestEtag;
  }
}

/** Cache lookup ID untuk sifat & kecepatan agar tidak fetch berulang kali */
let lookupCache: {
  confidentiality: Record<string, string>;
  urgency: Record<string, string>;
} | null = null;

async function getLookupIdMap() {
  if (
    lookupCache &&
    Object.keys(lookupCache.confidentiality).length > 0 &&
    Object.keys(lookupCache.urgency).length > 0
  ) {
    return lookupCache;
  }

  const [confList, urgList] = await Promise.all([
    fetchLookups("CONFIDENTIALITY").catch(() => []),
    fetchLookups("URGENCY").catch(() => []),
  ]);

  const confMap: Record<string, string> = {};
  for (const c of confList) {
    confMap[c.code.toUpperCase()] = String(c.id);
  }

  const urgMap: Record<string, string> = {};
  for (const u of urgList) {
    urgMap[u.code.toUpperCase()] = String(u.id);
  }

  if (Object.keys(confMap).length > 0 || Object.keys(urgMap).length > 0) {
    lookupCache = { confidentiality: confMap, urgency: urgMap };
  }
  return { confidentiality: confMap, urgency: urgMap };
}

export interface FormInputValues {
  letterTypeId?: string | null;
  letterTypeCode?: string | null;
  templateId?: string | null;
  classificationId?: string | null;
  kecepatan?: string | null;
  sifat?: string | null;
  title?: string | null;
  summary?: string | null;
  specialNote?: string | null;
  reviewers?: Array<{ employeeId?: string; [key: string]: unknown }> | null;
  approvers?: Array<{ employeeId?: string; [key: string]: unknown }> | null;
  toUnits?: Array<{ id?: string; [key: string]: unknown }> | null;
  ccUnits?: Array<{ id?: string; [key: string]: unknown }> | null;
  externalRecipient?: {
    name?: string | null;
    address?: string | null;
    attention?: string | null;
  } | null;
}

function mapRecipientInput(
  entry?: { id?: string; [key: string]: unknown } | null
): { unitId: string } | { aliasLevelId: string } | null {
  if (!entry || !entry.id) return null;
  const rawId = String(entry.id).trim();
  const aliasMatch = /^alias-(\d+)$/.exec(rawId);
  if (aliasMatch) {
    return { aliasLevelId: aliasMatch[1] };
  }
  if (/^\d+$/.test(rawId)) {
    return { unitId: rawId };
  }
  return null;
}

/**
 * Mengonversi detail surat dari backend (`LetterDetail`) menjadi nilai form `BuatSuratFormValues`
 * dan daftar lampiran `LetterAttachment[]` (menghindari duplikasi kode di halaman Buat Surat).
 */
export function detailToFormValues(detail: LetterDetail): {
  formValues: BuatSuratFormValues;
  attachments: LetterAttachment[];
} {
  const reviewers: StepEntry[] = (detail.steps ?? [])
    .filter((s) => s.role === "REVIEWER")
    .sort((a, b) => a.order - b.order)
    .map((s, idx) => ({
      id: `rev-${s.user.id || idx}-${s.order}`,
      employeeId: String(s.user.id),
      employeeName: s.user.name,
      jabatan: s.user.positionName,
      unitName: s.user.unitName,
      kodeAlias: "R",
    }));

  const approvers: StepEntry[] = (detail.steps ?? [])
    .filter((s) => s.role === "APPROVER")
    .sort((a, b) => a.order - b.order)
    .map((s, idx) => ({
      id: `app-${s.user.id || idx}-${s.order}`,
      employeeId: String(s.user.id),
      employeeName: s.user.name,
      jabatan: s.user.positionName,
      unitName: s.user.unitName,
      kodeAlias: "A",
    }));

  const toUnits = (detail.recipients?.to ?? [])
    .map((r) => {
      if (r.type === "UNIT" && r.unit) {
        return { id: String(r.unit.id), name: r.unit.name };
      }
      if (r.type === "ALIAS" && r.alias) {
        return { id: `alias-${r.alias.levelId}`, name: r.alias.label };
      }
      return null;
    })
    .filter((u): u is { id: string; name: string } => u !== null);

  const ccUnits = (detail.recipients?.cc ?? [])
    .map((r) => {
      if (r.type === "UNIT" && r.unit) {
        return { id: String(r.unit.id), name: r.unit.name };
      }
      if (r.type === "ALIAS" && r.alias) {
        return { id: `alias-${r.alias.levelId}`, name: r.alias.label };
      }
      return null;
    })
    .filter((u): u is { id: string; name: string } => u !== null);

  const formValues: BuatSuratFormValues = {
    letterTypeId: detail.letterType?.id ? String(detail.letterType.id) : "",
    letterTypeCode: detail.letterType?.code ?? "",
    templateId: detail.template?.id ? String(detail.template.id) : undefined,
    classificationId: detail.classification?.id
      ? String(detail.classification.id)
      : "",
    kecepatan:
      (detail.urgency?.code as "BIASA" | "SEGERA" | "SANGAT_SEGERA") || "BIASA",
    sifat:
      (detail.confidentiality?.code as "BIASA" | "TERBATAS" | "RAHASIA") ||
      "BIASA",
    title: detail.title ?? "",
    specialNote: detail.specialNote ?? "",
    summary: detail.summary ?? "",
    reviewers,
    approvers,
    toUnits,
    ccUnits,
    externalRecipient: {
      name: detail.external?.recipient?.name ?? "",
      address: detail.external?.recipient?.address ?? "",
      attention: detail.external?.recipient?.attention ?? "",
    },
  };

  const attachments: LetterAttachment[] = (detail.attachments ?? []).map(
    (a) => ({
      id: a.id,
      fileName: a.fileName,
      sizeBytes: a.size,
      mimeType: a.contentType,
      uploadedAt: a.uploadedAt,
    })
  );

  return { formValues, attachments };
}

/** Mengonversi nilai form BuatSuratFormValues ke payload LetterInput backend */
export async function formValuesToLetterInput(values: FormInputValues) {
  const { confidentiality: confMap, urgency: urgMap } = await getLookupIdMap();

  const confidentialityId = values.sifat
    ? /^\d+$/.test(String(values.sifat).trim())
      ? String(values.sifat).trim()
      : confMap[values.sifat.toUpperCase()]
    : undefined;

  const urgencyId = values.kecepatan
    ? /^\d+$/.test(String(values.kecepatan).trim())
      ? String(values.kecepatan).trim()
      : urgMap[values.kecepatan.toUpperCase()]
    : undefined;

  const reviewerIds = (values.reviewers ?? [])
    .map((r) => String(r.employeeId).trim())
    .filter((id) => id && /^\d+$/.test(id));

  const approverIds = (values.approvers ?? [])
    .map((a) => String(a.employeeId).trim())
    .filter((id) => id && /^\d+$/.test(id));

  const toRecipients = (values.toUnits ?? [])
    .map(mapRecipientInput)
    .filter(
      (r): r is { unitId: string } | { aliasLevelId: string } => r !== null
    );

  const ccRecipients = (values.ccUnits ?? [])
    .map(mapRecipientInput)
    .filter(
      (r): r is { unitId: string } | { aliasLevelId: string } => r !== null
    );

  const resolvedLetterTypeId =
    values.letterTypeId && /^\d+$/.test(String(values.letterTypeId).trim())
      ? String(values.letterTypeId).trim()
      : null;

  const rawTemplateId =
    values.templateId && /^\d+$/.test(String(values.templateId).trim())
      ? String(values.templateId).trim()
      : null;

  // Jika pengguna memilih kombinasi Jenis Surat dan Template dari jenis surat berbeda
  // (mis. Memo Internal + template Nota Dinas Internal), tetap izinkan dokumen .docx
  // dipakai di editor, tetapi kirim templateId: null ke backend agar tidak ditolak checkTemplate
  const selectedTpl = rawTemplateId
    ? getCachedTemplates().find((t) => String(t.id) === rawTemplateId)
    : undefined;
  const isTemplateTypeMismatch = Boolean(
    selectedTpl?.letterType?.id &&
      resolvedLetterTypeId &&
      String(selectedTpl.letterType.id) !== resolvedLetterTypeId
  );

  const isExternalOutgoing =
    (values.letterTypeCode ?? "").toUpperCase() === "SURAT_EKSTERNAL";

  const extName = values.externalRecipient?.name?.trim() || "";
  const extAddress = values.externalRecipient?.address?.trim() || null;
  const extAttention = values.externalRecipient?.attention?.trim() || null;

  return {
    letterTypeId: resolvedLetterTypeId,
    templateId: isTemplateTypeMismatch ? null : rawTemplateId,
    classificationId:
      values.classificationId &&
      /^\d+$/.test(String(values.classificationId).trim())
        ? String(values.classificationId).trim()
        : null,
    confidentialityId: confidentialityId ?? null,
    urgencyId: urgencyId ?? null,
    title: values.title?.trim() || null,
    summary: values.summary?.trim() || null,
    specialNote: values.specialNote?.trim() || null,
    reviewerIds,
    approverIds,
    recipients: isExternalOutgoing
      ? { to: [], cc: [] }
      : {
          to: toRecipients,
          cc: ccRecipients,
        },
    externalRecipient: isExternalOutgoing
      ? {
          name: extName,
          address: extAddress,
          attention: extAttention,
        }
      : undefined,
  };
}

/** Menyimpan draft surat (POST jika baru, PUT jika sudah ada ID + ETag) */
export async function saveDraft(
  values: FormInputValues,
  draftId?: string | null,
  currentEtag?: string | null
): Promise<SaveDraftResult> {
  const payload = await formValuesToLetterInput(values);

  if (!draftId) {
    // POST /api/v1/letters
    const res = await fetch(`${API}/letters`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    if (!res.ok) {
      const err = (await res.json().catch(() => ({}))) as {
        error?: { code?: string; message?: string };
      };
      throw new DraftSaveError(
        err.error?.message ?? `Gagal membuat draf (HTTP ${res.status})`,
        res.status,
        { code: err.error?.code }
      );
    }

    const etag = res.headers.get("etag") || res.headers.get("x-etag");
    const json = (await res.json()) as { data: LetterDetail };
    return { id: json.data.id, etag, detail: json.data };
  } else {
    // PUT /api/v1/letters/{draftId}
    let etagToUse = currentEtag;

    // Jika ETag belum tersedia, fetch data terbaru dari server untuk memperoleh ETag
    if (!etagToUse) {
      try {
        const current = await getDraftLetter(draftId);
        etagToUse = current.etag;
      } catch (err) {
        console.warn("Gagal mengambil ETag sebelum update draf:", err);
      }
    }

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };
    if (etagToUse) {
      headers["If-Match"] = etagToUse;
    }

    let res = await fetch(`${API}/letters/${draftId}`, {
      method: "PUT",
      headers,
      body: JSON.stringify(payload),
    });

    // Hanya untuk 428 (Precondition Required / belum ada If-Match):
    // ambil ETag mutakhir lalu coba kirim kembali sekali
    if (res.status === 428) {
      try {
        const fresh = await getDraftLetter(draftId);
        if (fresh.etag) {
          headers["If-Match"] = fresh.etag;
          res = await fetch(`${API}/letters/${draftId}`, {
            method: "PUT",
            headers,
            body: JSON.stringify(payload),
          });
        }
      } catch {
        // Lanjutkan ke penanganan error di bawah
      }
    }

    // 412 Precondition Failed (VERSION_CONFLICT):
    // Jangan menimpa diam-diam. Ambil detail & ETag terbaru lalu beri tahu pengguna.
    if (res.status === 412) {
      const err = (await res.json().catch(() => ({}))) as {
        error?: { code?: string; message?: string };
      };
      const fresh = await getDraftLetter(draftId).catch(() => null);
      throw new DraftSaveError(
        err.error?.message ??
          "Draf telah diubah di sesi atau tab lain (412). Data terbaru telah dimuat, silakan periksa kembali sebelum menyimpan.",
        412,
        {
          code: err.error?.code ?? "VERSION_CONFLICT",
          latestDetail: fresh?.detail,
          latestEtag: fresh?.etag ?? null,
        }
      );
    }

    // 409 Conflict (INVALID_STATE):
    // Surat sudah diproses (bukan DRAFT / RETURNED_TO_DRAFTER), jangan di-retry PUT.
    if (res.status === 409) {
      const err = (await res.json().catch(() => ({}))) as {
        error?: { code?: string; message?: string };
      };
      const fresh = await getDraftLetter(draftId).catch(() => null);
      throw new DraftSaveError(
        err.error?.message ??
          "Surat ini sedang diproses dan tidak lagi dalam status draf yang dapat diedit.",
        409,
        {
          code: err.error?.code ?? "INVALID_STATE",
          latestDetail: fresh?.detail,
          latestEtag: fresh?.etag ?? null,
        }
      );
    }

    if (!res.ok) {
      const err = (await res.json().catch(() => ({}))) as {
        error?: { code?: string; message?: string };
      };
      throw new DraftSaveError(
        err.error?.message ?? `Gagal memperbarui draf (HTTP ${res.status})`,
        res.status,
        { code: err.error?.code }
      );
    }

    const etag = res.headers.get("etag") || res.headers.get("x-etag");
    const json = (await res.json()) as { data: LetterDetail };
    return { id: draftId, etag, detail: json.data };
  }
}

/** Mengirim draft ke Reviewer pertama (POST /api/v1/letters/{id}/submit) */
export async function submitLetter(letterId: string): Promise<LetterDetail> {
  const res = await fetch(`${API}/letters/${letterId}/submit`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as {
      error?: { message?: string };
    };
    throw new Error(
      err.error?.message ?? `Gagal mengirim surat (HTTP ${res.status})`
    );
  }

  const json = (await res.json()) as { data: LetterDetail };
  return json.data;
}

/** Mengambil detail draf surat beserta ETag-nya */
export async function getDraftLetter(
  letterId: string
): Promise<{ detail: LetterDetail; etag: string | null }> {
  const res = await fetch(`${API}/letters/${letterId}`, {
    method: "GET",
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as {
      error?: { message?: string };
    };
    throw new Error(
      err.error?.message ?? `Gagal memuat surat (HTTP ${res.status})`
    );
  }

  const etag = res.headers.get("etag") || res.headers.get("x-etag");
  const json = (await res.json()) as { data: LetterDetail };
  return { detail: json.data, etag };
}

// ─── Surat Masuk Eksternal (POST /api/v1/letters/incoming) ───

export interface CreateIncomingLetterInput {
  file: File;
  title: string;
  sender: string;
  referenceNumber: string;
  letterDate: string; // YYYY-MM-DD
  userIds?: string[];
  aliasLevelIds?: string[];
  note?: string;
}

/**
 * Mencatat surat masuk eksternal beserta scan PDF dan mendisposisikannya
 * ke penerima internal (`POST /api/v1/letters/incoming`).
 */
export async function createIncomingLetter(
  input: CreateIncomingLetterInput
): Promise<LetterDetail> {
  const form = new FormData();
  form.append("file", input.file, input.file.name || "Surat_Masuk_Eksternal.pdf");
  form.append("title", input.title.trim());
  form.append("sender", input.sender.trim());
  form.append("referenceNumber", input.referenceNumber.trim());
  form.append("letterDate", input.letterDate.trim());
  if (input.userIds && input.userIds.length > 0) {
    form.append("userIds", input.userIds.join(","));
  }
  if (input.aliasLevelIds && input.aliasLevelIds.length > 0) {
    form.append("aliasLevelIds", input.aliasLevelIds.join(","));
  }
  if (input.note && input.note.trim()) {
    form.append("note", input.note.trim());
  }

  const res = await fetch(`${API}/letters/incoming`, {
    method: "POST",
    body: form,
    credentials: "include",
  });

  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as {
      error?: { message?: string };
    };
    throw new Error(
      err.error?.message ?? `Gagal mencatat surat masuk eksternal (HTTP ${res.status})`
    );
  }

  const json = (await res.json()) as { data: LetterDetail };
  return json.data;
}

