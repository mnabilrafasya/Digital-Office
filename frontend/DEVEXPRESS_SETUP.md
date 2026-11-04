# Panduan Penggunaan & Setup DevExpress Rich Text Editor

> **Lokasi Utama:** Halaman **Buat Surat** (`/surat/baru`) — **Section 3: Isi Surat & Tata Naskah Digital**  
> **Lokasi Review & Approval:** Halaman **Detail Surat** (`/surat/[id]`)  
> **Halaman Sandbox Uji:** `/dev/editor` (hanya aktif saat mode development)

---

## 1. Ringkasan Fitur Editor

DevExpress RichEdit adalah editor naskah dinas berbasis web dengan antarmuka dan ukuran halaman setara **Microsoft Word**. Fitur yang sudah tersedia di frontend:

- **Standar Halaman & Penggaris Microsoft Word Indonesia**:
  - Satuan penggaris menggunakan **Centimeter (`cm`)** (area ketik `0–16 cm` di atas kertas A4 `21 cm x 29,7 cm` dengan margin Normal Word `2,54 cm` di keempat sisi).
  - Klik pada kotak **Font Name** atau **Font Size** di Ribbon otomatis memblok teksnya sehingga pengguna dapat langsung mengetik nama atau ukuran huruf.
  - AutoCorrect Bahasa Indonesia aktif untuk memperbaiki salah ketik umum secara otomatis.
- **Template Naskah Dinas Resmi (`.docx` Asli)**:
  1. **Surat Dinas Resmi Standar** (`public/templates/surat-dinas-standar.docx`)
  2. **Nota Dinas Internal** (`public/templates/nota-dinas.docx`)
  3. **Surat Undangan Rapat** (`public/templates/undangan-rapat.docx`)
  4. **Surat Perintah Tugas / SPT** (`public/templates/surat-tugas.docx`)
  5. **Dokumen Kosong (Blank)** (`public/templates/dokumen-kosong.docx`)
- **Sinkronisasi Otomatis Data Form**: Mengisi otomatis judul/perihal, unit penerima (Kepada Yth), dan pejabat approver (kolom tanda tangan) dari Section 1 ke dalam naskah `.docx`.
- **Integrasi Penyimpanan Versi `.docx` ke MinIO Backend**:
  - Otomatis mengunggah file `.docx` ke `PUT /api/v1/letters/{id}/document` saat Drafter klik **Simpan Draft** atau **Kirim Surat**.
  - Reviewer aktif dapat mengedit naskah langsung di halaman `/surat/[id]` dan menyimpan revisi sebagai versi `.docx` baru.
- **Buka & Unduh Dokumen Lokal (`.docx`, `.rtf`, `.txt`)**, **Cetak / Simpan PDF**, **Zoom Interaktif (`50%–200%`)**, dan **Mode Layar Penuh (Fullscreen)**.

---

## 2. Catatan Mengenai Lisensi DevExpress

- **Status Evaluasi:** DevExpress menyediakan evaluasi lokal selama 30 hari. Pada masa evaluasi lokal ini, seluruh fitur editor berfungsi normal 100%.
- **Kunci Lisensi (Opsional):** Jika tim sudah memiliki kunci lisensi komersial dari DevExpress:
  1. Buka file `frontend/.env.local`
  2. Isi nilai:
     ```env
     NEXT_PUBLIC_DEVEXPRESS_LICENSE_KEY=kunci_lisensi_anda_disini
     ```
  3. Restart dev server (`npm run dev`).

---

## 3. Struktur File Komponen di Frontend

```
frontend/
├── public/
│   └── templates/                          # File .docx template fisik (A4, margin 2,54 cm, ###NoSurat###)
│       ├── surat-dinas-standar.docx
│       ├── nota-dinas.docx
│       ├── undangan-rapat.docx
│       ├── surat-tugas.docx
│       └── dokumen-kosong.docx
└── src/
    ├── lib/
    │   ├── api/
    │   │   └── letter-document.ts          # Klien API PUT/GET /api/v1/letters/{id}/document & /versions
    │   └── templates/
    │       └── letter-templates.ts         # Registri template .docx & sinkronisasi placeholder ke editor
    ├── components/
    │   ├── review/
    │   │   └── document-viewer/
    │   │       └── review-document-inner.tsx # Viewer/Editor naskah di halaman Detail, Review & Approval
    │   └── surat/
    │       ├── basic-fields.tsx            # Dropdown jenis & template di Section 1
    │       └── letter-editor/
    │           ├── index.tsx               # Wrapper dinamis (ssr: false) dengan loading skeleton
    │           ├── rich-edit-surat-inner.tsx # Komponen inti DevExpress RichEdit Section 3
    │           ├── letter-editor-toolbar.tsx # Command bar atas gaya Word (Template, Berkas, Zoom)
    │           ├── word-editor-config.ts   # Konfigurasi standar Word (cm, A4, Ribbon, AutoCorrect ID)
    │           └── use-select-on-focus.ts  # Auto-select input font/size & shortcut Ctrl+S
    └── app/
        ├── (app)/surat/baru/page.tsx       # Halaman Buat Surat (unggah .docx otomatis saat Simpan/Submit)
        └── dev/editor/                     # Halaman sandbox dev (/dev/editor)
```
