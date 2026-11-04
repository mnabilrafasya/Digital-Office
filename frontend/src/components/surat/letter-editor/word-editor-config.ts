// components/surat/letter-editor/word-editor-config.ts
// Konfigurasi standar DevExpress RichEdit agar tampil dan berperilaku seperti Microsoft Word:
// - Tinggi eksplisit (px/vh) agar saat adjustControl() DevExpress mengatur canvas.style.height = "0",
//   elemen induk tidak kolaps menjadi 0px.
// - Satuan penggaris sentimeter (0 s/d 15,92 cm area ketik pada kertas A4 margin Normal 2,54 cm)
// - PrintLayout view dengan Horizontal Ruler aktif
// - Penyederhanaan Ribbon (sembunyikan File, MailMerge, References; fokus Home, Insert, PageLayout, View)
// - AutoCorrect dasar Bahasa Indonesia & penomoran otomatis

import type { Options } from "devexpress-richedit";

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

  // Biarkan options.fonts memakai default bawaan DevExpress (sudah mencakup 30+ font Office
  // seperti Calibri, Arial, Times New Roman, Tahoma, Cambria, Bookman Old Style lengkap dengan metriknya).

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
