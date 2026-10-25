// ─────────────────────────────────────────────
// lib/templates/letter-templates.ts
// Format & Template Tata Naskah Dinas Resmi (Frontend)
// Siap dimuat ke DevExpress RichEdit DocumentFormat.Html
// ─────────────────────────────────────────────

export interface LetterTemplateContext {
  title?: string;
  sifat?: "BIASA" | "TERBATAS" | "RAHASIA" | string;
  kecepatan?: "BIASA" | "SEGERA" | "SANGAT_SEGERA" | string;
  classificationCode?: string;
  classificationName?: string;
  recipients?: Array<{ name: string; position?: string }>;
  approver?: { name: string; position?: string; employeeNumber?: string };
  drafter?: { name: string; position?: string };
  date?: string;
  letterNumberPlaceholder?: string;
  attachmentsCount?: number;
}

export interface LetterTemplateDefinition {
  id: string;
  name: string;
  description: string;
  letterTypeKeywords: string[];
  generateHtml: (ctx: LetterTemplateContext) => string;
}

function formatDateIndo(dateStr?: string): string {
  if (!dateStr) {
    const now = new Date();
    return new Intl.DateTimeFormat("id-ID", {
      day: "numeric",
      month: "long",
      year: "numeric",
    }).format(now);
  }
  return dateStr;
}

// ─── 1. Template Surat Dinas Resmi Standar ─────────────────────────
function generateSuratDinasHtml(ctx: LetterTemplateContext): string {
  const tanggal = formatDateIndo(ctx.date);
  const perihal = ctx.title?.trim() || "[Perihal / Hal Surat]";
  const sifat = ctx.sifat || "BIASA";
  const lampiran = ctx.attachmentsCount && ctx.attachmentsCount > 0
    ? `${ctx.attachmentsCount} Berkas`
    : "—";
  const penerima = ctx.recipients && ctx.recipients.length > 0
    ? ctx.recipients.map((r) => r.name).join("<br/>")
    : "Yth. Pimpinan Unit Kerja Terkait<br/>di Tempat";
  const approverName = ctx.approver?.name || "[Nama Pejabat Penandatangan]";
  const approverJabatan = ctx.approver?.position || "[Jabatan Pejabat Penandatangan]";
  const approverNip = ctx.approver?.employeeNumber ? `NIP. ${ctx.approver.employeeNumber}` : "NIP. ..............................";

  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <style>
    body {
      font-family: 'Times New Roman', Times, serif;
      font-size: 12pt;
      line-height: 1.35;
      color: #000000;
      margin: 0;
      padding: 10px;
    }
    .kop-table {
      width: 100%;
      border-collapse: collapse;
      margin-bottom: 2px;
    }
    .kop-header {
      text-align: center;
    }
    .kop-instansi {
      font-size: 14pt;
      font-weight: bold;
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }
    .kop-subinstansi {
      font-size: 13pt;
      font-weight: bold;
      text-transform: uppercase;
    }
    .kop-alamat {
      font-size: 9.5pt;
      margin-top: 3px;
    }
    .kop-garis-atas {
      border-top: 2px solid #000;
      margin-top: 6px;
      margin-bottom: 1px;
    }
    .kop-garis-bawah {
      border-top: 1px solid #000;
      margin-bottom: 18px;
    }
    .meta-table {
      width: 100%;
      border-collapse: collapse;
      margin-bottom: 18px;
      font-size: 11pt;
    }
    .meta-table td {
      vertical-align: top;
      padding: 2px 0;
    }
    .tujuan-section {
      margin-bottom: 18px;
      font-size: 11pt;
    }
    .isi-paragraf {
      text-align: justify;
      text-indent: 36pt;
      margin-bottom: 12px;
      font-size: 11pt;
    }
    .ttd-table {
      width: 100%;
      border-collapse: collapse;
      margin-top: 24px;
      font-size: 11pt;
    }
    .tembusan {
      margin-top: 24px;
      font-size: 9.5pt;
    }
  </style>
</head>
<body>
  <!-- KOP SURAT -->
  <table class="kop-table">
    <tr>
      <td class="kop-header">
        <div class="kop-instansi">PT PUPUK INDONESIA (PERSERO)</div>
        <div class="kop-subinstansi">DIREKTORAT TEKNOLOGI INFORMASI</div>
        <div class="kop-alamat">Jl. Wijaya II No. 45, Kebayoran Baru, Jakarta Selatan 12160 | Telp. (021) 7279-9999 | www.pupuk-indonesia.com</div>
      </td>
    </tr>
  </table>
  <div class="kop-garis-atas"></div>
  <div class="kop-garis-bawah"></div>

  <!-- TANGGAL & METADATA SURAT -->
  <table class="meta-table">
    <tr>
      <td style="width: 14%;">Nomor</td>
      <td style="width: 2%;">:</td>
      <td style="width: 44%;">###NoSurat###</td>
      <td style="width: 40%; text-align: right;">Jakarta, ${tanggal}</td>
    </tr>
    <tr>
      <td>Sifat</td>
      <td>:</td>
      <td>${sifat}</td>
      <td></td>
    </tr>
    <tr>
      <td>Lampiran</td>
      <td>:</td>
      <td>${lampiran}</td>
      <td></td>
    </tr>
    <tr>
      <td>Perihal</td>
      <td>:</td>
      <td><strong>${perihal}</strong></td>
      <td></td>
    </tr>
  </table>

  <!-- KEPADA YTH -->
  <div class="tujuan-section">
    Kepada Yth.<br/>
    <strong>${penerima}</strong><br/>
    di Tempat
  </div>

  <!-- SALAM PEMBUKA -->
  <div style="margin-bottom: 10px; font-size: 11pt;">Dengan hormat,</div>

  <!-- ISI SURAT -->
  <div class="isi-paragraf">
    Sehubungan dengan pelaksanaan program strategis dan tata kelola operasional di lingkungan PT Pupuk Indonesia (Persero), dengan ini kami sampaikan pokok-pokok informasi dan koordinasi sebagai berikut:
  </div>

  <div class="isi-paragraf">
    1. [Uraikan latar belakang atau konteks surat dinas secara jelas dan ringkas di sini].
  </div>

  <div class="isi-paragraf">
    2. [Uraikan isi pokok, instruksi, permohonan, atau pemberitahuan yang ingin disampaikan kepada pihak penerima].
  </div>

  <div class="isi-paragraf">
    Demikian surat ini kami sampaikan untuk menjadi perhatian dan dapat ditindaklanjuti sebagaimana mestinya. Atas perhatian dan kerja sama yang baik, kami ucapkan terima kasih.
  </div>

  <!-- TANDA TANGAN / APPROVER -->
  <table class="ttd-table">
    <tr>
      <td style="width: 50%;"></td>
      <td style="width: 50%; text-align: center;">
        <div>${approverJabatan}</div>
        <div style="height: 65px; display: flex; align-items: center; justify-content: center; font-style: italic; color: #666;">
          (Ruang Tanda Tangan Digital / QR Code Verifikasi)
        </div>
        <div><strong>${approverName}</strong></div>
        <div>${approverNip}</div>
      </td>
    </tr>
  </table>

  <!-- TEMBUSAN -->
  <div class="tembusan">
    <strong>Tembusan:</strong><br/>
    1. Direksi PT Pupuk Indonesia (Persero)<br/>
    2. Arsip Digital Office
  </div>
</body>
</html>`;
}

// ─── 2. Template Nota Dinas (Internal) ─────────────────────────────
function generateNotaDinasHtml(ctx: LetterTemplateContext): string {
  const tanggal = formatDateIndo(ctx.date);
  const perihal = ctx.title?.trim() || "[Perihal / Hal Nota Dinas]";
  const sifat = ctx.sifat || "BIASA";
  const penerima = ctx.recipients && ctx.recipients.length > 0
    ? ctx.recipients.map((r) => r.name).join(", ")
    : "Pimpinan Unit Terkait";
  const pengirim = ctx.drafter?.position || "Staf Pengembang Sistem / Unit TI";
  const approverName = ctx.approver?.name || "[Nama Pejabat]";
  const approverJabatan = ctx.approver?.position || "[Jabatan Pejabat]";

  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <style>
    body {
      font-family: Arial, Helvetica, sans-serif;
      font-size: 11pt;
      line-height: 1.4;
      color: #000;
      margin: 0;
      padding: 10px;
    }
    .nd-title {
      text-align: center;
      font-size: 14pt;
      font-weight: bold;
      text-decoration: underline;
      margin-bottom: 2px;
      text-transform: uppercase;
    }
    .nd-number {
      text-align: center;
      font-size: 10pt;
      margin-bottom: 20px;
    }
    .nd-table {
      width: 100%;
      border-collapse: collapse;
      margin-bottom: 14px;
      font-size: 10.5pt;
    }
    .nd-table td {
      padding: 4px 0;
      vertical-align: top;
    }
    .divider {
      border-top: 2px solid #000;
      margin-bottom: 16px;
    }
    .nd-isi {
      text-align: justify;
      margin-bottom: 12px;
    }
    .ttd-box {
      margin-top: 28px;
      float: right;
      width: 260px;
      text-align: center;
    }
  </style>
</head>
<body>
  <div class="nd-title">NOTA DINAS</div>
  <div class="nd-number">Nomor: ###NoSurat###</div>

  <table class="nd-table">
    <tr>
      <td style="width: 18%;"><strong>Yth.</strong></td>
      <td style="width: 2%;">:</td>
      <td style="width: 80%;">${penerima}</td>
    </tr>
    <tr>
      <td><strong>Dari</strong></td>
      <td>:</td>
      <td>${pengirim}</td>
    </tr>
    <tr>
      <td><strong>Tanggal</strong></td>
      <td>:</td>
      <td>${tanggal}</td>
    </tr>
    <tr>
      <td><strong>Sifat</strong></td>
      <td>:</td>
      <td>${sifat}</td>
    </tr>
    <tr>
      <td><strong>Perihal</strong></td>
      <td>:</td>
      <td><strong>${perihal}</strong></td>
    </tr>
  </table>

  <div class="divider"></div>

  <div class="nd-isi">
    <strong>1. Pengantar / Latar Belakang</strong><br/>
    Menindaklanjuti kebutuhan operasional dan peningkatan efisiensi proses bisnis, bersama ini disampaikan pokok evaluasi dan pertimbangan teknis terkait ${perihal}.
  </div>

  <div class="nd-isi">
    <strong>2. Pokok Pembahasan & Rekomendasi</strong><br/>
    [Tuliskan analisa masalah, opsi tindakan, atau kebutuhan sumber daya di bagian ini].
  </div>

  <div class="nd-isi">
    <strong>3. Kesimpulan & Permohonan Arahan</strong><br/>
    Demikian Nota Dinas ini kami sampaikan untuk menjadi bahan pertimbangan. Mohon arahan dan persetujuan Bapak/Ibu lebih lanjut.
  </div>

  <table style="width: 100%; margin-top: 30px;">
    <tr>
      <td style="width: 50%;"></td>
      <td style="width: 50%; text-align: center;">
        <div><strong>${approverJabatan}</strong></div>
        <div style="height: 60px; display: flex; align-items: center; justify-content: center; font-style: italic; color: #777;">
          (Tanda Tangan Elektronik)
        </div>
        <div><strong>${approverName}</strong></div>
      </td>
    </tr>
  </table>
</body>
</html>`;
}

// ─── 3. Template Undangan Rapat / Kegiatan ─────────────────────────
function generateUndanganHtml(ctx: LetterTemplateContext): string {
  const tanggal = formatDateIndo(ctx.date);
  const perihal = ctx.title?.trim() || "Undangan Rapat Koordinasi";
  const penerima = ctx.recipients && ctx.recipients.length > 0
    ? ctx.recipients.map((r) => r.name).join("<br/>")
    : "Bapak/Ibu Pimpinan Unit Kerja<br/>di Tempat";
  const approverName = ctx.approver?.name || "[Nama Pejabat]";
  const approverJabatan = ctx.approver?.position || "[Jabatan Pejabat]";

  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <style>
    body {
      font-family: 'Times New Roman', Times, serif;
      font-size: 12pt;
      line-height: 1.35;
      margin: 0;
      padding: 10px;
    }
    .kop-instansi {
      text-align: center;
      font-weight: bold;
      font-size: 13pt;
      text-transform: uppercase;
    }
    .kop-garis {
      border-top: 2px solid #000;
      margin: 8px 0 16px 0;
    }
    .agenda-table {
      margin: 12px 0 16px 20px;
      font-size: 11pt;
    }
    .agenda-table td {
      padding: 3px 6px;
      vertical-align: top;
    }
  </style>
</head>
<body>
  <div class="kop-instansi">PT PUPUK INDONESIA (PERSERO)<br/><span style="font-size: 11pt; font-weight: normal;">DIREKTORAT TEKNOLOGI INFORMASI</span></div>
  <div class="kop-garis"></div>

  <table style="width: 100%; font-size: 11pt; margin-bottom: 16px;">
    <tr>
      <td style="width: 15%;">Nomor</td>
      <td style="width: 2%;">:</td>
      <td style="width: 43%;">###NoSurat###</td>
      <td style="width: 40%; text-align: right;">Jakarta, ${tanggal}</td>
    </tr>
    <tr>
      <td>Sifat</td>
      <td>:</td>
      <td>Penting</td>
      <td></td>
    </tr>
    <tr>
      <td>Lampiran</td>
      <td>:</td>
      <td>1 (satu) Berkas Susunan Acara</td>
      <td></td>
    </tr>
    <tr>
      <td>Perihal</td>
      <td>:</td>
      <td><strong>${perihal}</strong></td>
      <td></td>
    </tr>
  </table>

  <div style="font-size: 11pt; margin-bottom: 14px;">
    Kepada Yth.<br/>
    <strong>${penerima}</strong><br/>
    di Tempat
  </div>

  <p style="font-size: 11pt;">Dengan hormat,</p>
  <p style="text-align: justify; text-indent: 30pt; font-size: 11pt;">
    Dalam rangka koordinasi dan penyelarasan target operasional, kami mengharapkan kehadiran Bapak/Ibu dalam rapat koordinasi yang akan dilaksanakan pada:
  </p>

  <table class="agenda-table">
    <tr>
      <td style="width: 120px;"><strong>Hari / Tanggal</strong></td>
      <td>:</td>
      <td>Senin, ${tanggal}</td>
    </tr>
    <tr>
      <td><strong>Waktu</strong></td>
      <td>:</td>
      <td>09.00 WIB s.d. selesai</td>
    </tr>
    <tr>
      <td><strong>Tempat / Media</strong></td>
      <td>:</td>
      <td>Ruang Rapat TI Lantai 4 / Hybrid Zoom Meeting</td>
    </tr>
    <tr>
      <td><strong>Agenda</strong></td>
      <td>:</td>
      <td>${perihal}</td>
    </tr>
  </table>

  <p style="text-align: justify; text-indent: 30pt; font-size: 11pt;">
    Mengingat pentingnya agenda tersebut, dimohon kehadiran Bapak/Ibu tepat pada waktunya. Demikian undangan ini kami sampaikan, atas kehadiran dan kerja samanya diucapkan terima kasih.
  </p>

  <table style="width: 100%; margin-top: 24px; font-size: 11pt;">
    <tr>
      <td style="width: 50%;"></td>
      <td style="width: 50%; text-align: center;">
        <div>${approverJabatan}</div>
        <div style="height: 60px; display: flex; align-items: center; justify-content: center; font-style: italic; color: #888;">
          (Ruang Tanda Tangan)
        </div>
        <div><strong>${approverName}</strong></div>
      </td>
    </tr>
  </table>
</body>
</html>`;
}

// ─── 4. Template Surat Tugas (SPT) ─────────────────────────────────
function generateSuratTugasHtml(ctx: LetterTemplateContext): string {
  const tanggal = formatDateIndo(ctx.date);
  const perihal = ctx.title?.trim() || "Penugasan Implementasi Sistem";
  const approverName = ctx.approver?.name || "[Nama Pejabat]";
  const approverJabatan = ctx.approver?.position || "[Jabatan Pejabat]";

  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <style>
    body {
      font-family: 'Times New Roman', Times, serif;
      font-size: 12pt;
      line-height: 1.35;
      margin: 0;
      padding: 10px;
    }
    .title {
      text-align: center;
      font-weight: bold;
      font-size: 14pt;
      text-decoration: underline;
      margin-bottom: 2px;
    }
    .nomor {
      text-align: center;
      font-size: 10.5pt;
      margin-bottom: 20px;
    }
    .isi {
      text-align: justify;
      font-size: 11pt;
      margin-bottom: 12px;
    }
    .table-petugas {
      width: 100%;
      border-collapse: collapse;
      margin: 12px 0;
      font-size: 10.5pt;
    }
    .table-petugas th, .table-petugas td {
      border: 1px solid #000;
      padding: 5px 8px;
    }
  </style>
</head>
<body>
  <div style="text-align: center; font-weight: bold; font-size: 12pt; text-transform: uppercase;">PT PUPUK INDONESIA (PERSERO)</div>
  <div style="border-top: 2px solid #000; margin: 6px 0 18px 0;"></div>

  <div class="title">SURAT PERINTAH TUGAS</div>
  <div class="nomor">Nomor: ###NoSurat###</div>

  <div class="isi">
    Yang bertanda tangan di bawah ini, <strong>${approverJabatan}</strong> PT Pupuk Indonesia (Persero), dengan ini menugaskan kepada pegawai berikut:
  </div>

  <table class="table-petugas">
    <thead>
      <tr style="background-color: #f2f2f2;">
        <th style="width: 6%;">No</th>
        <th style="width: 34%;">Nama Pegawai</th>
        <th style="width: 25%;">NIK / NIP</th>
        <th style="width: 35%;">Jabatan / Unit</th>
      </tr>
    </thead>
    <tbody>
      <tr>
        <td style="text-align: center;">1</td>
        <td><strong>Andi Kurniawan</strong></td>
        <td>TI-2024-001</td>
        <td>Staf Pengembang Sistem</td>
      </tr>
      <tr>
        <td style="text-align: center;">2</td>
        <td><strong>Budi Santoso</strong></td>
        <td>TI-2022-015</td>
        <td>Koordinator Pengembangan Sistem</td>
      </tr>
    </tbody>
  </table>

  <div class="isi">
    <strong>Untuk:</strong> Melaksanakan penugasan ${perihal} sesuai dengan ketentuan dan prosedur standar operasional perusahaan.
  </div>

  <div class="isi">
    Demikian Surat Perintah Tugas ini dibuat untuk dapat dilaksanakan dengan penuh tanggung jawab dan menyampaikan laporan hasil pelaksanaan setelah kegiatan selesai.
  </div>

  <table style="width: 100%; margin-top: 30px; font-size: 11pt;">
    <tr>
      <td style="width: 50%;"></td>
      <td style="width: 50%; text-align: center;">
        <div>Ditetapkan di: Jakarta<br/>Pada tanggal: ${tanggal}</div>
        <div style="margin-top: 10px;"><strong>${approverJabatan}</strong></div>
        <div style="height: 55px; display: flex; align-items: center; justify-content: center; font-style: italic; color: #888;">
          (Tanda Tangan)
        </div>
        <div><strong>${approverName}</strong></div>
      </td>
    </tr>
  </table>
</body>
</html>`;
}

// ─── 5. Template Dokumen Kosong (Blank) ─────────────────────────────
function generateBlankHtml(): string {
  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <style>
    body {
      font-family: 'Times New Roman', Times, serif;
      font-size: 12pt;
      line-height: 1.4;
      margin: 0;
      padding: 10px;
    }
  </style>
</head>
<body>
  <p>Tulis naskah surat dinas Anda di sini...</p>
</body>
</html>`;
}

// ─── Daftar Template Resmi Frontend ────────────────────────────────
export const LETTER_TEMPLATE_LIST: LetterTemplateDefinition[] = [
  {
    id: "tmpl-dinas-standar",
    name: "Surat Dinas Resmi Standar",
    description: "Format surat resmi dengan Kop PT Pupuk Indonesia, Nomor ###NoSurat###, Yth, TTD & Tembusan.",
    letterTypeKeywords: ["surat", "antar", "keluar", "standar", "korporat"],
    generateHtml: generateSuratDinasHtml,
  },
  {
    id: "tmpl-nota-dinas",
    name: "Nota Dinas Internal",
    description: "Format korespondensi internal (Yth, Dari, Tanggal, Perihal, Pembahasan, TTD Elektronik).",
    letterTypeKeywords: ["nota", "dinas", "internal"],
    generateHtml: generateNotaDinasHtml,
  },
  {
    id: "tmpl-undangan",
    name: "Surat Undangan Rapat",
    description: "Format undangan kegiatan/rapat koordinasi lengkap dengan tabel Hari, Waktu, Tempat, Agenda.",
    letterTypeKeywords: ["undangan", "rapat", "acara"],
    generateHtml: generateUndanganHtml,
  },
  {
    id: "tmpl-tugas",
    name: "Surat Perintah Tugas (SPT)",
    description: "Format penugasan kedinasan resmi dengan tabel daftar pegawai dan pejabat penandatangan.",
    letterTypeKeywords: ["tugas", "perintah", "spt"],
    generateHtml: generateSuratTugasHtml,
  },
  {
    id: "tmpl-kosong",
    name: "Dokumen Kosong (Blank)",
    description: "Mulai dari halaman kosong tanpa format awal.",
    letterTypeKeywords: [],
    generateHtml: generateBlankHtml,
  },
];

export function getTemplateById(id?: string): LetterTemplateDefinition {
  const found = LETTER_TEMPLATE_LIST.find((t) => t.id === id);
  return found || LETTER_TEMPLATE_LIST[0]; // default surat dinas standar
}
