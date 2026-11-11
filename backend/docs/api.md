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

## Master data

Hak = **role** + **posisi**. Posisi DIREKTUR, SVP, VP, AVP otomatis bisa approve dan disposisi, apa pun role-nya.

| Role | Bisa |
|---|---|
| SUPERADMIN | Kelola perusahaan |
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
| `PUT /company` `{letterNumberFormat, watermarkText}` | Simpan format nomor dan watermark (kosong = tanpa watermark) |
| `GET /company/number-preview?format=` | Contoh nomor surat |
| `GET /company/watermark-preview?text=` | Gambar watermark |

Token format nomor: `{SEQ}`, `{COMP}`, `{YYYY}` (wajib), `{MM_ROMAN}`.

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
