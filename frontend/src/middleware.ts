// ─────────────────────────────────────────────
// middleware.ts
// Redirect otomatis:
//   - Belum login + akses halaman app → redirect /login
//   - Sudah login + akses /login → redirect /surat/daftar
// ─────────────────────────────────────────────

import { NextRequest, NextResponse } from "next/server";

// Path publik yang tidak butuh login
const PUBLIC_PATHS = ["/login"];

// Path yang dilewati middleware (aset statis, route API, dll)
const SKIP_PREFIXES = [
  "/_next/",
  "/api/",
  "/favicon.ico",
];

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;

  // Lewati aset statis dan API routes
  if (SKIP_PREFIXES.some((p) => pathname.startsWith(p))) {
    return NextResponse.next();
  }

  const token = req.cookies.get("token")?.value;
  const isPublic = PUBLIC_PATHS.some((p) => pathname === p || pathname.startsWith(p + "/"));

  // Belum login, coba akses halaman app
  if (!token && !isPublic) {
    const loginUrl = req.nextUrl.clone();
    loginUrl.pathname = "/login";
    // Simpan halaman asal agar bisa redirect balik setelah login
    loginUrl.searchParams.set("next", pathname);
    return NextResponse.redirect(loginUrl);
  }

  // Sudah login, coba akses /login → langsung ke app
  if (token && isPublic) {
    const homeUrl = req.nextUrl.clone();
    homeUrl.pathname = "/surat/daftar";
    homeUrl.search = "";
    return NextResponse.redirect(homeUrl);
  }

  return NextResponse.next();
}

export const config = {
  // Jalankan middleware di semua path kecuali yang sudah di-skip di atas
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
