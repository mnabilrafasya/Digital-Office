// components/surat/letter-editor/word-editor-config.ts
// Konfigurasi standar DevExpress RichEdit agar tampil dan berperilaku seperti Microsoft Word:
// - Tinggi eksplisit (px/vh) agar saat adjustControl() DevExpress mengatur canvas.style.height = "0",
//   elemen induk tidak kolaps menjadi 0px.
// - Satuan penggaris sentimeter (0 s/d 15,92 cm area ketik pada kertas A4 margin Normal 2,54 cm)
// - PrintLayout view dengan Horizontal Ruler aktif
// - Penyederhanaan Ribbon (sembunyikan File, MailMerge, References; fokus Home, Insert, PageLayout, View)
// - AutoCorrect dasar Bahasa Indonesia & penomoran otomatis

import type { Options } from "devexpress-richedit";

import "./official-fonts.css";

// Tipe modul dinamis DevExpress RichEdit
export type RichEditModule = typeof import("devexpress-richedit") & {
  license?: (key: string) => void;
};

/** Daftar koreksi otomatis (AutoCorrect) salah ketik umum Bahasa Indonesia */
export const INDONESIAN_AUTOCORRECT_RULES = [
  { replace: "yg", with: "yang" },
  { replace: "dgn", with: "dengan" },
  { replace: "utk", with: "untuk" },
  { replace: "dlm", with: "dalam" },
  { replace: "pd", with: "pada" },
  { replace: "sdh", with: "sudah" },
  { replace: "blm", with: "belum" },
  { replace: "krn", with: "karena" },
  { replace: "tsb", with: "tersebut" },
  { replace: "kpd", with: "kepada" },
  { replace: "shg", with: "sehingga" },
  { replace: "spt", with: "seperti" },
  { replace: "terimakasih", with: "terima kasih" },
  { replace: "kerjasama", with: "kerja sama" },
  { replace: "tindaklanjut", with: "tindak lanjut" },
  { replace: "ditindak lanjuti", with: "ditindaklanjuti" },
  { replace: "bertandatangan", with: "bertanda tangan" },
  { replace: "tandatangan", with: "tanda tangan" },
];

/**
 * Font naskah yang boleh dipilih beserta file-nya di public/fonts. File saat ini adalah font
 * terbuka (OFL) yang ukuran hurufnya identik dengan font Office: Liberation untuk Arial, Times New
 * Roman, dan Courier New; Carlito untuk Calibri; Caladea untuk Cambria. Untuk memakai font
 * Microsoft asli cukup mengganti file dan URI di sini, tanpa mengubah kode lain.
 */
const fontFiles = (base: string) => ({
  regularFontUri: `/fonts/${base}-Regular.ttf`,
  boldFontUri: `/fonts/${base}-Bold.ttf`,
  italicFontUri: `/fonts/${base}-Italic.ttf`,
  boldItalicFontUri: `/fonts/${base}-BoldItalic.ttf`,
});

export const OFFICIAL_FONTS = {
  defaultFolder: "/fonts/",
  fonts: [
    { name: "Arial", fontFamily: "Arial", ...fontFiles("LiberationSans") },
    { name: "Times New Roman", fontFamily: "Times New Roman", ...fontFiles("LiberationSerif") },
    { name: "Courier New", fontFamily: "Courier New", ...fontFiles("LiberationMono") },
    { name: "Calibri", fontFamily: "Calibri", ...fontFiles("Carlito") },
    { name: "Cambria", fontFamily: "Cambria", ...fontFiles("Caladea") },
  ],
  mappings: { defaultFontName: "Arial" },
};

/**
 * Memuat modul DevExpress RichEdit sekaligus file font resmi (official-fonts.css). Font dimuat lebih
 * dulu supaya editor mengukur dan menggambar teks dengan file yang sama dengan PDF resmi.
 */
export async function loadRichEdit(): Promise<RichEditModule> {
  const faces = OFFICIAL_FONTS.fonts.flatMap(({ fontFamily }) =>
    ["400", "700", "italic 400", "italic 700"].map((style) => `${style} 16px "${fontFamily}"`)
  );
  const [mod] = await Promise.all([
    import("devexpress-richedit") as Promise<RichEditModule>,
    // Font gagal dimuat tidak menghentikan editor (DevExpress memakai font pengganti)
    Promise.all(faces.map((f) => document.fonts.load(f).catch(() => []))),
  ]);
  return mod;
}

export interface ConfigureWordOptionsParams {
  mod: RichEditModule;
  readOnly: boolean;
  showRibbon: boolean;
  initialZoom: number;
  /**
   * PENTING: Wajib berupa ukuran eksplisit (misal "980px" atau "calc(100vh - 110px)"),
   * JANGAN "100%" jika parent memiliki height: auto, karena adjustControl() DevExpress
   * sempat menyetel canvas.style.height = "0" untuk mengukur clientHeight elemen.
   */
  height?: string;
  onDocumentChanged?: () => void;
  onKeyDown?: (
    sender: unknown,
    args: { handled: boolean; htmlEvent: KeyboardEvent }
  ) => void;
}

/**
 * Membangun objek Options DevExpress RichEdit dengan standar UX Microsoft Word.
 */
export function createWordEditorOptions({
  mod,
  readOnly,
  showRibbon,
  initialZoom,
  height = "980px",
  onDocumentChanged,
  onKeyDown,
}: ConfigureWordOptionsParams): Options {
  const options: Options = mod.createOptions();
  options.width = "100%";
  options.height = height;
  options.readOnly = readOnly;

  // Satuan penggaris: Sentimeter (A4 = 21 cm x 29,7 cm; margin 2,54 cm -> area ketik ~15,9 cm)
  options.unit = mod.RichEditUnit?.Centimeter ?? 0;

  // Tampilan PrintLayout (kertas putih di atas kanvas abu-abu seperti Word) + penggaris aktif
  options.view = {
    zoomLevel: initialZoom,
    viewType: mod.ViewType?.PrintLayout ?? 1,
    printLayoutViewSettings: {
      showHorizontalRuler: true,
    },
  };

  // Font dibatasi ke daftar yang file-nya tersedia di public/fonts, supaya tampilan di layar dan
  // PDF resmi (ekspor DevExpress) memakai file font yang sama. Font lain di naskah dipetakan ke Arial.
  options.fonts = OFFICIAL_FONTS;

  // Konfigurasi AutoCorrect ala Microsoft Word
  options.autoCorrect = {
    correctTwoInitialCapitals: true,
    detectUrls: true,
    enableAutomaticNumbering: true,
    replaceTextAsYouType: true,
    caseSensitiveReplacement: false,
    replaceInfoCollection: INDONESIAN_AUTOCORRECT_RULES,
  };

  // Cetak & Simpan PDF berbasis browser (ClientHtml): mendukung seluruh font sistem & A4 presisi
  options.printing = {
    mode: mod.PrintMode?.Html ?? 1,
    closePrintDialogWithHtmlPreview: true,
  };

  // Nonaktifkan dialog konfirmasi bawaan browser dari DevExpress saat dokumen di-replace/submit,
  // karena form Buat Surat sudah memiliki konfirmasi keluar sendiri.
  options.confirmOnLosingChanges = { enabled: false };

  // Konfigurasi Ribbon: sembunyikan tab yang tidak relevan (File bawaan, MailMerge, References)
  // karena aksi file & sinkronisasi variabel ditangani oleh toolbar aplikasi di atas editor.
  options.ribbon.visible = showRibbon;
  if (showRibbon && mod.RibbonTabType) {
    options.ribbon.removeTab(mod.RibbonTabType.File);
    options.ribbon.removeTab(mod.RibbonTabType.MailMerge);
    options.ribbon.removeTab(mod.RibbonTabType.References);
    options.ribbon.activeTabIndex = 0;
  }

  if (onDocumentChanged) {
    options.events.documentChanged = () => {
      onDocumentChanged();
    };
  }

  if (onKeyDown) {
    options.events.keyDown = (s, args) => {
      onKeyDown(s, args);
    };
  }

  return options;
}
