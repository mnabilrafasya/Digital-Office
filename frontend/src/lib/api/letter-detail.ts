// ─────────────────────────────────────────────
// lib/api/letter-detail.ts
// Integrasi nyata ke backend  GET /api/v1/letters/{id}
// POST /api/v1/letters/{id}/approve | return | submit
// ─────────────────────────────────────────────

// ─── Tipe sesuai model backend (model/letter.go) ───

export interface LookupRef {
  id: string;     // backend: int64 json:"id,string"
  code: string;
  name: string;
}

export interface UnitRef {
  id: string;
  code: string;
  name: string;
}

export interface PersonRef {
  id: string;
  name: string;
  positionName: string;
}

export interface StepUser {
  id: string;
  name: string;
  positionName: string;
  unitName: string;
}

export type StepState = "DONE" | "ACTIVE" | "PENDING";

export interface LetterStep {
  role: "REVIEWER" | "APPROVER";
  order: number;
  state: StepState;
  /** Catatan pengembalian — ada bila step pernah mengembalikan surat */
  returnNote?: string;
  user: StepUser;
}

export interface RecipientUnit {
  type: "UNIT";
  unit: UnitRef;
  resolvedHead: PersonRef | null;
}

export interface RecipientAlias {
  type: "ALIAS";
  alias: {
    levelId: string;
    code: string;
    label: string;
    memberCount: number;
  };
}

export type LetterRecipient = RecipientUnit | RecipientAlias;

export interface Attachment {
  id: string;
  fileName: string;
  size: number;
  contentType: string;
  uploadedAt: string;
  uploadedBy: { id: string; name: string };
}

export interface LetterDetail {
  id: string;
  status: string;
  letterNumber: string | null;
  letterDate: string | null;
  title: string | null;
  summary: string | null;
  specialNote: string | null;
  letterType: LookupRef | null;
  template?: LookupRef | null;
  classification: LookupRef | null;
  confidentiality: LookupRef | null;
  urgency: LookupRef | null;
  senderUnit: UnitRef;
  drafter: PersonRef;
  steps: LetterStep[];
  recipients: { to: LetterRecipient[]; cc: LetterRecipient[] };
  attachments: Attachment[];
  currentVersion: number;
  currentAssignee: PersonRef | null;
  lastReturnNote: string | null;
  allowedActions: string[];
  createdAt: string;
  updatedAt: string;
}

// ─── Helper fetch ────────────────────────────────

const API = "/api/v1";

async function apiFetch<T>(
  path: string,
  init?: RequestInit
): Promise<{ data: T; etag?: string }> {
  const res = await fetch(`${API}${path}`, {
    cache: "no-store",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({})) as {
      error?: { message?: string };
    };
    throw new Error(
      body.error?.message ?? `HTTP ${res.status}`
    );
  }

  const json = (await res.json()) as { data: T };
  const etag = res.headers.get("ETag") ?? undefined;
  return { data: json.data, etag };
}

// ─── GET detail surat ────────────────────────────

export async function getLetter(id: string): Promise<LetterDetail | null> {
  try {
    const { data } = await apiFetch<LetterDetail>(`/letters/${encodeURIComponent(id)}`);
    return data;
  } catch (e) {
    const err = e as Error;
    if (err.message.startsWith("HTTP 404")) return null;
    throw e;
  }
}

// ─── Aksi workflow ───────────────────────────────

export type ReviewActionType = "APPROVE" | "RETURN" | "SUBMIT" | "CANCEL";

export interface ReviewActionResult {
  ok: boolean;
  data?: LetterDetail;
  error?: string;
}

/**
 * Panggil endpoint workflow:
 *  - APPROVE → POST /letters/{id}/approve   { note? }
 *  - RETURN  → POST /letters/{id}/return    { note }
 *  - SUBMIT  → POST /letters/{id}/submit    (tanpa body)
 */
export async function reviewAction(
  id: string,
  action: ReviewActionType,
  note?: string
): Promise<ReviewActionResult> {
  const endpoint =
    action === "APPROVE" ? "approve"
    : action === "RETURN" ? "return"
    : action === "CANCEL" ? "cancel"
    : "submit";

  const hasBody = action !== "SUBMIT";
  const body    = hasBody ? JSON.stringify({ note: note ?? null }) : undefined;

  try {
    const { data } = await apiFetch<LetterDetail>(
      `/letters/${encodeURIComponent(id)}/${endpoint}`,
      { method: "POST", body }
    );
    return { ok: true, data };
  } catch (e) {
    const err = e as Error;
    return { ok: false, error: err.message };
  }
}

// ─── Util ────────────────────────────────────────

export function formatBytes(bytes: number): string {
  if (bytes < 1024)            return `${bytes} B`;
  if (bytes < 1024 * 1024)     return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

/**
 * Cek apakah surat ini dalam status "perlu diedit drafter".
 * Drafter harus diarahkan ke form edit, bukan halaman review.
 */
export function isEditableByDrafter(status: string, allowedActions: string[]): boolean {
  return (
    (status === "DRAFT" || status === "RETURNED_TO_DRAFTER") &&
    allowedActions.includes("EDIT")
  );
}
