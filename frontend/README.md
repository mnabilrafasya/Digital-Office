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

Frontend menggunakan sistem autentikasi cookie `httpOnly` yang dikelola oleh Next.js Server Actions dan dilindungi oleh `middleware.ts`.

### Akun Uji Coba (Seed Accounts)
Pada halaman login (`/login`), terdapat fitur **"Akses Cepat Developer"** untuk login instan dalam 1 klik tanpa perlu mengetik kredensial secara manual:

| Peran | Nama | NIP / NIK | Fungsi Uji Coba |
|---|---|---|---|
| **Drafter** | Andi Kurniawan | `2015047` | Membuat dan mengedit draf surat |
| **Reviewer** | Fajar Nugroho | `2012060` | Meninjau dan memverifikasi surat |
| **Approver** | Agus Setiawan | `1995021` | Menyetujui dan menandatangani surat |
| **Admin** | Rudi Hartono | `2014090` | Manajemen master data & sistem |

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

*(Catatan: Fitur penyimpanan surat sungguhan dan upload lampiran menunggu endpoint `POST /api/v1/letters` dari tim backend).*
