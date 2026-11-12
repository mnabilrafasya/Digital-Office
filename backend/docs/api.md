# Back-end Digital Office

Jalankan dari folder `backend`.

## Perintah harian

| Perintah | Fungsi |
|---|---|
| `docker compose up -d` | Menyalakan SQL Server, MinIO, Gotenberg |
| `go run ./cmd/api` | Menjalankan server di `http://localhost:8080` |
| `go run ./cmd/migrate up` lalu `go run ./cmd/seed all` | Setelah `git pull` (aman, tidak menghapus data) |
| `go test ./...` | Menjalankan semua test |

Pertama kali: isi `.env` (dari `.env.example`), simpan snapshot HC di `data/hc/organization.json`, lalu jalankan keempat perintah di atas.

## Migration

| Perintah | Fungsi |
|---|---|
| `go run ./cmd/migrate up` | Menerapkan migration baru (development: membuat database bila belum ada) |
| `go run ./cmd/migrate version` | Melihat versi database |
| `go run ./cmd/migrate down N` | Membongkar N migration terakhir (data di dalamnya hilang) |
| `go run ./cmd/migrate force V` | Memperbaiki status *dirty* setelah migration gagal |
| `go run ./cmd/migrate reset --yes` | Development saja: hapus database, buat ulang, lalu `up` (semua data hilang) |

## Seed

| Perintah | Fungsi |
|---|---|
| `go run ./cmd/seed all` | `reference` + `dev` + `templates` |
| `go run ./cmd/seed reference` | Data acuan: perusahaan, role, permission, lookup, posisi |
| `go run ./cmd/seed dev` | Data dummy PSP & PKG (development). Database kosong: diisi semua; sudah berisi data: hanya menambah yang belum ada |
| `go run ./cmd/seed templates` | Template naskah ke MinIO (PSP 5, PKG 1) |
| `go run ./cmd/sync-hc` | Menambah unit PSP baru dari snapshot HC (`-api` dari API HC, `-company KODE` untuk perusahaan lain) |

Semua seed aman diulang dan tidak menimpa pengaturan admin. Password akun dummy = `SEED_PASSWORD` di `.env`.

## Konvensi API

- Semua endpoint di bawah `/api/v1`, kecuali `GET /health`. Selain login, wajib header `Authorization: Bearer <accessToken>`.
- Response sukses `{"data": ...}` (daftar: `+ "meta": {page, pageSize, total}`); error `{"error": {"code", "message"}}`.
- Kode HTTP: 400 isian salah, 401 belum login, 403 tidak punya hak, 404 tidak ada/tidak boleh dilihat, 409 tidak sesuai status/giliran, 412 `If-Match` kedaluwarsa, 428 `If-Match` tidak dikirim.
- Semua ID dikirim sebagai string; surat memakai uuid. Paginasi `page` (≥1) dan `pageSize` (1–100, bawaan 20).
- Unggahan `multipart/form-data`, field `file`, maks 25 MB.
- Ubah draf wajib `If-Match: <ETag>` dari response terakhir.

## Akun dan data acuan

| Endpoint | Fungsi |
|---|---|
| `POST /auth/login` `{employeeNumber, password}` | Login → `accessToken`, `expiresIn`, `user` |
| `POST /auth/logout` | Logout dari semua perangkat |
| `GET /auth/me` | User login: jabatan, unit, perusahaan, role, `permissions` |
| `GET /lookups?category=` | Pilihan: `LETTER_TYPE`, `LETTER_STATUS`, `CLASSIFICATION`, `CONFIDENTIALITY`, `URGENCY` |
| `GET /units?q=` | Unit aktif perusahaan (pilihan penerima) |
| `GET /users/candidates?type=REVIEWER\|APPROVER&q=&limit=` | Calon reviewer/approver |
| `GET /recipients/aliases` | Alias penerima per posisi (VP, AVP, ...) beserta `levelId` |

## Surat

**Jenis surat** (`letterType.code`) dan arah (`direction`):

| Jenis | `direction` | Ringkas |
|---|---|---|
| Nota dinas, surat edaran, dll. | `INTERNAL` | Penerima unit/alias, wajib penanda QR, terkirim otomatis saat approver terakhir setuju |
| `SURAT_EKSTERNAL` | `OUTGOING` | Satu tujuan luar, tanpa tembusan, tanpa QR, ditandatangani basah lalu scan diunggah |
| `SURAT_MASUK_EKSTERNAL` | `INCOMING` | Scan surat dari luar dicatat lalu langsung didisposisikan |

**Status:** `DRAFT` → `IN_REVIEW` → `IN_APPROVAL` → `SENT` (internal) atau `APPROVED` → `SENT` (eksternal keluar). Dikembalikan = `RETURNED_TO_DRAFTER`. Surat masuk = `RECEIVED`.

**Draf**

| Endpoint | Fungsi |
|---|---|
| `GET /letters?scope=mine\|tasks&status=&q=` | Surat saya / tugas saya (menunggu tindakan saya, termasuk `APPROVED` milik approver terakhir) |
| `POST /letters` | Buat draf (body boleh kosong). 201 + ETag |
| `GET /letters/{id}` | Detail: `steps`, `currentAssignee`, `allowedActions`, `approval`, `external`, ETag |
| `PUT /letters/{id}` + `If-Match` | Ganti seluruh isi draf (hanya drafter, saat `DRAFT`/`RETURNED_TO_DRAFTER`) |
| `GET /letters/{id}/history` | Riwayat aksi alur |

Isi draf: `letterTypeId, templateId, classificationId, confidentialityId, urgencyId, title, summary, specialNote, reviewerIds[], approverIds[]`, ditambah:
- internal: `recipients: {to: [{unitId} | {aliasLevelId}], cc: [...]}`;
- eksternal keluar: `externalRecipient: {name, address, attention}` (`recipients` harus kosong).

Syarat ajukan: semua isian wajib terisi, minimal 1 reviewer dan 1 approver (maks 10, berurutan), naskah sudah diunggah, dan untuk internal naskah berisi `###QRCode###`.

**Alur** (tombol mengikuti `allowedActions`)

| Endpoint | Siapa | Hasil |
|---|---|---|
| `POST /letters/{id}/submit` | Drafter | Ke reviewer 1 (ajukan ulang juga mulai dari reviewer 1) |
| `POST /letters/{id}/approve` `{note?}` | Reviewer/approver yang mendapat giliran | Ke reviewer/approver berikutnya. Approver terakhir: nomor + tanggal terbit, internal → `SENT`, eksternal → `APPROVED` |
| `POST /letters/{id}/return` `{note}` | Reviewer/approver yang mendapat giliran | Selalu kembali ke drafter (`RETURNED_TO_DRAFTER`) |

**Naskah dan PDF**

| Endpoint | Fungsi |
|---|---|
| `PUT /letters/{id}/document` (file .docx) | Unggah naskah (drafter, saat draf). 201 + versi baru |
| `GET /letters/{id}/versions` | Daftar versi naskah |
| `GET /letters/{id}/document?version=&inline=1` | Unduh: .docx sebelum terbit, PDF resmi sesudahnya. `inline=1` untuk tampil di browser |
| `GET /letters/{id}/document/filled` | .docx terisi otomatis (NoSurat, TglSurat, Sifat, QR) untuk diekspor editor |
| `GET /letters/{id}/document/status` | `pdfReady`: apakah PDF resmi sudah disimpan |
| `PUT /letters/{id}/document/final` (file PDF) | Simpan PDF hasil ekspor editor (sekali; 201 baru, 200 sudah ada) |
| `GET /letters/{id}/qr` | PNG QR persetujuan (internal saja) |
| `POST /letters/{id}/attachments` (file) | Tambah lampiran (saat draf). 201 + ETag |
| `GET /letters/{id}/attachments/{attachmentId}` | Unduh lampiran |
| `DELETE /letters/{id}/attachments/{attachmentId}` | Hapus lampiran (saat draf). 204 + ETag |

Penanda naskah: `###NoSurat###`, `###TglSurat###`, `###Sifat###`, `###QRCode###` (huruf besar/kecil bebas, posisi bebas). Pada surat eksternal penanda QR dikosongkan. PDF resmi yang diunduh/dicetak diberi watermark perusahaan, kecuali scan bertanda tangan.

**Surat eksternal**

| Endpoint | Fungsi |
|---|---|
| `PUT /letters/{id}/document/signed` (file PDF) | Approver terakhir mengunggah scan bertanda tangan basah saat `APPROVED` → `SENT`. Scan menjadi PDF resmi dan arsipnya masuk inbox drafter, reviewer, dan approver (tab Eksternal, hanya dibaca) |
| `POST /letters/incoming` (multipart) | Siapa pun mencatat surat masuk. Field: `file` (PDF), `title`, `sender`, `referenceNumber`, `letterDate` (YYYY-MM-DD), `userIds` dan/atau `aliasLevelIds` (dipisah koma, total 1–50), `note`. 201, status `RECEIVED` |

Surat masuk: pengunggah mendapat item `DISPOSED`, tiap tujuan (alias dipecah ke anggotanya) mendapat item disposisi `UNREAD` berisi `note`. Selanjutnya diteruskan seperti surat biasa lewat inbox.

Cara cetak surat eksternal keluar: `GET document/filled` → ekspor PDF di editor → `PUT document/final` → `GET document` (ber-watermark) → cetak, tanda tangan basah, scan → `PUT document/signed`.

## Template naskah

| Endpoint | Fungsi |
|---|---|
| `GET /templates?letterTypeId=` | Template aktif perusahaan |
| `GET /templates/{id}/file` | Unduh .docx |
| `POST /templates` (multipart: `file`, `name`, `description`, `letterTypeId`) | Tambah template |
| `PUT /templates/{id}` (multipart, file opsional) | Ganti data template (pembuat) |

## Inbox

| Endpoint | Fungsi |
|---|---|
| `GET /inbox?status=&category=INTERNAL\|EXTERNAL&q=` | Surat yang saya terima + `counts` per status. `category` kosong = semua |
| `GET /inbox/{itemId}` | Detail item + riwayat penerusan + `allowedActions`. `UNREAD` otomatis menjadi `READ` |
| `POST /inbox/{itemId}/disposition` `{userIds[], unitIds[], note?}` | Disposisi (butuh hak disposisi) → `DISPOSED` |
| `POST /inbox/{itemId}/redirect` `{userId, note?}` | Alihkan ke orang lain → `REDIRECTED` |

Status item: `UNREAD`, `READ`, `DISPOSED`, `REDIRECTED`. Tiap item membawa `letter.direction` (`INTERNAL`/`OUTGOING`/`INCOMING`) dan `letter.external` (tujuan atau pengirim, nomor, tanggal surat luar). Arsip `OUTGOING` tidak bisa didisposisi atau dialihkan (409).

## Lacak dan audit

| Endpoint | Fungsi |
|---|---|
| `GET /letters/{id}/tracking` | Posisi surat sekarang, linimasa, dan distribusi penerima |
| `GET /letters/{id}/audit` | Audit log satu surat (`AUDIT_VIEW`) |
| `GET /audit-logs?from=&to=&actorId=&action=&entityType=&q=` | Audit log perusahaan (`AUDIT_VIEW`) |
| `GET /audit-logs/{logId}` | Detail log |
| `GET /audit-logs/export?...` | Unduh CSV (filter sama) |
| `GET /monitoring/letters?status=&minWaitingHours=` | Surat yang tertahan di satu orang |

## Master data (admin)

Hak = **role** + **posisi**. Posisi DIREKTUR, SVP, VP, AVP otomatis bisa approve dan disposisi, apa pun role-nya.

| Role | Bisa |
|---|---|
| SUPERADMIN | Kelola perusahaan, ditambah semua hak ADMIN |
| ADMIN | Pengaturan perusahaan, unit, jabatan, user, template, audit log |
| PEGAWAI | Membuat dan me-review surat, membuat template |

Semua di bawah `/api/v1`. Tidak ada hapus permanen (cukup nonaktif), semua perubahan tercatat di audit log, dan admin hanya mengelola perusahaannya sendiri.

**SUPERADMIN — perusahaan**

| Endpoint | Fungsi |
|---|---|
| `GET /admin/companies` | Daftar perusahaan |
| `POST /admin/companies` `{code, name, city}` | Tambah perusahaan |
| `GET /admin/companies/{code}` | Detail |
| `PUT /admin/companies/{code}` `{name, city}` | Ubah nama dan kota |
| `PATCH /admin/companies/{code}/status` `{active}` | Aktif/nonaktif (nonaktif = user-nya tidak bisa login) |
| `GET /admin/companies/{code}/units` | Unit perusahaan itu |
| `POST /admin/companies/{code}/admins` `{unitId \| unitName, positionName, employeeNumber, name, email, password}` | Tambah admin perusahaan |

**ADMIN — pengaturan perusahaan**

| Endpoint | Fungsi |
|---|---|
| `GET /company` | Lihat pengaturan |
| `PUT /company` `{letterNumberFormat, watermarkText}` | Simpan format nomor dan watermark (kosong = tanpa watermark). Watermark hanya ditempel di PDF resmi (unduh/cetak) surat yang sudah disetujui, tidak di tampilan layar |
| `GET /company/number-preview?format=` | Contoh nomor surat |
| `GET /company/watermark-preview?text=` | Gambar watermark |

**Nomor surat**

- Format bawaan `{SEQ}/{COMP}/{MM_ROMAN}/{YYYY}`, contoh `015/PSP/XI/2026`. Keempat token wajib ada; teks lain bebas (huruf, angka, tanda baca, maks 100 karakter).
- `{SEQ}` = nomor urut berjalan (minimal 3 digit) per perusahaan, untuk semua jenis surat, dan mulai lagi dari `001` setiap bulan. `{COMP}` = kode perusahaan, `{MM_ROMAN}` = bulan romawi, `{YYYY}` = tahun.
- Nomor diberikan saat approver terakhir menyetujui; bulan dan tahun mengikuti waktu persetujuan (`APP_TIMEZONE`). Surat masuk eksternal tidak diberi nomor.
- Format baru hanya berlaku untuk surat yang disetujui sesudahnya. Format tersimpan yang tidak memenuhi aturan otomatis diganti format bawaan.

**ADMIN — unit dan jabatan**

| Endpoint | Fungsi |
|---|---|
| `GET /admin/units` | Daftar unit |
| `POST /admin/units` `{name, unitType, parentId}` | Tambah unit |
| `PUT /admin/units/{id}` | Ubah unit |
| `PATCH /admin/units/{id}/status` `{active}` | Aktif/nonaktif (ditolak bila masih ada sub-unit atau user aktif) |
| `GET /admin/position-levels` | Pilihan posisi |
| `GET /admin/positions?unitId=` | Daftar jabatan |
| `POST /admin/positions` `{unitId, name, levelId, isUnitHead}` | Tambah jabatan (maks 1 pimpinan unit) |
| `PUT /admin/positions/{id}` | Ubah jabatan |
| `PATCH /admin/positions/{id}/status` `{active}` | Aktif/nonaktif (ditolak bila masih dipegang user aktif) |

**ADMIN — user**

| Endpoint | Fungsi |
|---|---|
| `GET /admin/users?q=&unitId=&roleId=&status=` | Daftar user |
| `GET /admin/users/{id}` | Detail |
| `POST /admin/users` `{employeeNumber, name, email, positionId, roleId, password}` | Tambah user |
| `PUT /admin/users/{id}` `{employeeNumber, name, email, positionId, roleId}` | Ubah, mutasi, ganti role |
| `PATCH /admin/users/{id}/status` `{active}` | Aktif/nonaktif |
| `POST /admin/users/{id}/reset-password` `{password}` | Reset password |
| `GET /admin/roles` | Pilihan role |

Mutasi, ganti role, nonaktif, dan reset password langsung mengakhiri sesi user. Admin tidak bisa menonaktifkan atau mengganti role dirinya sendiri, dan tidak bisa mengurus akun SUPERADMIN.

**ADMIN — template**

| Endpoint | Fungsi |
|---|---|
| `GET /admin/templates` | Semua template + jumlah pemakaian |
| `PATCH /admin/templates/{id}/status` `{active}` | Aktif/nonaktif (nonaktif = hilang dari pilihan surat) |

## Master jenis surat dan klasifikasi

Permission `ORG_MANAGE` (ADMIN), selalu di perusahaan pengguna:

| Endpoint | Fungsi |
|---|---|
| `GET /api/v1/admin/lookups/{category}?q=&status=` | Daftar standar global dan data perusahaan (termasuk nonaktif) |
| `POST /api/v1/admin/lookups/{category}` | Tambah `{code, name, note, sortOrder}` |
| `PUT /api/v1/admin/lookups/{category}/{id}` | Ubah `{name, note, sortOrder}`; kode tetap |
| `PATCH /api/v1/admin/lookups/{category}/{id}/status` | Aktif/nonaktif `{active}` |

`category` hanya `LETTER_TYPE` atau `CLASSIFICATION`. Data standar global memiliki
`readOnly: true` dan tidak dapat diubah atau dinonaktifkan. Tambahan data tersimpan
khusus perusahaan pengguna. Kode tidak boleh menduplikasi kode global/perusahaan.
Tidak ada hapus permanen; seluruh mutasi dan audit dilakukan dalam satu transaksi.
Data nonaktif tidak muncul pada `GET /lookups` tetapi riwayat surat tetap tersimpan.
Endpoint ini menggunakan tabel yang ada; tidak memerlukan migration tambahan.

## Akun development

Akun development, termasuk **Arif Wicaksono (2014091, SUPERADMIN)**, didefinisikan
bersama di `database/seeds/dev/003_users.sql`. Password awal mengikuti
`SEED_PASSWORD` di `.env`. Seed tidak mengganti data/password akun yang sudah ada.

Gunakan perintah yang sama pada semua PC, dari folder `backend`:

```sh
go run ./cmd/migrate up
go run ./cmd/seed all
```

Database baru menjalankan seluruh seed development. Database yang sudah berisi user
hanya menjalankan seed pengguna dasar dan seed lanjutan yang aman diulang; akun
hanya ditambahkan bila NIK belum ada. Tidak perlu reset database atau perintah akun khusus.
Seed development membutuhkan `APP_ENV=development` dan data unit HC sesuai panduan awal.

Approve dan disposisi SUPERADMIN tetap mengikuti posisinya, sama seperti role lain.
