# Digital Office: Konteks Proyek (Frontend)

Dipakai sebagai konteks untuk developer dan AI coding assistant. **Jangan baca seluruhnya**: buka bagian yang relevan.

| Tugas | Bagian |
|---|---|
| Apa pun | 1, 5, 6 |
| Menyambung ke backend (login, Buat Surat) | 7, 8 |
| Daftar Surat | 9, 13 |
| Review | 10, 13 |
| Inbox / Detail / Dashboard | 11 |
| Menulis data mock | 12 |
| Prompt siap tempel | 15 |

Label sumber: **[KONFIRMASI]** arahan pembimbing/ringkasan DB · **[BACKEND]** ada di `backend/docs/api.md` · **[GAMBAR]** terlihat di mockup (label UI mengikuti ini) · **[SISTEM LAMA]** foto sistem berjalan · **[USULAN]** belum dikonfirmasi, jangan dianggap fakta.

---

## 1. Ringkasan

Sistem persuratan **Digital Office** (PT Pupuk Sriwidjaja Palembang), multi-perusahaan: **data tiap perusahaan terpisah total** (backend menyaring otomatis dari perusahaan user). [KONFIRMASI]
- Frontend: Next.js (Nabil). Backend: Golang (rekan magang), API di `/api/v1`, sudah berjalan sebagian (bagian 7).
- Deliverable magang: mockup, ERD, flowchart; presentasi maks. 10 menit per orang. [KONFIRMASI]

## 2. Stack dan kondisi repo

Next 16.4.0 (App Router, `src/`, Turbopack), React 19.3.0, TypeScript 5, Tailwind v4 lewat `@tailwindcss/turbopack` (tidak ada `postcss.config.mjs`, itu normal; jangan diubah ke `@tailwindcss/postcss`), shadcn/ui (base-nova) + lucide-react, react-hook-form + zod, TanStack Query, date-fns. `reactCompiler: true`.

Kondisi saat ini:
- `next.config.ts`: `cacheComponents` dan `partialPrefetching` **aktif**, dan `/` di-redirect ke `/login`. Sidebar dibungkus `<Suspense>` (kemungkinan karena itu). **[REKOMENDASI]** matikan `cacheComponents`/`partialPrefetching` sebelum menyambung login: membaca cookie/sesi terjadi saat request, dan hampir semua halaman per-user sehingga caching tidak berguna.
- Login `components/auth/login-form.tsx` masih UI saja. Belum ada `proxy.ts`, session, `lib/api/*`, maupun `QueryClientProvider` (TanStack Query terpasang tapi belum dipakai).
- `msw` terpasang tapi tidak dipakai (mock lewat fungsi di `lib/api/*`, bukan MSW). Paket `cn` di `package.json` bukan utilitas `cn` kita (itu ada di `lib/utils.ts`), aman dihapus.
- Halaman Buat Surat: `app/(app)/surat/baru/page.tsx` (orkestrator, client) + `components/surat/*` + `hooks/use-autosave.ts`. Autosave dan submit masih `setTimeout`.
- Menu sidebar yang belum berhalaman jatuh ke `app/(app)/[...slug]/page.tsx` (placeholder).

## 3. Fokus dan urutan

Urutan: **Daftar Surat -> Review -> integrasi Buat Surat + login ke backend -> Inbox/Detail -> Dashboard**. Daftar Surat dan Review dikerjakan dengan data mock yang bentuknya meniru backend, karena endpoint list dan aksi review belum ada (bagian 7).

Editor ala Word **ditunda** (keputusan TipTap/Lexical vs editor Office embed belum diambil). Tempatnya selalu `EditorPlaceholder`/`DocumentPreviewPlaceholder`.

## 4. Domain dan alur [KONFIRMASI]

Surat dibuat **drafter**, dicek **reviewer** (berurutan), disetujui **approver**, lalu dikirim ke **pimpinan unit penerima**. Pimpinan bisa mendisposisi, staf bisa mengalihkan.

- Buat surat: tiap simpan = versi baru (tidak menimpa). Status `DRAFT`.
- Review berurutan: kirim -> reviewer 1 (`IN_REVIEW`). Setuju -> tahap berikutnya. Kembalikan + catatan -> drafter (`RETURNED_TO_DRAFTER`), drafter kirim ulang **hanya ke reviewer yang mengembalikan**. Reviewer terakhir setuju -> approver (`IN_APPROVAL`).
- Approval: approver tidak bisa mengedit, hanya setuju atau kembalikan. Kembalikan -> reviewer terakhir (`RETURNED_TO_REVIEWER`); setelah setuju lagi, kembali **hanya ke approver yang mengembalikan**. Approver terakhir = penanda tangan "atas nama".
- Terbit (satu transaksi): nomor otomatis (mis. `015/TI/PSP/X/2026`), tanggal = waktu disetujui, file FINAL dengan QR + timestamp, surat masuk inbox pimpinan unit penerima. Status `SENT`.
- Inbox: `UNREAD` -> `READ`; disposisi ke orang/unit, berjenjang (`DISPOSED`); staf bisa mengalihkan (`REDIRECTED`). Inbox read-only.
- Batal: status `CANCELLED`, alasan wajib, watermark "Dibatalkan", nomor terpakai tidak dipakai ulang.
- Semua aksi tercatat (audit).

## 5. Status

Enum mengikuti kode lookup backend (`LETTER_STATUS`, `INBOX_STATUS`) dan **tidak boleh diubah**:
```ts
type LetterStatus = "DRAFT" | "IN_REVIEW" | "RETURNED_TO_DRAFTER" | "IN_APPROVAL"
  | "RETURNED_TO_REVIEWER" | "SENT" | "CANCELLED";
type InboxStatus = "UNREAD" | "READ" | "DISPOSED" | "REDIRECTED"; // DISPOSED ada di backend
```
**Perbaiki `InboxStatus` di `lib/types.ts`: tambahkan `DISPOSED`** (sebelumnya dianggap turunan; ternyata status resmi).

Label UI (di `lib/status.ts`, satu-satunya tempat) mengikuti **gambar**, bukan nama lookup backend:

| Enum | Label UI [GAMBAR] | Nama lookup backend | Tab Daftar Surat |
|---|---|---|---|
| `DRAFT` | Draft | Draf | Draft |
| `IN_REVIEW` | Menunggu Review | Sedang Direview | Menunggu Review |
| `RETURNED_TO_DRAFTER` | Dikembalikan | Dikembalikan ke Drafter | Dikembalikan |
| `RETURNED_TO_REVIEWER` | Dikembalikan [USULAN] | Dikembalikan ke Reviewer | Dikembalikan |
| `IN_APPROVAL` | Menunggu Approval | Menunggu Persetujuan | Menunggu Approval |
| `SENT` | Disetujui | Terkirim | Disetujui |
| `CANCELLED` | Dibatalkan | Dibatalkan | Dibatalkan |

Nama di backend berbeda dari gambar. Selama belum diputuskan, UI memakai kolom "Label UI". Status halaman Review memakai pola `Menunggu Reviewer 2 (<nama>)` dari `currentAssignee`.

Tab Inbox [GAMBAR]: Unread, Read, Disposed, Redirected = `UNREAD`, `READ`, `DISPOSED`, `REDIRECTED`.

## 6. Aturan UI

- **Sifat** = lookup `CONFIDENTIALITY`: `BIASA`, `TERBATAS`, `RAHASIA`. **Kecepatan** = lookup `URGENCY`: `BIASA`, `SEGERA`, `SANGAT_SEGERA` (tiga nilai, termasuk Biasa). Dua field berbeda; "Penting" bukan Sifat.
- Teks bantu kecepatan/sifat diambil dari `note` lookup backend (mis. Segera = "Ditanggapi maksimal 2x24 jam"), **bukan** teks "< 24 Jam Kerja" dari mockup lama.
- Nomor dan tanggal surat sebelum disetujui = placeholder "dibuat otomatis".
- Tujuan/Tembusan = unit atau alias level ("Seluruh VP"), bukan orang. Pimpinan unit dicari backend.
- Tombol aksi di halaman surat **dirender dari `allowedActions`** milik surat (bagian 7), bukan dari tebakan peran.
- Menu sidebar mengikuti `permissions` dari `/auth/me`: `LETTER_CREATE` (Buat Surat), `LETTER_REVIEW` (Menunggu Review), `LETTER_APPROVE` (Menunggu Approval), `INBOX_DISPOSE` (disposisi), `TEMPLATE_CREATE`, `AUDIT_VIEW`. Menu tanpa permission disembunyikan; yang tidak butuh permission tetap tampil.
- Kembalikan membuka modal (catatan wajib). Batalkan membuka dialog (alasan wajib; menjelaskan surat tidak kembali ke drafter). Status selalu berlabel teks, tidak hanya warna.
- Token warna ada di `globals.css` (`@theme`); jangan hardcode hex. Font Inter/Plus Jakarta Sans, teks isi 14-16px. Hindari gradient ungu/biru, glassmorphism, neon, kartu terlalu bulat, ilustrasi dekoratif, emoji.

## 7. Integrasi backend [BACKEND]

Sumber kebenaran: `backend/docs/api.md` (baca bagian endpoint yang dikerjakan saja). Ringkasan keadaan per 20 Okt 2026:

**Sudah ada, boleh dipanggil sungguhan:**
`POST /auth/login` (`employeeNumber`, `password` -> `accessToken` Bearer 8 jam + `user`), `POST /auth/logout` (204, mengakhiri semua sesi user), `GET /auth/me` (user + `permissions`), `GET /lookups?category=LETTER_TYPE|CONFIDENTIALITY|URGENCY|CLASSIFICATION|LETTER_STATUS|INBOX_STATUS`, `GET /units?q=`, `GET /users/candidates?type=REVIEWER|APPROVER&q=&limit=` (perlu `LETTER_CREATE`), `GET /recipients/aliases`, `POST /letters` (buat draf), `GET /letters/{uuid}`, `PUT /letters/{uuid}` (simpan draf).

**Belum ada di backend (pakai mock, bentuk data sama dengan backend):**
daftar surat (list), daftar template, upload lampiran, **kirim/submit** ke reviewer, aksi review (setuju/kembalikan/batalkan), inbox dan disposisi. `allowedActions` saat ini hanya berisi `EDIT`.

**Aturan format:** sukses `{ "data": ... }`, error `{ "error": { "code", "message" } }` (kode: `VALIDATION_ERROR` 400, `INVALID_CREDENTIALS`/`UNAUTHORIZED` 401, `FORBIDDEN` 403, `NOT_FOUND` 404, `INVALID_STATE` 409, `VERSION_CONFLICT` 412, `PRECONDITION_REQUIRED` 428, `TOO_MANY_REQUESTS` 429 + `Retry-After`). Field camelCase. Semua ID **string**, surat = **uuid**.

**Cara frontend memanggil (arsitektur) [USULAN]:** panggil backend lewat **route handler Next** (`app/api/[...path]/route.ts`, atau server action), bukan langsung dari browser. Alasan: (1) token disimpan di cookie httpOnly sehingga JavaScript browser tidak bisa menaruhnya di header `Authorization`; handler membacanya dan meneruskannya; (2) CORS backend tidak mengekspos header `ETag`/`Location`/`Retry-After`, jadi browser lintas-origin tidak bisa membacanya; lewat handler tidak ada masalah CORS. Alternatif: minta backend menambah `ExposedHeaders`. URL backend dari env (`BACKEND_URL`, default `http://localhost:8080`). 401 -> hapus cookie dan arahkan ke `/login`.

**Simpan draf (autosave) dan ETag:**
1. Draf belum ada -> `POST /letters` (body boleh kosong atau parsial) -> simpan `id` (uuid) dan header `ETag`.
2. Berikutnya `PUT /letters/{id}` dengan header `If-Match: <ETag terakhir>`; response membawa ETag baru (**selalu berubah tiap simpan**).
3. Karena ETag berubah tiap simpan, simpan **harus berurutan** (antrian/mutex): dua PUT bersamaan membuat yang kedua `412`. Perbaiki `use-autosave` agar tidak menjalankan save paralel (termasuk tombol Simpan Draft saat autosave berjalan).
4. `412 VERSION_CONFLICT` -> ambil ulang detail, beri tahu "surat diubah di tempat lain". `409 INVALID_STATE` -> surat sudah diproses, form jadi read-only. `400` -> tampilkan pesan.
5. `PUT` **mengganti seluruh isi** (field yang tidak dikirim dikosongkan): kirim seluruh form tiap kali. Draf boleh **parsial**: autosave tidak boleh diblok validasi zod; zod hanya untuk tombol Submit.

**Pemetaan form Buat Surat -> body API:**

| Form | API |
|---|---|
| `letterTypeId`, `classificationId`, `templateId` | sama (id string dari lookup; template belum ada endpoint-nya) |
| `sifat` (kode) | `confidentialityId` (cari id dari lookup `CONFIDENTIALITY` berdasar `code`) |
| `kecepatan` (kode) | `urgencyId` (lookup `URGENCY`) |
| `title`, `specialNote`, `summary` | sama (maks 250/500/1000; emoji dihitung 2) |
| `reviewers[]` (urut) | `reviewerIds: string[]` (maks 10, tanpa duplikat; drafter boleh jadi reviewer) |
| `approvers[0]` | `approverId` (**tepat satu**; bukan drafter, bukan reviewer surat yang sama) |
| `toUnits[]` / `ccUnits[]` | `recipients.to` / `recipients.cc`: `[{ unitId }]` atau `[{ aliasLevelId }]` (maks 50; unit/alias tidak boleh dobel, termasuk antara to dan cc) |

**Bentuk detail surat** (`GET /letters/{id}`): `id`, `status`, `letterNumber`, `letterDate`, `title`, `summary`, `specialNote`, `letterType`/`classification`/`confidentiality`/`urgency`/`template` (`{id, code, name}`), `senderUnit`, `drafter` (`{id, name, positionName}`), `steps[]` (`{role: REVIEWER|APPROVER, order, state: PENDING|..., user:{id,name,positionName,unitName}}`), `recipients.{to,cc}` (`{type: UNIT|ALIAS, unit|alias, resolvedHead}`), `attachments[]`, `currentVersion`, `currentAssignee`, `lastReturnNote`, `allowedActions[]`, `createdAt`, `updatedAt`. Perbarui `lib/types.ts` mengikuti bentuk ini (tipe lama `Letter` di repo tidak sama dengan backend).

**Login:** form mengirim `employeeNumber` + `password`; `401` -> "NIP / NIK atau kata sandi tidak sesuai"; `429` -> tampilkan menunggu `Retry-After` detik. Token kedaluwarsa 8 jam -> 401 -> login ulang. Opsi "Ingat saya" menentukan masa cookie (pengaruh ke backend belum ada; token tetap 8 jam). Tab "Daftar (Register)" tidak dibuat (pegawai disiapkan admin) kecuali pembimbing mengonfirmasi.

## 8. Buat Surat (sudah dibangun; yang perlu diubah saat integrasi)

Tampilan sudah selesai sesuai gambar (Section 1 Detail, Section 2 Lampiran, Section 3 placeholder, action bar, dialog). Penyesuaian untuk integrasi, berurutan:
1. Isi dropdown dari lookup (Jenis Surat, Klasifikasi, Sifat, Kecepatan) dan hilangkan data mock yang hardcode. **Kecepatan = tiga opsi** (Biasa/Segera/Sangat Segera), label dan teks bantu dari lookup.
2. Reviewer/Approver: kandidat dari `/users/candidates` (pencarian di server, `limit` ≤ 50). **Approver hanya satu** (sesuai `approverId`); ubah picker approver jadi pilihan tunggal, atau konfirmasi ke backend bila approver bertahap harus didukung. Drafter boleh jadi reviewer tetapi tidak boleh jadi approver (kandidat `APPROVER` otomatis tanpa drafter).
3. Tujuan/Tembusan: unit dari `/units?q=`, ditambah opsi **alias** dari `/recipients/aliases` ("Seluruh VP (2)"). Tolak duplikat lintas Tujuan dan Tembusan.
4. Autosave sungguhan sesuai bagian 7 (POST pertama, PUT + `If-Match`, antrian, 412/409).
5. Lampiran, Template, Submit, **tetap mock** sampai backend punya endpoint-nya.

## 9. Daftar Surat [GAMBAR + USULAN]

Route: `app/(app)/surat/daftar/page.tsx`. Data mock lewat `lib/api/letters.ts` (`listLetters(params)`), bentuk di bagian 13.

**Layout (dari gambar):**
- Breadcrumb `Beranda > Penciptaan Surat > Daftar Surat`, judul "Daftar Surat", subjudul "Kelola, pantau proses review & persetujuan, serta tindak lanjut surat yang Anda susun." Tombol "+ Buat Surat Baru" (kanan atas, ke `/surat/baru`).
- **Empat kartu ringkasan:** Total Naskah Anda · Perlu Tindakan (Revisi) · Proses Review & Approval (rincian "n Review • n Approval") · Disetujui. Angka dari `counts`. Hapus teks karangan di mockup ("Verifikasi BSrE Valid", "Tahun Anggaran 2025").
- **Tab status dengan jumlah:** Semua, Draft, Menunggu Review, Dikembalikan, Menunggu Approval, Disetujui, Dibatalkan (pemetaan di bagian 5; "Dikembalikan" = `RETURNED_TO_DRAFTER` + `RETURNED_TO_REVIEWER`).
- **Baris filter:** kolom cari (hint `Ctrl + K`), pilihan Jenis Surat ("Semua Jenis Surat"), Sifat ("Semua Sifat"), rentang tanggal, tombol "Terapkan Filter" dan "Reset".
- **Tabel:** Nomor Surat · Judul/Perihal · Pengirim (unit) · Tujuan · Jenis & Sifat · Tanggal Dibuat · Status · Tahap Saat Ini · Tindakan. Header lengket, baris punya hover.
- **Footer:** "Menampilkan x - y dari n surat", "Tampilkan: 10 per halaman", pagination (Sebelumnya/angka/Selanjutnya).

**Isi sel [USULAN]:**
- Nomor Surat: nomor bila `SENT`, selain itu "Dibuat otomatis". Judul kosong (draf) -> "(Tanpa perihal)".
- Jenis & Sifat: nama jenis + badge Sifat. Tujuan: nama unit pertama + "+n lainnya".
- Tahap Saat Ini dari `currentAssignee`: "Reviewer 2 · <nama>" / "Approver · <nama>"; `DRAFT` -> "Drafter (Anda)"; `SENT`/`CANCELLED` -> "—".
- Tindakan menurut status: `DRAFT` -> "Lanjutkan" (ke form, hanya bila `allowedActions` memuat `EDIT`); `RETURNED_*` -> "Revisi" (tampilkan `lastReturnNote` pada tooltip/baris); selain itu -> "Lihat" (ke halaman Review/Detail).

**Perilaku:** tab, filter, pencarian, halaman disimpan di **query string URL** (bisa di-bookmark; tombol Back bekerja). Pencarian di-debounce. Pagination di sisi server (kontrak bagian 13). State: loading (skeleton baris), kosong ("Belum ada surat" + ajakan Buat Surat Baru; beda untuk hasil filter kosong), error (pesan + coba lagi). Tidak ada angka karangan; angka kartu harus sama dengan jumlah tab.

## 10. Review [GAMBAR + USULAN]

Route: `app/(app)/surat/[id]/page.tsx` (`id` = uuid). Data: `getLetter(id)` (backend `GET /letters/{id}` bila ada drafnya, mock untuk status lanjut). Satu halaman, beberapa **varian menurut `allowedActions` dan posisi user** (bukan menurut tebakan peran):

| Varian | Kapan | Yang tampil |
|---|---|---|
| Reviewer aktif | user = `currentAssignee` role `REVIEWER` | Area dokumen **placeholder editor** + tombol simpan perubahan (dinonaktifkan sampai editor ada), action bar: Batalkan Naskah · Kembalikan (Revisi) · Setujui Naskah |
| Approver aktif | user = `currentAssignee` role `APPROVER` | Dokumen **read-only**, action bar: Kembalikan (Revisi) · Setujui Naskah (Batalkan hanya bila `allowedActions` memuatnya; siapa boleh batal belum dikonfirmasi) |
| Drafter, dikembalikan | status `RETURNED_*`, `EDIT` diizinkan | Banner catatan revisi (`lastReturnNote`) + tombol "Revisi" ke form |
| Hanya lihat | peserta lain/surat selesai | Tanpa action bar |

**Layout (dari gambar):** breadcrumb `Beranda > Penciptaan Surat > Daftar Surat > Review > <kode>`; baris chip (Sifat, Kecepatan, Status `Menunggu Reviewer N (<nama>)`); judul surat. Dua kolom:
- **Kiri:** pratinjau dokumen (toolbar: zoom 100%, Sesuaikan Halaman, Cetak Salinan, Unduh Draf PDF, layar penuh; tombol yang belum didukung dinonaktifkan) lalu kartu **Lampiran Berkas Pendukung (n Berkas)** (total ukuran; tiap berkas: ikon tipe, nama, ukuran, Pratinjau, Unduh). Dokumen/lampiran = mock/placeholder.
- **Kanan:** kartu **Metadata & Ringkasan Surat** (Jenis Surat, Klasifikasi, Pejabat Pembuat/drafter, Pejabat Penandatangan/approver, Tujuan sebagai chip unit, Tembusan, Isi Ringkas Arsip) dan kartu **Alur & Riwayat** (stepper dari `steps`: tiap tahap berstatus selesai/aktif/menunggu, dengan catatan bila dikembalikan; kode `state` dari backend dipetakan di satu tempat).
- **Action bar lengket di bawah:** "Anda meninjau sebagai: Reviewer N (<nama>)" + tombol sesuai varian. **Tidak ada hitung mundur SLA** (tidak ada di backend); cukup teks urgensi dari lookup.

**Modal [KONFIRMASI + USULAN]:**
- *Kembalikan*: catatan wajib (min. 10 karakter [USULAN]); teks tujuan kembali: reviewer -> "Drafter", approver -> "Reviewer terakhir".
- *Batalkan*: alasan wajib; peringatan surat tidak kembali ke drafter dan diberi watermark "Dibatalkan".
- *Setujui*: konfirmasi; reviewer terakhir -> "diteruskan ke Approver"; approver terakhir -> "nomor surat dibuat otomatis dan surat dikirim ke pimpinan unit penerima".
- Setelah aksi: kembali ke daftar atau perbarui halaman, status dan riwayat ikut berubah (mock memperbarui data in-memory).

Aksi (`APPROVE`, `RETURN`, `CANCEL`) ada di mock sebagai fungsi `reviewAction(id, action, note?)`; namanya diusulkan ke backend (bagian 13).

## 11. Halaman lain (ringkas)

**Shell [SISTEM LAMA + GAMBAR]:** header navy (logo, Home, Supporting Documents, notifikasi, `[NIK] Nama`, Logout) dan sidebar: Dashboard; Profil (Edit Profil); Penciptaan Surat (Buat Surat Baru, Buat SP/ASP, Daftar Surat, Menunggu Review, Menunggu Approval); Master Data; Manajemen Arsip Aktif/Inaktif; Permission Document; Inbox; Notifikasi; Rekap; Pencarian. Jangan tambah menu karangan.
**Inbox:** tab Unread/Read/Disposed/Redirected; kolom Aksi (Detail), Surat, Judul, Keterangan (Disposisi dari/CC dari/Surat dari), Pengirim (unit), Approver (orang), Tanggal (terbaru dulu), Sifat, Kecepatan; baris filter; pagination.
**Detail Disetujui:** preview final + QR/timestamp, metadata, Cetak Salinan, Unduh PDF Resmi, **Disposisi**, **Alihkan** (satu set tombol).
**Dashboard [USULAN]:** kartu ringkasan, "Perlu Tindakan Anda", aktivitas terbaru. Isi belum dikonfirmasi.

## 12. Aturan konten dan data mock

- Teks UI Bahasa Indonesia; identifier bahasa Inggris.
- Mock **meniru bentuk backend** (ID string, uuid surat, `steps`, `currentAssignee`, `allowedActions`, waktu ISO `+07:00`), supaya saat endpoint siap cukup mengganti isi fungsi di `lib/api/*`.
- Data fiktif dan konsisten. Orang berbeda untuk drafter, reviewer 1, reviewer 2, approver, dan user login. Boleh memakai akun uji dari `api.md` bagian 5 (sudah fiktif), jangan nama/NIK asli dari screenshot.
- Satu nama perusahaan, satu rangkaian tanggal (2026), angka antar halaman nyambung (kartu = jumlah tab = jumlah baris).
- Tanpa klaim tanpa sumber (ISO 15489/27001, BSrE, FortiSandbox, SHA-256/MD5, "Sah Secara Hukum", hitung mundur SLA). Yang ada di arahan: QR dengan timestamp persetujuan.

## 13. Kontrak yang diusulkan ke backend [USULAN]

Pagination backend sudah punya format `meta: { page, pageSize, total }` (default 20, maks 100), jadi list sebaiknya memakai bentuk yang sama.

```
GET /api/v1/letters?scope=mine|review|approval&status=&q=&letterTypeId=&confidentialityId=&from=&to=&page=&pageSize=
-> { data: LetterListItem[], meta: { page, pageSize, total }, counts: { ALL, DRAFT, IN_REVIEW, RETURNED, IN_APPROVAL, SENT, CANCELLED } }

LetterListItem = { id, status, letterNumber|null, title|null, letterType, confidentiality, urgency,
  senderUnit, recipientsSummary: { first: string|null, total: number },
  createdAt, currentAssignee: { id, name, role, order }|null, lastReturnNote|null, allowedActions[] }

POST /api/v1/letters/{id}/submit            // kirim draf ke reviewer 1 (If-Match)
POST /api/v1/letters/{id}/actions           // { action: "APPROVE"|"RETURN"|"CANCEL", note? } (If-Match)
GET  /api/v1/letter-templates?letterTypeId=
POST /api/v1/letters/{id}/attachments       // multipart (PDF/DOCX/XLSX/PPTX, maks 25 MB)
```
`scope=mine` untuk Daftar Surat, `review`/`approval` untuk menu "Menunggu Review/Approval". Frontend mock memakai bentuk ini; bila backend memutuskan lain, ubah di `lib/api/letters.ts` dan `lib/types.ts` saja.

## 14. Pertanyaan terbuka

Ke backend: (1) endpoint list, submit, aksi review, template, lampiran di bagian 13; (2) **approver tunggal** di API, sedangkan ringkasan DB menyebut approver berurutan: mana yang benar?; (3) tambah `ExposedHeaders` (`ETag`, `Location`, `Retry-After`) di CORS atau konfirmasi pakai route handler Next; (4) nama label status backend ("Terkirim", "Sedang Direview") vs gambar ("Disetujui", "Menunggu Review"); (5) nilai `allowedActions` selain `EDIT` dan nilai `state` pada `steps`.
Ke pembimbing: (1) siapa boleh membatalkan surat dan di tahap mana; (2) apakah tab Daftar (Register) ada; (3) Lupa Kata Sandi seperti apa; (4) isi Dashboard dan brand guideline; (5) penanganan .docx dan server editor; (6) nama tombol Submit vs "Kirim ke Reviewer".

## 15. Prompt siap tempel

**A. Daftar Surat**
```
Baca CONTEXT_CORE.md, lalu PROJECT_CONTEXT.md bagian 5, 6, 9, 12, 13 saja.
Bangun halaman Daftar Surat di src/app/(app)/surat/daftar/ sesuai bagian 9.
Data lewat src/lib/api/letters.ts (listLetters) dengan MOCK yang bentuknya persis kontrak bagian 13 (LetterListItem, meta, counts). Perbarui lib/types.ts (tambah LetterListItem, tambah DISPOSED di InboxStatus). Jangan panggil fetch dari komponen.
Aturan: tab/filter/halaman di query string; status pakai StatusBadge dan lib/status.ts; token warna dari globals.css; teks Indonesia; kartu = jumlah tab = jumlah baris; tanpa klaim karangan.
Pecah komponen (summary-cards, status-tabs, filter-bar, letters-table, pagination) di components/daftar-surat/, tiap file di bawah 200 baris. Tambahkan state loading/kosong/error.
Kerjakan bertahap dan berhenti di tiap tahap untuk saya cek: (1) types + mock, (2) kartu + tab + filter, (3) tabel + pagination, (4) state kosong/error. Mulai dari tahap 1; jelaskan alasan keputusan teknis secara singkat.
```

**B. Review**
```
Baca CONTEXT_CORE.md, lalu PROJECT_CONTEXT.md bagian 5, 6, 10, 12, 13 saja.
Bangun halaman Review di src/app/(app)/surat/[id]/ sesuai bagian 10: varian reviewer aktif, approver aktif, drafter-dikembalikan, hanya-lihat. Tombol dirender dari allowedActions, bukan dari tebakan peran.
Data lewat lib/api/letters.ts (getLetter, reviewAction) dengan MOCK berbentuk detail surat backend (steps, currentAssignee, lastReturnNote, allowedActions). Tidak ada editor Word: area dokumen = placeholder. Tanpa hitung mundur SLA.
Modal Kembalikan (catatan wajib), Batalkan (alasan wajib), Setujui (konfirmasi) di components/review/. Setelah aksi, mock memperbarui data in-memory sehingga status dan riwayat berubah.
File di bawah 200 baris. Bertahap, berhenti di tiap tahap: (1) types + mock + layout statis, (2) metadata + alur & riwayat, (3) action bar + modal + aksi mock. Mulai dari tahap 1.
```

**C. Integrasi Buat Surat + login**
```
Baca CONTEXT_CORE.md, lalu PROJECT_CONTEXT.md bagian 7 dan 8, dan backend/docs/api.md bagian Autentikasi, lookups, units, candidates, aliases, dan Draf surat.
Matikan cacheComponents dan partialPrefetching di next.config.ts. Buat route handler app/api/[...path]/route.ts yang meneruskan ke BACKEND_URL dengan Authorization dari cookie httpOnly. Sambungkan login (form -> /auth/login), proxy.ts (tanpa cookie -> /login), logout, dan user di header dari /auth/me.
Lalu sambungkan Buat Surat sesuai bagian 8: lookup, kandidat reviewer/approver, unit + alias, autosave POST lalu PUT + If-Match dengan antrian (tanpa save paralel), penanganan 412/409/400. Lampiran, template, submit tetap mock.
Bertahap dan berhenti di tiap tahap; jelaskan alasan keputusan teknis secara singkat.
```

## 16. Masalah mockup yang diketahui (jangan dibawa ke kode)

Reviewer/approver/drafter/user login diisi orang yang sama; nomor dan tanggal surat terisi sebelum disetujui (campur 2025/2026); badge "Sifat: Penting"; chip unit tujuan terduplikasi; tabel Inbox/Daftar Surat kosong atau terpotong; tombol duplikat di Detail; dua nama perusahaan berbeda di kop; detail karangan (bagian 12); nama/NIK di screenshot terlihat asli.
