# DevExpress Rich Text Editor — Setup & Panduan Tim

> **Status:** Spike / Pengujian Lokal  
> **Hanya tersedia saat `NODE_ENV=development`**  
> Jangan deploy ke production sebelum ada lisensi komersial.

---

## 1. Apa ini?

DevExpress RichEdit adalah editor dokumen berbasis web yang tampilannya mirip Microsoft Word. Bisa membuka, mengedit, dan mengekspor file **.docx**, **.rtf**, dan **.txt** langsung di browser — tanpa server tambahan untuk rendering.

Halaman uji tersedia di: **`http://localhost:3000/dev/editor`**

---

## 2. Persyaratan

- Node.js ≥ 18 (cek: `node --version`)
- npm ≥ 9 (cek: `npm --version`)
- Semua dependency frontend sudah diinstall (lihat langkah 3)

---

## 3. Cara Setup (pertama kali / clone ulang)

### Langkah 1: Install semua dependency frontend

```bash
cd frontend
npm install
```

> Ini sudah mencakup `devextreme` dan `devexpress-richedit` yang ditambahkan pada 25 Okt 2026.
> Kalau error `devexpress-richedit not found`, jalankan:
> ```bash
> npm install devextreme devexpress-richedit
> ```

### Langkah 2: Salin file environment

```bash
# Di folder frontend/
copy .env.example .env.local
```

Edit `.env.local` dan isi:

```env
# URL Backend Go (wajib)
BACKEND_URL=http://localhost:8080

# Password akun seed untuk Akses Cepat Developer
# Samakan dengan SEED_PASSWORD di backend/.env
SEED_PASSWORD=<isi_dari_backend_env>

# (Opsional) Kunci lisensi DevExpress — lihat Langkah 3
NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY=
```

### Langkah 3: (Opsional) Kunci Lisensi DevExpress

Tanpa kunci lisensi, editor **tetap berjalan** tapi muncul watermark "DevExpress — Unlicensed" di layar dan konsol browser. Untuk evaluasi 30 hari ini sudah cukup.

Untuk menghilangkan watermark:

1. Daftar / masuk ke [devexpress.com](https://www.devexpress.com/)
2. Buka **DevExpress Download Manager**
3. Salin **License Key**
4. Tambahkan ke `.env.local`:
   ```env
   NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY=kunci_anda_disini
   ```

> **Penting:** File `.env.local` sudah ada di `.gitignore`. Jangan pernah commit kunci lisensi ke repository.

### Langkah 4: Jalankan server development

```bash
# Di folder frontend/
npm run dev
```

Buka browser: **http://localhost:3000/dev/editor**

---

## 4. Cara Menggunakan Halaman Editor

| Aksi | Cara |
|------|------|
| Buka file | Klik tombol **"Buka File"** → pilih `.docx`, `.rtf`, atau `.txt` dari komputer |
| Edit dokumen | Langsung di area editor (toolbar Microsoft Word-like) |
| Simpan hasil | Klik **"Simpan sebagai .docx"** → file terunduh ke komputer |
| Dokumen baru | Refresh halaman |

---

## 5. Struktur File Editor

```
frontend/src/app/dev/
├── layout.tsx              # Layout kosong (tanpa sidebar/header), diblokir di production
└── editor/
    ├── page.tsx            # Server component — guard production, render client
    ├── editor-client.tsx   # Client wrapper + dynamic import (ssr: false)
    └── rich-edit-wrapper.tsx  # Komponen inti DevExpress RichEdit
```

---

## 6. Keterbatasan Saat Ini (Spike)

| Keterbatasan | Keterangan |
|---|---|
| Dokumen tidak tersimpan ke backend | Hanya simpan ke komputer lokal. Backend belum punya endpoint dokumen. |
| Template belum tersedia | Backend belum ada endpoint daftar template. |
| Variabel `###NoSurat###` belum diganti otomatis | Perlu integrasi backend saat nomor surat dibuat. |
| Tanda tangan & QR Code | Bergantung pada backend: dibuat saat approver menyetujui. |
| Editor belum terpasang di halaman Buat Surat / Review | Menunggu backend siap; sekarang masih placeholder. |

---

## 7. Yang Dibutuhkan dari Backend (Tim Backend)

Agar editor bisa dipakai di production, backend perlu menyediakan:

| Endpoint | Keterangan |
|---|---|
| `GET /letters/{id}/document` | Unduh file .docx versi terkini surat |
| `PUT /letters/{id}/document` | Simpan dokumen (reviewer bisa edit; approver read-only) |
| `GET /templates` | Daftar template surat (.docx dengan variabel `###NoSurat###`, dll) |
| `GET /templates/{id}` | Unduh template tertentu |
| `POST /letters/{id}/finalize` | Buat file FINAL + QR code setelah approver setuju |

Sampai endpoint ini tersedia, halaman `/dev/editor` hanya dipakai untuk demo dan eksplorasi teknis.

---

## 8. Troubleshooting

### Editor tidak muncul / blank
- Buka DevTools browser (F12) → tab Console → cari error merah
- Pastikan `npm install` sudah dijalankan dan tidak ada error
- Coba hard refresh: `Ctrl+Shift+R`

### Error: `Cannot find module 'devexpress-richedit'`
```bash
cd frontend
npm install devextreme devexpress-richedit
```

### Muncul pesan "Unlicensed"
- Normal untuk evaluasi. Tambahkan `NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY` di `.env.local` untuk menghilangkannya.

### Halaman 404 di production
- Benar dan disengaja. Halaman `/dev/*` diblokir di production.

---

## 9. Referensi

- [Dokumentasi DevExpress RichEdit](https://docs.devexpress.com/AspNetCore/400373/rich-edit)
- [npm: devexpress-richedit](https://www.npmjs.com/package/devexpress-richedit)
- [Download Manager DevExpress (untuk kunci lisensi)](https://www.devexpress.com/ClientCenter/DownloadManager/)
