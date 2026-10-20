# Dokumentasi API Digital Office

Versi API: `v1`. Semua endpoint aplikasi berada di bawah prefix `/api/v1`.

## 1. Cara menjalankan

Dari folder `backend`, dengan Docker Desktop berjalan dan `.env` sudah diisi (salin dari `.env.example`):

```powershell
docker compose up -d          # SQL Server dan MinIO
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
| GET | `/api/v1/letters/{letterId}` | Ya | drafter atau peserta surat |
| PUT | `/api/v1/letters/{letterId}` | Ya | drafter |

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

Detail surat beserta header `ETag`.

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
    "allowedActions": ["EDIT"],
    "createdAt": "2026-10-20T09:15:00+07:00",
    "updatedAt": "2026-10-20T10:02:11+07:00"
  }
}
```

- `allowedActions` adalah tombol yang boleh ditampilkan untuk user yang sedang melihat. Saat ini hanya `EDIT` (drafter, status `DRAFT` atau `RETURNED_TO_DRAFTER`).
- `resolvedHead` (pimpinan unit penerima) baru terisi setelah surat ditandatangani.
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
