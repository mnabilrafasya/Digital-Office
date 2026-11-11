package surat

import (
	"context"

	"office.backend/internal/platform/pdf"
	"office.backend/internal/platform/watermark"
)

// Watermark naskah resmi: teks watermark perusahaan (companies.watermark_text, diatur admin
// perusahaan) miring dan abu-abu transparan. Watermark hanya ada di PDF resmi, yaitu setelah surat
// disetujui approver dan diekspor: ditempel backend di setiap halaman saat PDF resmi disimpan
// (SaveFinalPDF), sehingga unduhan dan cetak selalu ber-watermark. Naskah yang tampil di layar (mode
// baca) dan draf tidak ber-watermark.

// watermarkConfidentiality adalah sifat surat (kode lookup CONFIDENTIALITY) yang diberi watermark.
var watermarkConfidentiality = map[string]bool{"BIASA": true, "TERBATAS": true, "RAHASIA": true}

// watermarkText mengembalikan teks watermark untuk surat d, atau "" bila perusahaannya tidak memakai
// watermark atau sifat suratnya tidak diberi watermark.
func watermarkText(d *LetterDetail) string {
	if d.WatermarkText == nil || d.Confidentiality == nil || !watermarkConfidentiality[d.Confidentiality.Code] {
		return ""
	}
	return watermark.Normalize(*d.WatermarkText)
}

// addPDFWatermark menempel watermark surat d ke setiap halaman PDF resmi. Mengembalikan PDF hasil dan
// teks watermark yang ditempel ("" = PDF tidak diubah).
func addPDFWatermark(ctx context.Context, d *LetterDetail, data []byte) ([]byte, string, error) {
	text := watermarkText(d)
	if text == "" {
		return data, "", nil
	}
	img, err := watermark.PNG(text)
	if err != nil {
		return nil, "", err
	}
	out, err := pdf.AddWatermark(ctx, data, img)
	if err != nil {
		return nil, "", err
	}
	return out, text, nil
}
