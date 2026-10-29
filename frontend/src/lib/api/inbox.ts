// ─────────────────────────────────────────────
// lib/api/inbox.ts
// Integrasi ke backend: GET /api/v1/inbox, GET /inbox/{itemId},
// POST /inbox/{itemId}/disposition | redirect
// ─────────────────────────────────────────────

import type { LookupRef, UnitRef, PersonRef } from "./letter-detail";

// ─── Tipe ────────────────────────────────────

export interface InboxSource {
  type: "DIRECT" | "FORWARD";
  recipientType: "TO" | "CC" | null;
  forwardType: "DISPOSITION" | "REDIRECT" | null;
  forwardedBy: PersonRef | null;
  note: string | null;
}

export interface InboxLetter {
  id: string;
  letterNumber: string;
  title: string | null;
  letterDate: string;
  letterType: LookupRef;
  confidentiality: LookupRef;
  urgency: LookupRef;
  senderUnit: UnitRef;
  approver: PersonRef;
}

export interface InboxItem {
  id: string;
  status: "UNREAD" | "READ" | "DISPOSED" | "REDIRECTED";
  receivedAt: string;
  readAt: string | null;
  source: InboxSource;
  letter: InboxLetter;
}

export interface ForwardRecord {
  id: string;
  type: "DISPOSITION" | "REDIRECT";
  targetType: "USER" | "UNIT";
  targetUser: PersonRef | null;
  targetUnit: UnitRef | null;
  recipient: PersonRef;
  note: string | null;
  createdAt: string;
}

export interface InboxItemDetail extends InboxItem {
  forwards: ForwardRecord[];
  allowedActions: ("DISPOSE" | "REDIRECT")[];
}

export interface InboxCounts {
  UNREAD: number;
  READ: number;
  DISPOSED: number;
  REDIRECTED: number;
}

export interface PaginationMeta {
  page: number;
  pageSize: number;
  total: number;
}

export interface InboxListResponse {
  data: InboxItem[];
  meta: PaginationMeta;
  counts: InboxCounts;
}

// ─── Helper ──────────────────────────────────

const API = "/api/v1";

async function apiFetch<T>(path: string, init?: RequestInit): Promise<{ data: T }> {
  const res = await fetch(`${API}${path}`, {
    cache: "no-store",
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
    ...init,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({})) as { error?: { message?: string } };
    throw new Error(body.error?.message ?? `HTTP ${res.status}`);
  }
  const json = (await res.json()) as { data: T; meta?: PaginationMeta; counts?: InboxCounts };
  return { data: json as unknown as T };
}

// ─── Fetch daftar inbox ───────────────────────

export async function getInboxList(params: {
  status?: string;
  q?: string;
  page?: number;
  pageSize?: number;
}): Promise<InboxListResponse> {
  const qs = new URLSearchParams();
  if (params.status) qs.set("status", params.status);
  if (params.q)      qs.set("q", params.q);
  if (params.page)   qs.set("page", String(params.page));
  if (params.pageSize) qs.set("pageSize", String(params.pageSize));

  const query = qs.toString() ? `?${qs.toString()}` : "";
  const res = await fetch(`${API}/inbox${query}`, { cache: "no-store" });
  if (!res.ok) {
    const body = await res.json().catch(() => ({})) as { error?: { message?: string } };
    throw new Error(body.error?.message ?? `HTTP ${res.status}`);
  }
  return res.json() as Promise<InboxListResponse>;
}

// ─── Fetch satu item inbox ────────────────────

export async function getInboxItem(itemId: string): Promise<InboxItemDetail> {
  const res = await fetch(`${API}/inbox/${encodeURIComponent(itemId)}`, { cache: "no-store" });
  if (!res.ok) {
    const body = await res.json().catch(() => ({})) as { error?: { message?: string } };
    throw new Error(body.error?.message ?? `HTTP ${res.status}`);
  }
  const json = (await res.json()) as { data: InboxItemDetail };
  return json.data;
}

// ─── Disposisi ────────────────────────────────

export async function disposeInboxItem(
  itemId: string,
  payload: { userIds?: string[]; unitIds?: string[]; note?: string }
): Promise<InboxItemDetail> {
  const res = await fetch(`${API}/inbox/${encodeURIComponent(itemId)}/disposition`, {
    method: "POST",
    cache: "no-store",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({})) as { error?: { message?: string } };
    throw new Error(body.error?.message ?? `HTTP ${res.status}`);
  }
  const json = (await res.json()) as { data: InboxItemDetail };
  return json.data;
}

// ─── Pengalihan / Redirect ────────────────────

export async function redirectInboxItem(
  itemId: string,
  payload: { userId: string; note?: string }
): Promise<InboxItemDetail> {
  const res = await fetch(`${API}/inbox/${encodeURIComponent(itemId)}/redirect`, {
    method: "POST",
    cache: "no-store",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({})) as { error?: { message?: string } };
    throw new Error(body.error?.message ?? `HTTP ${res.status}`);
  }
  const json = (await res.json()) as { data: InboxItemDetail };
  return json.data;
}

// ─── Label helper ─────────────────────────────

export function inboxStatusLabel(status: InboxItem["status"]): string {
  const map: Record<InboxItem["status"], string> = {
    UNREAD: "Belum Dibaca",
    READ: "Sudah Dibaca",
    DISPOSED: "Didisposisikan",
    REDIRECTED: "Dialihkan",
  };
  return map[status] ?? status;
}

export function inboxSourceLabel(source: InboxSource): string {
  if (source.type === "DIRECT") {
    return source.recipientType === "CC" ? "Tembusan (CC)" : "Tujuan Utama";
  }
  if (source.forwardType === "DISPOSITION") {
    return `Disposisi dari ${source.forwardedBy?.name ?? "—"}`;
  }
  return `Dialihkan dari ${source.forwardedBy?.name ?? "—"}`;
}

export async function getInboxUnreadCount(): Promise<number> {
  try {
    const result = await getInboxList({ pageSize: 1 });
    return result.counts.UNREAD;
  } catch {
    return 0;
  }
}
