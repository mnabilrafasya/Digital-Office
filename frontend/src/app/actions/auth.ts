// ─────────────────────────────────────────────
// app/actions/auth.ts
// Server Actions untuk login dan logout.
// Tidak perlu route handler terpisah — Next.js
// Server Actions berjalan di server dan bisa
// mengakses / set cookie httpOnly.
// ─────────────────────────────────────────────

"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";

const BACKEND_URL = process.env.BACKEND_URL ?? "http://localhost:8080";

export interface LoginResult {
  success: boolean;
  error?: string;
}

/** Login: kirim NIK + password ke backend, simpan token di cookie httpOnly. */
export async function loginAction(
  employeeNumber: string,
  password: string,
): Promise<LoginResult> {
  let res: Response;
  try {
    res = await fetch(`${BACKEND_URL}/api/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ employeeNumber, password }),
      cache: "no-store",
    });
  } catch {
    return {
      success: false,
      error: "Backend tidak dapat dihubungi. Pastikan server berjalan.",
    };
  }

  if (!res.ok) {
    if (res.status === 401) {
      return {
        success: false,
        error: "NIP / NIK atau kata sandi tidak sesuai.",
      };
    }
    if (res.status === 429) {
      const wait = Number(res.headers.get("retry-after"));
      return {
        success: false,
        error:
          wait > 0
            ? `Terlalu banyak percobaan. Coba lagi dalam ${Math.ceil(wait / 60)} menit.`
            : "Terlalu banyak percobaan. Coba lagi beberapa saat lagi.",
      };
    }
    if (res.status === 400) {
      return { success: false, error: "NIK dan password tidak boleh kosong." };
    }
    return {
      success: false,
      error: `Server error (${res.status}). Coba beberapa saat lagi.`,
    };
  }

  const json = await res.json();
  const token: string = json.data.accessToken;
  const expiresIn: number = json.data.expiresIn ?? 28800; // default 8 jam

  // Simpan token di cookie httpOnly — JS browser tidak bisa baca,
  // aman dari serangan XSS.
  const store = await cookies();
  store.set("token", token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    maxAge: expiresIn,
    path: "/",
  });

  return { success: true };
}

/**
 * Login dev (Akses Cepat Developer):
 * Menggunakan SEED_PASSWORD dari environment (.env.local),
 * sehingga tiap anggota tim dapat login otomatis sesuai konfigurasi env masing-masing.
 */
export async function devLoginAction(
  employeeNumber: string,
): Promise<LoginResult> {
  // Server action bisa dipanggil siapa pun: tolak mentah-mentah di production.
  if (process.env.NODE_ENV === "production") {
    return { success: false, error: "Tidak tersedia." };
  }

  const seedPassword = process.env.SEED_PASSWORD;

  if (!seedPassword) {
    return {
      success: false,
      error:
        "SEED_PASSWORD belum diatur di .env.local frontend. Silakan tambahkan SEED_PASSWORD sesuai backend Anda.",
    };
  }

  return loginAction(employeeNumber, seedPassword);
}

/** Logout: hapus cookie token lalu redirect ke halaman login. */
export async function logoutAction() {
  const store = await cookies();
  const token = store.get("token")?.value;

  // Cabut token di backend (semua perangkat keluar). Gagal pun cookie tetap dihapus.
  if (token) {
    try {
      await fetch(`${BACKEND_URL}/api/v1/auth/logout`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        cache: "no-store",
      });
    } catch {
      // backend tidak terjangkau: lanjut hapus cookie lokal
    }
  }

  store.delete("token");
  redirect("/login");
}
