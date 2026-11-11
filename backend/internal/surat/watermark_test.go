package surat

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"office.backend/internal/auth"
	"office.backend/internal/platform/pdf"
)

func TestWatermarkText(t *testing.T) {
	saved := watermarkConfidentiality
	defer func() { watermarkConfidentiality = saved }()
	watermarkConfidentiality = map[string]bool{"RAHASIA": true}

	for code, want := range map[string]string{"RAHASIA": "PUPUK SRIWIDJAJA", "BIASA": "", "": ""} {
		d := &LetterDetail{WatermarkText: ptr(" PUPUK  SRIWIDJAJA ")}
		if code != "" {
			d.Confidentiality = &LookupRef{Code: code}
		}
		if got := watermarkText(d); got != want {
			t.Errorf("sifat %q: %q, want %q", code, got, want)
		}
	}
	// Perusahaan tanpa watermark
	watermarkConfidentiality = saved
	for _, text := range []*string{nil, ptr(""), ptr("   ")} {
		if got := watermarkText(&LetterDetail{WatermarkText: text, Confidentiality: &LookupRef{Code: "BIASA"}}); got != "" {
			t.Errorf("teks %v: %q, want tanpa watermark", text, got)
		}
	}
}

// Naskah yang tampil di layar (mode baca) tidak ber-watermark; watermark hanya di PDF resmi.
func TestFilledDocumentWithoutWatermark(t *testing.T) {
	svc, store, _, _, _ := sentLetter(true)
	store.detail.Confidentiality = &LookupRef{Code: "BIASA", Name: "Biasa"}
	store.detail.WatermarkText = ptr("PUPUK SRIWIDJAJA")
	f, err := svc.FilledDocument(context.Background(), andiUser, testLetterID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f.Body)
	for name := range docxFiles(t, data) {
		if strings.Contains(name, "watermark") {
			t.Errorf("naskah mode baca memuat watermark: %s", name)
		}
	}
}

// PDF resmi yang disimpan ber-watermark di setiap halaman; perusahaan tanpa watermark: PDF apa adanya.
func TestFinalPDFHasWatermark(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		text *string
		want bool
	}{
		{"PSP", ptr("PUPUK SRIWIDJAJA"), true},
		{"tanpa watermark", nil, false},
	} {
		svc, store, _, _, audit := sentLetter(true)
		store.detail.Confidentiality = &LookupRef{Code: "BIASA", Name: "Biasa"}
		store.detail.WatermarkText = tc.text
		src := minimalPDF(10)
		if _, _, err := svc.SaveFinalPDF(ctx, andiUser, FinalPDFInput{LetterID: testLetterID, Data: src}); err != nil {
			t.Fatalf("%s: SaveFinalPDF: %v", tc.name, err)
		}
		f, err := svc.OpenDocument(ctx, andiUser, testLetterID, 0)
		if err != nil {
			t.Fatalf("%s: OpenDocument: %v", tc.name, err)
		}
		stored, _ := io.ReadAll(f.Body)
		if got := !bytes.Equal(stored, src); got != tc.want {
			t.Errorf("%s: PDF diberi watermark = %v, want %v", tc.name, got, tc.want)
		}
		if n, err := pdf.PageCount(ctx, stored); err != nil || n != 1 {
			t.Errorf("%s: PDF tersimpan rusak: %d halaman, %v", tc.name, n, err)
		}
		var snap finalPDFSnapshot
		for _, e := range audit.entries {
			if e.Action == "FINAL_PDF" {
				snap, _ = e.Snapshot.(finalPDFSnapshot)
			}
		}
		wantText := ""
		if tc.want {
			wantText = "PUPUK SRIWIDJAJA"
		}
		if snap.Watermark != wantText || snap.FileSize != int64(len(stored)) {
			t.Errorf("%s: audit %+v", tc.name, snap)
		}
	}
}

// Penerima juga mendapat PDF salinannya ber-watermark.
func TestRecipientCopyPDFHasWatermark(t *testing.T) {
	svc, store, _, inbox, _ := sentLetter(true)
	inbox.holders = map[int64]bool{50: true}
	store.detail.Confidentiality = &LookupRef{Code: "RAHASIA", Name: "Rahasia"}
	store.detail.WatermarkText = ptr("PETROKIMIA GRESIK")
	recipient := &auth.CurrentUser{ID: 50, Company: auth.UserCompany{Code: "PSP"}}
	ctx := context.Background()
	src := minimalPDF(30)
	if _, _, err := svc.SaveFinalPDF(ctx, recipient, FinalPDFInput{LetterID: testLetterID, Data: src}); err != nil {
		t.Fatal(err)
	}
	f, err := svc.OpenDocument(ctx, recipient, testLetterID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if stored, _ := io.ReadAll(f.Body); bytes.Equal(stored, src) {
		t.Error("PDF salinan penerima harus ber-watermark")
	}
}
