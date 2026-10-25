// ─────────────────────────────────────────────
// lib/templates/letter-templates.ts
// Format & Template Tata Naskah Dinas Resmi (Frontend)
// Dirancang khusus untuk DevExpress RichEdit HtmlImporter
// Wajib menggunakan inline style (DevExpress mengabaikan blok <style>)
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
  const lampiran =
    ctx.attachmentsCount && ctx.attachmentsCount > 0
      ? `${ctx.attachmentsCount} Berkas`
      : "—";
  const penerima =
    ctx.recipients && ctx.recipients.length > 0
      ? ctx.recipients.map((r) => r.name).join("<br/>")
      : "Yth. Pimpinan Unit Kerja Terkait<br/>di Tempat";
  const approverName = ctx.approver?.name || "[Nama Pejabat Penandatangan]";
  const approverJabatan =
    ctx.approver?.position || "[Jabatan Pejabat Penandatangan]";
  const approverNip = ctx.approver?.employeeNumber
    ? `NIP. ${ctx.approver.employeeNumber}`
    : "NIP. ..............................";

  return `
  <!-- KOP SURAT RESMI -->
  <table style="width: 100%; border-collapse: collapse; margin-bottom: 2px;">
    <tr>
      <td style="text-align: center; vertical-align: top;">
        <p style="font-family: 'Times New Roman', serif; font-size: 14pt; font-weight: bold; text-transform: uppercase; margin: 0 0 2px 0;">PT PUPUK SRIWIDJAJA PALEMBANG</p>
        <p style="font-family: 'Times New Roman', serif; font-size: 12.5pt; font-weight: bold; text-transform: uppercase; margin: 0 0 3px 0;">DIREKTORAT TEKNOLOGI INFORMASI</p>
        <p style="font-family: 'Times New Roman', serif; font-size: 9.5pt; margin: 0 0 4px 0;">Jalan Mayor Zen, Kalidoni, Kota Palembang, Sumatera Selatan</p>
      </td>
    </tr>
  </table>
  <hr style="border-top: 2px solid #000; border-bottom: none; margin: 2px 0 16px 0;" />

  <!-- TANGGAL & METADATA SURAT -->
  <table style="width: 100%; border-collapse: collapse; font-family: 'Times New Roman', serif; font-size: 11pt; margin-bottom: 16px;">
    <tr>
      <td style="width: 15%; padding: 2px 0; vertical-align: top;">Nomor</td>
      <td style="width: 2%; padding: 2px 0; vertical-align: top;">:</td>
      <td style="width: 43%; padding: 2px 0; vertical-align: top;">###NoSurat###</td>
      <td style="width: 40%; padding: 2px 0; vertical-align: top; text-align: right;">Jakarta, ${tanggal}</td>
    </tr>
    <tr>
      <td style="padding: 2px 0; vertical-align: top;">Sifat</td>
      <td style="padding: 2px 0; vertical-align: top;">:</td>
      <td style="padding: 2px 0; vertical-align: top;">${sifat}</td>
      <td></td>
    </tr>
    <tr>
      <td style="padding: 2px 0; vertical-align: top;">Lampiran</td>
      <td style="padding: 2px 0; vertical-align: top;">:</td>
      <td style="padding: 2px 0; vertical-align: top;">${lampiran}</td>
      <td></td>
    </tr>
    <tr>
      <td style="padding: 2px 0; vertical-align: top;">Perihal</td>
      <td style="padding: 2px 0; vertical-align: top;">:</td>
      <td style="padding: 2px 0; vertical-align: top;"><strong>${perihal}</strong></td>
      <td></td>
    </tr>
  </table>

  <!-- KEPADA YTH -->
  <p style="font-family: 'Times New Roman', serif; font-size: 11pt; margin: 0 0 14px 0; line-height: 1.4;">
    Kepada Yth.<br/>
    <strong>${penerima}</strong><br/>
    di Tempat
  </p>

  <!-- SALAM PEMBUKA -->
  <p style="font-family: 'Times New Roman', serif; font-size: 11pt; margin: 0 0 10px 0;">Dengan hormat,</p>

  <!-- ISI SURAT -->
  <p style="font-family: 'Times New Roman', serif; font-size: 11pt; text-align: justify; text-indent: 36pt; margin: 0 0 10px 0; line-height: 1.4;">
    Sehubungan dengan pelaksanaan program kerja dan operasional di lingkungan PT SRIWIDJAJA Palembang, dengan ini kami sampaikan pokok-pokok pemberitahuan dan koordinasi sebagai berikut:
  </p>

  <p style="font-family: 'Times New Roman', serif; font-size: 11pt; text-align: justify; text-indent: 36pt; margin: 0 0 10px 0; line-height: 1.4;">
    1. [Uraikan latar belakang atau konteks perihal surat dinas secara jelas dan ringkas di bagian ini].
  </p>

  <p style="font-family: 'Times New Roman', serif; font-size: 11pt; text-align: justify; text-indent: 36pt; margin: 0 0 10px 0; line-height: 1.4;">
    2. [Uraikan isi pokok, instruksi, permohonan, atau pemberitahuan yang ingin disampaikan kepada pihak penerima].
  </p>

  <p style="font-family: 'Times New Roman', serif; font-size: 11pt; text-align: justify; text-indent: 36pt; margin: 0 0 16px 0; line-height: 1.4;">
    Demikian surat ini kami sampaikan untuk menjadi perhatian dan dapat ditindaklanjuti sebagaimana mestinya. Atas perhatian dan kerja sama yang baik, kami ucapkan terima kasih.
  </p>

  <!-- TANDA TANGAN / APPROVER -->
  <table style="width: 100%; border-collapse: collapse; font-family: 'Times New Roman', serif; font-size: 11pt; margin-top: 24px;">
    <tr>
      <td style="width: 50%;"></td>
      <td style="width: 50%; text-align: center; vertical-align: top;">
        <p style="margin: 0 0 4px 0;"><strong>${approverJabatan}</strong></p>
        <p style="margin: 35px 0 35px 0; font-style: italic; color: #888888; font-size: 9.5pt;">(Ruang Tanda Tangan Digital / QR Code)</p>
        <p style="margin: 0 0 2px 0;"><strong><u>${approverName}</u></strong></p>
        <p style="margin: 0; font-size: 10pt;">${approverNip}</p>
      </td>
    </tr>
  </table>

  <!-- TEMBUSAN -->
  <p style="font-family: 'Times New Roman', serif; font-size: 9.5pt; margin-top: 24px; line-height: 1.3;">
    <strong>Tembusan:</strong><br/>
    1. Direksi PT Pupuk Sriwidjaja Palembang<br/>
    2. Arsip Digital Office
  </p>
`;
}

// ─── 2. Template Nota Dinas (Internal) ─────────────────────────────
function generateNotaDinasHtml(ctx: LetterTemplateContext): string {
  const tanggal = formatDateIndo(ctx.date);
  const perihal = ctx.title?.trim() || "[Perihal / Hal Nota Dinas]";
  const sifat = ctx.sifat || "BIASA";
  const penerima =
    ctx.recipients && ctx.recipients.length > 0
      ? ctx.recipients.map((r) => r.name).join(", ")
      : "Pimpinan Unit Terkait";
  const pengirim = ctx.drafter?.position || "Staf Pengembang Sistem / Unit TI";
  const approverName = ctx.approver?.name || "[Nama Pejabat]";
  const approverJabatan = ctx.approver?.position || "[Jabatan Pejabat]";

  return `
  <p style="font-family: Arial, sans-serif; text-align: center; font-size: 14pt; font-weight: bold; text-decoration: underline; margin: 0 0 2px 0; text-transform: uppercase;">NOTA DINAS</p>
  <p style="font-family: Arial, sans-serif; text-align: center; font-size: 10pt; margin: 0 0 16px 0;">Nomor: ###NoSurat###</p>

  <table style="width: 100%; border-collapse: collapse; font-family: Arial, sans-serif; font-size: 10.5pt; margin-bottom: 12px;">
    <tr>
      <td style="width: 18%; padding: 3px 0; vertical-align: top;"><strong>Yth.</strong></td>
      <td style="width: 2%; padding: 3px 0; vertical-align: top;">:</td>
      <td style="width: 80%; padding: 3px 0; vertical-align: top;">${penerima}</td>
    </tr>
    <tr>
      <td style="padding: 3px 0; vertical-align: top;"><strong>Dari</strong></td>
      <td style="padding: 3px 0; vertical-align: top;">:</td>
      <td style="padding: 3px 0; vertical-align: top;">${pengirim}</td>
    </tr>
    <tr>
      <td style="padding: 3px 0; vertical-align: top;"><strong>Tanggal</strong></td>
      <td style="padding: 3px 0; vertical-align: top;">:</td>
      <td style="padding: 3px 0; vertical-align: top;">${tanggal}</td>
    </tr>
    <tr>
      <td style="padding: 3px 0; vertical-align: top;"><strong>Sifat</strong></td>
      <td style="padding: 3px 0; vertical-align: top;">:</td>
      <td style="padding: 3px 0; vertical-align: top;">${sifat}</td>
    </tr>
    <tr>
      <td style="padding: 3px 0; vertical-align: top;"><strong>Perihal</strong></td>
      <td style="padding: 3px 0; vertical-align: top;">:</td>
      <td style="padding: 3px 0; vertical-align: top;"><strong>${perihal}</strong></td>
    </tr>
  </table>

  <hr style="border-top: 2px solid #000; border-bottom: none; margin: 0 0 14px 0;" />

  <p style="font-family: Arial, sans-serif; font-size: 11pt; text-align: justify; margin: 0 0 10px 0; line-height: 1.4;">
    <strong>1. Pengantar / Latar Belakang</strong><br/>
    Menindaklanjuti kebutuhan operasional dan peningkatan efisiensi proses bisnis, bersama ini disampaikan pokok evaluasi dan pertimbangan teknis terkait ${perihal}.
  </p>

  <p style="font-family: Arial, sans-serif; font-size: 11pt; text-align: justify; margin: 0 0 10px 0; line-height: 1.4;">
    <strong>2. Pokok Pembahasan & Rekomendasi</strong><br/>
    [Tuliskan analisa masalah, opsi tindakan, atau kebutuhan sumber daya di bagian ini secara rinci].
  </p>

  <p style="font-family: Arial, sans-serif; font-size: 11pt; text-align: justify; margin: 0 0 16px 0; line-height: 1.4;">
    <strong>3. Kesimpulan & Permohonan Arahan</strong><br/>
    Demikian Nota Dinas ini kami sampaikan untuk menjadi bahan pertimbangan. Mohon arahan dan persetujuan Bapak/Ibu lebih lanjut.
  </p>

  <table style="width: 100%; border-collapse: collapse; font-family: Arial, sans-serif; font-size: 10.5pt; margin-top: 24px;">
    <tr>
      <td style="width: 50%;"></td>
      <td style="width: 50%; text-align: center; vertical-align: top;">
        <p style="margin: 0 0 4px 0;"><strong>${approverJabatan}</strong></p>
        <p style="margin: 35px 0 35px 0; font-style: italic; color: #888888; font-size: 9pt;">(Tanda Tangan Elektronik)</p>
        <p style="margin: 0;"><strong><u>${approverName}</u></strong></p>
      </td>
    </tr>
  </table>
`;
}

// ─── 3. Template Undangan Rapat / Kegiatan ─────────────────────────
function generateUndanganHtml(ctx: LetterTemplateContext): string {
  const tanggal = formatDateIndo(ctx.date);
  const perihal = ctx.title?.trim() || "Undangan Rapat Koordinasi";
  const penerima =
    ctx.recipients && ctx.recipients.length > 0
      ? ctx.recipients.map((r) => r.name).join("<br/>")
      : "Bapak/Ibu Pimpinan Unit Kerja<br/>di Tempat";
  const approverName = ctx.approver?.name || "[Nama Pejabat]";
  const approverJabatan = ctx.approver?.position || "[Jabatan Pejabat]";

  return `
  <table style="width: 100%; border-collapse: collapse; margin-bottom: 2px;">
    <tr>
      <td style="text-align: center; vertical-align: top;">
        <p style="font-family: 'Times New Roman', serif; font-size: 13pt; font-weight: bold; text-transform: uppercase; margin: 0 0 2px 0;">PT PUPUK SRIWIDJAJA PALEMBANG</p>
        <p style="font-family: 'Times New Roman', serif; font-size: 11pt; margin: 0 0 4px 0;">DIREKTORAT TEKNOLOGI INFORMASI</p>
      </td>
    </tr>
  </table>
  <hr style="border-top: 2px solid #000; border-bottom: none; margin: 2px 0 14px 0;" />

  <table style="width: 100%; border-collapse: collapse; font-family: 'Times New Roman', serif; font-size: 11pt; margin-bottom: 14px;">
    <tr>
      <td style="width: 15%; padding: 2px 0; vertical-align: top;">Nomor</td>
      <td style="width: 2%; padding: 2px 0; vertical-align: top;">:</td>
      <td style="width: 43%; padding: 2px 0; vertical-align: top;">###NoSurat###</td>
      <td style="width: 40%; padding: 2px 0; vertical-align: top; text-align: right;">Jakarta, ${tanggal}</td>
    </tr>
    <tr>
      <td style="padding: 2px 0; vertical-align: top;">Sifat</td>
      <td style="padding: 2px 0; vertical-align: top;">:</td>
      <td style="padding: 2px 0; vertical-align: top;">Penting</td>
      <td></td>
    </tr>
    <tr>
      <td style="padding: 2px 0; vertical-align: top;">Lampiran</td>
      <td style="padding: 2px 0; vertical-align: top;">:</td>
      <td style="padding: 2px 0; vertical-align: top;">1 (satu) Berkas Susunan Acara</td>
      <td></td>
    </tr>
    <tr>
      <td style="padding: 2px 0; vertical-align: top;">Perihal</td>
      <td style="padding: 2px 0; vertical-align: top;">:</td>
      <td style="padding: 2px 0; vertical-align: top;"><strong>${perihal}</strong></td>
      <td></td>
    </tr>
  </table>

  <p style="font-family: 'Times New Roman', serif; font-size: 11pt; margin: 0 0 12px 0; line-height: 1.4;">
    Kepada Yth.<br/>
    <strong>${penerima}</strong><br/>
    di Tempat
  </p>

  <p style="font-family: 'Times New Roman', serif; font-size: 11pt; margin: 0 0 10px 0;">Dengan hormat,</p>

  <p style="font-family: 'Times New Roman', serif; font-size: 11pt; text-align: justify; text-indent: 30pt; margin: 0 0 10px 0; line-height: 1.4;">
    Dalam rangka koordinasi dan penyelarasan target operasional, kami mengharapkan kehadiran Bapak/Ibu dalam rapat koordinasi yang akan diselenggarakan pada:
  </p>

  <table style="margin: 8px 0 14px 24px; font-family: 'Times New Roman', serif; font-size: 11pt; border-collapse: collapse;">
    <tr>
      <td style="width: 140px; padding: 3px 0; vertical-align: top;"><strong>Hari / Tanggal</strong></td>
      <td style="width: 15px; padding: 3px 0; vertical-align: top;">:</td>
      <td style="padding: 3px 0; vertical-align: top;">Senin, ${tanggal}</td>
    </tr>
    <tr>
      <td style="padding: 3px 0; vertical-align: top;"><strong>Waktu</strong></td>
      <td style="padding: 3px 0; vertical-align: top;">:</td>
      <td style="padding: 3px 0; vertical-align: top;">09.00 WIB s.d. selesai</td>
    </tr>
    <tr>
      <td style="padding: 3px 0; vertical-align: top;"><strong>Tempat / Media</strong></td>
      <td style="padding: 3px 0; vertical-align: top;">:</td>
      <td style="padding: 3px 0; vertical-align: top;">Ruang Rapat Utama Lantai 4 / Hybrid Zoom Meeting</td>
    </tr>
    <tr>
      <td style="padding: 3px 0; vertical-align: top;"><strong>Agenda</strong></td>
      <td style="padding: 3px 0; vertical-align: top;">:</td>
      <td style="padding: 3px 0; vertical-align: top;">${perihal}</td>
    </tr>
  </table>

  <p style="font-family: 'Times New Roman', serif; font-size: 11pt; text-align: justify; text-indent: 30pt; margin: 0 0 16px 0; line-height: 1.4;">
    Mengingat pentingnya agenda tersebut, dimohon kehadiran Bapak/Ibu tepat pada waktunya. Demikian undangan ini kami sampaikan, atas kehadiran dan kerja samanya diucapkan terima kasih.
  </p>

  <table style="width: 100%; border-collapse: collapse; font-family: 'Times New Roman', serif; font-size: 11pt; margin-top: 24px;">
    <tr>
      <td style="width: 50%;"></td>
      <td style="width: 50%; text-align: center; vertical-align: top;">
        <p style="margin: 0 0 4px 0;"><strong>${approverJabatan}</strong></p>
        <p style="margin: 35px 0 35px 0; font-style: italic; color: #888888; font-size: 9.5pt;">(Ruang Tanda Tangan)</p>
        <p style="margin: 0;"><strong><u>${approverName}</u></strong></p>
      </td>
    </tr>
  </table>
`;
}

// ─── 4. Template Surat Tugas (SPT) ─────────────────────────────────
function generateSuratTugasHtml(ctx: LetterTemplateContext): string {
  const tanggal = formatDateIndo(ctx.date);
  const perihal = ctx.title?.trim() || "Penugasan Implementasi Sistem";
  const approverName = ctx.approver?.name || "[Nama Pejabat]";
  const approverJabatan = ctx.approver?.position || "[Jabatan Pejabat]";

  return `
  <table style="width: 100%; border-collapse: collapse; margin-bottom: 2px;">
    <tr>
      <td style="text-align: center; vertical-align: top;">
        <p style="font-family: 'Times New Roman', serif; font-size: 12pt; font-weight: bold; text-transform: uppercase; margin: 0 0 2px 0;">PT PUPUK SRIWIDJAJA PALEMBANG</p>
      </td>
    </tr>
  </table>
  <hr style="border-top: 2px solid #000; border-bottom: none; margin: 2px 0 16px 0;" />

  <p style="font-family: 'Times New Roman', serif; text-align: center; font-size: 14pt; font-weight: bold; text-decoration: underline; margin: 0 0 2px 0;">SURAT PERINTAH TUGAS</p>
  <p style="font-family: 'Times New Roman', serif; text-align: center; font-size: 10.5pt; margin: 0 0 18px 0;">Nomor: ###NoSurat###</p>

  <p style="font-family: 'Times New Roman', serif; font-size: 11pt; text-align: justify; margin: 0 0 12px 0; line-height: 1.4;">
    Yang bertanda tangan di bawah ini, <strong>${approverJabatan}</strong> PT Pupuk Sriwidjaja Palembang, dengan ini menugaskan kepada pegawai berikut:
  </p>

  <table style="width: 100%; border-collapse: collapse; font-family: 'Times New Roman', serif; font-size: 10.5pt; margin: 10px 0 14px 0;">
    <thead>
      <tr style="background-color: #f2f2f2;">
        <th style="border: 1px solid #000; padding: 4px 6px; width: 6%; text-align: center;">No</th>
        <th style="border: 1px solid #000; padding: 4px 6px; width: 34%; text-align: left;">Nama Pegawai</th>
        <th style="border: 1px solid #000; padding: 4px 6px; width: 25%; text-align: left;">NIK / NIP</th>
        <th style="border: 1px solid #000; padding: 4px 6px; width: 35%; text-align: left;">Jabatan / Unit</th>
      </tr>
    </thead>
    <tbody>
      <tr>
        <td style="border: 1px solid #000; padding: 4px 6px; text-align: center;">1</td>
        <td style="border: 1px solid #000; padding: 4px 6px;"><strong>Andi Kurniawan</strong></td>
        <td style="border: 1px solid #000; padding: 4px 6px;">TI-2024-001</td>
        <td style="border: 1px solid #000; padding: 4px 6px;">Staf Pengembang Sistem</td>
      </tr>
      <tr>
        <td style="border: 1px solid #000; padding: 4px 6px; text-align: center;">2</td>
        <td style="border: 1px solid #000; padding: 4px 6px;"><strong>Budi Santoso</strong></td>
        <td style="border: 1px solid #000; padding: 4px 6px;">TI-2022-015</td>
        <td style="border: 1px solid #000; padding: 4px 6px;">Koordinator Pengembangan Sistem</td>
      </tr>
    </tbody>
  </table>

  <p style="font-family: 'Times New Roman', serif; font-size: 11pt; text-align: justify; margin: 0 0 10px 0; line-height: 1.4;">
    <strong>Untuk:</strong> Melaksanakan penugasan ${perihal} sesuai dengan ketentuan dan prosedur standar operasional perusahaan.
  </p>

  <p style="font-family: 'Times New Roman', serif; font-size: 11pt; text-align: justify; margin: 0 0 16px 0; line-height: 1.4;">
    Demikian Surat Perintah Tugas ini dibuat untuk dapat dilaksanakan dengan penuh tanggung jawab dan menyampaikan laporan hasil pelaksanaan setelah kegiatan selesai.
  </p>

  <table style="width: 100%; border-collapse: collapse; font-family: 'Times New Roman', serif; font-size: 11pt; margin-top: 24px;">
    <tr>
      <td style="width: 50%;"></td>
      <td style="width: 50%; text-align: center; vertical-align: top;">
        <p style="margin: 0 0 2px 0;">Ditetapkan di: Jakarta</p>
        <p style="margin: 0 0 6px 0;">Pada tanggal: ${tanggal}</p>
        <p style="margin: 0 0 4px 0;"><strong>${approverJabatan}</strong></p>
        <p style="margin: 35px 0 35px 0; font-style: italic; color: #888888; font-size: 9.5pt;">(Tanda Tangan)</p>
        <p style="margin: 0;"><strong><u>${approverName}</u></strong></p>
      </td>
    </tr>
  </table>
`;
}

// ─── 5. Template Dokumen Kosong (Blank) ─────────────────────────────
function generateBlankHtml(): string {
  return `<p style="font-family: 'Times New Roman', serif; font-size: 12pt; margin: 0;">&nbsp;</p>`;
}

// ─── Daftar Template Resmi Frontend ────────────────────────────────
export const LETTER_TEMPLATE_LIST: LetterTemplateDefinition[] = [
  {
    id: "tmpl-dinas-standar",
    name: "Surat Dinas Resmi Standar",
    description:
      "Format surat resmi dengan Kop PT Pupuk Sriwidjaja Palembang, Nomor ###NoSurat###, Yth, TTD & Tembusan.",
    letterTypeKeywords: ["surat", "antar", "keluar", "standar", "korporat"],
    generateHtml: generateSuratDinasHtml,
  },
  {
    id: "tmpl-nota-dinas",
    name: "Nota Dinas Internal",
    description:
      "Format korespondensi internal (Yth, Dari, Tanggal, Perihal, Pembahasan, TTD Elektronik).",
    letterTypeKeywords: ["nota", "dinas", "internal"],
    generateHtml: generateNotaDinasHtml,
  },
  {
    id: "tmpl-undangan",
    name: "Surat Undangan Rapat",
    description:
      "Format undangan kegiatan/rapat koordinasi lengkap dengan tabel Hari, Waktu, Tempat, Agenda.",
    letterTypeKeywords: ["undangan", "rapat", "acara"],
    generateHtml: generateUndanganHtml,
  },
  {
    id: "tmpl-tugas",
    name: "Surat Perintah Tugas (SPT)",
    description:
      "Format penugasan kedinasan resmi dengan tabel daftar pegawai dan pejabat penandatangan.",
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
