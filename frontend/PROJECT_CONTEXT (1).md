# Digital Office: Konteks Proyek (Frontend)

Dokumen ini dipakai dua cara:
1. **Konteks** untuk developer dan AI coding assistant (taruh di root repo atau `frontend/`).
2. **Prompt**: bagian 15 siap ditempel ke AI untuk membangun halaman Buat Surat.

Label sumber:
- **[KONFIRMASI]** dari arahan pembimbing atau ringkasan desain database
- **[GAMBAR]** terlihat di mockup/screenshot yang sudah dibuat (label UI mengikuti ini)
- **[SISTEM LAMA]** terlihat di foto sistem Digital Office yang berjalan
- **[USULAN]** belum dikonfirmasi pembimbing. Jangan diperlakukan sebagai fakta.

---

## 1. Ringkasan

Sistem persuratan **Digital Office** (PT Pupuk Sriwidjaja Palembang), multi-group company: **data tiap perusahaan terpisah total**. [KONFIRMASI]
- Cakupan: Dashboard, Pembuatan Surat, Inbox. Pendukung: Daftar Surat, Review, Detail Surat. [KONFIRMASI]
- Frontend: **Next.js** (Nabil). Backend: **Golang** (rekan magang). Fokus tahap ini: frontend dan alur. [KONFIRMASI]
- Deliverable: mockup, ERD, flowchart. Presentasi maks. 10 menit per orang. [KONFIRMASI]
- Kerja: Agile/Scrum di ClickUp (format story `FE-03. Buat Surat | Sebagai <peran>, saya ingin ...`).

## 2. Tech stack dan status repo

Repo `Digital-Office/{frontend,backend}`. Terpasang di `frontend/package.json`:

| Paket | Versi |
|---|---|
| next | 16.4.0 |
| react, react-dom | 19.3.0 |
| typescript | ^5 |
| tailwindcss | ^4 |
| eslint, eslint-config-next | ^9, 16.4.0 |
| babel-plugin-react-compiler | 1.0.0 |

- npm, App Router, folder `src/`, Turbopack.
- `next.config.ts`: `cacheComponents`, `partialPrefetching`, `reactCompiler` aktif. Perhatikan aturan caching Next 16 saat fetching.
- **Tailwind v4 lewat `@tailwindcss/turbopack`** (rule `*.css` di `turbopack.rules`). Ini setup yang benar dari create-next-app terbaru, **tidak ada `postcss.config.mjs` dan itu normal**. Jangan "memperbaiki" ke `@tailwindcss/postcss`.
- Tanpa Word editor di tahap ini, jadi **tidak ada** TipTap/Lexical/OnlyOffice yang dipasang.

**Library yang dipasang untuk Buat Surat [USULAN]:**

| Kebutuhan | Library |
|---|---|
| Komponen UI | shadcn/ui + `lucide-react` |
| Form dan validasi | `react-hook-form` + `zod` + `@hookform/resolvers` |
| Data fetching/mutation | TanStack Query |
| Tanggal (WIB) | date-fns |
| Mock API | MSW (Mock Service Worker) |

TanStack Table dipasang nanti saat mengerjakan Inbox/Daftar Surat.

## 3. Fokus saat ini: halaman Buat Surat tanpa editor Word

**Dikerjakan:** Section 1 Detail Surat, Section 2 Lampiran, action bar, simpan otomatis, dialog konfirmasi, validasi, data mock.

**Tidak dikerjakan sekarang:**
- Antarmuka ala Word (tab File/Home/Insert, toolbar, penggaris, halaman A4, kolaborator). Section 3 diganti **kartu placeholder** (lihat bagian 7).
- Backend sungguhan. Semua lewat MSW.
- Tombol **Pratinjau Cetak PDF** (butuh dokumen). Tampilkan nonaktif dengan tooltip "Tersedia setelah editor surat siap", atau sembunyikan.

Konsekuensi: validasi dan Submit **tidak** menuntut isi surat. Submit di tahap ini hanya mengirim metadata ke mock.

## 4. Domain dan alur [KONFIRMASI]

Surat dibuat **drafter**, dicek **reviewer** (berurutan), disetujui **approver** (berurutan), lalu dikirim ke **pimpinan unit** penerima. Pimpinan bisa **mendisposisi**, staf bisa **mengalihkan**.

**Persiapan data (admin):** struktur organisasi (perusahaan, unit, jabatan, pegawai); isi dropdown (jenis surat, sifat, kecepatan, label status); template `.docx` berisi kop dan variabel seperti `###NoSurat###`.

**Buat surat:** drafter mengisi jenis surat, template, sifat, kecepatan, perihal, ringkasan, keterangan khusus, unit penerima (Kepada/TO dan Tembusan/CC), daftar reviewer dan approver beserta urutan. Tiap Simpan = **versi baru** (versi lama tidak ditimpa). Status `DRAFT`.

**Review (berurutan):** kirim masuk ke reviewer tahap 1 (`IN_REVIEW`). Reviewer boleh mengedit dan menyimpan. Setuju lanjut ke tahap berikutnya. Kembalikan + catatan ke drafter (`RETURNED_TO_DRAFTER`), drafter kirim ulang **hanya ke reviewer yang mengembalikan**. Reviewer terakhir setuju masuk ke approver (`IN_APPROVAL`).

**Approval (berurutan):** approver tidak bisa mengedit, hanya setuju atau kembalikan. Kembalikan ke reviewer terakhir (`RETURNED_TO_REVIEWER`); setelah reviewer setuju lagi, surat kembali **hanya ke approver yang mengembalikan**. Approver terakhir = penanda tangan "atas nama".

**Surat terbit (satu transaksi DB oleh approver terakhir):** nomor otomatis (contoh `015/TI/PSP/X/2026`), tanggal surat = waktu disetujui, backend membuat file FINAL dengan QR + timestamp, sistem mencari pimpinan tiap unit penerima, surat masuk inbox mereka. Status `SENT`.

**Inbox dan disposisi:** `UNREAD` ke `READ`. Disposisi ke orang atau satu unit, berjenjang. Staf bisa mengalihkan (`REDIRECTED`). Inbox read-only.

**Batal:** surat tidak dihapus, status `CANCELLED`, alasan wajib, watermark "Dibatalkan", nomor terpakai tidak dipakai ulang.

**Tracing:** simpan, kirim, baca, setuju, kembalikan, batal, ganti peserta semuanya tercatat.

## 5. Status: label UI mengikuti gambar, enum mengikuti alur

Enum (dari ringkasan DB, **jangan diubah**):
```ts
export type LetterStatus =
  | "DRAFT" | "IN_REVIEW" | "RETURNED_TO_DRAFTER"
  | "IN_APPROVAL" | "RETURNED_TO_REVIEWER" | "SENT" | "CANCELLED";

export type InboxStatus = "UNREAD" | "READ" | "REDIRECTED";
```

Label UI (satu-satunya tempat label didefinisikan, di `lib/status.ts`):

| Enum | Label UI [GAMBAR] | Tab Daftar Surat | Badge |
|---|---|---|---|
| `DRAFT` | Draft | Draft | Info |
| `IN_REVIEW` | Menunggu Review | Menunggu Review | Menunggu |
| `RETURNED_TO_DRAFTER` | Dikembalikan | Dikembalikan | Bahaya |
| `RETURNED_TO_REVIEWER` | Dikembalikan [USULAN] | Dikembalikan | Bahaya |
| `IN_APPROVAL` | Menunggu Approval | Menunggu Approval | Menunggu |
| `SENT` | Disetujui | Disetujui | Sukses |
| `CANCELLED` | Dibatalkan | Dibatalkan | Bahaya |

Catatan:
- Gambar memakai **"Disetujui"**. Dalam alur, persetujuan akhir dan pengiriman adalah satu transaksi, jadi `SENT` = "Disetujui". Detail memakai chip "Disetujui" dan panel "Status: Disetujui Penuh".
- Status di halaman Review mengikuti pola gambar: `Menunggu Reviewer 2 (<nama>)`. Nama tahap dan orang berasal dari `steps`, bukan dari enum.
- `RETURNED_TO_REVIEWER` belum ada di gambar. Usulan: tampil "Dikembalikan", kolom Tahap Saat Ini berisi "Reviewer N (dikembalikan oleh Approver M)". Konfirmasi ke backend.
- Tab Inbox [GAMBAR]: **Unread** (`UNREAD`), **Read** (`READ`), **Disposed**, **Redirected** (`REDIRECTED`). "Disposed" diturunkan dari data disposisi (entitas terpisah), bukan nilai `InboxStatus`. Angka di gambar hanya mock.

## 6. Aturan UI penting

- **Sifat** (kerahasiaan) dan **Kecepatan Tanggapan** adalah dua field terpisah.
  - Sifat: `Biasa`, `Terbatas`, `Rahasia`. "Penting" bukan nilai Sifat.
  - Kecepatan: `Segera`, `Sangat Segera` (form menampilkan "Segera (< 24 Jam Kerja)").
- Nomor dan tanggal surat sebelum disetujui = placeholder "dibuat otomatis".
- **Tujuan dan Tembusan adalah unit**, bukan orang. Pimpinan unit dicari sistem.
- Review: reviewer melihat editor aktif, approver read-only.
- **Kembalikan** membuka modal, Catatan Revisi wajib. **Batalkan** membuka dialog, alasan wajib, dan menjelaskan surat tidak kembali ke drafter.
- Status selalu berlabel teks, tidak hanya warna.

## 7. Halaman Buat Surat (FE-03) [GAMBAR + KONFIRMASI]

Route: `src/app/(app)/surat/baru/page.tsx`.

**Header halaman:** breadcrumb `Beranda > Penciptaan Surat > Buat Surat Baru`, judul "Buat Surat Baru", indikator "Tersimpan Otomatis (jam WIB)" dengan titik hijau, tombol **Muat Draf Terakhir** (kanan atas). Banner info: "Mohon untuk tidak copy-paste langsung dari Microsoft Word berformat asing untuk menjaga kerapian font dan format baku penomoran surat korporat." [SISTEM LAMA]. Karena belum ada editor, banner ini boleh disembunyikan sampai Section 3 aktif.

### Section 1: Detail Surat

| Field | Kontrol | Wajib | Aturan |
|---|---|---|---|
| Jenis Surat | select, default "Surat Antar Unit Kerja" | ya | dari master mock |
| Template Surat | select | tidak | dari master mock |
| Klasifikasi | combobox dengan ikon cari | ya | cari by kode/nama |
| Kecepatan Tanggapan | select | ya | Segera / Sangat Segera |
| Sifat Kerahasiaan | radio Biasa/Terbatas/Rahasia, default Biasa | ya | teks bantu di kanan, mis. "Sifat 'Biasa' dapat diakses oleh seluruh staf pengadministrasi unit terkait" (isi per pilihan, tentukan fiktif) |
| Judul / Perihal | input | ya | maks 250, penghitung "n / 250", centang hijau **hanya jika valid dan tidak kosong** |
| Reviewer | daftar bertahap | min. 1 | chip "Tahap N" + nama + (unit) + tombol hapus; "+ Tambah Reviewer" |
| Approver | daftar bertahap | min. 1 | pola sama dengan Reviewer |
| Tujuan Surat | pemilih **unit**, multi, chip | min. 1 | tombol "Pilih Penerima", "+ Tambah Instansi / Unit"; **tidak boleh duplikat** |
| Tembusan (CC) | pemilih unit, multi | tidak | tombol "Pilih Tembusan"; unit tidak boleh ada di Tujuan |
| Keterangan Khusus | input teks | tidak | |
| Isi Ringkas Arsip | textarea | tidak | |

Perilaku `StepPicker` (Reviewer/Approver): cari pegawai, tambah, hapus, ubah urutan (nomor tahap otomatis menyesuaikan). Orang yang sama di dua tahap: tampilkan peringatan non-blokir sampai pembimbing menjawab (bagian 12).

### Section 2: Lampiran
- Panel "Unggah Berkas Baru" (format PDF, DOCX, XLSX, PPTX) + tombol "Pilih File dari Komputer", dan dropzone seret-lepas (maks 25 MB per berkas).
- Kolom saring nama berkas "Saring nama berkas lampiran...", teks "Menampilkan x dari y berkas".
- Tabel: `#`, Aksi (lihat, hapus), Nama Berkas & Format (ikon tipe + keterangan), Ukuran, Waktu Unggah.
- State: kosong ("Belum ada lampiran"), mengunggah (progress), error (tipe/ukuran tidak valid, tombol coba lagi).

### Section 3: Isi Surat (placeholder, tanpa editor Word)
Kartu dengan judul "Isi Surat & Tata Naskah Digital" dan teks netral, mis. "Editor naskah menyusul. Isi surat disusun dari template yang dipilih." Tidak ada toolbar, penggaris, atau halaman A4. Komponen dipisah (`EditorPlaceholder`) agar mudah diganti editor nanti.

### Action bar bawah (sticky)
**Simpan Draft** (konfirmasi dulu), **Pratinjau Cetak PDF** (nonaktif, lihat bagian 3), **Submit** (konfirmasi dulu, aktif jika form valid), **Batal / Kembali ke Daftar Surat** (peringatan jika ada perubahan belum tersimpan).

### Dialog
- Simpan Draft: "Simpan draft surat ini?" (Batal / Simpan). Tiap simpan membuat versi baru di mock.
- Submit: "Kirim surat ke Reviewer Tahap 1?" (Batal / Kirim). Status menjadi `IN_REVIEW`.
- Keluar dengan perubahan belum tersimpan: "Perubahan belum tersimpan. Tinggalkan halaman?"

### Simpan otomatis dan Muat Draf Terakhir [USULAN]
Autosave debounce, indikator menampilkan jam terakhir tersimpan. "Muat Draf Terakhir" mengisi form dari draf terakhir milik user (mock).

## 8. Halaman lain (ringkas, dikerjakan setelah Buat Surat)

**Shell [SISTEM LAMA + GAMBAR]:** header navy (logo "Digital Office / Sistem Persuratan", Home, Supporting Documents, notifikasi, `[NIK] Nama`, Logout). Sidebar: Dashboard; Profil (Edit Profil); Penciptaan Surat (Buat Surat Baru, Buat SP/ASP, Daftar Surat, Menunggu Review, Menunggu Approval); Master Data; Manajemen Arsip Aktif; Manajemen Arsip Inaktif; Permission Document; Inbox (badge); Notifikasi (badge); Rekap; Pencarian. Jangan tambah menu karangan.

**Inbox:** tab Unread/Read/Disposed/Redirected; kolom Aksi (Detail), Surat, Judul Surat, Keterangan (Disposisi dari/CC dari/Surat dari), Pengirim (unit), Approver (orang), Tanggal (terbaru di atas), Sifat, Kecepatan; baris filter di bawah header; pagination.

**Daftar Surat:** tab sesuai tabel bagian 5; kolom Nomor Surat, Judul/Perihal, Pengirim, Tujuan, Jenis & Sifat, Tanggal Dibuat, Status, Tahap Saat Ini, Tindakan. Kartu ringkasan boleh ada tetapi tanpa klaim "BSrE/TTE" karangan.

**Review:** chip Sifat/Kecepatan/Status, judul, metadata, stepper alur dan riwayat (termasuk jalur kembali), lampiran, area dokumen. Action bar: "Anda meninjau sebagai: Reviewer N", **Batalkan Naskah**, **Kembalikan (Revisi)**, **Setujui Naskah**. Approver: Setujui dan Kembalikan (siapa boleh Batalkan: lihat bagian 12).

**Detail Disetujui:** preview final dengan QR dan timestamp, metadata, aksi Cetak Salinan, Unduh PDF Resmi, **Disposisi**, **Alihkan**. Satu set tombol saja.

**Dashboard [USULAN]:** kartu ringkasan, "Perlu Tindakan Anda", arus surat per minggu, aktivitas terbaru. Konfirmasi isinya.

## 9. Design system [USULAN]

Navy dari sistem lama. Opsi aksen: navy + teal atau navy + amber (belum diputuskan). Jika ada brand guideline resmi, itu mengalahkan palet ini.

| Token | Hex | Pemakaian |
|---|---|---|
| `primary-dark` | `#1B2A4A` | sidebar, header, judul |
| `primary` | `#1E2A78` | tombol utama, link |
| `accent` | `#D97706` | highlight secukupnya |
| `bg` | `#F7F7F5` | latar halaman |
| `surface` | `#FFFFFF` | kartu, tabel |
| `panel` | `#EEF1F5` | panel sekunder |
| `text` | `#1F2937` | teks isi |
| `text-muted` | `#6B7280` | teks sekunder |

Warna status (selalu dengan label teks):

| Makna | Teks | Latar |
|---|---|---|
| Sukses/Disetujui | `#166534` | `#DCFCE7` |
| Menunggu/Segera | `#92400E` | `#FEF3C7` |
| Bahaya/Sangat Segera/Batal/Dikembalikan | `#991B1B` | `#FEE2E2` |
| Info/Draft | `#1E40AF` | `#DBEAFE` |
| Netral/Biasa | `#475569` | `#F1F5F9` |

Kontras belum diuji, cek dulu. Aturan: definisikan token di `@theme` (Tailwind v4) di `globals.css`, **tanpa hardcode hex di komponen**; font Inter atau Plus Jakarta Sans; teks isi 14-16px; hindari gradient ungu/biru, glassmorphism, neon, kartu terlalu bulat, ilustrasi dekoratif, emoji; status sebagai badge lembut berlabel.

## 10. Aturan konten dan data mock

- Teks UI **Bahasa Indonesia**; identifier kode **bahasa Inggris**.
- **Jangan masukkan data pegawai asli** (nama, NIK, nomor surat, URL internal) dari screenshot ke kode, mock, atau tool AI. Pakai nama dan unit fiktif.
- Satu tahun dan satu rangkaian tanggal konsisten di semua mock.
- Drafter, reviewer 1, reviewer 2, approver, dan user login adalah **orang berbeda** di mock.
- Satu nama organisasi fiktif di seluruh mock.
- Jangan pakai klaim tanpa sumber (ISO 15489/27001, BSrE BSSN, FortiSandbox, SHA-256, MD5, "Sah Secara Hukum"). Yang ada di arahan: QR dengan timestamp persetujuan.
- Angka antar halaman nyambung (sidebar, dashboard, tab).

## 11. Kontrak data [USULAN, sepakati dengan backend Golang]

```ts
export type Sifat = "BIASA" | "TERBATAS" | "RAHASIA";
export type Kecepatan = "SEGERA" | "SANGAT_SEGERA";

export interface WorkflowStep {
  id: string;
  role: "REVIEWER" | "APPROVER";
  order: number;                       // tahap 1, 2, ...
  employeeId: string;
  status: "PENDING" | "ACTIVE" | "DONE";
  actions: StepAction[];               // riwayat per tahap
}

export interface StepAction {
  type: "APPROVED" | "RETURNED";
  note?: string;                       // wajib bila RETURNED
  at: string;                          // ISO
}

export interface LetterAttachment {
  id: string; fileName: string; mimeType: string;
  sizeBytes: number; uploadedAt: string;
}

export interface Letter {
  id: string;
  companyId: string;                   // isolasi data per perusahaan
  status: LetterStatus;
  drafterId: string;
  number: string | null;               // null sampai SENT
  letterDate: string | null;           // = waktu approval akhir
  title: string;                       // maks 250
  summary?: string;
  specialNote?: string;
  letterTypeId: string;
  templateId?: string;
  classificationId: string;
  sifat: Sifat;
  kecepatan: Kecepatan;
  toUnitIds: string[];
  ccUnitIds: string[];
  steps: WorkflowStep[];
  returnedByStepId?: string;           // untuk aturan "kirim ulang hanya ke yang mengembalikan"
  attachments: LetterAttachment[];
  currentVersion: number;              // tiap simpan = versi baru
  cancelReason?: string;               // wajib bila CANCELLED
}
```

Mock dengan MSW dulu. Bila backend punya OpenAPI, generate tipe TS otomatis. Setiap tabel backend membawa `company_id`, frontend tidak menampilkan data lintas perusahaan.

## 12. Pertanyaan terbuka (tanyakan ke pembimbing/backend)

1. Siapa boleh membatalkan surat dan di tahap mana? (Arahan awal: reviewer/approver bisa batalkan. Ringkasan DB: approver hanya setuju atau kembalikan.)
2. Tab Inbox "Disposed": turunan dari disposisi atau status sendiri? (Foto lama: On Progress/Closed/Cancelled.)
3. Boleh orang yang sama jadi reviewer sekaligus approver, atau drafter sekaligus reviewer?
4. Tampilan `RETURNED_TO_REVIEWER` di Daftar Surat drafter.
5. Isi Dashboard dan brand guideline resmi.
6. Penanganan .docx dan server editor (ditunda bersama Word editor).
7. Nama tombol: "Submit" vs "Kirim ke Reviewer", "Setujui Naskah".

## 13. Editor surat (DITUNDA)

Tidak dikerjakan sekarang. Saat dimulai: dokumen disimpan sebagai .docx dengan variabel template, tiap simpan = versi baru. Pilihan: TipTap/Lexical (fleksibel, kolaborasi via Yjs, tapi tidak membaca .docx native) atau editor Office yang di-embed seperti OnlyOffice (mirip Word, butuh server). Mulai dengan spike 1-2 hari dan keputusan bersama backend dan pembimbing.

## 14. Task FE-03 (ClickUp)

Story: (1) lihat form pembuatan surat; (2) pilih reviewer dan approver; (3) unggah lampiran; (4) kotak konfirmasi sebelum simpan. Tambahan usulan: pemilih Tujuan/Tembusan; Simpan draft dan Muat Draf Terakhir. Story "Isi surat dengan editor" ditunda.

Urutan kerja: dependency, design token, types + zod + mock, shell + `StatusBadge`, form statis, validasi, StepPicker, UnitPicker, lampiran, dialog, autosave.

Acceptance criteria:
- **Story 1:** field sesuai bagian 7, wajib divalidasi (zod), penghitung judul, pesan error Bahasa Indonesia.
- **Story 2:** picker bisa dicari, label tahap, tambah/hapus/ubah urutan, min. 1 reviewer dan 1 approver.
- **Story 3:** validasi tipe dan ukuran, tabel dengan lihat/hapus/filter, state kosong/loading/error.
- **Story 4:** dialog konfirmasi untuk Simpan Draft dan Submit, peringatan perubahan belum tersimpan saat Batal/Kembali.

Konvensi [USULAN]: satu branch per story (`feature/FE-03-buat-surat`), link branch/PR di task, status digeser harian.

## 15. Prompt siap tempel: halaman Buat Surat

```
Baca PROJECT_CONTEXT.md di repo ini sebagai sumber kebenaran, lalu bangun halaman Buat Surat (FE-03) di frontend/ (Next.js 16 App Router, React 19, Tailwind v4, TypeScript, npm).

Ruang lingkup:
- Route src/app/(app)/surat/baru/page.tsx, Section 1 Detail Surat, Section 2 Lampiran, action bar sticky, dialog konfirmasi, autosave indicator + Muat Draf Terakhir.
- JANGAN buat editor ala Word. Section 3 hanya kartu placeholder (komponen EditorPlaceholder). Pratinjau Cetak PDF dinonaktifkan.
- Submit tidak menuntut isi surat; kirim metadata ke mock.

Aturan wajib:
- Field, label, dan validasi persis seperti bagian 7. Status mengikuti bagian 5 (label dari lib/status.ts, enum LetterStatus tidak diubah).
- Sifat (Biasa/Terbatas/Rahasia) dan Kecepatan (Segera/Sangat Segera) dua field terpisah. "Penting" bukan Sifat.
- Tujuan dan Tembusan = unit, bukan orang, tanpa duplikat. Reviewer/Approver bertahap, min. 1 masing-masing.
- Form: react-hook-form + zod (@hookform/resolvers). Data: TanStack Query + MSW. Tanggal: date-fns (WIB).
- Komponen UI shadcn/ui + lucide-react. Token warna di @theme globals.css, tanpa hardcode hex. Badge status berlabel teks.
- Teks UI Bahasa Indonesia, identifier bahasa Inggris. Jangan pakai data pegawai asli; semua mock fiktif dan konsisten (drafter, reviewer, approver, user login orang berbeda).
- Jangan ubah next.config.ts atau setup Tailwind (@tailwindcss/turbopack sudah benar).
- Jangan tambah menu sidebar atau klaim karangan (BSrE, ISO, FortiSandbox, hash).

Kerjakan bertahap dan berhenti di tiap tahap untuk saya cek: (1) install dependency + token + types/zod/mock, (2) shell + StatusBadge, (3) form statis, (4) validasi, (5) StepPicker dan UnitPicker, (6) lampiran, (7) dialog + autosave.
Mulai dari tahap 1 saja dan jelaskan alasan setiap keputusan teknis secara singkat.
```

## 16. Masalah mockup yang diketahui (jangan dibawa ke kode)

- Reviewer, approver, drafter, dan user login diisi orang yang sama.
- Nomor dan tanggal surat terisi sebelum disetujui; tanggal campur 2025 dan 2026.
- Badge "Sifat: Penting" salah.
- Chip unit tujuan terduplikasi; judul kosong tetapi ada penghitung dan centang hijau.
- Tabel Inbox dan Daftar Surat kosong atau terpotong; teks terpotong/bertumpuk di Detail; tombol aksi duplikat.
- Dua nama perusahaan berbeda di kop (Review dan Detail).
- Detail karangan tanpa sumber (bagian 10).
- Nama/NIK pada screenshot terlihat asli, jangan dipakai.
