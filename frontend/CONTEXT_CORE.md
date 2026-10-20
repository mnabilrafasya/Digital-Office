# Digital Office (frontend) — konteks inti

Baca file ini dulu. Detail ada di `PROJECT_CONTEXT.md`: buka hanya bagian yang relevan dengan tugas (lihat "Peta bagian" di atasnya). Kontrak API resmi: `../backend/docs/api.md` (sumber kebenaran untuk integrasi; backend selalu menang bila bertentangan dengan mock).
Sebelum menulis kode Next.js, baca panduan terkait di `node_modules/next/dist/docs/` (lihat `AGENTS.md`).

## Stack
Next 16.4 (App Router, `src/`), React 19.3, TypeScript, Tailwind v4 (`@tailwindcss/turbopack`, jangan diubah ke postcss), shadcn/ui (base-nova) + lucide-react, react-hook-form + zod, TanStack Query. `reactCompiler` aktif.

## Aturan
- Teks UI Bahasa Indonesia, identifier bahasa Inggris. Token warna di `globals.css` (`@theme`), tanpa hardcode hex.
- Label status hanya di `lib/status.ts`. Enum `LetterStatus`/`InboxStatus` (`lib/types.ts`) mengikuti kode lookup backend, jangan diubah.
- Sifat (CONFIDENTIALITY: Biasa/Terbatas/Rahasia) dan Kecepatan (URGENCY: Biasa/Segera/Sangat Segera) adalah dua field berbeda. Tujuan/Tembusan = unit (atau alias level), bukan orang.
- Tombol aksi di halaman surat dirender dari `allowedActions` milik surat, bukan dari tebakan peran.
- Semua ID dari backend adalah string (surat = uuid). Waktu ISO dengan offset (+07:00).
- Data mock FIKTIF saja. Jangan masukkan nama/NIK/nomor surat asli.
- File komponen di bawah ~200 baris. Server Component secara default, `"use client"` hanya bila perlu.
- Jangan tambah menu sidebar atau klaim karangan (BSrE, ISO, FortiSandbox, hash, SLA hitung mundur).

## Lapisan data
Akses data hanya lewat `src/lib/api/*`. Endpoint yang sudah ada di backend dipanggil sungguhan, sisanya mock dengan bentuk data yang sama dengan backend (lihat PROJECT_CONTEXT bagian 7). Jangan memanggil `fetch` dari komponen.

## Kondisi repo (per 20 Okt 2026)
- Login: UI saja, belum terhubung. Belum ada `proxy.ts`, session, atau `lib/api/*`.
- Buat Surat: tampilan selesai (tanpa editor Word; Section 3 placeholder), autosave/submit masih `setTimeout`.
- Sidebar menu belum berhalaman -> `(app)/[...slug]` placeholder.
- Berikutnya: Daftar Surat -> Review -> integrasi Buat Surat + login -> Inbox/Detail -> Dashboard.

## Struktur
`app/(auth)/login` · `app/(app)/{layout,surat/baru,[...slug]}` · `components/{auth,shell,surat,ui}` · `hooks/use-autosave.ts` · `lib/{types,schemas,status,mock-data,utils}`
