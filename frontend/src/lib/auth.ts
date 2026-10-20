// ─────────────────────────────────────────────
// lib/auth.ts
// Helper auth: baca sesi dari cookie (server-side).
// Dipakai oleh middleware, layout, dan server actions.
// ─────────────────────────────────────────────

import { cookies } from "next/headers";

const BACKEND_URL = process.env.BACKEND_URL ?? "http://localhost:8080";

export interface AuthUser {
  id: string;
  employeeNumber: string;
  name: string;
  email: string;
  position: { id: string; name: string; isUnitHead: boolean };
  unit: { id: string; code: string; name: string };
  company: { code: string; name: string };
  role: { code: string; name: string };
  permissions: string[];
}

/** Ambil token dari cookie httpOnly "token". Null jika belum login. */
export async function getToken(): Promise<string | null> {
  const store = await cookies();
  return store.get("token")?.value ?? null;
}

/** Ambil data user saat ini dari backend via GET /auth/me.
 *  Null jika token tidak ada atau kadaluarsa (401).
 */
export async function getCurrentUser(): Promise<AuthUser | null> {
  const token = await getToken();
  if (!token) return null;

  try {
    const res = await fetch(`${BACKEND_URL}/api/v1/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    });
    if (!res.ok) return null;
    const json = await res.json();
    return json.data as AuthUser;
  } catch {
    return null;
  }
}
