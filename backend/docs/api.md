# Perintah Back-end Digital Office

Jalankan dari folder `backend`, berurutan.

| No | Perintah | Fungsi |
|---|---|---|
| 1 | `docker compose up -d` | Menjalankan SQL Server, MinIO, dan Gotenberg |
| 2 | Simpan snapshot HC ke `data/hc/organization.json` | Data unit kerja PSP (didapat lewat jalur internal, jangan di-commit) |
| 3 | `go run ./cmd/migrate reset --yes` | Mengosongkan database lalu membuat ulang strukturnya |
| 4 | `go run ./cmd/seed all` | Mengisi data acuan, unit PSP dari snapshot HC, akun dummy, dan template naskah |
| 5 | `go run ./cmd/api` | Menjalankan server di `http://localhost:8080` |

Perintah tambahan:

| Perintah | Fungsi |
|---|---|
| `go run ./cmd/migrate up` | Menerapkan migration baru tanpa menghapus data |
| `go run ./cmd/seed reference` | Memperbarui data acuan (role, permission, lookup) tanpa menghapus data; aman diulang |
| `go run ./cmd/seed templates` | Mengunggah template naskah bawaan ke MinIO (database lama yang belum punya template) |
| `go run ./cmd/sync-hc` | Menambah unit PSP baru dari snapshot HC (unit yang sudah ada tidak diubah) |
| `go run ./cmd/sync-hc -api` | Sama, langsung dari API HC (butuh VPN dan token di `.env`) |
| `go test ./...` | Menjalankan semua test |

Memperbarui database lama ke versi master data: `go run ./cmd/migrate up` lalu `go run ./cmd/seed reference`.

## Hak akses

Hak user = permission **role** + permission dari **posisi** (level jabatan).

| Role | Permission |
|---|---|
| `SUPERADMIN` | `LETTER_CREATE`, `LETTER_REVIEW`, `TEMPLATE_CREATE`, `COMPANY_MANAGE` |
| `ADMIN` | `LETTER_CREATE`, `LETTER_REVIEW`, `TEMPLATE_CREATE`, `AUDIT_VIEW`, `COMPANY_SETTINGS`, `ORG_MANAGE`, `USER_MANAGE`, `TEMPLATE_MANAGE` |
| `PEGAWAI` | `LETTER_CREATE`, `LETTER_REVIEW`, `TEMPLATE_CREATE` |

Posisi `DIREKTUR`, `SVP`, `VP`, `AVP` otomatis mendapat `LETTER_APPROVE` dan `INBOX_DISPOSE`, apa pun role-nya. Posisi `STAF` (atau kosong) tidak. Jabatan "pimpinan unit" hanya menentukan penerima surat yang ditujukan ke unit.

## Endpoint master data

Semua di bawah `/api/v1`, wajib login. Admin selalu mengelola perusahaannya sendiri. Tidak ada hapus permanen; semua cukup dinonaktifkan dan setiap perubahan tercatat di audit log.

| Endpoint | Permission | Fungsi |
|---|---|---|
| `GET /admin/companies?q=&status=&page=&pageSize=` | `COMPANY_MANAGE` | Daftar perusahaan beserta jumlah unit, user, surat |
| `POST /admin/companies` `{code, name, city}` | `COMPANY_MANAGE` | Tambah perusahaan (posisi standar ikut dibuat; format nomor bawaan) |
| `GET /admin/companies/{code}` | `COMPANY_MANAGE` | Detail perusahaan |
| `PUT /admin/companies/{code}` `{name, city}` | `COMPANY_MANAGE` | Ubah identitas |
| `PATCH /admin/companies/{code}/status` `{active}` | `COMPANY_MANAGE` | Aktif/nonaktif (nonaktif: semua user-nya tidak bisa login) |
| `GET /admin/companies/{code}/units?q=` | `COMPANY_MANAGE` | Unit aktif perusahaan, untuk memilih unit admin pertama |
| `POST /admin/companies/{code}/admins` `{unitId \| unitName, positionName, employeeNumber, name, email, password}` | `COMPANY_MANAGE` | Tambah admin perusahaan (jabatan baru berposisi STAF) |
| `GET /company` | `COMPANY_SETTINGS` | Pengaturan perusahaan + contoh nomor surat |
| `PUT /company` `{letterNumberFormat, watermarkText}` | `COMPANY_SETTINGS` | Simpan format nomor dan watermark (kosong = tanpa watermark) |
| `GET /company/number-preview?format=` | `COMPANY_SETTINGS` | Validasi + contoh nomor (token `{SEQ}`, `{COMP}`, `{MM_ROMAN}`, `{YYYY}`; wajib `{SEQ}`, `{COMP}`, `{YYYY}`) |
| `GET /company/watermark-preview?text=` | `COMPANY_SETTINGS` | Gambar PNG watermark (tanpa `text` = yang tersimpan) |
| `GET /admin/units?q=&status=` | `ORG_MANAGE` | Semua unit + pimpinan dan jumlah isinya |
| `POST /admin/units` `{name, unitType, parentId}` | `ORG_MANAGE` | Tambah unit |
| `PUT /admin/units/{unitId}` | `ORG_MANAGE` | Ubah nama, jenis, induk (tidak boleh melingkar) |
| `PATCH /admin/units/{unitId}/status` `{active}` | `ORG_MANAGE` | Nonaktif ditolak bila masih ada sub-unit aktif atau user aktif |
| `GET /admin/position-levels` | `ORG_MANAGE` | Pilihan posisi (DIREKTUR, SVP, VP, AVP, STAF) |
| `GET /admin/positions?unitId=&q=&status=&page=&pageSize=` | `ORG_MANAGE` | Daftar jabatan + pemegangnya |
| `POST /admin/positions` `{unitId, name, levelId, isUnitHead}` | `ORG_MANAGE` | Tambah jabatan (posisi wajib; maks 1 pimpinan aktif per unit) |
| `PUT /admin/positions/{positionId}` | `ORG_MANAGE` | Ubah jabatan (hak pemegangnya ikut berubah) |
| `PATCH /admin/positions/{positionId}/status` `{active}` | `ORG_MANAGE` | Nonaktif ditolak bila masih dipegang user aktif |
| `GET /admin/users?q=&unitId=&roleId=&status=&page=&pageSize=` | `USER_MANAGE` | Daftar user |
| `GET /admin/users/{userId}` | `USER_MANAGE` | Detail user |
| `POST /admin/users` `{employeeNumber, name, email, positionId, roleId, password}` | `USER_MANAGE` | Tambah user (password minimal 8 karakter) |
| `PUT /admin/users/{userId}` `{employeeNumber, name, email, positionId, roleId}` | `USER_MANAGE` | Ubah, mutasi, ganti role (mutasi/ganti role mengakhiri sesinya) |
| `PATCH /admin/users/{userId}/status` `{active}` | `USER_MANAGE` | Aktif/nonaktif (mengakhiri sesinya) |
| `POST /admin/users/{userId}/reset-password` `{password}` | `USER_MANAGE` | Ganti password (mengakhiri sesinya) |
| `GET /admin/roles` | `USER_MANAGE` | Pilihan role (SUPERADMIN hanya untuk pemegang `COMPANY_MANAGE`) |
| `GET /admin/templates?status=&q=` | `TEMPLATE_MANAGE` | Semua template + jumlah surat yang memakainya |
| `PATCH /admin/templates/{templateId}/status` `{active}` | `TEMPLATE_MANAGE` | Aktif/nonaktif (nonaktif: hilang dari pilihan surat; draf lama tetap aman) |

Pengaman user: admin tidak bisa menonaktifkan atau mengganti role dirinya sendiri, admin terakhir perusahaan tidak bisa dicabut, dan akun/role SUPERADMIN hanya diurus pemegang `COMPANY_MANAGE`.
