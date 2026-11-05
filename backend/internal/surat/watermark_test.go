package surat

import (
	"bytes"
	"context"
	"image/png"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestRenderWatermark(t *testing.T) {
	data, err := watermarkPNG()
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("bukan PNG: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != b.Dy() || b.Dx() < 800 {
		t.Fatalf("ukuran gambar = %v, want persegi yang cukup tajam", b)
	}
	alphaAt := func(fx, fy float64) uint32 {
		_, _, _, a := img.At(b.Min.X+int(fx*float64(b.Dx())), b.Min.Y+int(fy*float64(b.Dy()))).RGBA()
		return a >> 8
	}
	// Pojok kiri atas dan kanan bawah kosong; teks melintasi tengah (miring naik ke kanan)
	if alphaAt(0.02, 0.02) != 0 || alphaAt(0.98, 0.98) != 0 {
		t.Error("latar watermark harus transparan")
	}
	var maxAlpha, painted uint32
	for y := b.Min.Y; y < b.Max.Y; y += 4 {
		for x := b.Min.X; x < b.Max.X; x += 4 {
			_, _, _, a := img.At(x, y).RGBA()
			if a>>8 > maxAlpha {
				maxAlpha = a >> 8
			}
			if a > 0 {
				painted++
			}
		}
	}
	if maxAlpha == 0 || maxAlpha > watermarkAlpha+1 || painted == 0 {
		t.Errorf("transparansi watermark salah: alpha maks %d (want 1..%d)", maxAlpha, watermarkAlpha)
	}
	if again, _ := watermarkPNG(); !bytes.Equal(again, data) {
		t.Error("gambar watermark harus dibuat sekali dan sama")
	}
}

func TestWatermarkOnTemplates(t *testing.T) {
	wm, _ := watermarkPNG()
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
	wm, _ := watermarkPNG()
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
		d := &LetterDetail{}
		if code != "" {
			d.Confidentiality = &LookupRef{Code: code}
		}
		got, err := watermarkFor(d)
		if err != nil || (got != nil) != want {
			t.Errorf("sifat %q: watermark ada = %v, want %v (err %v)", code, got != nil, want, err)
		}
	}
}

func TestFilledDocumentHasWatermark(t *testing.T) {
	svc, store, _, _, _ := sentLetter(true)
	ctx := context.Background()
	store.detail.Confidentiality = &LookupRef{Code: "BIASA", Name: "Biasa"}
	f, err := svc.FilledDocument(ctx, andiUser, testLetterID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f.Body)
	if files := docxFiles(t, data); files["word/media/watermark.png"] == "" || !strings.Contains(files[watermarkHeaderPart], "rIdWatermark") {
		t.Error("naskah terisi surat terbit harus ber-watermark")
	}
}
