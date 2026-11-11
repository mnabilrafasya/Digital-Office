package surat

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// docxFiles membuka .docx menjadi peta nama file -> isi.
func docxFiles(t *testing.T, docx []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		t.Fatalf("hasil bukan .docx: %v", err)
	}
	files := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		data, _ := io.ReadAll(rc)
		rc.Close()
		files[f.Name] = string(data)
	}
	return files
}

// paragraphTexts mengambil teks per paragraf (baris baru <w:br/> ditulis "|"), sekaligus
// memastikan XML-nya valid.
func paragraphTexts(t *testing.T, part string) []string {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(part))
	var (
		out    []string
		cur    strings.Builder
		inText bool
	)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("XML tidak valid: %v\n%s", err, part)
		}
		switch el := tok.(type) {
		case xml.StartElement:
			inText = el.Name.Local == "t"
			if el.Name.Local == "br" {
				cur.WriteString("|")
			}
		case xml.EndElement:
			inText = false
			if el.Name.Local == "p" {
				out = append(out, cur.String())
				cur.Reset()
			}
		case xml.CharData:
			if inText {
				cur.Write(el)
			}
		}
	}
	return out
}

const wDoc = `<?xml version="1.0" encoding="UTF-8"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>%s</w:body></w:document>`

func docWith(paragraphs string) string { return strings.Replace(wDoc, "%s", paragraphs, 1) }

func TestFillPlaceholdersText(t *testing.T) {
	doc := docWith(
		// penanda terpecah di dua run dengan format berbeda
		`<w:p><w:r><w:t>Nomor: ###*No</w:t></w:r><w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">Surat*### (resmi)</w:t></w:r></w:p>` +
			// format lama, karakter khusus, dan penanda isi yang harus dibiarkan
			`<w:p><w:r><w:t>Perihal: ###Perihal### &amp; ###*Poin1*###</w:t></w:r></w:p>` +
			// beberapa baris, huruf besar-kecil berbeda
			`<w:p><w:r><w:t>###*KEPADA*###</w:t></w:r></w:p>` +
			// penanda yang terpotong di dua paragraf tidak dianggap penanda
			`<w:p><w:r><w:t>###*No</w:t></w:r></w:p><w:p><w:r><w:t>Surat*###</w:t></w:r></w:p>` +
			`<w:p><w:r><w:tab/><w:t xml:space="preserve"/><w:t>Tanpa penanda</w:t></w:r></w:p>`)
	header := docWith(`<w:p><w:r><w:t>Nomor ###*NoSurat*###</w:t></w:r></w:p>`)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range map[string]string{
		"[Content_Types].xml": `<Types><Default Extension="xml" ContentType="application/xml"/></Types>`,
		"word/document.xml":   doc,
		"word/header1.xml":    header,
		"word/styles.xml":     "<w:t>###*NoSurat*###</w:t>",
	} {
		w, _ := zw.Create(name)
		w.Write([]byte(content))
	}
	zw.Close()

	values := map[string]string{phNoSurat: "015/PSP/X/2026"}
	out, err := fillPlaceholders(buf.Bytes(), values, []byte("png"))
	if err != nil {
		t.Fatalf("fillPlaceholders: %v", err)
	}
	files := docxFiles(t, out)
	want := []string{
		"Nomor: 015/PSP/X/2026 (resmi)",
		"Perihal: ###Perihal### & ###*Poin1*###", // bukan penanda sistem: dibiarkan
		"###*KEPADA*###",
		"###*No", "Surat*###",
		"Tanpa penanda",
	}
	if got := paragraphTexts(t, files["word/document.xml"]); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("document.xml:\n got %q\nwant %q", got, want)
	}
	if !strings.Contains(files["word/document.xml"], `<w:rPr><w:b/></w:rPr>`) {
		t.Error("format run ikut hilang")
	}
	if got := paragraphTexts(t, files["word/header1.xml"]); len(got) != 1 || got[0] != "Nomor 015/PSP/X/2026" {
		t.Errorf("header1.xml = %q", got)
	}
	if files["word/styles.xml"] != "<w:t>###*NoSurat*###</w:t>" || len(files) != 4 {
		t.Errorf("bagian lain tidak boleh berubah, dan tanpa penanda QR tidak ada gambar yang ditambahkan: %d file", len(files))
	}

	if _, err := fillPlaceholders([]byte("bukan zip"), values, nil); err == nil {
		t.Error("naskah rusak harus error")
	}
}

func TestFillPlaceholdersQR(t *testing.T) {
	doc := docWith(`<w:p><w:r><w:rPr><w:b/></w:rPr><w:t>Tanda tangan: ###*QR</w:t><w:t>Code*### selesai</w:t></w:r></w:p>`)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range map[string]string{
		"[Content_Types].xml":           `<Types><Default Extension="xml" ContentType="application/xml"/></Types>`,
		"word/document.xml":             doc,
		"word/media/qr-persetujuan.png": "gambar lain dengan nama sama",
		"word/_rels/document.xml.rels":  `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="x" Target="styles.xml"/></Relationships>`,
	} {
		w, _ := zw.Create(name)
		w.Write([]byte(content))
	}
	zw.Close()

	out, err := fillPlaceholders(buf.Bytes(), nil, []byte("\x89PNG-qr"))
	if err != nil {
		t.Fatal(err)
	}
	files := docxFiles(t, out)
	if got := paragraphTexts(t, files["word/document.xml"]); len(got) != 1 || got[0] != "Tanda tangan:  selesai" {
		t.Errorf("teks di sekitar QR = %q", got)
	}
	d := files["word/document.xml"]
	if !strings.Contains(d, `<wp:extent cx="1080000" cy="1080000"/>`) || !strings.Contains(d, `r:embed="rIdQrPersetujuan"`) || strings.Contains(d, "###") {
		t.Errorf("gambar QR 3x3 cm tidak masuk: %s", d)
	}
	if files["word/media/qr-persetujuan-2.png"] != "\x89PNG-qr" || files["word/media/qr-persetujuan.png"] != "gambar lain dengan nama sama" {
		t.Error("file gambar QR salah nama atau menimpa gambar lain")
	}
	rels := files["word/_rels/document.xml.rels"]
	if !strings.Contains(rels, `Id="rId1"`) || !strings.Contains(rels, `Id="rIdQrPersetujuan" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/qr-persetujuan-2.png"`) {
		t.Errorf("relasi gambar salah: %s", rels)
	}
	if !strings.Contains(files["[Content_Types].xml"], `<Default Extension="png" ContentType="image/png"/>`) {
		t.Errorf("tipe png tidak didaftarkan: %s", files["[Content_Types].xml"])
	}

	// Penanda QR di beberapa tempat (isi dan footer): semuanya diganti QR
	var many bytes.Buffer
	zw = zip.NewWriter(&many)
	for name, content := range map[string]string{
		"[Content_Types].xml": `<Types><Default Extension="xml" ContentType="application/xml"/></Types>`,
		"word/document.xml":   docWith(`<w:p><w:r><w:t>###*QRCode*###</w:t></w:r></w:p><w:p><w:r><w:t>###qrcode###</w:t></w:r></w:p>`),
		"word/footer1.xml":    docWith(`<w:p><w:r><w:t>Paraf ###*QRCode*###</w:t></w:r></w:p>`),
	} {
		w, _ := zw.Create(name)
		w.Write([]byte(content))
	}
	zw.Close()
	out, err = fillPlaceholders(many.Bytes(), nil, []byte("png"))
	if err != nil {
		t.Fatal(err)
	}
	files = docxFiles(t, out)
	if strings.Count(files["word/document.xml"], "<w:drawing>") != 2 || strings.Count(files["word/footer1.xml"], "<w:drawing>") != 1 ||
		files["word/_rels/footer1.xml.rels"] == "" {
		t.Error("setiap penanda QR (termasuk di footer) harus diganti QR")
	}

	// Tanpa penanda QR: tidak ada QR yang ditambahkan (QR hanya di posisi penanda)
	out, _ = fillPlaceholders(makeDocx(docWith(`<w:p><w:r><w:t>Tanpa QR</w:t></w:r></w:p>`)), nil, []byte("png"))
	files = docxFiles(t, out)
	if strings.Contains(files["word/document.xml"], "<w:drawing>") || files["word/media/qr-persetujuan.png"] != "" {
		t.Error("tanpa penanda QR tidak boleh ada QR")
	}
}

func TestHasQRPlaceholder(t *testing.T) {
	cases := []struct {
		name  string
		parts map[string]string
		want  bool
	}{
		{"isi surat", map[string]string{"word/document.xml": docWith(`<w:p><w:r><w:t>TTD ###*QRCode*###</w:t></w:r></w:p>`)}, true},
		{"terpecah beberapa run", map[string]string{"word/document.xml": docWith(`<w:p><w:r><w:t>###*QR</w:t></w:r><w:r><w:rPr><w:b/></w:rPr><w:t>Code*###</w:t></w:r></w:p>`)}, true},
		{"format lama, huruf kecil", map[string]string{"word/document.xml": docWith(`<w:p><w:r><w:t>###qrcode###</w:t></w:r></w:p>`)}, true},
		{"di footer", map[string]string{"word/document.xml": docWith(`<w:p/>`), "word/footer2.xml": docWith(`<w:p><w:r><w:t>###*QRCode*###</w:t></w:r></w:p>`)}, true},
		{"di tabel", map[string]string{"word/document.xml": docWith(`<w:tbl><w:tr><w:tc><w:p><w:r><w:t>###*QRCode*###</w:t></w:r></w:p></w:tc></w:tr></w:tbl>`)}, true},
		{"tidak ada", map[string]string{"word/document.xml": docWith(`<w:p><w:r><w:t>###*NoSurat*###</w:t></w:r></w:p>`)}, false},
		{"terpotong dua paragraf", map[string]string{"word/document.xml": docWith(`<w:p><w:r><w:t>###*QR</w:t></w:r></w:p><w:p><w:r><w:t>Code*###</w:t></w:r></w:p>`)}, false},
		{"hanya di bagian non-teks", map[string]string{"word/document.xml": docWith(`<w:p/>`), "word/styles.xml": "<w:t>###*QRCode*###</w:t>"}, false},
	}
	for _, tc := range cases {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		for name, content := range tc.parts {
			w, _ := zw.Create(name)
			w.Write([]byte(content))
		}
		zw.Close()
		if got, err := hasQRPlaceholder(buf.Bytes()); err != nil || got != tc.want {
			t.Errorf("%s: %v (err %v), want %v", tc.name, got, err, tc.want)
		}
	}
	if _, err := hasQRPlaceholder([]byte("bukan zip")); err == nil {
		t.Error("naskah rusak harus error")
	}
}

func TestLetterPlaceholders(t *testing.T) {
	d := &LetterDetail{
		LetterNumber:    ptr("001/PSP/XI/2026"),
		Title:           ptr("Undangan Rapat"),
		Confidentiality: &LookupRef{Name: "Terbatas"},
		SenderUnit:      UnitRef{Name: "Departemen Teknologi Informasi PSP"},
		Approval: &Approval{ApprovedAt: time.Date(2026, 11, 5, 8, 30, 0, 0, time.UTC),
			ApprovedBy: PersonRef{Name: "Agus Setiawan", PositionName: "VP Teknologi Informasi"}, EmployeeNumber: "1995021"},
	}
	// Hanya NoSurat, TglSurat, dan Sifat (QR diisi terpisah); perihal, pengirim, dan penandatangan tidak
	want := map[string]string{phNoSurat: "001/PSP/XI/2026", phTglSurat: "5 November 2026", phSifat: "Terbatas"}
	got := letterPlaceholders(d)
	if len(got) != len(want) {
		t.Errorf("penanda = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if len(letterPlaceholders(&LetterDetail{})) != 0 {
		t.Error("data kosong tidak boleh menghasilkan penanda")
	}
}

// TestFillTemplateFiles mengisi kelima template bawaan: keempat penanda sistem terisi, QR masuk di
// posisinya, dan penanda lain (diisi drafter) tetap ada.
func TestFillTemplateFiles(t *testing.T) {
	system := regexp.MustCompile(`(?i)###\*?(NoSurat|TglSurat|Sifat|QRCode)\*?###`)
	values := map[string]string{phNoSurat: "001/PSP/XI/2026", phTglSurat: "5 November 2026", phSifat: "Biasa"}
	for _, name := range []string{"surat-dinas-standar", "nota-dinas", "undangan-rapat", "surat-tugas", "dokumen-kosong"} {
		src, err := os.ReadFile("../../database/seeds/templates/" + name + ".docx")
		if err != nil {
			t.Fatal(err)
		}
		out, err := fillPlaceholders(src, values, []byte("\x89PNG"))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if d := docxFiles(t, out)["word/document.xml"]; strings.Count(d, "<w:drawing><wp:inline") != 1 {
			t.Errorf("%s: QR harus satu, di posisi penanda", name)
		}
		if ok, _ := hasQRPlaceholder(src); !ok {
			t.Errorf("%s: template bawaan harus memuat penanda QR", name)
		}
		text := strings.Join(paragraphTexts(t, docxFiles(t, out)["word/document.xml"]), "\n")
		if m := system.FindString(text); m != "" {
			t.Errorf("%s: penanda sistem tersisa %s", name, m)
		}
		if !strings.Contains(text, "001/PSP/XI/2026") {
			t.Errorf("%s: nomor surat tidak terisi", name)
		}
		srcText := strings.Join(paragraphTexts(t, docxFiles(t, src)["word/document.xml"]), " ")
		if strings.Contains(srcText, "TglSurat") != strings.Contains(text, "5 November 2026") {
			t.Errorf("%s: tanggal surat tidak terisi", name) // Surat Tugas memang tanpa tanggal
		}
		if !strings.Contains(text, "###*NamaPenandatangan*###") {
			t.Errorf("%s: penanda yang diisi drafter ikut terisi/terhapus", name)
		}
	}
}

func TestFilledDocumentQRAtPlaceholder(t *testing.T) {
	svc, store, _, _, _ := sentLetter(false)
	files := svc.repos.Files.(*fakeFiles)
	files.objects = map[string][]byte{"draf-qr": makeDocx(docWith(`<w:p><w:r><w:t>###*QRCode*###</w:t></w:r></w:p>`))}
	store.versions = []DocumentVersion{{VersionNumber: 1, Type: VersionDraft, FileName: "Naskah.docx", ObjectKey: "draf-qr"}}

	f, err := svc.FilledDocument(context.Background(), andiUser, testLetterID)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(f.Body)
	d := docxFiles(t, got)["word/document.xml"]
	if strings.Count(d, "<w:drawing>") != 1 || strings.Contains(d, "###") || strings.Contains(d, `<w:jc w:val="right"/>`) {
		t.Errorf("QR harus menggantikan penanda, bukan ditambahkan di akhir:\n%s", d)
	}
}
