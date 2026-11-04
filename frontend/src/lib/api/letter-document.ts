// lib/api/letter-document.ts
// Klien API untuk naskah surat (.docx per versi), PDF resmi, dan lampiran surat (MinIO):
// - PUT /api/v1/letters/{id}/document (unggah .docx sebagai versi baru)
// - GET /api/v1/letters/{id}/document?version= (unduh .docx atau PDF resmi saat SENT)
// - GET /api/v1/letters/{id}/versions (daftar riwayat versi naskah)
// - POST /api/v1/letters/{id}/attachments (unggah lampiran pendukung + ETag baru)
// - GET /api/v1/letters/{id}/attachments/{attachmentId} (unduh lampiran pendukung)
// - DELETE /api/v1/letters/{id}/attachments/{attachmentId} (hapus lampiran + ETag baru)

import type { LetterAttachment } from "@/lib/types";

const API = "/api/v1";

export interface DocumentVersion {
  versionNumber: number;
  type: "DRAFT" | "FINAL";
  fileName: string;
  fileExtension: string;
  fileSize: number;
  checksum: string;
  createdBy: {
    id: string;
    name: string;
    positionName: string;
  };
  createdByRole: "DRAFTER" | "REVIEWER" | "APPROVER";
  createdAt: string;
}

export interface UploadDocumentResult {
  version: DocumentVersion;
  etag: string | null;
}

export interface BackendAttachmentItem {
  id: string;
  fileName: string;
  size: number;
  contentType: string;
  checksum?: string;
  uploadedAt: string;
  uploadedBy: {
    id: string;
    name: string;
  };
}

export interface UploadAttachmentResult {
  attachment: LetterAttachment;
  etag: string | null;
}

/**
 * Mengunggah file/blob .docx dari editor ke server sebagai versi baru.
 * Mengembalikan versi baru serta ETag surat terbaru (penting untuk PUT /letters/{id} berikutnya).
 */
export async function uploadLetterDocument(
  letterId: string,
  fileOrBlob: Blob,
  fileName = "Naskah_Dinas.docx"
): Promise<UploadDocumentResult> {
  const safeName = fileName.toLowerCase().endsWith(".docx")
    ? fileName
    : `${fileName}.docx`;

  const file =
    fileOrBlob instanceof File
      ? fileOrBlob
      : new File([fileOrBlob], safeName, {
          type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        });

  const form = new FormData();
  form.append("file", file, safeName);

  const res = await fetch(
    `${API}/letters/${encodeURIComponent(letterId)}/document`,
    {
      method: "PUT",
      body: form,
      credentials: "include",
    }
  );

  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as {
      error?: { message?: string };
    };
    throw new Error(
      err.error?.message ?? `Gagal menyimpan naskah .docx (HTTP ${res.status})`
    );
  }

  const etag = res.headers.get("etag") || res.headers.get("x-etag");
  const json = (await res.json()) as { data: DocumentVersion };
  return { version: json.data, etag };
}

/**
 * Mengunduh naskah .docx versi terbaru (atau nomor versi tertentu) dari server
 * untuk dibuka di dalam DevExpress RichEdit. Mengembalikan null bila naskah belum pernah diunggah (404).
 */
export async function fetchLetterDocumentBlob(
  letterId: string,
  version?: number
): Promise<{ blob: Blob; fileName: string } | null> {
  const qs = version && version >= 1 ? `?version=${version}` : "";
  const res = await fetch(
    `${API}/letters/${encodeURIComponent(letterId)}/document${qs}`,
    { cache: "no-store", credentials: "include" }
  );

  if (res.status === 404) {
    return null;
  }

  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as {
      error?: { message?: string };
    };
    throw new Error(
      err.error?.message ?? `Gagal mengunduh naskah (HTTP ${res.status})`
    );
  }

  const disposition = res.headers.get("content-disposition") ?? "";
  const match = /filename="?([^";]+)"?/i.exec(disposition);
  const fileName = match?.[1]?.trim() || "Naskah_Dinas.docx";
  const blob = await res.blob();
  return { blob, fileName };
}

/**
 * Mengambil daftar riwayat versi naskah surat (terbaru dulu).
 */
export async function getLetterVersions(
  letterId: string
): Promise<DocumentVersion[]> {
  try {
    const res = await fetch(
      `${API}/letters/${encodeURIComponent(letterId)}/versions`,
      { cache: "no-store", credentials: "include" }
    );
    if (!res.ok) return [];
    const json = (await res.json()) as { data: DocumentVersion[] };
    return json.data ?? [];
  } catch {
    return [];
  }
}

/**
 * Mengunggah satu file lampiran pendukung ke MinIO (`POST /api/v1/letters/{letterId}/attachments`).
 * Mengembalikan objek `LetterAttachment` dan `ETag` terbaru surat.
 */
export async function uploadLetterAttachment(
  letterId: string,
  file: File
): Promise<UploadAttachmentResult> {
  const form = new FormData();
  form.append("file", file, file.name);

  const res = await fetch(
    `${API}/letters/${encodeURIComponent(letterId)}/attachments`,
    {
      method: "POST",
      body: form,
      credentials: "include",
    }
  );

  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as {
      error?: { message?: string };
    };
    throw new Error(
      err.error?.message ?? `Gagal mengunggah lampiran (HTTP ${res.status})`
    );
  }

  const etag = res.headers.get("etag") || res.headers.get("x-etag");
  const json = (await res.json()) as { data: BackendAttachmentItem };
  const item = json.data;

  return {
    attachment: {
      id: item.id,
      fileName: item.fileName,
      mimeType: item.contentType,
      sizeBytes: item.size,
      uploadedAt: item.uploadedAt,
    },
    etag,
  };
}

/**
 * Menghapus satu lampiran surat dari MinIO (`DELETE /api/v1/letters/{letterId}/attachments/{attachmentId}`).
 * Mengembalikan `ETag` terbaru surat dari response `204 No Content`.
 */
export async function deleteLetterAttachment(
  letterId: string,
  attachmentId: string
): Promise<{ etag: string | null }> {
  const res = await fetch(
    `${API}/letters/${encodeURIComponent(letterId)}/attachments/${encodeURIComponent(attachmentId)}`,
    {
      method: "DELETE",
      credentials: "include",
    }
  );

  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as {
      error?: { message?: string };
    };
    throw new Error(
      err.error?.message ?? `Gagal menghapus lampiran (HTTP ${res.status})`
    );
  }

  const etag = res.headers.get("etag") || res.headers.get("x-etag");
  return { etag };
}

/**
 * URL unduhan langsung untuk berkas lampiran surat (`GET /api/v1/letters/{letterId}/attachments/{attachmentId}`).
 */
export function getLetterAttachmentDownloadUrl(
  letterId: string,
  attachmentId: string
): string {
  return `${API}/letters/${encodeURIComponent(letterId)}/attachments/${encodeURIComponent(attachmentId)}`;
}
