// ─────────────────────────────────────────────
// lib/api/form.ts
// Fungsi data untuk form Buat Surat.
// Semua pemanggilan lewat proxy Next (/api/v1/...) sehingga
// token tidak perlu diletakkan di JS browser.
// Sumber kebenaran: backend/docs/api.md
// ─────────────────────────────────────────────

// ─── Tipe respons backend ────────────────────

export interface LookupItem {
  id: string;
  code: string;
  name: string;
  note: string | null;
  sortOrder: number;
}

export interface UnitItem {
  id: string;
  code: string;
  name: string;
  unitType: string;
  parentId: string | null;
  headPositionName: string | null;
}

export interface CandidateItem {
  id: string;
  name: string;
  employeeNumber: string;
  positionName: string;
  unitName: string;
  kodeAlias?: "R" | "A" | string;
  kode_alias?: "R" | "A" | string;
}

// ─── Helper fetch ────────────────────────────

/** Prefix proxy: semua request lewat /api/ (route handler Next). */
const API = "/api/v1";

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${API}${path}`, { cache: "no-store" });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(
      (body as { error?: { message?: string } }).error?.message ??
        `HTTP ${res.status}`
    );
  }
  const json = (await res.json()) as { data: T };
  return json.data;
}

// ─── Lookup ──────────────────────────────────

type LookupCategory =
  | "LETTER_TYPE"
  | "CONFIDENTIALITY"
  | "URGENCY"
  | "CLASSIFICATION"
  | "LETTER_STATUS"
  | "INBOX_STATUS";

export async function fetchLookups(
  category: LookupCategory
): Promise<LookupItem[]> {
  return get<LookupItem[]>(`/lookups?category=${category}`);
}

// ─── Units ───────────────────────────────────

export async function fetchUnits(q?: string): Promise<UnitItem[]> {
  const search = q ? `?q=${encodeURIComponent(q)}` : "";
  return get<UnitItem[]>(`/units${search}`);
}

// ─── Candidates (Reviewer / Approver) ────────

export interface CandidateQuery {
  type: "REVIEWER" | "APPROVER" | "R" | "A" | string;
  kodeAlias?: "R" | "A" | string;
  kode_alias?: "R" | "A" | string;
  q?: string;
  limit?: number;
}

export async function fetchCandidates(
  query: CandidateQuery
): Promise<CandidateItem[]> {
  // Normalisasi Kode Alias / type: 'R' -> 'REVIEWER', 'A' -> 'APPROVER'
  const rawType = (query.kodeAlias || query.kode_alias || query.type || "REVIEWER").toUpperCase();
  const normalizedType =
    rawType === "A" || rawType === "APPROVER" ? "APPROVER" : "REVIEWER";

  const params = new URLSearchParams({ type: normalizedType });
  if (query.q) params.set("q", query.q);
  if (query.limit) params.set("limit", String(query.limit));
  return get<CandidateItem[]>(`/users/candidates?${params.toString()}`);
}
