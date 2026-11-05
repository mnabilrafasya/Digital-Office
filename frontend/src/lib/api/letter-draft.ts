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
import type { LetterDetail } from "./letter-detail";

const API = "/api/v1";

export interface SaveDraftResult {
  id: string;
  etag: string | null;
  detail: LetterDetail;
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

  const approverId =
    values.approvers &&
    values.approvers.length > 0 &&
    values.approvers[0].employeeId &&
    /^\d+$/.test(String(values.approvers[0].employeeId).trim())
      ? String(values.approvers[0].employeeId).trim()
      : undefined;

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

  return {
    letterTypeId: resolvedLetterTypeId,
    templateId: isTemplateTypeMismatch ? null : rawTemplateId,
    classificationId:
      values.classificationId && /^\d+$/.test(String(values.classificationId).trim())
        ? String(values.classificationId).trim()
        : null,
    confidentialityId: confidentialityId ?? null,
    urgencyId: urgencyId ?? null,
    title: values.title?.trim() || null,
    summary: values.summary?.trim() || null,
    specialNote: values.specialNote?.trim() || null,
    reviewerIds,
    approverId: approverId ?? null,
    recipients: {
      to: toRecipients,
      cc: ccRecipients,
    },
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
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error?.message ?? `Gagal membuat draf (HTTP ${res.status})`);
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

    // Jika server merespons 428 (Precondition Required) atau 409 (Conflict),
    // otomatis ambil ETag mutakhir lalu coba kirim kembali
    if (res.status === 428 || res.status === 409) {
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
        // Abaikan kegagalan refresh, lanjutkan ke penanganan error standar
      }
    }

    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error?.message ?? `Gagal memperbarui draf (HTTP ${res.status})`);
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
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error?.message ?? `Gagal mengirim surat (HTTP ${res.status})`);
  }

  const json = (await res.json()) as { data: LetterDetail };
  return json.data;
}

/** Mengambil detail draf surat beserta ETag-nya */
export async function getDraftLetter(letterId: string): Promise<{ detail: LetterDetail; etag: string | null }> {
  const res = await fetch(`${API}/letters/${letterId}`, {
    method: "GET",
    headers: { "Content-Type": "application/json" },
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error?.message ?? `Gagal memuat surat (HTTP ${res.status})`);
  }

  const etag = res.headers.get("etag") || res.headers.get("x-etag");
  const json = (await res.json()) as { data: LetterDetail };
  return { detail: json.data, etag };
}
