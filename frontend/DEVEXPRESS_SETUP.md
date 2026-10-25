# Panduan Penggunaan & Setup DevExpress Rich Text Editor

> **Lokasi Utama:** Halaman **Buat Surat** (`/surat/baru`) — **Section 3: Isi Surat & Tata Naskah Digital**  
> **Halaman Sandbox Uji:** `/dev/editor` (hanya aktif saat mode development)

---

## 1. Ringkasan Fitur Editor

DevExpress RichEdit adalah editor naskah dinas berbasis web dengan antarmuka toolbar mirip **Microsoft Word**. Fitur yang sudah tersedia di frontend:

- ✅ **Pengeditan Tata Naskah Lengkap**: Pengetikan, pemformatan teks (Bold, Italic, Underline, Font, Ukuran), paragraf (Rata Kiri, Kanan, Tengah, Justify), tabel, penomoran/bullet, dan margin.
- ✅ **Template Naskah Dinas Resmi Siap Pakai**:
  1. **Surat Dinas Resmi Standar** (Kop PT Pupuk Sriwidjaja, Nomor `###NoSurat###`, Sifat, Lampiran, Perihal, Yth, TTD & Tembusan)
  2. **Nota Dinas Internal** (Format internal Yth, Dari, Tanggal, Sifat, Perihal, Bab Pembahasan, TTD Elektronik)
  3. **Surat Undangan Rapat** (Format undangan dinas lengkap dengan tabel Hari, Waktu, Tempat/Media, Agenda)
  4. **Surat Perintah Tugas / SPT** (Format penugasan dinas dengan tabel daftar pegawai yang ditugaskan)
  5. **Dokumen Kosong (Blank)** (Halaman putih bersih tanpa format awal)
- ✅ **Sinkronisasi Otomatis Data Form**: Mengambil otomatis judul/perihal, sifat surat, unit penerima (Kepada Yth), dan pejabat approver (kolom tanda tangan) dari Section 1.
- ✅ **Buka Dokumen Lokal (`.docx`, `.rtf`, `.txt`)**: Membuka draft file Word dari komputer langsung ke dalam editor.
- ✅ **Unduh Dokumen (`.docx`)**: Menyimpan / mengekspor naskah yang sedang diedit menjadi file `.docx`.
- ✅ **Mode Layar Penuh (Fullscreen)**: Memperbesar editor ke layar penuh agar nyaman mengetik naskah panjang tanpa terganggu scroll halaman form.

---

## 2. Panduan Penggunaan untuk Pengguna / Drafter

### A. Mengakses Editor
1. Buka aplikasi di browser (misal `http://localhost:3000`).
2. Login sebagai pengguna/drafter.
3. Klik menu **"Buat Surat"** di sidebar kiri.
4. Isi **Section 1: Detail Surat** (Jenis Surat, Klasifikasi, Sifat, Perihal, Penerima, dan Reviewer/Approver).
5. (Opsional) Tambahkan berkas pendukung pada **Section 2: Lampiran**.
6. Gulir ke bawah ke **Section 3: Isi Surat & Tata Naskah Digital**. Editor Word akan otomatis memuat template yang sesuai.

### B. Memilih dan Mengganti Template Naskah
- Pada toolbar atas Section 3, temukan dropdown **"Template Naskah"**.
- Pilih template yang diinginkan:
  - *Surat Dinas Resmi Standar*
  - *Nota Dinas Internal*
  - *Surat Undangan Rapat*
  - *Surat Perintah Tugas (SPT)*
  - *Dokumen Kosong (Blank)*
- Editor akan langsung berganti ke format template yang dipilih beserta variabel isiannya.

### C. Menyinkronkan Perubahan dari Section 1
- Jika Anda mengubah judul perihal, sifat surat, unit penerima, atau pejabat penandatangan di Section 1:
- Klik tombol **"Sinkronkan Form"** pada toolbar Section 3.
- Variabel naskah di editor akan diperbarui dengan data terbaru dari Section 1.

### D. Mengetik dan Memformat Surat
- Klik pada area lembar kerja dokumen untuk mulai mengetik.
- Gunakan toolbar Ribbon di bagian atas editor:
  - **Tab Home**: Ganti jenis huruf (Font), ukuran, warna teks, perataan teks (kiri/tengah/kanan/justify), spasi baris, dan bullet/penomoran.
  - **Tab Insert**: Menyisipkan tabel, simbol, atau jeda halaman.
  - **Tab Page Layout**: Mengatur margin halaman, orientasi (portrait/landscape), dan ukuran kertas.

### E. Membuka Dokumen Word dari Komputer
- Jika Anda sudah memiliki draf di komputer dalam format `.docx`:
- Klik tombol **"Buka .docx"** di toolbar atas Section 3.
- Pilih file dari komputer Anda. Dokumen akan terbuka langsung di dalam editor web.

### F. Menyimpan / Mengunduh Hasil Ketikan
- Klik tombol **"Unduh .docx"** di toolbar atas Section 3.
- File Microsoft Word `.docx` akan langsung terunduh ke komputer Anda.

### G. Mode Layar Penuh (Fullscreen)
- Untuk kenyamanan mengetik surat panjang tanpa terganggu form lainnya:
- Klik tombol ikon **Maximize / Layar Penuh** di ujung kanan toolbar Section 3.
- Untuk kembali ke tampilan form biasa, klik tombol ikon **Minimize** di tempat yang sama.

---

## 3. Catatan Mengenai Lisensi DevExpress

- **Status Evaluasi:** DevExpress menyediakan evaluasi lokal selama 30 hari. Pada masa evaluasi lokal ini, seluruh fitur editor (buka file, edit naskah, format tabel, ekspor docx) berfungsi normal 100%.
- **Kunci Lisensi (Opsional):** Jika tim Anda sudah memiliki kunci lisensi komersial dari DevExpress:
  1. Buka file `frontend/.env.local`
  2. Isi nilai:
     ```env
     NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY=kunci_lisensi_anda_disini
     ```
  3. Restart dev server (`npm run dev`).
  4. File `.env.local` sudah masuk ke `.gitignore`, sehingga kunci lisensi tidak akan bocor ke git repository.

---

## 4. Struktur File Komponen di Frontend

```
frontend/src/
├── lib/
│   └── templates/
│       └── letter-templates.ts         # Definisi & generator HTML template dinas (Surat Dinas, ND, Undangan, SPT)
├── components/
│   └── surat/
│       ├── basic-fields.tsx            # Dropdown jenis & template di Section 1
│       └── letter-editor/
│           ├── index.tsx               # Wrapper dinamis (ssr: false) dengan loading skeleton
│           └── rich-edit-surat-inner.tsx # Komponen inti DevExpress RichEdit Section 3
└── app/
    ├── (app)/surat/baru/page.tsx       # Halaman Buat Surat (menggunakan LetterEditorSection)
    └── dev/editor/                     # Halaman sandbox dev (/dev/editor)
```

---

## 5. Yang Diperlukan dari Tim Backend untuk Tahap Lanjutan

Agar naskah yang diketik di editor ini dapat tersimpan permanen di database server dan dapat dibaca oleh Reviewer dan Approver:

1. **Simpan & Muat Dokumen Per Versi**:
   - `PUT /api/v1/letters/{id}/document`: Menerima file/blob `.docx` untuk disimpan di server (tiap simpan = versi baru).
   - `GET /api/v1/letters/{id}/document`: Mengunduh file `.docx` versi terkini dari surat tersebut.
2. **Pemberian Nomor Otomatis**:
   - Backend mengganti placeholder `###NoSurat###` dengan nomor dinas resmi (misal: `015/TI/PSP/X/2026`) saat status surat berubah menjadi `SENT` (disetujui Approver).
3. **Dokumen Final + QR Code**:
   - Backend memproses pembuatan file PDF/DOCX final ber-QR Code penandatanganan elektronik untuk dikirimkan ke Inbox penerima.
