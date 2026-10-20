# Dokumentasi API digioffice

Versi API: `v1`. Semua endpoint aplikasi berada di bawah prefix `/api/v1`.
Cara menyiapkan dan menjalankan server: lihat [setup.md](setup.md).

## 1. Format response

| Jenis | Bentuk |
|---|---|
| Sukses | `{ "data": ... }` |
| Daftar berhalaman | `{ "data": [...], "meta": { "page": 1, "pageSize": 20, "total": 57 } }` |
| Error | `{ "error": { "code": "KODE", "message": "Pesan untuk pengguna" } }` |

| HTTP | `code` | Kapan |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Input tidak valid atau kurang (termasuk JSON rusak atau body lebih dari 1 MB) |
| 401 | `INVALID_CREDENTIALS` | Login gagal (pesan selalu "NIK atau password salah") |
| 401 | `UNAUTHORIZED` | Token tidak ada, rusak, kedaluwarsa, sudah logout, atau user dinonaktifkan |
| 403 | `FORBIDDEN` | Peran user tidak pernah boleh melakukan tindakan ini (contoh: approver mengedit isi surat) |
| 404 | `NOT_FOUND` | Route/data tidak ada, **atau surat yang tidak boleh dilihat user** (keberadaannya tidak dibocorkan) |
| 405 | `METHOD_NOT_ALLOWED` | Method HTTP salah |
| 409 | `INVALID_STATE` | Aksi tidak sesuai status surat saat ini (contoh: klik kirim dua kali, bukan giliran user) |
| 412 | `VERSION_CONFLICT` | `If-Match` tidak sama dengan versi terbaru; header `ETag` berisi versi terbaru |
| 413 | `FILE_TOO_LARGE` | File melebihi `UPLOAD_MAX_FILE_SIZE_MB` |
| 428 | `PRECONDITION_REQUIRED` | Header `If-Match` wajib tetapi tidak dikirim |
| 429 | `TOO_MANY_REQUESTS` | Terlalu banyak percobaan login; header `Retry-After` berisi detik tunggu |
| 500 | `INTERNAL_ERROR` | Error tak terduga. Detail hanya di log server |
| 507 | `STORAGE_FULL` | Sisa kapasitas storage di bawah batas aman |

Aturan umum:

- JSON camelCase. Semua ID numerik dikirim sebagai **string** (`"id": "8"`).
- **Surat diidentifikasi dengan uuid** (`"id": "6f1c2a9e-..."`), bukan ID numerik.
- Waktu dikirim dalam RFC 3339 dengan offset zona `APP_TIMEZONE` (contoh `2026-10-20T09:15:00+07:00`).
- Body JSON maksimal 1 MB. Header `ETag`, `Location`, dan `Retry-After` dapat dibaca JavaScript (CORS expose).
- Parameter pencarian `q`: spasi di awal/akhir dibuang, kosong berarti tidak mencari, maksimal 100 karakter; `%`, `_`, `[` dicari apa adanya.
- Semua data otomatis disaring ke **perusahaan user yang login**.

## 2. Autentikasi

1. `POST /api/v1/auth/login` menghasilkan `accessToken` (berlaku `JWT_TTL`, default 8 jam).
2. Kirim di setiap request yang wajib login: `Authorization: Bearer <accessToken>`.
3. Data user dimuat ulang dari database setiap request. User/role yang dinonaktifkan langsung ditolak (401).
4. `POST /api/v1/auth/logout` mengakhiri **semua sesi user di semua perangkat** (token lama apa pun menjadi tidak berlaku).
5. Pembatasan login gagal: 5 kali per 15 menit untuk kombinasi IP + NIK, dan 30 kali per 15 menit per IP. Login sukses mereset hitungan IP + NIK. Saat terkena: 429 dan `Retry-After`.
   Catatan: hitungan disimpan di memori server. Bila API dijalankan lebih dari satu instance, hitungan perlu dipindah ke penyimpanan bersama.

Login NIK + password bersifat sementara sampai SSO tersedia.

## 3. Ringkasan endpoint

| Method | Path | Login | Permission / syarat |
|---|---|---|---|
| GET | `/health` | Tidak | - |
| POST | `/api/v1/auth/login` | Tidak | - |
| POST | `/api/v1/auth/logout` | Ya | - |
| GET | `/api/v1/auth/me` | Ya | - |
| GET | `/api/v1/lookups` | Ya | - |
| GET | `/api/v1/units` | Ya | - |
| GET | `/api/v1/recipients/aliases` | Ya | - |
| GET | `/api/v1/users/candidates` | Ya | `LETTER_CREATE` |
| POST | `/api/v1/letters` | Ya | `LETTER_CREATE` |
| GET | `/api/v1/letters` | Ya | - |
| GET | `/api/v1/letters/{letterId}` | Ya | drafter atau peserta |
| PUT | `/api/v1/letters/{letterId}` | Ya | drafter, `If-Match` wajib |
| DELETE | `/api/v1/letters/{letterId}` | Ya | drafter, `If-Match` wajib |
| GET | `/api/v1/letters/{letterId}/history` | Ya | drafter atau peserta |
| POST | `/api/v1/letters/{letterId}/submit` | Ya | drafter |
| POST | `/api/v1/letters/{letterId}/approve` | Ya | reviewer assignee, `LETTER_REVIEW` |
| POST | `/api/v1/letters/{letterId}/return` | Ya | reviewer/approver assignee |
| POST | `/api/v1/letters/{letterId}/sign` | Ya | approver assignee, `LETTER_APPROVE` |
| POST | `/api/v1/letters/{letterId}/attachments/upload-url` | Ya | drafter |
| POST | `/api/v1/letters/{letterId}/attachments` | Ya | drafter |
| DELETE | `/api/v1/letters/{letterId}/attachments/{attachmentId}` | Ya | drafter |
| GET | `/api/v1/letters/{letterId}/attachments/{attachmentId}/download-url` | Ya | drafter atau peserta |
| POST | `/api/v1/letters/{letterId}/versions/upload-url` | Ya | drafter, atau reviewer assignee |
| POST | `/api/v1/letters/{letterId}/versions` | Ya | drafter, atau reviewer assignee |
| GET | `/api/v1/letters/{letterId}/versions` | Ya | drafter atau peserta |
| GET | `/api/v1/letters/{letterId}/versions/{versionNumber}/download-url` | Ya | drafter atau peserta |

Kode permission: `LETTER_CREATE`, `LETTER_REVIEW`, `LETTER_APPROVE`, `INBOX_DISPOSE`, `TEMPLATE_CREATE`, `AUDIT_VIEW`.

## 4. Endpoint dasar (Sprint 1)

### GET `/health`
`{ "database": "ok", "storage": "ok" }`. HTTP 503 dengan nilai `"error"` pada bagian yang bermasalah. Tidak memakai format `{"data": ...}`.

### POST `/api/v1/auth/login`
Request `{ "employeeNumber": "2015047", "password": "..." }`.
Response 200 `{ "data": { "accessToken": "eyJ...", "tokenType": "Bearer", "expiresIn": 28800, "user": { ...sama dengan /auth/me... } } }`.
Error: 400 (field kosong), 401 `INVALID_CREDENTIALS`, 429 `TOO_MANY_REQUESTS`.

### POST `/api/v1/auth/logout`
Tanpa body. Response 204. Semua token user sebelumnya menjadi tidak berlaku (401).

### GET `/api/v1/auth/me`
```json
{
  "data": {
    "id": "8", "employeeNumber": "2015047", "name": "Andi Kurniawan", "email": "andi.kurniawan@pusri.test",
    "position": { "id": "13", "name": "Staf Pengembangan Aplikasi", "isUnitHead": false,
                  "level": { "code": "STAF", "name": "Staf" } },
    "unit": { "id": "9", "code": "PA", "name": "Bagian Pengembangan Aplikasi" },
    "company": { "code": "PSP", "name": "PT Pupuk Sriwidjaja Palembang" },
    "role": { "code": "PEGAWAI", "name": "Pegawai" },
    "permissions": ["LETTER_CREATE", "LETTER_REVIEW", "TEMPLATE_CREATE"]
  }
}
```
`position.level` bernilai `null` bila jabatan belum punya level. Angka ID di contoh hanya ilustrasi.

### GET `/api/v1/lookups?category=...`
`category` wajib: `LETTER_TYPE`, `CONFIDENTIALITY`, `URGENCY`, `CLASSIFICATION`, `LETTER_STATUS`, `INBOX_STATUS`. Lookup global ditambah milik perusahaan user, urut `sortOrder` lalu `name`.
Elemen: `{ "id", "code", "name", "color", "note", "sortOrder" }`. Error 400 bila category tidak dikenal.

### GET `/api/v1/units?q=...`
Unit aktif perusahaan user, urut nama. Elemen: `{ "id", "code", "name", "unitType", "parentId", "headPositionName" }`.

### GET `/api/v1/users/candidates?type=REVIEWER|APPROVER&q=...&limit=...`
Kandidat aktif satu perusahaan (`REVIEWER`: punya `LETTER_REVIEW`, user login boleh muncul; `APPROVER`: punya `LETTER_APPROVE`, user login tidak muncul). `q` mencari nama user dan nama jabatan. `limit` 1 sampai 50 (default 20).
Elemen: `{ "id", "name", "employeeNumber", "positionName", "unitName" }`.

## 5. Penerima dan alias

### GET `/api/v1/recipients/aliases`
Alias penerima = level jabatan perusahaan user yang punya minimal 1 anggota. Anggota alias adalah user aktif dengan jabatan **pimpinan unit** (`isUnitHead`) pada level tersebut.
```json
{ "data": [ { "levelId": "60", "code": "VP", "label": "Seluruh VP", "memberCount": 2 } ] }
```

Penerima surat adalah **unit** (pimpinan unit dicari sistem) atau **alias**, masing-masing di `to` (Kepada) atau `cc` (Tembusan). Satu unit tidak boleh muncul di `to` dan `cc` sekaligus; satu alias juga tidak boleh muncul dua kali. Saat ditandatangani, penerima diterjemahkan menjadi orang dan dideduplikasi: seseorang yang terkena `to` dan `cc` menerima sebagai `to`.

## 6. Surat (draf)

### POST `/api/v1/letters` dan PUT `/api/v1/letters/{letterId}`
Semua field opsional selama draf. `PUT` **mengganti seluruh** isian (field yang tidak dikirim berarti kosong), cocok untuk simpan otomatis yang mengirim seluruh isi form.
```json
{
  "letterTypeId": "12", "templateId": null, "classificationId": "40", "confidentialityId": "1", "urgencyId": "5",
  "title": "Undangan Rapat Koordinasi Sistem Informasi", "summary": "...", "specialNote": "...",
  "reviewerIds": ["9", "4"],
  "approverId": "3",
  "recipients": { "to": [ { "unitId": "7" }, { "aliasLevelId": "60" } ], "cc": [ { "unitId": "9" } ] }
}
```
- Urutan `reviewerIds` = urutan tahap. Setiap penerima berisi tepat satu dari `unitId` atau `aliasLevelId`. ID kosong (`""`) dianggap `null`.
- Panjang maksimal: `title` 250, `specialNote` 500, `summary` 1000 karakter.
- Referensi selalu divalidasi (kategori lookup benar dan aktif, unit/template/alias aktif dan satu perusahaan, reviewer punya `LETTER_REVIEW`, approver punya `LETTER_APPROVE`, approver bukan drafter dan bukan reviewer, tanpa duplikat).
- Kelengkapan (jenis surat, klasifikasi, sifat, kecepatan, perihal, minimal 1 reviewer, tepat 1 approver, minimal 1 penerima `to`) diperiksa saat submit, dan juga saat `PUT` di status `RETURNED_TO_DRAFTER`.
- `POST` → 201 + header `Location: /api/v1/letters/{uuid}` + `ETag`. `PUT` → 200 + `ETag` baru.
- Unit pengirim = unit drafter saat draf dibuat.

Error: 400, 403 (bukan drafter), 404, 409 (surat sudah diproses), 412, 428.

### DELETE `/api/v1/letters/{letterId}`
Wajib `If-Match`. Hanya draf (`DRAFT`) yang belum pernah dikirim. Response 204. File lampiran dan versinya dihapus dari storage.

### GET `/api/v1/letters/{letterId}`
Detail + header `ETag`. Bila user adalah assignee dan belum membaca sejak transisi terakhir, aksi `READ` dicatat di riwayat (ETag tidak berubah).
```json
{
  "data": {
    "id": "6f1c2a9e-...", "status": "IN_REVIEW", "letterNumber": null, "letterDate": null,
    "title": "...", "summary": "...", "specialNote": "...",
    "letterType": { "id": "12", "code": "NOTA_DINAS", "name": "Nota Dinas" },
    "classification": { "id": "40", "code": "TI.01", "name": "..." },
    "confidentiality": { "id": "1", "code": "BIASA", "name": "Biasa" },
    "urgency": { "id": "5", "code": "SEGERA", "name": "Segera" },
    "template": null,
    "senderUnit": { "id": "8", "code": "PA", "name": "..." },
    "drafter": { "id": "15", "name": "Andi Kurniawan", "positionName": "..." },
    "steps": [
      { "role": "REVIEWER", "order": 1, "state": "DONE",    "user": { "id": "9", "name": "...", "positionName": "...", "unitName": "..." } },
      { "role": "REVIEWER", "order": 2, "state": "ACTIVE",  "user": { } },
      { "role": "APPROVER", "order": 1, "state": "PENDING", "user": { } }
    ],
    "recipients": {
      "to": [
        { "type": "UNIT",  "unit": { "id": "7", "code": "SDM", "name": "..." }, "resolvedHead": null },
        { "type": "ALIAS", "alias": { "levelId": "60", "code": "VP", "label": "Seluruh VP", "memberCount": 2 } }
      ],
      "cc": [ { "type": "UNIT", "unit": { }, "resolvedHead": null } ]
    },
    "attachments": [
      { "id": "uuid", "fileName": "...", "size": 1024, "contentType": "...", "checksum": "sha256 hex",
        "uploadedAt": "...", "uploadedBy": { "id": "15", "name": "..." } }
    ],
    "currentVersion": 2,
    "currentAssignee": { "id": "4", "name": "...", "positionName": "..." },
    "lastReturnNote": null,
    "allowedActions": ["APPROVE", "RETURN", "UPLOAD_VERSION"],
    "createdAt": "2026-10-20T09:15:00+07:00", "updatedAt": "2026-10-20T10:02:11+07:00"
  }
}
```
- `steps[].state`: `DONE`, `ACTIVE`, `PENDING`.
- `resolvedHead` terisi setelah surat ditandatangani (pimpinan unit yang menerima).
- `letterDate` = tanggal tanda tangan (zona `APP_TIMEZONE`), `null` sebelum `SENT`.
- `lastReturnNote` = catatan pengembalian terakhir saat status `RETURNED_*`.
- `allowedActions` (gabungan): `EDIT`, `DELETE`, `SUBMIT`, `APPROVE`, `RETURN`, `SIGN`, `UPLOAD_ATTACHMENT`, `UPLOAD_VERSION`, sudah memperhitungkan status, giliran, dan permission user.

### GET `/api/v1/letters?scope=mine|tasks&status=...&q=...&page=1&pageSize=20`
- `scope=mine` (default): surat buatan user. `scope=tasks`: surat yang menunggu tindakan user (`IN_REVIEW`, `IN_APPROVAL`, `RETURNED_TO_REVIEWER`, `RETURNED_TO_DRAFTER`).
- `status` opsional, boleh beberapa dipisah koma. `q` mencari di perihal dan nomor surat.
- Urut `updatedAt` terbaru. `pageSize` maksimal 100.
- Elemen: `id`, `letterNumber`, `title`, `status`, `letterType`, `confidentiality`, `urgency`, `senderUnit`, `currentAssignee`, `currentStep` (`{ "role", "order" }` atau `null`), `createdAt`, `updatedAt`.

### GET `/api/v1/letters/{letterId}/history`
Riwayat urut waktu: `{ "action", "actorRole", "actor", "target", "fromStatus", "toStatus", "note", "versionNumber", "createdAt" }`. Aksi: `SAVE` (versi dokumen diunggah), `SUBMIT`, `READ`, `APPROVE`, `RETURN`, `SIGN`.

## 7. Alur persetujuan

| Endpoint | Body | Aktor |
|---|---|---|
| `POST .../submit` | - | drafter |
| `POST .../approve` | `{ "note": "opsional" }` | reviewer yang sedang menjadi assignee |
| `POST .../return` | `{ "note": "wajib, 1-1000 karakter" }` | reviewer atau approver yang sedang menjadi assignee |
| `POST .../sign` | - | approver yang sedang menjadi assignee |

Semua aksi menjawab 200 + detail surat terbaru + `ETag`. Peserta yang bukan assignee saat ini (termasuk klik dua kali) mendapat 409 `INVALID_STATE`; yang bukan peserta mendapat 404; peran tanpa permission yang dibutuhkan mendapat 403.

### Mesin status

| # | Aksi | Aktor | Dari | Ke | Assignee baru | Catatan |
|---|---|---|---|---|---|---|
| 1 | SUBMIT | drafter | DRAFT | IN_REVIEW | reviewer 1 | nama unit pengirim disimpan (snapshot) |
| 2 | SUBMIT | drafter | RETURNED_TO_DRAFTER | IN_REVIEW | **reviewer 1 (ulang dari awal)** | |
| 3 | APPROVE | reviewer aktif, bukan terakhir | IN_REVIEW | IN_REVIEW | reviewer berikutnya | |
| 4 | APPROVE | reviewer terakhir | IN_REVIEW | IN_APPROVAL | approver | |
| 5 | RETURN | reviewer aktif | IN_REVIEW | RETURNED_TO_DRAFTER | drafter | |
| 6 | RETURN | approver | IN_APPROVAL | RETURNED_TO_REVIEWER | reviewer terakhir yang meneruskan | |
| 7 | APPROVE | reviewer yang dikembalikan | RETURNED_TO_REVIEWER | IN_APPROVAL | **approver langsung** | |
| 8 | RETURN | reviewer yang dikembalikan | RETURNED_TO_REVIEWER | RETURNED_TO_DRAFTER | drafter | |
| 9 | SIGN | approver | IN_APPROVAL | SENT | - | nomor surat, tanggal, inbox penerima |

Saat submit dan sign: setiap unit penerima harus punya pimpinan aktif dan setiap alias minimal 1 anggota; bila tidak, 409 dengan nama unit/alias bermasalah. Penerima tugas berikutnya yang sudah nonaktif juga menghasilkan 409.

### Nomor surat
Dibuat saat SIGN dari `companies.letter_number_format` (default `{SEQ}/{UNIT}/{COMP}/{MM_ROMAN}/{YYYY}`), contoh `001/PA/PSP/X/2026`. `{SEQ}` urut per unit pengirim per tahun (minimal 3 digit). Dua tanda tangan bersamaan di unit yang sama dijamin mendapat nomor berbeda.

## 8. Lampiran dan versi dokumen

Byte file **tidak pernah lewat server API**. Browser mengunggah langsung ke MinIO; server hanya memberi izin, lalu mencatat.

```
1. FE -> API    POST .../upload-url { fileName, size }
   API          cek akses, status, ukuran, kapasitas storage
   API -> FE    { uploadUrl, method, fields, uploadToken, expiresAt, maxSizeBytes }
2. FE -> MinIO  POST multipart ke uploadUrl: semua fields, lalu file (field "file") paling akhir
                MinIO sendiri menolak file di luar 1..maxSizeBytes
3. FE -> API    POST .../attachments atau .../versions { uploadToken }
   API          verifikasi token, cek object, salin ke lokasi final, hitung SHA-256, catat di database
```

### POST `.../attachments/upload-url` dan `.../versions/upload-url`
Request `{ "fileName": "Lampiran Anggaran.xlsx", "size": 482113 }`. Response:
```json
{
  "data": {
    "uploadUrl": "http://localhost:9000/documents/", "method": "POST",
    "fields": { "Content-Type": "", "bucket": "documents", "key": "staging/...", "policy": "...",
                "x-amz-algorithm": "...", "x-amz-credential": "...", "x-amz-date": "...", "x-amz-signature": "..." },
    "uploadToken": "eyJ...", "expiresAt": "...", "maxSizeBytes": 26214400
  }
}
```
Field `Content-Type` sudah disediakan: isi nilainya dengan tipe file (boleh dibiarkan kosong), **jangan menambah field Content-Type kedua** (MinIO akan menolak dengan 403).
Error: 400 (ukuran 0), 403, 404, 409 (status tidak mengizinkan), 413, 507.

### POST `.../attachments` dan `.../versions`
Request `{ "uploadToken": "..." }`. Response 201 saat pertama kali dicatat, 200 untuk konfirmasi ulang dengan token yang sama (idempoten, tidak membuat duplikat). Header `ETag` berisi versi surat terbaru.
- Lampiran: `{ "id", "fileName", "size", "contentType", "checksum", "uploadedAt", "uploadedBy" }`.
- Versi: `{ "versionNumber", "versionType", "fileName", "fileExtension", "size", "checksum", "createdBy", "createdByRole", "createdAt" }`; setiap versi baru mencatat aksi `SAVE` di riwayat.

Error: 400 (token tidak valid/kedaluwarsa, file belum diunggah), 403, 404, 409, 413 (file di storage melebihi batas; object dihapus).

### Lainnya
- `DELETE .../attachments/{attachmentId}`: 204 + `ETag`.
- `GET .../versions`: daftar versi, terbaru di atas.
- `GET .../attachments/{attachmentId}/download-url` dan `GET .../versions/{versionNumber}/download-url`: `{ "url", "fileName", "expiresAt" }`. URL berlaku `DOWNLOAD_URL_TTL` dan **selalu memaksa unduhan** (`Content-Disposition: attachment`, `application/octet-stream`), sehingga file tidak pernah dibuka atau dijalankan di browser.

Siapa boleh: lampiran hanya drafter saat `DRAFT`/`RETURNED_TO_DRAFTER`; versi dokumen drafter saat `DRAFT`/`RETURNED_TO_DRAFTER` atau reviewer assignee saat `IN_REVIEW`/`RETURNED_TO_REVIEWER`; unduh oleh drafter dan semua peserta. Tidak ada pembatasan jenis file, jumlah lampiran, atau unggahan bersamaan.

## 9. Batasan Sprint 2

- Dokumen surat (versi) **belum wajib** saat submit, karena editor dokumen di front-end belum ada.
- `CANCEL` (batalkan surat) dan `REASSIGN` (ganti peserta di tengah alur) belum tersedia.
- Penerima surat belum bisa melihat surat lewat API; inbox (`inbox_items`) sudah diisi saat SIGN dan akan dibuka di Sprint 3.
- Pembatasan login disimpan di memori satu instance server.
- Logout mengakhiri sesi di semua perangkat (belum ada logout per perangkat).

## 10. Panduan integrasi front-end

1. **Tombol aksi** (Simpan, Kirim, Setujui, Kembalikan, Tanda Tangan, Unggah) masuk mode loading dan nonaktif sampai response datang. Bila gagal, aktifkan lagi dan tampilkan `error.message`.
2. **ETag**: simpan `ETag` dari setiap response surat; `PUT`/`DELETE` wajib mengirim `If-Match` berisi nilai itu apa adanya. Jalankan mutasi ke surat yang sama **berurutan** (antrean di klien); unggahan ke MinIO tetap boleh paralel. Bila 412: ambil ulang detail. Bila perbedaannya hanya lampiran, ulangi simpan dengan `ETag` baru; bila isinya berbeda, tampilkan "Draf diubah di tab/perangkat lain".
3. **Upload** 3 langkah: upload-url, lalu POST multipart ke MinIO (semua `fields` dulu, file terakhir; progress lewat XHR), lalu konfirmasi. Isi nilai field `Content-Type` yang sudah ada, jangan menambah field baru. Back-end tidak membatasi jenis file; pembatasan jenis di front-end murni keputusan UX.
4. **Dropdown** memakai `id` dari `/lookups` (bukan `code`). Form saat ini masih memakai kode untuk sifat dan kecepatan, dan pilihan kecepatan `BIASA` belum ada.
5. **Approver tunggal**: pemilih approver menjadi pilihan tunggal (`approverId`). Reviewer tetap bertahap, minimal 1.
6. **Penerima**: Kepada dan Tembusan bisa berisi unit atau alias dari `/recipients/aliases`.
7. **Tombol** yang tampil mengikuti `allowedActions` dari detail surat.
8. **401**: arahkan ke halaman login (token kedaluwarsa atau sudah logout). **429**: tampilkan pesan dan tunggu `Retry-After`.

## 11. Akun uji (data dummy development)

Semua akun memakai password yang sama: **lihat `SEED_PASSWORD` di `.env`**. Semua nama, NIK, dan email fiktif.

| Peran dalam uji | Nama | NIK | Role |
|---|---|---|---|
| Drafter | Andi Kurniawan (Staf Pengembangan Aplikasi) | 2015047 | PEGAWAI |
| Reviewer 1 | Fajar Nugroho (Analis Sistem Informasi) | 2012060 | PEGAWAI |
| Reviewer 2 | Dewi Lestari (AVP Pengembangan Aplikasi) | 2003035 | PIMPINAN |
| Approver | Agus Setiawan (SVP Teknologi Informasi) | 1995021 | PIMPINAN |
| Approver lain | Ratna Kusumawati (Direktur Keuangan dan Umum) | 1889002 | PIMPINAN |
| Penerima (tujuan Departemen SDM) | Sri Wahyuni (VP Sumber Daya Manusia) | 1998102 | PIMPINAN |
| Admin | Rudi Hartono (Administrator Sistem) | 2014090 | ADMIN |
| User nonaktif (login ditolak) | Slamet Riyadi | 1990170 | PEGAWAI |
| User perusahaan lain (PKG) | Lestari Ningsih | 2019203 | PEGAWAI |
