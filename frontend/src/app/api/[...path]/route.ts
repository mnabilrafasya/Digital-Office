// ─────────────────────────────────────────────
// app/api/[...path]/route.ts
// Proxy ke backend Go. Membaca token dari:
//   1. Cookie "token" (httpOnly, setelah login selesai diintegrasikan)
//   2. Env DEV_TOKEN (sementara, untuk development sebelum login)
// Alasan proxy:
//   - Token disimpan di cookie httpOnly → JS browser tidak bisa
//     menaruhnya di header Authorization secara langsung.
//   - Header ETag / Location / Retry-After dari backend tidak
//     diekspos CORS ke browser; lewat proxy tidak ada masalah itu.
// ─────────────────────────────────────────────

import { cookies } from "next/headers";
import { NextRequest, NextResponse } from "next/server";

const BACKEND_URL = process.env.BACKEND_URL ?? "http://localhost:8080";

// Header yang ikut diteruskan dari backend ke browser
const FORWARDED_RESPONSE_HEADERS = [
  "content-type",
  "etag",
  "location",
  "retry-after",
  "x-request-id",
];

async function handler(
  req: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  const { path } = await params;
  const backendPath = path.join("/");

  // Baca token dari cookie httpOnly "token"
  const cookieStore = await cookies();
  const token = cookieStore.get("token")?.value;

  // Next.js route berada di /api/[...path], sehingga params.path tidak menyertakan "api".
  // Backend Go mewajibkan prefix /api/v1/..., sehingga kita tambahkan "api/".
  const normalizedPath = backendPath.startsWith("api/") ? backendPath : `api/${backendPath}`;
  const search = req.nextUrl.search;
  const targetUrl = `${BACKEND_URL}/${normalizedPath}${search}`;

  // Teruskan body untuk POST / PUT
  const body =
    req.method !== "GET" && req.method !== "HEAD"
      ? await req.arrayBuffer()
      : undefined;

  const headers: Record<string, string> = {
    "Content-Type": req.headers.get("content-type") ?? "application/json",
  };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  // Header If-Match (ETag surat) perlu diteruskan untuk PUT
  const ifMatch = req.headers.get("if-match");
  if (ifMatch) headers["If-Match"] = ifMatch;

  let response: Response;
  try {
    response = await fetch(targetUrl, {
      method: req.method,
      headers,
      body,
      // Penting: jangan cache di sisi server Next
      cache: "no-store",
    });
  } catch {
    return NextResponse.json(
      { error: { code: "BACKEND_UNAVAILABLE", message: "Backend tidak dapat dihubungi." } },
      { status: 503 }
    );
  }

  // Status tanpa isi (204, 205, 304) wajib body null.
  const noBody = [204, 205, 304].includes(response.status);
  const responseBody = noBody ? null : await response.arrayBuffer();

  const nextResponse = new NextResponse(responseBody, {
    status: response.status,
  });

  // Token ditolak backend: hapus cookie agar navigasi berikutnya ke /login.
  if (response.status === 401 && token) {
    nextResponse.cookies.delete("token");
  }

  // Teruskan header backend yang relevan ke browser
  for (const h of FORWARDED_RESPONSE_HEADERS) {
    const v = response.headers.get(h);
    if (v) nextResponse.headers.set(h, v);
  }

  // Khusus ETag: pastikan 'etag' dan 'x-etag' tersedia dan dapat dibaca oleh client fetch
  const etagVal = response.headers.get("etag");
  if (etagVal) {
    nextResponse.headers.set("etag", etagVal);
    nextResponse.headers.set("x-etag", etagVal);
  }
  nextResponse.headers.set(
    "Access-Control-Expose-Headers",
    "ETag, etag, x-etag, Location, location, Retry-After, retry-after"
  );

  return nextResponse;
}

export const GET = handler;
export const POST = handler;
export const PUT = handler;
export const PATCH = handler;
export const DELETE = handler;
