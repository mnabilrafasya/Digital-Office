# Digital Office (frontend): Panduan Proyek dan Masalah

Satu-satunya file konteks proyek. Untuk AI coding assistant: **baca bagian 1-4 dulu, lalu hanya bagian yang relevan dengan tugas.** Kontrak API resmi: `../backend/docs/api.md` (sumber kebenaran; bila berbeda dari file ini atau mock, backend menang). Sebelum menulis kode Next.js, baca panduan di `node_modules/next/dist/docs/` (lihat `AGENTS.md`). Diperbarui 25 Okt 2026.

| Tugas                               | Bagian |
| ----------------------------------- | ------ |
| Apa pun                             | 1-4    |
| Perbaikan masalah                   | 7      |
| Menyambung/ubah pemanggilan backend | 5      |
| Daftar Surat / Review               | 6      |
| Editor Word (DevExpress)            | 8      |
| Inbox / Dashboard                   | 9      |
| Pertanyaan terbuka                  | 10     |
| Prompt siap tempel                  | 11     |

Label: **[KONFIRMASI]** arahan pembimbing/ringkasan DB · **[BACKEND]** ada di `api.md` · **[GAMBAR]** mockup · **[USULAN]** belum dikonfirmasi.

---

## 1. Ringkasan dan aturan

Sistem persuratan **Digital Office** (PT Pupuk Sriwidjaja Palembang), multi-perusahaan: data tiap perusahaan terpisah total (backend menyaring otomatis). Frontend: Next.js (Nabil). Backend: Golang (rekan magang), API `/api/v1`. Deliverable magang: mockup, ERD, flowchart, presentasi maks. 10 menit per orang. [KONFIRMASI]

Aturan kode (berlaku selalu):

- Teks UI Bahasa Indonesia, identifier bahasa Inggris. Token warna di `globals.css`, tanpa hardcode hex. Font Inter/Plus Jakarta Sans; hindari gradient ungu/biru, glassmorphism, neon, ilustrasi dekoratif, emoji.
- Label status hanya di `lib/status.ts`. Enum `LetterStatus`/`InboxStatus` (`lib/types.ts`) = kode lookup backend, jangan diubah.
- **Sifat** (lookup `CONFIDENTIALITY`: Biasa/Terbatas/Rahasia) dan **Kecepatan** (lookup `URGENCY`: Biasa/Segera/Sangat Segera) dua field berbeda; label dan teks bantu dari lookup. Tujuan/Tembusan = unit atau alias level, bukan orang.
- Tombol aksi surat **dirender dari `allowedActions`** milik surat [BACKEND] (`EDIT`, `SUBMIT`, `APPROVE`, `RETURN`), bukan tebakan dari status/peran. Jangan mengarang endpoint atau aksi yang belum ada di `api.md`.
- ID dari backend string (surat = uuid); waktu ISO dengan offset +07:00.
- File/komponen di bawah ~200 baris; tanpa `any`; data fetching lewat TanStack Query atau `lib/api/*` (bukan `setState` di dalam `useEffect`, bukan `fetch` di komponen).
- Data mock fiktif; jangan masukkan nama/NIK/nomor surat asli. Tanpa klaim karangan (BSrE, ISO, hash, "Sah Secara Hukum", hitung mundur SLA) dan tanpa angka palsu (badge sidebar).

## 2. Stack dan arsitektur

Next 16.4.0 (App Router, `src/`, Turbopack), React 19.3, TypeScript 5, Tailwind v4 lewat `@tailwindcss/turbopack` (tidak ada `postcss.config.mjs`, itu normal; jangan diubah), shadcn/ui (base-nova) + lucide-react, react-hook-form + zod, TanStack Query, date-fns. `reactCompiler` aktif; `cacheComponents`/`partialPrefetching` mati. Proteksi rute Next 16 bernama `proxy.ts` (`middleware.ts` deprecated).

Alur data: **browser -> route handler `app/api/[...path]/route.ts` -> backend Go** (`BACKEND_URL`, default `http://localhost:8080`). Token JWT di cookie httpOnly `token` (diset server action login `app/actions/auth.ts`); proxy memasangnya ke header `Authorization` dan meneruskan `If-Match`/`ETag`. Alasan: JS browser tidak bisa membaca cookie httpOnly dan CORS backend tidak mengekspos `ETag`.

Struktur: `app/(auth)/login` · `app/(app)/surat/{baru,daftar,review,approval,[id]}` · `app/api/[...path]` · `app/actions/auth.ts` · `middleware.ts` · `components/{auth,shell,surat,daftar-surat,review,ui}` · `hooks/` · `lib/{api,auth,types,schemas,status,mock-data,utils}`.

## 3. Status pekerjaan (25 Okt 2026)

| Fitur                                                                                             | Status                                                                             |
| ------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| Login, logout, `/auth/me`                                                                         | Tersambung (kekurangan: bagian 7)                                                  |
| Buat Surat: form, simpan draf **manual** (tombol Simpan Draft), edit draf `?draft=<uuid>`, submit | Tersambung. **Autosave sengaja dihapus** (keputusan Nabil); label "Draf tersimpan" |
| Buat Surat: lampiran                                                                              | Simulasi (tidak tersimpan; backend belum ada)                                      |
| Buat Surat: template                                                                              | Mock (tidak dikirim)                                                               |
| Buat Surat: alias penerima ("Seluruh VP")                                                         | Belum (hanya unit)                                                                 |
| Daftar Surat (tab, filter, pagination; scope Naskah Saya / Tugas Saya)                            | Tersambung sebagian                                                                |
| Menunggu Review / Approval                                                                        | Redirect ke Daftar Surat `scope=tasks`                                             |
| Review/Detail `/surat/[id]` (submit, approve, return)                                             | Tersambung                                                                         |
| Review: batalkan                                                                                  | UI ada, tersembunyi (backend belum punya aksi)                                     |
| Review: setujui oleh approver (tanda tangan)                                                      | Belum (backend belum ada; approver baru bisa mengembalikan)                        |
| Riwayat aksi (`GET /letters/{id}/history`)                                                        | Belum dipakai                                                                      |
| Editor Word (DevExpress)                                                                          | Belum (placeholder)                                                                |
| Inbox, Detail Disetujui, disposisi, alihkan                                                       | Belum (backend belum ada)                                                          |
| Dashboard                                                                                         | Belum (isi belum dikonfirmasi)                                                     |

## 4. Domain, alur, status

Surat dibuat **drafter**, dicek **reviewer** (berurutan), disetujui **approver**, dikirim ke **pimpinan unit penerima**; pimpinan bisa mendisposisi, staf bisa mengalihkan. [KONFIRMASI]

- Tiap simpan dokumen = versi baru (.docx). Reviewer boleh mengedit dokumen; approver tidak.
- Kembalikan + catatan: reviewer -> drafter (`RETURNED_TO_DRAFTER`); approver -> reviewer terakhir (`RETURNED_TO_REVIEWER`).
- Approver terakhir menyetujui = penanda tangan "atas nama"; satu transaksi: nomor otomatis (mis. `015/TI/PSP/X/2026`), tanggal = waktu disetujui, file FINAL + QR timestamp, masuk inbox pimpinan unit penerima, status `SENT`.
- Inbox read-only: `UNREAD` -> `READ`; disposisi `DISPOSED` (orang/unit, berjenjang); alihkan `REDIRECTED`.
- Batal: `CANCELLED`, alasan wajib, watermark "Dibatalkan", nomor tidak dipakai ulang. Semua aksi diaudit.

**Alur di backend sekarang** [BACKEND]: `SUBMIT` -> reviewer 1; `APPROVE` -> reviewer berikutnya, reviewer terakhir -> `IN_APPROVAL`; reviewer yang dikembalikan approver lalu `APPROVE` -> langsung `IN_APPROVAL`; kirim ulang dari `RETURNED_TO_DRAFTER` -> **mulai dari reviewer 1**; approver **tunggal**. Dua hal ini berbeda dari ringkasan DB (kirim ulang hanya ke yang mengembalikan; approver berurutan): bagian 10.

```ts
type LetterStatus =
  | "DRAFT"
  | "IN_REVIEW"
  | "RETURNED_TO_DRAFTER"
  | "IN_APPROVAL"
  | "RETURNED_TO_REVIEWER"
  | "SENT"
  | "CANCELLED";
type InboxStatus = "UNREAD" | "READ" | "DISPOSED" | "REDIRECTED";
```

Label UI (mengikuti **gambar**, bukan nama lookup backend):

| Enum                                          | Label UI          | Nama di backend                    | Tab Daftar Surat  |
| --------------------------------------------- | ----------------- | ---------------------------------- | ----------------- |
| `DRAFT`                                       | Draft             | Draf                               | Draft             |
| `IN_REVIEW`                                   | Menunggu Review   | Sedang Direview                    | Menunggu Review   |
| `RETURNED_TO_DRAFTER`, `RETURNED_TO_REVIEWER` | Dikembalikan      | Dikembalikan ke Drafter / Reviewer | Dikembalikan      |
| `IN_APPROVAL`                                 | Menunggu Approval | Menunggu Persetujuan               | Menunggu Approval |
| `SENT`                                        | Disetujui         | Terkirim                           | Disetujui         |
| `CANCELLED`                                   | Dibatalkan        | Dibatalkan                         | Dibatalkan        |

## 5. Integrasi backend [BACKEND]

**Sudah ada:** `POST /auth/login` (`employeeNumber`, `password`), `POST /auth/logout` (204; mengakhiri **semua** sesi user di semua perangkat), `GET /auth/me` (+`permissions`); `GET /lookups?category=`, `GET /units?q=`, `GET /users/candidates?type=REVIEWER|APPROVER`, `GET /recipients/aliases`; `POST /letters`, `GET /letters` (`scope=mine|tasks`, `status` boleh banyak dipisah koma, `q`, `page`, `pageSize`), `GET/PUT /letters/{uuid}` (PUT wajib `If-Match`), `GET /letters/{uuid}/history`, `POST /letters/{uuid}/submit|approve|return` (`return` wajib `note`, 1-1000 karakter).
**Belum ada:** tanda tangan approver + nomor + kirim ke inbox, batal, dokumen/versi .docx, lampiran, daftar template, inbox/disposisi/alihkan, hapus draf.

Format: sukses `{ "data": ... }` (list: `meta {page,pageSize,total}`); error `{ "error": {code,message} }`: 400 `VALIDATION_ERROR`, 401 `UNAUTHORIZED`/`INVALID_CREDENTIALS`, 403 `FORBIDDEN`, 404 `NOT_FOUND`, 409 `INVALID_STATE`, 412 `VERSION_CONFLICT`, 428 `PRECONDITION_REQUIRED`, 429 `TOO_MANY_REQUESTS` (+`Retry-After`).

`GET /letters` item: `id, letterNumber, title, status, letterType, confidentiality, urgency, senderUnit, currentAssignee, currentStep{role,order}, createdAt, updatedAt`. **Tidak ada:** penerima, `lastReturnNote`, `allowedActions`, jumlah per status, filter jenis/sifat/tanggal. `scope=tasks` = surat `IN_REVIEW`/`IN_APPROVAL`/`RETURNED_TO_REVIEWER`/`RETURNED_TO_DRAFTER` yang menunggu saya.

**Form Buat Surat -> body API:** `letterTypeId`, `classificationId` (id lookup); `sifat` -> `confidentialityId`, `kecepatan` -> `urgencyId` (cari id dari lookup berdasar `code`); `title/specialNote/summary` (maks 250/500/1000); `reviewers[]` -> `reviewerIds` (urut, maks 10, tanpa duplikat); `approvers[0]` -> `approverId` (**tepat satu**); `toUnits/ccUnits` -> `recipients.to/cc` berisi `{unitId}` atau `{aliasLevelId}` (maks 50, tanpa duplikat lintas to/cc). `PUT` mengganti seluruh isi; ETag berubah tiap simpan (jangan kirim dua PUT bersamaan). `412` = ambil ulang detail dan beri tahu pengguna, jangan menimpa diam-diam; `409` = surat sedang diproses, muat ulang. Draf boleh parsial; kelengkapan dicek saat submit.

**Login:** `401` -> "NIP / NIK atau kata sandi tidak sesuai"; `429` -> tampilkan tunggu `Retry-After` detik; token 8 jam. Tab "Daftar (Register)" tidak dibuat (pegawai disiapkan admin) kecuali pembimbing mengonfirmasi. "Akses Cepat Developer" di login hanya untuk masa pengembangan dan **akan dihapus sebelum production** (keputusan Nabil); selama masih ada, wajib dinonaktifkan di production (bagian 7, butir 1).

## 6. Daftar Surat dan Review

**Daftar Surat** `/surat/daftar`: scope "Naskah Saya (Drafter)" = `mine`, "Tugas Saya (Review & Approval)" = `tasks`; empat kartu ringkasan; tab status + jumlah; filter; tabel (Nomor Surat, Judul, Pengirim, Tujuan, Jenis & Sifat, Tanggal Dibuat, Status, Tahap Saat Ini, Tindakan); pagination; state di query string. `allowedActions` di list saat ini **ditebak di klien** dari status (butir 10 di bagian 7).

**Review/Detail** `/surat/[id]`: chip Sifat/Kecepatan/Status (`Menunggu Reviewer N (<nama>)`), judul, pratinjau dokumen (placeholder), lampiran, metadata, alur (`LetterSteps`). Action bar lengket muncul bila `allowedActions` memuat `APPROVE`/`RETURN`/`CANCEL`. Modal Kembalikan/Batalkan wajib catatan (UI min. 10 karakter; backend 1-1000). Drafter dengan `EDIT` diarahkan ke form. Approver saat ini hanya bisa **Kembalikan**.

## 7. Masalah di frontend (urut prioritas; centang saat selesai)

**P0: keamanan dan bug nyata**

1. [ ] **Akses Cepat Developer terbuka di production.** `devLoginAction` adalah server action yang bisa dipanggil siapa pun; bila `SEED_PASSWORD` terisi di server production, penyerang bisa login sebagai akun seed mana pun. Fallback `NEXT_PUBLIC_SEED_PASSWORD` menaruh password di bundle browser. Perbaiki: tolak di action bila `NODE_ENV === "production"`, sembunyikan panelnya, hapus fallback `NEXT_PUBLIC_`. Hapus seluruh fitur sebelum production.
2. [ ] **Proxy crash (500) untuk respons tanpa isi** (mis. `POST /auth/logout` -> 204; terbukti di log uji): `new NextResponse(arrayBuffer, {status:204})` dilarang. Beri body `null` untuk 204/205/304.
3. [ ] **Logout tidak memanggil backend**; hanya menghapus cookie, token tetap valid ≤ 8 jam. Panggil `POST /auth/logout` dulu (efek: semua perangkat keluar), lalu hapus cookie.
4. [ ] **Sesi kedaluwarsa tidak ditangani.** `middleware` hanya mengecek keberadaan cookie; layout tidak memvalidasi. Token ditolak backend (logout di perangkat lain, user nonaktif) membuat user terjebak: `/login` dialihkan ke `/surat/daftar` yang penuh error. Perbaiki: bila `/auth/me` atau API menjawab 401, hapus cookie dan alihkan ke `/login`; pengalihan "sudah login" dilakukan di halaman login dengan memeriksa sesi valid. Ganti `middleware.ts` jadi `proxy.ts`.
5. [ ] **Filter Daftar Surat menipu.** Filter jenis/sifat/tanggal dijalankan di klien pada satu halaman saja, `meta.total` dan jumlah tab tidak ikut berubah. Sembunyikan sampai backend mendukung (bagian 10), sisakan `q` dan status.
6. [ ] **Jumlah tab/kartu salah di atas 100 surat** (`counts` dari satu request `pageSize=100`). Minta backend menyediakan hitungan per status.
7. [ ] **"Menunggu Review" melewatkan surat dikembalikan approver.** Redirect memakai `status=IN_REVIEW`; pakai `IN_REVIEW,RETURNED_TO_REVIEWER`. "Menunggu Approval" = `IN_APPROVAL`.
8. [ ] **Approver di form bisa sampai 10, backend hanya satu**; UI mengirim `approvers[0]` dan membuang sisanya tanpa pemberitahuan. Seharusnya 10 max tetap.

**P1: kualitas dan kebenaran tampilan** 9. [ ] ESLint **22 error**: `no-explicit-any` 16 (`letter-draft.ts` 4, `baru/page.tsx` 12), `set-state-in-effect` 4 (`[id]/page`, `daftar-surat-view`, `basic-fields`, `classification-combobox`), `react-hooks/refs` 2 (`baru/page.tsx`). Ganti pola `useEffect + setState` dengan `useQuery`. 10. [ ] `lib/api/letters.ts` memalsukan data: `allowedActions` ditebak (`REVIEW`/`APPROVE` tidak ada di backend), `recipientsSummary` kosong (kolom Tujuan selalu kosong), `lastReturnNote` selalu null. Hapus tebakan; tampilkan "—" atau minta backend menambah field. 11. [ ] `surat/baru/page.tsx` 435 baris dan `surat/[id]/page.tsx` 501 baris; pemetaan detail -> nilai form ditulis dua kali. Ekstrak `detailToFormValues()` dan pecah komponen. 12. [ ] Riwayat/catatan kembali: `LetterSteps` memakai `step.returnNote` yang **tidak ada** di backend. Pakai `GET /letters/{id}/history` dan `lastReturnNote` (banner revisi). 13. [ ] Sidebar: badge palsu (Inbox 5, Notifikasi 3); menu belum difilter `permissions` (`LETTER_CREATE`, `LETTER_REVIEW`, `LETTER_APPROVE`, `INBOX_DISPOSE`, `TEMPLATE_CREATE`, `AUDIT_VIEW`). 14. [ ] Lampiran tampak berhasil tetapi tidak tersimpan: beri keterangan "belum tersimpan di server" atau nonaktifkan. 15. [ ] Alias penerima: memuat draf memetakan alias menjadi `alias-<id>` yang dibuang saat disimpan (kehilangan data diam-diam). Dukung alias atau beri peringatan. 16. [ ] `letter-draft.ts` auto-retry saat 409/428 kurang tepat (409 = surat sedang diproses; 412 harus ditampilkan, bukan ditimpa). Membuka `/surat/baru?draft=<id>` untuk surat yang tidak bisa diedit dialihkan ke `/surat/<id>`. 17. [ ] Bersihkan: hapus `hooks/use-autosave.ts` (autosave dibatalkan), paket `cn` (bukan utilitas kita), `msw` (tidak dipakai), `kodeAlias`/`kode_alias`, komentar `DEV_TOKEN`, redirect ganda `/` (di `next.config.ts` dan `app/page.tsx`).

## 8. Editor Word (DevExpress) [KONFIRMASI pembimbing]

Pembimbing meminta editor mirip Word dengan **DevExpress**. Tujuan terdekat: **demo lokal ke pembimbing** (tanpa hosting; cukup `npm run dev`).

- Paket npm `devexpress-richedit` (bersama `devextreme`): Rich Text Editor web yang membuka/menyimpan DOCX, RTF, TXT dan mengekspor PDF. Hanya jalan di browser: muat di komponen client dengan `dynamic(() => import(...), { ssr: false })`.
- **Lisensi:** menurut dokumentasi lisensi DevExtreme, evaluasi **30 hari** diizinkan; selama itu muncul pesan "unlicensed" di layar/konsol. Tiap developer perlu kunci sendiri dari DevExpress Download Manager (`config({ licenseKey })`; simpan di file yang di-`.gitignore`). Untuk dipakai di production perlu lisensi komersial: tanyakan ke pembimbing. Pastikan ketentuan evaluasi khusus `devexpress-richedit` di dokumentasi DevExpress.
- Spike (1-2 hari): halaman uji `/dev/editor` (hanya development) yang membuka dan menyimpan .docx lokal; catat langkah instal dan masalah di Next 16 + React 19. Jangan mengubah halaman Buat Surat/Review dulu.
- Untuk benar-benar dipakai butuh backend: endpoint **dokumen per versi** (.docx), daftar **template** (variabel `###NoSurat###`), dan pembuatan file FINAL + QR. Tanpa itu editor hanya bisa dites lokal.
- Aturan: reviewer aktif boleh mengedit; approver read-only. Tempat di UI: Section 3 Buat Surat dan area dokumen di Review (`EditorPlaceholder`).

## 9. Inbox dan Dashboard

**Inbox:** tab Unread/Read/Disposed/Redirected; kolom Aksi (Detail), Surat, Judul, Keterangan (Disposisi dari/CC dari/Surat dari), Pengirim (unit), Approver (orang), Tanggal (terbaru dulu), Sifat, Kecepatan; baris filter; pagination. Backend belum ada: kerjakan dengan mock berbentuk list surat lalu ganti. **Detail Disetujui:** preview final + QR/timestamp, metadata, Cetak Salinan, Unduh PDF Resmi, Disposisi (orang/unit, berjenjang), Alihkan. **Dashboard [USULAN]:** kartu ringkasan, "Perlu Tindakan Anda", aktivitas terbaru; isi belum dikonfirmasi. Menu lain (Profil, Master Data, Arsip, Notifikasi, Rekap, Pencarian) di luar cakupan sampai ada arahan.

## 10. Pertanyaan terbuka

**Ke backend:** (1) tanda tangan approver (nomor, QR, `SENT`, kirim inbox), batal, dokumen/versi .docx, lampiran, template, inbox/disposisi/alihkan; (2) `GET /letters`: tambah ringkasan penerima, `lastReturnNote`, `allowedActions`, `counts` per status, filter jenis/sifat/tanggal; (3) approver tunggal (API) vs berurutan (ringkasan DB); (4) kirim ulang setelah dikembalikan ke drafter: mulai reviewer 1 (API) vs hanya ke yang mengembalikan (ringkasan DB); (5) label status backend ("Terkirim", "Sedang Direview") vs gambar ("Disetujui", "Menunggu Review"); (6) apakah backend bisa mengirim token lewat cookie httpOnly langsung.
**Ke pembimbing:** siapa boleh membatalkan surat dan di tahap mana; apakah tab Daftar (Register) ada; Lupa Kata Sandi seperti apa; isi Dashboard dan brand guideline; lisensi DevExpress; nama tombol Submit vs "Kirim ke Reviewer".

## 11. Prompt siap tempel

**A. Perbaikan (bagian 7)**

```
Baca PROJECT_CONTEXT.md bagian 1, 2, 5, 7 saja. Kerjakan butir P0 1-8 lalu P1 9-17 satu per satu; berhenti setelah tiap butir untuk saya cek (centang butirnya di file).
Untuk butir 4, ganti middleware.ts jadi proxy.ts sesuai docs Next di node_modules. Jelaskan alasan keputusan teknis secara singkat. Jangan mengubah hal di luar butir.
```

**B. Spike editor DevExpress**

```
Baca PROJECT_CONTEXT.md bagian 1, 2, 8 saja.
Buat halaman uji /dev/editor (hanya development) yang me-render devexpress-richedit lewat dynamic import ssr:false dan bisa membuka serta menyimpan file .docx lokal. Kunci lisensi lewat file yang di-.gitignore. Catat langkah instalasi dan masalah di Next 16 + React 19. Jangan menyentuh Buat Surat dan Review.
```
