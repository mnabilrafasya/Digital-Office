package surat

import (
	"bytes"
	"context"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"office.backend/internal/platform/watermark"
)

// testWatermark adalah watermark contoh untuk uji naskah.
func testWatermark() ([]byte, error) { return watermark.PNG("PUPUK SRIWIDJAJA") }

func TestWatermarkOnTemplates(t *testing.T) {
	wm, _ := testWatermark()
	for _, name := range []string{"surat-dinas-standar", "nota-dinas", "undangan-rapat", "surat-tugas", "dokumen-kosong"} {
		src, err := os.ReadFile("../../database/seeds/templates/" + name + ".docx")
		if err != nil {
			t.Fatal(err)
		}
		out, err := fillPlaceholders(src, map[string]string{phNoSurat: "001/PSP/XI/2026"}, []byte("\x89PNG"), wm)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		files := docxFiles(t, out)
		hdr := files["word/header1.xml"]
		if !strings.Contains(hdr, `behindDoc="1"`) || !strings.Contains(hdr, `relativeFrom="page"`) || !strings.Contains(hdr, `r:embed="rIdWatermark"`) {
			t.Errorf("%s: watermark tidak masuk header", name)
		}
		paragraphTexts(t, hdr) // XML header tetap valid
		if !strings.Contains(files["word/_rels/header1.xml.rels"], `Id="rIdWatermark"`) || files["word/media/watermark.png"] != string(wm) {
			t.Errorf("%s: relasi/gambar watermark tidak ada", name)
		}
		// Template A4 (11909 x 16834 twip): watermark di tengah halaman
		w := int64(watermarkWidthEMU)
		if want := (int64(11909)*emuPerTwip - w) / 2; !strings.Contains(hdr, "<wp:posOffset>"+strconv.FormatInt(want, 10)+"</wp:posOffset>") {
			t.Errorf("%s: posisi horizontal bukan tengah halaman (want %d)", name, want)
		}
		if strings.Contains(files["word/document.xml"], "rIdWatermark") {
			t.Errorf("%s: watermark tidak boleh di isi naskah (header sudah ada)", name)
		}
	}
}

func TestWatermarkCreatesHeader(t *testing.T) {
	wm, _ := testWatermark()
	sect := `<w:sectPr><w:pgSz w:w="11906" w:h="16838"/></w:sectPr>`
	out, err := fillPlaceholders(makeDocx(docWith(`<w:p><w:r><w:t>Tanpa header</w:t></w:r></w:p>`+sect)), nil, []byte("png"), wm)
	if err != nil {
		t.Fatal(err)
	}
	files := docxFiles(t, out)
	doc := files["word/document.xml"]
	if !strings.Contains(doc, `<w:sectPr><w:headerReference`) || !strings.Contains(doc, `r:id="rIdWatermarkHeader"`) {
		t.Errorf("header tidak dipasang di pengaturan halaman:\n%s", doc)
	}
	paragraphTexts(t, doc)
	if hdr := files[watermarkHeaderPart]; !strings.Contains(hdr, `behindDoc="1"`) {
		t.Errorf("header watermark tidak dibuat: %q", hdr)
	}
	if !strings.Contains(files["word/_rels/document.xml.rels"], `Target="header-watermark.xml"`) ||
		!strings.Contains(files["[Content_Types].xml"], `PartName="/word/header-watermark.xml"`) ||
		!strings.Contains(files["word/_rels/header-watermark.xml.rels"], `Id="rIdWatermark"`) {
		t.Error("relasi/tipe header watermark tidak lengkap")
	}
}

func TestWatermarkPerConfidentiality(t *testing.T) {
	saved := watermarkConfidentiality
	defer func() { watermarkConfidentiality = saved }()
	watermarkConfidentiality = map[string]bool{"RAHASIA": true}

	for code, want := range map[string]bool{"RAHASIA": true, "BIASA": false, "": false} {
		d := &LetterDetail{WatermarkText: ptr("PUPUK SRIWIDJAJA")}
		if code != "" {
			d.Confidentiality = &LookupRef{Code: code}
		}
		got, err := watermarkFor(d)
		if err != nil || (got != nil) != want {
			t.Errorf("sifat %q: watermark ada = %v, want %v (err %v)", code, got != nil, want, err)
		}
	}
}

// Watermark mengikuti perusahaan surat: tanpa teks = tanpa watermark, teks berbeda = gambar berbeda.
func TestWatermarkPerCompany(t *testing.T) {
	biasa := &LookupRef{Code: "BIASA"}
	for _, text := range []*string{nil, ptr(""), ptr("   ")} {
		if got, err := watermarkFor(&LetterDetail{WatermarkText: text, Confidentiality: biasa}); err != nil || got != nil {
			t.Errorf("teks %v: watermark ada = %v (err %v), want tidak ada", text, got != nil, err)
		}
	}
	psp, err := watermarkFor(&LetterDetail{WatermarkText: ptr(" PUPUK  SRIWIDJAJA "), Confidentiality: biasa})
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := watermarkFor(&LetterDetail{WatermarkText: ptr("PETROKIMIA GRESIK"), Confidentiality: biasa})
	if err != nil {
		t.Fatal(err)
	}
	want, _ := testWatermark()
	if !bytes.Equal(psp, want) || bytes.Equal(psp, pkg) {
		t.Error("watermark harus mengikuti teks perusahaan (spasi berlebih dirapikan)")
	}
}

func TestFilledDocumentHasWatermark(t *testing.T) {
	svc, store, _, _, _ := sentLetter(true)
	ctx := context.Background()
	store.detail.Confidentiality = &LookupRef{Code: "BIASA", Name: "Biasa"}
	store.detail.WatermarkText = ptr("PUPUK SRIWIDJAJA")
	f, err := svc.FilledDocument(ctx, andiUser, testLetterID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f.Body)
	if files := docxFiles(t, data); files["word/media/watermark.png"] == "" || !strings.Contains(files[watermarkHeaderPart], "rIdWatermark") {
		t.Error("naskah terisi surat terbit harus ber-watermark")
	}
}
