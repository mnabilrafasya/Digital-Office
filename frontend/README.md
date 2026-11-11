# Digital Office — Frontend

Aplikasi antarmuka web persuratan digital menggunakan Next.js (App Router), TypeScript, dan Tailwind CSS, terintegrasi dengan backend Go.

---

## 🚀 Panduan Memulai (Getting Started)

### 1. Konfigurasi Environment
Salin file `.env.example` menjadi `.env.local`:
```bash
cp .env.example .env.local
```
Isi konfigurasi default:
```env
# URL server backend Go (default: port 8080)
BACKEND_URL=http://localhost:8080
```

### 2. Menjalankan Server Development
Pastikan dependensi sudah terpasang, lalu jalankan server:
```bash
npm install
npm run dev
```
Aplikasi dapat diakses di [http://localhost:3000](http://localhost:3000).

---

## 🔐 Autentikasi & Akun Uji Coba

Frontend menggunakan sistem autentikasi cookie `httpOnly` yang dikelola oleh Next.js Server Actions dan dilindungi oleh `proxy.ts`.

### Akun Uji Coba (Seed Accounts)

Akun akses cepat, termasuk Arif dan Rudi, didefinisikan di `backend/database/seeds/dev/003_users.sql`.
Untuk menyiapkan backend di PC lain, gunakan `go run ./cmd/migrate up` lalu
`go run ./cmd/seed all` dari folder backend. Seed hanya menambah akun yang belum ada.
Pada halaman login (`/login`), terdapat fitur **"Akses Cepat Developer"** untuk login instan dalam 1 klik tanpa perlu mengetik kredensial secara manual:

| Peran | Nama | NIP / NIK | Fungsi Uji Coba |
|---|---|---|---|
| **Superadmin** | Arif Wicaksono | `2014091` | Seluruh akses master data & pengujian alur surat |
| **Drafter** | Andi Kurniawan | `2015047` | Membuat dan mengedit draf surat |
| **Reviewer** | Fajar Nugroho | `2012060` | Meninjau dan memverifikasi surat |
| **Approver** | Agus Setiawan | `1995021` | Menyetujui dan menandatangani surat |
| **Admin** | Rudi Hartono | `2014090` | Manajemen master data & sistem perusahaan |

> **Password dev / seed**: Mengikuti nilai `SEED_PASSWORD` di `.env.local` frontend (disamakan dengan `SEED_PASSWORD` pada `.env` backend).

---

## 📋 Status Integrasi Backend

### 1. Modul Autentikasi
- `POST /api/v1/auth/login` &rarr; Terintegrasi penuh (menyimpan session cookie).
- `GET /api/v1/auth/me` &rarr; Terintegrasi (menampilkan nama user dan jabatan pada Header).
- Middleware proteksi rute &rarr; Mengarahkan pengunjung belum login ke `/login`.

### 2. Modul Buat Surat (`/surat/baru`)
- **Jenis Surat** &rarr; Mengambil data live via `GET /api/v1/lookups?category=LETTER_TYPE`.
- **Klasifikasi Arsip** &rarr; Pencarian live combobox via `GET /api/v1/lookups?category=CLASSIFICATION`.
- **Kecepatan Surat** &rarr; Mengambil data live via `GET /api/v1/lookups?category=URGENCY`.
- **Unit Tujuan & Tembusan** &rarr; Pencarian live via `GET /api/v1/units?q=...`.
- **Reviewer & Approver** &rarr; Pencarian kandidat pegawai live via `GET /api/v1/users/candidates?type=...`.

### 3. Modul Master Data (`/master-data`)
Halaman `/master-data` memakai API backend melalui proxy sesi `/api/v1`. Kategori dan tindakan mengikuti permission dari `/auth/me`:

- `COMPANY_MANAGE`: perusahaan, aktif/nonaktif, dan tambah admin perusahaan.
- `ORG_MANAGE`: unit kerja, jabatan, serta jenis surat (`LETTER_TYPE`) dan klasifikasi arsip (`CLASSIFICATION`) khusus perusahaan.
- `USER_MANAGE`: user, mutasi jabatan/role, aktif/nonaktif, reset password.
- `TEMPLATE_MANAGE`: daftar administratif serta aktif/nonaktif template.
- `TEMPLATE_CREATE`: tambah/edit berkas template `.docx` dan melihat referensi persuratan (`CONFIDENTIALITY`, `URGENCY`).
- `COMPANY_SETTINGS`: format nomor surat dan watermark perusahaan, termasuk pratinjau.

Referensi global ditandai **Standar sistem** (`readOnly: true`) dan dilindungi dari perubahan. Tidak ada hapus permanen; data cukup dinonaktifkan agar riwayat surat tetap terjaga.

Pemeriksaan tipe:
```sh
npx tsc --noEmit --incremental false
```

### Monitoring Surat

Menu Monitoring Surat (`/rekap`) menggabungkan pantauan proses dan riwayat aktivitas.
Tab Proses Surat menampilkan surat, petugas saat ini, status, dan lama menunggu.
Tab Riwayat Aktivitas menampilkan waktu, pengguna, aktivitas berbahasa Indonesia,
dan data terkait. Detail perubahan ditampilkan sebagai ringkasan, tanpa JSON/IP.
Tautan lama `/permission` diarahkan ke `/rekap?tab=activity`. API audit yang sudah
ada tetap digunakan.
