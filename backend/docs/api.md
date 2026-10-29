# Dokumentasi API Digital Office

Versi API: `v1`. Semua endpoint aplikasi berada di bawah prefix `/api/v1`.

## 1. Cara menjalankan

Dari folder `backend`, dengan Docker Desktop berjalan dan `.env` sudah diisi (salin dari `.env.example`):

```powershell
docker compose up -d          # SQL Server, MinIO, dan Gotenberg (LibreOffice untuk PDF resmi)
go run ./cmd/migrate up       # development: buat database bila belum ada + jalankan migration
go run ./cmd/seed all         # data acuan + data dummy (dummy hanya development)
go run ./cmd/api              # server di http://localhost:8080
```

Konfigurasi yang dipakai API (di `.env`):

| Env | Keterangan |
|---|---|
| `JWT_SECRET` | Wajib untuk server API (tidak dibutuhkan `migrate` dan `seed`), minimal 32 karakter, dan tidak boleh berisi nilai contoh dari `.env.example`. Kunci tanda tangan token, jangan dibagikan. |
| `JWT_TTL` | Masa berlaku token, default `8h`. |
| `CORS_ALLOWED_ORIGINS` | Origin front-end yang diizinkan, dipisah koma. Default `http://localhost:3000`. |
| `APP_TIMEZONE` | Zona waktu bisnis, default `Asia/Jakarta`. Divalidasi saat start. |
| `APP_MEMORY_LIMIT_MB` | Batas memori Go (`debug.SetMemoryLimit`), default `0` = tanpa batas. |
| `TRUST_PROXY_HEADERS` | Default `false`. Isi `true` **hanya** bila API berada di belakang reverse proxy yang menimpa `X-Forwarded-For`/`X-Real-IP`. |
| `GOTENBERG_URL` | Alamat Gotenberg (LibreOffice) untuk membuat PDF resmi, default `http://localhost:3001` (port 3000 dipakai front-end). Bila mati, unduhan PDF resmi mendapat `503`; fitur lain tetap jalan. |
| `DB_TRUST_SERVER_CERTIFICATE` | Default `true` hanya saat `APP_ENV=development`; selain itu `false` (sertifikat TLS SQL Server diverifikasi). |

## 2. Format response

Sukses:

```json
{ "data": ... }
```

Error:

```json
{ "error": { "code": "KODE", "message": "Pesan untuk pengguna" } }
```

| HTTP | `code` | Kapan |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Input tidak valid atau kurang (termasuk body JSON rusak atau lebih dari 1 MB) |
| 401 | `INVALID_CREDENTIALS` | Login gagal (NIK atau password salah, user nonaktif, belum punya password) |
| 401 | `UNAUTHORIZED` | Token tidak ada, rusak, kedaluwarsa, sudah logout, atau user sudah dinonaktifkan |
| 403 | `FORBIDDEN` | Login valid tetapi tidak punya permission |
| 404 | `NOT_FOUND` | Route atau data tidak ditemukan, termasuk surat yang tidak boleh diakses user (supaya keberadaannya tidak bocor) |
| 405 | `METHOD_NOT_ALLOWED` | Method HTTP salah |
| 409 | `INVALID_STATE` | Tindakan tidak sesuai status surat saat ini |
| 412 | `VERSION_CONFLICT` | `If-Match` tidak sama dengan versi terbaru; header `ETag` berisi versi terbaru |
| 428 | `PRECONDITION_REQUIRED` | Header `If-Match` wajib tetapi tidak dikirim |
| 429 | `TOO_MANY_REQUESTS` | Terlalu banyak percobaan login gagal; header `Retry-After` berisi detik tunggu |
| 503 | `SERVICE_UNAVAILABLE` | Layanan pendukung (pembuat PDF) sedang tidak bisa dihubungi; coba lagi nanti |
| 500 | `INTERNAL_ERROR` | Error tak terduga. Detail hanya ada di log server. |

Aturan JSON:

- Nama field memakai camelCase.
- Semua ID dikirim sebagai **string**, misalnya `"id": "8"`.
- Body request maksimal 1 MB.

## 3. Autentikasi

1. Login lewat `POST /api/v1/auth/login` untuk mendapatkan `accessToken`.
2. Kirim token di setiap request yang wajib login:

   ```
   Authorization: Bearer <accessToken>
   ```

3. Token berlaku **8 jam** (`expiresIn` dalam detik). Setelah kedaluwarsa, API menjawab `401 UNAUTHORIZED` dan pengguna harus login ulang.
4. Data user dimuat ulang dari database di setiap request. Kalau user dinonaktifkan atau role-nya dinonaktifkan, token lamanya langsung ditolak (`401`).
5. `POST /api/v1/auth/logout` mengakhiri **semua sesi user di semua perangkat**: semua token yang dibuat sebelumnya langsung ditolak (`401`).
6. **Pembatasan login gagal:** 5 kali per 15 menit untuk kombinasi IP + NIK, dan 30 kali per 15 menit per IP. Jendela 15 menit dihitung sejak kegagalan pertama. Login sukses mereset hitungan IP + NIK. Saat terkena batas: `429` dengan header `Retry-After` (detik) dan pesan "Terlalu banyak percobaan login. Coba lagi dalam N menit."
   Catatan: hitungan disimpan di memori server. Bila API dijalankan lebih dari satu instance, hitungan perlu dipindah ke penyimpanan bersama.
7. IP klien diambil dari koneksi TCP. Header `X-Forwarded-For`/`X-Real-IP` hanya dipercaya bila `TRUST_PROXY_HEADERS=true`, supaya pembatasan login tidak bisa dilewati dengan memalsukan header.

Login dengan NIK + password bersifat sementara sampai SSO tersedia.

## 4. Endpoint

### Ringkasan

| Method | Path | Login | Permission |
|---|---|---|---|
| GET | `/health` | Tidak | - |
| POST | `/api/v1/auth/login` | Tidak | - |
| POST | `/api/v1/auth/logout` | Ya | - |
| GET | `/api/v1/auth/me` | Ya | - |
| GET | `/api/v1/lookups` | Ya | - |
| GET | `/api/v1/units` | Ya | - |
| GET | `/api/v1/users/candidates` | Ya | `LETTER_CREATE` |
| GET | `/api/v1/recipients/aliases` | Ya | `LETTER_CREATE` |
| POST | `/api/v1/letters` | Ya | `LETTER_CREATE` |
| GET | `/api/v1/letters` | Ya | - |
| GET | `/api/v1/letters/{letterId}` | Ya | drafter, peserta surat, atau penerima inbox (surat `SENT`) |
| PUT | `/api/v1/letters/{letterId}` | Ya | drafter |
| GET | `/api/v1/letters/{letterId}/history` | Ya | drafter atau peserta surat |
| GET | `/api/v1/letters/{letterId}/qr` | Ya | drafter, peserta surat, atau penerima inbox |
| GET | `/api/v1/letters/{letterId}/versions` | Ya | drafter atau peserta surat |
| GET | `/api/v1/letters/{letterId}/document` | Ya | drafter atau peserta surat (`.docx`); setelah `SENT` juga penerima inbox (PDF resmi) |
| PUT | `/api/v1/letters/{letterId}/document` | Ya | drafter saat draf, atau reviewer yang sedang memegang surat |
| POST | `/api/v1/letters/{letterId}/submit` | Ya | drafter |
| POST | `/api/v1/letters/{letterId}/approve` | Ya | reviewer (`LETTER_REVIEW`) atau approver (`LETTER_APPROVE`) yang sedang memegang surat |
| POST | `/api/v1/letters/{letterId}/return` | Ya | reviewer (`LETTER_REVIEW`) atau approver (`LETTER_APPROVE`) yang sedang memegang surat |
| GET | `/api/v1/inbox` | Ya | - (selalu inbox milik user yang login) |
| GET | `/api/v1/inbox/{itemId}` | Ya | pemilik item |
| POST | `/api/v1/inbox/{itemId}/disposition` | Ya | pemilik item (`INBOX_DISPOSE`) |
| POST | `/api/v1/inbox/{itemId}/redirect` | Ya | pemilik item |

Angka ID di contoh response hanya ilustrasi.

Semua data di endpoint yang wajib login otomatis disaring ke **perusahaan user yang login**. Perusahaan tidak bisa dipilih lewat parameter.

**Aturan parameter pencarian `q`** (berlaku di semua endpoint yang menerimanya): spasi di awal/akhir dibuang, kosong berarti tidak mencari, maksimal 100 karakter. Karakter `%`, `_`, dan `[` dicari apa adanya, bukan sebagai wildcard.

---

### GET `/health`

Mengecek koneksi database dan storage. Tidak memakai format `{"data": ...}`.

```json
{ "database": "ok", "storage": "ok" }
```

Bila salah satu tidak bisa dihubungi: HTTP 503 dengan nilai `"error"` pada bagian yang bermasalah.

---

### POST `/api/v1/auth/login`

Login dengan NIK dan password.

Request:

```json
{ "employeeNumber": "2015047", "password": "..." }
```

Response 200:

```json
{
  "data": {
    "accessToken": "eyJ...",
    "tokenType": "Bearer",
    "expiresIn": 28800,
    "user": { "...": "sama persis dengan isi data di GET /api/v1/auth/me" }
  }
}
```

Error:

| HTTP | `code` | Penyebab |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Body kosong, JSON rusak, atau `employeeNumber`/`password` kosong |
| 401 | `INVALID_CREDENTIALS` | Selalu dengan pesan "NIK atau password salah", apa pun penyebabnya (NIK tidak ada, password salah, user nonaktif, belum punya password, role nonaktif) |
| 429 | `TOO_MANY_REQUESTS` | Terlalu banyak percobaan gagal (lihat bagian 3); tunggu sesuai `Retry-After` |

---

### POST `/api/v1/auth/logout`

Wajib login, tanpa body. Response **204** tanpa isi. Semua token user sebelumnya (di semua perangkat) menjadi tidak berlaku.

Error: `401 UNAUTHORIZED`.

---

### GET `/api/v1/auth/me`

Data user yang sedang login. Wajib login.

Response 200:

```json
{
  "data": {
    "id": "8",
    "employeeNumber": "2015047",
    "name": "Andi Kurniawan",
    "email": "andi.kurniawan@pusri.test",
    "position": { "id": "13", "name": "Staf Pengembangan Aplikasi", "isUnitHead": false,
                  "level": { "code": "STAF", "name": "Staf" } },
    "unit": { "id": "9", "code": "PA", "name": "Bagian Pengembangan Aplikasi" },
    "company": { "code": "PSP", "name": "PT Pupuk Sriwidjaja Palembang" },
    "role": { "code": "PEGAWAI", "name": "Pegawai" },
    "permissions": ["LETTER_CREATE", "LETTER_REVIEW", "TEMPLATE_CREATE"]
  }
}
```

`position.level` adalah level jabatan (`DIREKTUR`, `SVP`, `VP`, `AVP`, `STAF`), atau `null` bila jabatan belum punya level.

Front-end memakai `permissions` untuk menampilkan atau menyembunyikan menu. Kode permission: `LETTER_CREATE`, `LETTER_REVIEW`, `LETTER_APPROVE`, `INBOX_DISPOSE`, `TEMPLATE_CREATE`, `AUDIT_VIEW`.

Error: `401 UNAUTHORIZED`.

---

### GET `/api/v1/lookups?category=...`

Pilihan dropdown. Wajib login.

| Parameter | Wajib | Keterangan |
|---|---|---|
| `category` | Ya | `LETTER_TYPE`, `CONFIDENTIALITY`, `URGENCY`, `CLASSIFICATION`, `LETTER_STATUS`, atau `INBOX_STATUS` |

Berisi lookup aktif yang berlaku untuk semua perusahaan ditambah milik perusahaan user. Urut `sortOrder`, lalu `name`.

Contoh: `GET /api/v1/lookups?category=LETTER_TYPE`

```json
{
  "data": [
    { "id": "7", "code": "NOTA_DINAS", "name": "Nota Dinas", "color": null,
      "note": "Komunikasi resmi antar unit atau pejabat", "sortOrder": 1 }
  ]
}
```

Error: `400 VALIDATION_ERROR` (category kosong/tidak dikenal), `401 UNAUTHORIZED`.

---

### GET `/api/v1/units?q=...`

Unit kerja aktif di perusahaan user, untuk memilih tujuan surat. Wajib login.

| Parameter | Wajib | Keterangan |
|---|---|---|
| `q` | Tidak | Cari di nama unit |

Urut berdasarkan nama. `parentId` bernilai `null` untuk direktorat. `headPositionName` adalah nama jabatan pimpinan unit (penerima surat), `null` bila unit belum punya jabatan pimpinan.

```json
{
  "data": [
    { "id": "7", "code": "SDM", "name": "Departemen Sumber Daya Manusia", "unitType": "DEPARTEMEN",
      "parentId": "2", "headPositionName": "VP Sumber Daya Manusia" }
  ]
}
```

Error: `400 VALIDATION_ERROR` (`q` lebih dari 100 karakter), `401 UNAUTHORIZED`.

---

### GET `/api/v1/users/candidates?type=...&q=...&limit=...`

Kandidat reviewer atau approver untuk form Buat Surat. Wajib login dan punya permission `LETTER_CREATE`.

| Parameter | Wajib | Keterangan |
|---|---|---|
| `type` | Ya | `REVIEWER` (punya `LETTER_REVIEW`) atau `APPROVER` (punya `LETTER_APPROVE`) |
| `q` | Tidak | Cari di **nama user dan nama jabatan** (NIK tidak ikut dicari) |
| `limit` | Tidak | 1 sampai 50, default 20 |

Aturan:

- Hanya user aktif di perusahaan yang sama, dengan jabatan, unit, dan role yang aktif.
- `REVIEWER`: user yang login **boleh** muncul (boleh memilih dirinya sendiri).
- `APPROVER`: user yang login **tidak** muncul (tidak menyetujui suratnya sendiri).
- Urut berdasarkan nama.

Contoh: `GET /api/v1/users/candidates?type=APPROVER&q=agus`

```json
{
  "data": [
    { "id": "6", "name": "Agus Setiawan, S.Kom., M.T.", "employeeNumber": "1995021",
      "positionName": "SVP Teknologi Informasi", "unitName": "Kompartemen Teknologi Informasi" }
  ]
}
```

Error: `400 VALIDATION_ERROR` (`type` tidak valid, `limit` di luar 1 sampai 50, `q` lebih dari 100 karakter), `401 UNAUTHORIZED`, `403 FORBIDDEN` (tidak punya `LETTER_CREATE`).

---

### GET `/api/v1/recipients/aliases`

Alias penerima untuk form Buat Surat. Alias = level jabatan perusahaan user (contoh `VP` = "Seluruh VP"), beranggotakan pimpinan unit aktif dengan level tersebut. Hanya alias dengan minimal 1 anggota yang ditampilkan, urut sesuai level.

```json
{ "data": [ { "levelId": "3", "code": "VP", "label": "Seluruh VP", "memberCount": 2 } ] }
```

Error: `401 UNAUTHORIZED`, `403 FORBIDDEN` (tidak punya `LETTER_CREATE`).

---

### Draf surat

Surat diidentifikasi dengan **uuid** (`letterId`), bukan ID angka. Surat hanya bisa dilihat drafter dan pesertanya (reviewer/approver) di perusahaan yang sama; selain itu dijawab `404`.

#### Body `POST /api/v1/letters` dan `PUT /api/v1/letters/{letterId}`

Semua field opsional selama masih draf. `PUT` **mengganti seluruh isi**: field yang tidak dikirim berarti dikosongkan (cocok untuk autosave yang mengirim seluruh form).

```json
{
  "letterTypeId": "7", "templateId": null, "classificationId": "36",
  "confidentialityId": "1", "urgencyId": "4",
  "title": "Undangan Rapat Koordinasi", "summary": "...", "specialNote": "...",
  "reviewerIds": ["10", "7"],
  "approverId": "6",
  "recipients": {
    "to": [ { "unitId": "7" }, { "aliasLevelId": "3" } ],
    "cc": [ { "unitId": "8" } ]
  }
}
```

| Field | Aturan |
|---|---|
| `letterTypeId`, `classificationId`, `confidentialityId`, `urgencyId` | Lookup aktif dengan kategori yang sesuai (global atau milik perusahaan user) |
| `templateId` | Template aktif milik perusahaan; kalau template dibuat untuk jenis surat tertentu, jenisnya harus sama dengan `letterTypeId` |
| `title` / `specialNote` / `summary` | Maksimal 250 / 500 / 1000 karakter (emoji dihitung 2). Spasi di awal/akhir dibuang; teks kosong = `null` |
| `reviewerIds` | Maksimal 10, tanpa duplikat, **urutan = urutan paraf**. Syaratnya sama dengan kandidat `REVIEWER`. Drafter boleh menjadi reviewer |
| `approverId` | Tepat satu orang. Syaratnya sama dengan kandidat `APPROVER`, bukan drafter, dan bukan reviewer surat yang sama |
| `recipients` | Setiap elemen berisi **tepat satu** dari `unitId` (unit aktif perusahaan) atau `aliasLevelId` (dari `/recipients/aliases`). Unit/alias yang sama tidak boleh muncul dua kali, termasuk di `to` dan `cc` sekaligus. Maksimal 50 |

Kelengkapan (jenis surat, klasifikasi, sifat, kecepatan, perihal, minimal 1 reviewer, approver, minimal 1 penerima `to`) belum diwajibkan saat draf. Kelengkapan diwajibkan saat surat dikirim, dan saat menyimpan surat berstatus `RETURNED_TO_DRAFTER`.

#### POST `/api/v1/letters`

Membuat draf (body boleh kosong). Unit pengirim dan perusahaan diambil dari user yang login. Response **201** berisi detail surat, dengan header `Location: /api/v1/letters/{uuid}` dan `ETag`.

#### GET `/api/v1/letters/{letterId}`

Detail surat beserta header `ETag`. Yang boleh membuka: drafter, reviewer/approver, dan **penerima inbox** (langsung atau lewat disposisi/pengalihan) untuk surat yang sudah `SENT`. Penerima inbox selalu mendapat `allowedActions: []`; aksinya ada di endpoint inbox.

```json
{
  "data": {
    "id": "00b5adec-8228-4f74-8b1e-28b065a29f2f",
    "status": "DRAFT",
    "letterNumber": null, "letterDate": null,
    "title": "Undangan Rapat Koordinasi", "summary": "...", "specialNote": null,
    "letterType": { "id": "7", "code": "NOTA_DINAS", "name": "Nota Dinas" },
    "classification": { "id": "36", "code": "...", "name": "..." },
    "confidentiality": { "id": "1", "code": "BIASA", "name": "Biasa" },
    "urgency": { "id": "4", "code": "...", "name": "..." },
    "template": null,
    "senderUnit": { "id": "5", "code": "PA", "name": "..." },
    "drafter": { "id": "8", "name": "Andi Kurniawan", "positionName": "Staf Pengembangan Aplikasi" },
    "steps": [
      { "role": "REVIEWER", "order": 1, "state": "PENDING", "user": { "id": "10", "name": "...", "positionName": "...", "unitName": "..." } },
      { "role": "REVIEWER", "order": 2, "state": "PENDING", "user": { } },
      { "role": "APPROVER", "order": 1, "state": "PENDING", "user": { } }
    ],
    "recipients": {
      "to": [
        { "type": "UNIT", "unit": { "id": "7", "code": "SDM", "name": "..." }, "resolvedHead": null },
        { "type": "ALIAS", "alias": { "levelId": "3", "code": "VP", "label": "Seluruh VP", "memberCount": 2 } }
      ],
      "cc": [ { "type": "UNIT", "unit": { "id": "8", "code": "AK", "name": "..." }, "resolvedHead": null } ]
    },
    "attachments": [],
    "currentVersion": 0,
    "currentAssignee": null,
    "lastReturnNote": null,
    "approval": null,
    "allowedActions": ["EDIT"],
    "createdAt": "2026-10-20T09:15:00+07:00",
    "updatedAt": "2026-10-20T10:02:11+07:00"
  }
}
```

- `allowedActions` adalah tombol yang boleh ditampilkan untuk user yang sedang melihat: `EDIT` dan `SUBMIT` (drafter, status `DRAFT` atau `RETURNED_TO_DRAFTER`), `APPROVE` dan `RETURN` (penerima tugas saat ini). Tombol disembunyikan bila permission perannya sudah dicabut.
- `steps[].state`: `DONE`, `ACTIVE` (sedang di tahap ini), atau `PENDING`.
- `currentAssignee`: orang yang sedang memegang surat (`null` saat draf).
- `lastReturnNote`: catatan pengembalian terakhir, hanya terisi saat status `RETURNED_TO_DRAFTER` atau `RETURNED_TO_REVIEWER`.
- Kalau yang membuka adalah `currentAssignee`, server mencatat aksi `READ` di riwayat (sekali sejak perpindahan terakhir).
- `resolvedHead` (pimpinan unit penerima) baru terisi setelah surat ditandatangani.
- `approval` bernilai `null` sampai approver menyetujui. Setelah itu berisi waktu dan siapa yang menyetujui, beserta alamat gambar QR:
  `{ "approvedAt": "2026-10-29T11:19:07+07:00", "approvedBy": { "id": "6", "name": "...", "positionName": "..." }, "qrUrl": "/api/v1/letters/{letterId}/qr" }`
- Waktu dikirim dalam zona `APP_TIMEZONE`.

#### PUT `/api/v1/letters/{letterId}`

Menyimpan draf. **Wajib** header `If-Match` berisi `ETag` terakhir yang diterima klien. Response 200 berisi detail terbaru dan `ETag` baru (selalu berubah setiap simpan). Reviewer, approver, dan penerima yang tidak berubah tidak ditulis ulang.

Alur untuk front-end: simpan `ETag` dari response terakhir, lalu kirim kembali di `If-Match`. Kalau mendapat `412`, ambil ulang detail dan beri tahu pengguna bahwa surat sudah diubah di tempat lain (misalnya di tab lain).

Error:

| HTTP | `code` | Kapan |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Isi tidak valid (lihat tabel aturan), `If-Match` berformat salah, atau surat `RETURNED_TO_DRAFTER` belum lengkap |
| 403 | `FORBIDDEN` | Peserta (reviewer/approver) mencoba mengubah surat |
| 404 | `NOT_FOUND` | Surat tidak ada atau user bukan drafter/peserta |
| 409 | `INVALID_STATE` | Surat sedang diproses (bukan `DRAFT`/`RETURNED_TO_DRAFTER`) |
| 412 | `VERSION_CONFLICT` | `If-Match` bukan versi terbaru |
| 428 | `PRECONDITION_REQUIRED` | `If-Match` tidak dikirim |

### Alur persetujuan

| Aksi | Pelaku | Dari status | Ke status | Surat berpindah ke |
|---|---|---|---|---|
| `SUBMIT` | drafter | `DRAFT` | `IN_REVIEW` | reviewer 1 |
| `SUBMIT` | drafter | `RETURNED_TO_DRAFTER` | `IN_REVIEW` | reviewer 1 (**diulang dari awal**) |
| `APPROVE` | reviewer, bukan terakhir | `IN_REVIEW` | `IN_REVIEW` | reviewer berikutnya |
| `APPROVE` | reviewer terakhir | `IN_REVIEW` | `IN_APPROVAL` | approver |
| `RETURN` | reviewer | `IN_REVIEW` | `RETURNED_TO_DRAFTER` | drafter |
| `RETURN` | approver | `IN_APPROVAL` | `RETURNED_TO_REVIEWER` | reviewer terakhir yang meneruskan |
| `APPROVE` | approver | `IN_APPROVAL` | `SENT` | inbox penerima (surat terbit; di riwayat tercatat `SIGN`) |
| `APPROVE` | reviewer yang dikembalikan | `RETURNED_TO_REVIEWER` | `IN_APPROVAL` | approver (**langsung**, tidak mengulang reviewer) |
| `RETURN` | reviewer yang dikembalikan | `RETURNED_TO_REVIEWER` | `RETURNED_TO_DRAFTER` | drafter |

Selain `SUBMIT`, hanya orang yang sedang memegang surat (`currentAssignee`) yang bisa beraksi. Setiap aksi tercatat di riwayat dan di audit (beserta IP), dalam transaksi yang sama.

#### POST `/api/v1/letters/{letterId}/submit` · `/approve` · `/return`

| Endpoint | Body |
|---|---|
| `submit` | tidak ada |
| `approve` | opsional: `{ "note": "..." }` |
| `return` | wajib: `{ "note": "..." }` (1 sampai 1000 karakter) |

Response **200** berisi detail surat terbaru (format sama dengan `GET /letters/{letterId}`) dan `ETag` baru.

Saat `submit`, server memeriksa:
- Isi surat lengkap.
- Semua rujukan masih valid (misalnya reviewer belum dinonaktifkan).
- Setiap unit penerima punya pimpinan aktif.
- Setiap alias punya anggota.

Di setiap aksi, server juga memastikan orang berikutnya masih aktif dan berwenang.

Saat **approver** `approve`, surat terbit dalam satu transaksi:
- Penerima diperiksa ulang (unit punya pimpinan aktif, alias punya anggota).
- Nomor surat dibuat otomatis dari `companies.letter_number_format` (default `{SEQ}/{UNIT}/{COMP}/{MM_ROMAN}/{YYYY}`, contoh `015/TI/PSP/X/2026`). `{SEQ}` adalah nomor urut per unit pengirim per tahun, minimal 3 digit.
- `letterDate` (tanggal surat) diisi waktu disetujui, dan approver dicatat sebagai penandatangan.
- Surat masuk inbox (`UNREAD`): pimpinan setiap unit Kepada/Tembusan dan semua anggota alias. Orang yang sama hanya menerima sekali, dan Kepada didahulukan atas Tembusan.
- Response berisi `status: "SENT"`, `letterNumber`, dan `approval` (lihat detail surat).

| HTTP | `code` | Kapan |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Surat belum lengkap, rujukan tidak valid, catatan `return` kosong atau lebih dari 1000 karakter |
| 403 | `FORBIDDEN` | Permission peran sudah dicabut (`LETTER_REVIEW` / `LETTER_APPROVE`) |
| 404 | `NOT_FOUND` | Surat tidak ada atau user bukan drafter/peserta |
| 409 | `INVALID_STATE` | Bukan giliran user, status sudah berubah (misalnya tombol diklik dua kali), penerima surat tidak bisa dijangkau, atau orang berikutnya sudah nonaktif |

Tombol yang diklik dua kali aman: request kedua menunggu request pertama selesai, lalu ditolak dengan `409` "Surat sudah diproses atau bukan giliran Anda...". Front-end cukup memuat ulang detail.

### Naskah surat (.docx per versi)

Naskah dari editor (DevExpress) disimpan sebagai file `.docx`. Setiap simpan menjadi versi baru dan tidak menimpa versi lama. Filenya disimpan di MinIO, datanya di `letter_versions`, dan riwayat mencatat aksi `SAVE`. `currentVersion` di detail surat adalah nomor versi terakhir (`0` = belum ada naskah).

| Aturan | Sebelum disetujui approver | Setelah `SENT` |
|---|---|---|
| Mengubah naskah | drafter (`DRAFT`/`RETURNED_TO_DRAFTER`) dan reviewer yang sedang memegang surat; approver hanya membaca | tidak bisa |
| Unduh (`GET /document`) | `.docx` | **PDF resmi ber-QR**; `.docx` lama ditolak `409` |
| QR | tidak tersedia (`409`) | `GET /letters/{letterId}/qr` |

#### PUT `/api/v1/letters/{letterId}/document`

Body `multipart/form-data` dengan field `file` berisi `.docx` (maksimal 10 MB). Response **201** berisi versi baru dan **`ETag` baru**. Simpan ETag ini untuk `PUT /letters/{letterId}` berikutnya; ETag lama akan mendapat `412`.

```json
{
  "data": {
    "versionNumber": 2, "type": "DRAFT", "fileName": "Naskah Rapat.docx", "fileExtension": "docx",
    "fileSize": 18240, "checksum": "19670d82...",
    "createdBy": { "id": "8", "name": "Andi Kurniawan", "positionName": "..." },
    "createdByRole": "DRAFTER", "createdAt": "2026-10-29T13:22:39+07:00"
  }
}
```

Error: `400 VALIDATION_ERROR` (bukan `.docx`, file kosong/rusak, lebih dari 10 MB, atau bukan multipart), `403 FORBIDDEN` (permission peran dicabut), `404 NOT_FOUND` (bukan drafter/peserta), `409 INVALID_STATE` (bukan giliran user mengubah naskah).

#### GET `/api/v1/letters/{letterId}/versions`

Daftar versi naskah, terbaru dulu (format item sama dengan response unggah). Untuk drafter dan peserta.

#### GET `/api/v1/letters/{letterId}/document?version=...`

Satu tombol unduh; format file mengikuti status surat. Response berupa file dengan `Content-Type` dan `Content-Disposition: attachment; filename="..."`.

| Status surat | File | Siapa |
|---|---|---|
| Belum disetujui approver | `.docx` versi terbaru (atau `?version=N`) | drafter dan peserta |
| `SENT` | PDF resmi ber-QR, nama file dari nomor surat (contoh `015-TI-PSP-X-2026.pdf`) | drafter, peserta, dan penerima inbox |

PDF resmi dibuat server sekali, saat pertama kali diunduh:
1. Penanda `###NoSurat###` di naskah `.docx` terakhir diganti nomor surat. Penanda harus ditulis dengan satu format (tidak dipecah tebal/miring di tengahnya).
2. `.docx` diubah ke PDF oleh LibreOffice (Gotenberg).
3. QR persetujuan ditempel di pojok kanan bawah halaman terakhir (lebar sekitar 3 cm).
4. Hasilnya disimpan sebagai versi `FINAL` (dibuat atas nama approver). Unduhan berikutnya memakai file yang sama persis, jadi tidak berubah walaupun data user diubah.

Unduhan pertama butuh waktu sekitar 0,5 sampai 2 detik; berikutnya langsung.

| HTTP | `code` | Kapan |
|---|---|---|
| 400 | `VALIDATION_ERROR` | `version` bukan angka mulai dari 1 |
| 404 | `NOT_FOUND` | "Naskah surat belum diunggah", "Versi naskah tidak ditemukan", atau user tidak berhak |
| 409 | `INVALID_STATE` | Surat `SENT` tetapi naskah belum pernah diunggah (PDF tidak bisa dibuat), atau meminta `.docx` (`?version=`) setelah `SENT` |
| 503 | `SERVICE_UNAVAILABLE` | Gotenberg tidak bisa dihubungi; coba lagi nanti |

#### GET `/api/v1/letters/{letterId}/qr`

Gambar **PNG** QR persetujuan untuk surat yang sudah disetujui approver. Aksesnya sama dengan detail surat. Lewat proxy front-end cukup dipakai sebagai `<img src="/api/v1/letters/{letterId}/qr">`. Isi QR berupa teks biasa:

```
Persetujuan Surat - Digital Office
Nomor: 015/TI/PSP/X/2026
Disetujui oleh: Agus Setiawan, S.Kom., M.T.
Jabatan: SVP Teknologi Informasi
Waktu: 29-10-2026 11:19:07 WIB
```

Isi QR tidak bisa dibuktikan keasliannya (tanpa kode verifikasi). Nama dan jabatan diambil dari data user saat QR diminta.

Error: `401 UNAUTHORIZED`, `404 NOT_FOUND` (surat tidak ada atau user bukan drafter/peserta), `409 INVALID_STATE` (surat belum disetujui approver).

---

### GET `/api/v1/letters?scope=...&status=...&q=...&page=...&pageSize=...`

| Parameter | Keterangan |
|---|---|
| `scope` | `mine` (default): surat yang saya buat. `tasks`: surat yang sedang menunggu tindakan saya (`IN_REVIEW`, `IN_APPROVAL`, `RETURNED_TO_REVIEWER`, `RETURNED_TO_DRAFTER`) |
| `status` | Opsional, boleh beberapa dipisah koma, contoh `IN_REVIEW,IN_APPROVAL` |
| `q` | Cari di perihal dan nomor surat (maksimal 100 karakter) |
| `page`, `pageSize` | Default 1 dan 20, `pageSize` maksimal 100 |

Urut dari yang terakhir diubah.

```json
{
  "data": [
    {
      "id": "00b5adec-...", "letterNumber": null, "title": "Undangan Rapat Koordinasi", "status": "IN_REVIEW",
      "letterType": { "id": "7", "code": "NOTA_DINAS", "name": "Nota Dinas" },
      "confidentiality": { }, "urgency": { },
      "senderUnit": { "id": "5", "code": "PA", "name": "..." },
      "currentAssignee": { "id": "7", "name": "Dewi Lestari, S.Kom.", "positionName": "..." },
      "currentStep": { "role": "REVIEWER", "order": 2 },
      "createdAt": "...", "updatedAt": "..."
    }
  ],
  "meta": { "page": 1, "pageSize": 20, "total": 1 }
}
```

`currentStep` bernilai `null` bila surat tidak sedang di reviewer atau approver.

Error: `400 VALIDATION_ERROR` (`scope`, `status`, `page`, atau `pageSize` tidak valid), `401 UNAUTHORIZED`.

---

### GET `/api/v1/letters/{letterId}/history`

Riwayat aksi surat, urut dari yang paling lama. Bisa dilihat drafter dan peserta.

```json
{
  "data": [
    { "action": "SUBMIT", "actorRole": "DRAFTER",
      "actor": { "id": "8", "name": "Andi Kurniawan", "positionName": "..." },
      "target": { "id": "10", "name": "Fajar Nugroho", "positionName": "..." },
      "fromStatus": "DRAFT", "toStatus": "IN_REVIEW", "note": null, "versionNumber": 0,
      "createdAt": "..." }
  ]
}
```

`action`: `SUBMIT`, `APPROVE`, `RETURN`, atau `READ` (penerima tugas membuka surat). `target` adalah orang yang menerima surat setelah aksi.

Error: `401 UNAUTHORIZED`, `404 NOT_FOUND`.

---

### GET `/api/v1/inbox?status=...&q=...&page=...&pageSize=...`

Surat masuk milik user yang login: surat terkirim (`SENT`) yang diterima langsung sebagai penerima (Kepada/Tembusan), atau lewat disposisi/pengalihan.

| Parameter | Keterangan |
|---|---|
| `status` | Opsional (tab), boleh beberapa dipisah koma: `UNREAD`, `READ`, `DISPOSED`, `REDIRECTED`. Kosong berarti semua |
| `q` | Cari di perihal dan nomor surat (maksimal 100 karakter) |
| `page`, `pageSize` | Default 1 dan 20, `pageSize` maksimal 100 |

Urut dari yang terbaru masuk.

```json
{
  "data": [
    {
      "id": "2", "status": "UNREAD", "receivedAt": "2026-10-20T11:00:00+07:00", "readAt": null,
      "source": {
        "type": "FORWARD", "recipientType": null, "forwardType": "DISPOSITION",
        "forwardedBy": { "id": "7", "name": "Dewi Lestari, S.Kom.", "positionName": "..." },
        "note": "Mohon ditindaklanjuti"
      },
      "letter": {
        "id": "80d2d908-...", "letterNumber": "015/TI/PSP/X/2026", "title": "Undangan Rapat",
        "letterDate": "2026-10-20T09:00:00+07:00",
        "letterType": { "id": "7", "code": "NOTA_DINAS", "name": "Nota Dinas" },
        "confidentiality": { }, "urgency": { },
        "senderUnit": { "id": "11", "code": "PA", "name": "Bagian Pengembangan Aplikasi" },
        "approver": { "id": "6", "name": "Agus Setiawan, S.Kom., M.T.", "positionName": "..." }
      }
    }
  ],
  "meta": { "page": 1, "pageSize": 20, "total": 1 },
  "counts": { "UNREAD": 1, "READ": 1, "DISPOSED": 0, "REDIRECTED": 0 }
}
```

- `id` adalah ID item inbox; `letter.id` (uuid) dipakai untuk membuka surat.
- Kolom Keterangan dibentuk dari `source`: `DIRECT` + `TO` = "Surat dari {senderUnit}", `DIRECT` + `CC` = "CC dari {senderUnit}", `FORWARD` + `DISPOSITION`/`REDIRECT` = "Disposisi/Dialihkan dari {forwardedBy}".
- `source.note` adalah catatan/instruksi dari yang mendisposisi atau mengalihkan (`null` untuk `DIRECT`).
- `letterDate` adalah tanggal surat (waktu disetujui); `approver` adalah penandatangan.
- `counts` selalu berisi keempat status, mengikuti `q` tetapi tidak mengikuti `status`, untuk angka di tab. `meta.total` mengikuti `status`.

Error: `400 VALIDATION_ERROR` (`status`, `q`, `page`, atau `pageSize` tidak valid), `401 UNAUTHORIZED`.

### Disposisi dan pengalihan

| Aksi | Siapa | Tujuan | Status item sumber |
|---|---|---|---|
| `DISPOSE` | pemilik item dengan permission `INBOX_DISPOSE` (pimpinan) | satu atau lebih orang dan/atau unit (maksimal 20) | `DISPOSED` |
| `REDIRECT` | pemilik item mana pun (misalnya surat salah alamat) | tepat satu orang | `REDIRECTED` |

- Hanya item `UNREAD`/`READ` yang bisa diteruskan, dan hanya sekali. Klik dua kali mendapat `409`.
- Penerima mendapat item inbox baru (`FORWARD`, `UNREAD`) dan sejak itu boleh membuka detail surat dan QR.
- Disposisi ke **unit** diterima pimpinan aktif unit itu. Penerima disposisi bisa mendisposisi lagi dari item miliknya (berjenjang), asalkan punya `INBOX_DISPOSE`.
- Tujuan wajib user/unit aktif di perusahaan yang sama, bukan diri sendiri, dan bukan unit yang dipimpin sendiri. Satu orang hanya menerima satu item per permintaan.
- Nama dan jabatan pengirim disimpan sebagai snapshot. Setiap aksi tercatat di audit (beserta IP).

#### GET `/api/v1/inbox/{itemId}`

Membuka satu item inbox milik user; item `UNREAD` otomatis menjadi `READ`. Isinya sama dengan satu baris daftar inbox, ditambah:

```json
{
  "data": {
    "id": "5", "status": "DISPOSED", "...": "(sama dengan item daftar inbox)",
    "forwards": [
      {
        "id": "3", "type": "DISPOSITION", "targetType": "UNIT",
        "targetUser": null, "targetUnit": { "id": "8", "code": "AK", "name": "Departemen Akuntansi dan Keuangan" },
        "recipient": { "id": "17", "name": "Joko Susilo, S.E., Ak.", "positionName": "..." },
        "note": "Mohon ditindaklanjuti", "createdAt": "2026-10-29T13:54:14+07:00"
      }
    ],
    "allowedActions": []
  }
}
```

- `forwards` adalah penerusan yang sudah dibuat dari item ini. `recipient` adalah orang yang menerima di inbox (untuk unit: pimpinannya).
- `allowedActions`: `DISPOSE` (bila punya `INBOX_DISPOSE`) dan `REDIRECT`, hanya selama item masih `UNREAD`/`READ`.
- Detail surat (isi, penerima, QR) diambil terpisah lewat `GET /letters/{letter.id}`.

Error: `401 UNAUTHORIZED`, `404 NOT_FOUND` (item tidak ada atau milik orang lain).

#### POST `/api/v1/inbox/{itemId}/disposition` · `/redirect`

| Endpoint | Body |
|---|---|
| `disposition` | `{ "userIds": ["16"], "unitIds": ["8"], "note": "..." }` (minimal satu tujuan; `note` opsional, maksimal 1000 karakter) |
| `redirect` | `{ "userId": "18", "note": "..." }` (`note` opsional) |

Response **200** berisi item terbaru (format sama dengan `GET /inbox/{itemId}`).

| HTTP | `code` | Kapan |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Tanpa tujuan, ID tidak valid, tujuan nonaktif atau beda perusahaan, ke diri sendiri, unit tanpa pimpinan, atau catatan terlalu panjang |
| 403 | `FORBIDDEN` | Disposisi tanpa permission `INBOX_DISPOSE` |
| 404 | `NOT_FOUND` | Item tidak ada atau milik orang lain |
| 409 | `INVALID_STATE` | Item sudah didisposisi atau dialihkan |

---

### Batasan Sprint 2 (sementara)

- PDF resmi hanya bisa dibuat bila naskah `.docx` pernah diunggah. Front-end belum mengunggah naskah dari editor (topik naskah sedang ditunda), jadi surat tanpa naskah mendapat `409` saat diunduh.
- Disposisi belum dibatasi ke unit bawahan (semua user/unit aktif di perusahaan boleh dituju), dan belum ada tanda "selesai ditindaklanjuti".
- Naskah belum diwajibkan saat `submit`, supaya alur front-end yang belum mengunggah naskah tetap berjalan. Lampiran belum ada.
- Draf belum bisa dihapus.

## 5. Akun uji (data dummy development)

Semua akun memakai password yang sama: **lihat `SEED_PASSWORD` di `.env`**. Semua nama, NIK, dan email fiktif.

| Peran dalam uji | Nama | NIK | Role |
|---|---|---|---|
| Drafter | Andi Kurniawan (Staf Pengembangan Aplikasi) | 2015047 | PEGAWAI |
| Reviewer 1 | Fajar Nugroho (Analis Sistem Informasi) | 2012060 | PEGAWAI |
| Reviewer 2 | Dewi Lestari (AVP Pengembangan Aplikasi) | 2003035 | PIMPINAN |
| Approver 1 | Agus Setiawan (SVP Teknologi Informasi) | 1995021 | PIMPINAN |
| Approver 2 | Ratna Kusumawati (Direktur Keuangan dan Umum) | 1889002 | PIMPINAN |
| Penerima (tujuan Departemen SDM) | Sri Wahyuni (VP Sumber Daya Manusia) | 1998102 | PIMPINAN |
| Admin | Rudi Hartono (Administrator Sistem) | 2014090 | ADMIN |
| User nonaktif (login harus ditolak) | Slamet Riyadi | 1990170 | PEGAWAI |
| User perusahaan lain (PKG) | Lestari Ningsih | 2019203 | PEGAWAI |
