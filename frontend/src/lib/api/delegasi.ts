// ─────────────────────────────────────────────
// lib/api/delegasi.ts
// Client API untuk Manajemen Delegasi Tugas Surat
// Mengikuti backend/internal/delegasi & backend/docs/delegasi.md
// ─────────────────────────────────────────────

import { fetchCandidates, type CandidateItem } from "@/lib/api/form";

export interface DelegationPerson {
  id: string;
  name: string;
  positionName: string;
}

export interface DelegationItem {
  id: string;
  user: DelegationPerson;
  delegate: DelegationPerson;
  from: string; // YYYY-MM-DD
  until: string; // YYYY-MM-DD
  note: string | null;
  status: "ACTIVE" | "UPCOMING" | "ENDED";
  createdBy: DelegationPerson;
  createdAt: string;
  updatedAt: string;
}

export interface MineDelegationsData {
  delegations: DelegationItem[];
  representing: DelegationItem[];
}

export interface AdminDelegationsResponse {
  data: DelegationItem[];
  meta: {
    page: number;
    pageSize: number;
    total: number;
  };
}

export interface CreateMineDelegationInput {
  substituteId: string;
  from: string;
  until: string;
  note?: string;
}

export interface CreateAdminDelegationInput {
  userId: string;
  substituteId: string;
  from: string;
  until: string;
  note?: string;
}

const API = "/api/v1";

async function request<T>(
  path: string,
  method = "GET",
  body?: unknown
): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    method,
    cache: "no-store",
    headers: body !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as {
      error?: { message?: string; code?: string };
    } | null;
    throw new Error(
      json?.error?.message ?? `Permintaan gagal (HTTP ${res.status}).`
    );
  }

  if (res.status === 204) {
    return undefined as T;
  }

  const json = (await res.json()) as { data: T };
  return json.data;
}

// ─── Pegawai: Delegasi Saya & Mewakili ────────

export async function getMineDelegations(): Promise<MineDelegationsData> {
  return request<MineDelegationsData>("/me/delegations");
}

export async function createMineDelegation(
  input: CreateMineDelegationInput
): Promise<DelegationItem> {
  return request<DelegationItem>("/me/delegations", "POST", input);
}

export async function updateMineDelegation(
  id: string,
  input: CreateMineDelegationInput
): Promise<DelegationItem> {
  return request<DelegationItem>(
    `/me/delegations/${encodeURIComponent(id)}`,
    "PUT",
    input
  );
}

export async function deleteMineDelegation(id: string): Promise<void> {
  return request<void>(`/me/delegations/${encodeURIComponent(id)}`, "DELETE");
}

// ─── Admin: Kelola Delegasi Semua Pegawai ────

export async function getAdminDelegations(params: {
  status?: string;
  userId?: string;
  page?: number;
  pageSize?: number;
}): Promise<AdminDelegationsResponse> {
  const sp = new URLSearchParams();
  if (params.status) sp.set("status", params.status);
  if (params.userId) sp.set("userId", params.userId);
  if (params.page) sp.set("page", String(params.page));
  if (params.pageSize) sp.set("pageSize", String(params.pageSize));

  const query = sp.toString() ? `?${sp.toString()}` : "";
  const res = await fetch(`${API}/admin/delegations${query}`, {
    cache: "no-store",
  });

  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as {
      error?: { message?: string };
    } | null;
    throw new Error(
      json?.error?.message ?? `Gagal mengambil delegasi admin (HTTP ${res.status}).`
    );
  }

  return res.json() as Promise<AdminDelegationsResponse>;
}

export async function createAdminDelegation(
  input: CreateAdminDelegationInput
): Promise<DelegationItem> {
  return request<DelegationItem>("/admin/delegations", "POST", input);
}

export async function updateAdminDelegation(
  id: string,
  input: CreateMineDelegationInput
): Promise<DelegationItem> {
  return request<DelegationItem>(
    `/admin/delegations/${encodeURIComponent(id)}`,
    "PUT",
    input
  );
}

export async function deleteAdminDelegation(id: string): Promise<void> {
  return request<void>(`/admin/delegations/${encodeURIComponent(id)}`, "DELETE");
}

// ─── Calon Pengganti (DELEGATE) ──────────────

export async function searchDelegateCandidates(
  q?: string
): Promise<CandidateItem[]> {
  return fetchCandidates({ type: "DELEGATE", q });
}

