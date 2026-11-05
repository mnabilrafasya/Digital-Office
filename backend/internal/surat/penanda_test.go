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

	"office.backend/internal/organisasi"
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

	values := map[string]string{phNoSurat: "015/PSP/X/2026", phPerihal: "Rapat <Q4> & evaluasi", phKepada: "VP Akuntansi\nSeluruh VP"}
	out, err := fillPlaceholders(buf.Bytes(), values, []byte("png"), nil)
	if err != nil {
		t.Fatalf("fillPlaceholders: %v", err)
	}
	files := docxFiles(t, out)
	want := []string{
		"Nomor: 015/PSP/X/2026 (resmi)",
		"Perihal: Rapat <Q4> & evaluasi & ###*Poin1*###",
		"VP Akuntansi|Seluruh VP",
		"###*No", "Surat*###",
		"Tanpa penanda",
		"", // tanpa penanda QR: paragraf QR ditambahkan di akhir
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
	if files["word/styles.xml"] != "<w:t>###*NoSurat*###</w:t>" || len(files) != 6 {
		t.Errorf("bagian lain tidak boleh berubah (yang bertambah hanya gambar QR dan relasinya): %d file", len(files))
	}

	if _, err := fillPlaceholders([]byte("bukan zip"), values, nil, nil); err == nil {
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

	out, err := fillPlaceholders(buf.Bytes(), nil, []byte("\x89PNG-qr"), nil)
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

	// Tanpa penanda QR: QR ditambahkan rata kanan di akhir isi, sebelum pengaturan halaman
	sect := `<w:sectPr><w:pgSz w:w="11906" w:h="16838"/></w:sectPr>`
	out, _ = fillPlaceholders(makeDocx(docWith(`<w:p><w:pPr>`+sect+`</w:pPr><w:r><w:t>Bab 1</w:t></w:r></w:p><w:p><w:r><w:t>Tanpa QR</w:t></w:r></w:p>`+sect)), nil, []byte("png"), nil)
	files = docxFiles(t, out)
	d = files["word/document.xml"]
	qrAt, lastSect := strings.Index(d, `<w:jc w:val="right"/></w:pPr><w:r><w:drawing>`), strings.LastIndex(d, "<w:sectPr>")
	if qrAt < strings.Index(d, "Tanpa QR") || qrAt > lastSect || strings.Count(d, "<w:drawing>") != 1 || files["word/media/qr-persetujuan.png"] != "png" {
		t.Errorf("QR tidak ditambahkan di akhir isi:\n%s", d)
	}
	paragraphTexts(t, d) // XML tetap valid
}

func TestLetterPlaceholders(t *testing.T) {
	d := &LetterDetail{
		LetterNumber:    ptr("001/PSP/XI/2026"),
		Title:           ptr("Undangan Rapat"),
		Confidentiality: &LookupRef{Name: "Terbatas"},
		SenderUnit:      UnitRef{Name: "Departemen Teknologi Informasi PSP"},
		Attachments:     []Attachment{{}, {}},
		Recipients: LetterRecipients{To: []LetterRecipient{
			{Unit: &UnitRef{Name: "Departemen Akuntansi"}, ResolvedHead: &PersonRef{Name: "Joko", PositionName: "VP Akuntansi"}},
			{Alias: &organisasi.Alias{Label: "Seluruh VP"}},
			{Unit: &UnitRef{Name: "Departemen Anggaran"}}, // pimpinan tidak tercatat: nama unit
		}},
		Approval: &Approval{ApprovedAt: time.Date(2026, 11, 5, 8, 30, 0, 0, time.UTC),
			ApprovedBy: PersonRef{Name: "Agus Setiawan", PositionName: "VP Teknologi Informasi"}, EmployeeNumber: "1995021"},
	}
	want := map[string]string{
		phNoSurat: "001/PSP/XI/2026", phTglSurat: "5 November 2026", phPerihal: "Undangan Rapat", phSifat: "Terbatas",
		phLampiran: "2 (dua) berkas", phKepada: "VP Akuntansi\nPara VP\nDepartemen Anggaran", phTembusan: "-",
		phDari: "Departemen Teknologi Informasi PSP", phJabatanTTD: "VP Teknologi Informasi", phNamaTTD: "Agus Setiawan",
		phNoBadge: "1995021",
	}
	got := letterPlaceholders(d, officialCopy{})
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("jumlah penanda = %d, want %d", len(got), len(want))
	}
	for n, w := range map[int]string{1: "satu", 11: "sebelas", 12: "dua belas", 20: "dua puluh", 21: "dua puluh satu"} {
		if terbilang(n) != w {
			t.Errorf("terbilang(%d) = %q, want %q", n, terbilang(n), w)
		}
	}
	if attachmentCountText(0) != "-" {
		t.Error("tanpa lampiran harus -")
	}

	// Salinan penerima: "Kepada" hanya jabatan pemilik salinan, penanda lain sama dengan salinan induk
	copyVals := letterPlaceholders(d, officialCopy{OwnerID: 50, PositionName: "VP Operasi Pabrik I"})
	for k, v := range want {
		if k == phKepada {
			v = "VP Operasi Pabrik I"
		}
		if copyVals[k] != v {
			t.Errorf("salinan: %s = %q, want %q", k, copyVals[k], v)
		}
	}
}

// TestFillTemplateFiles mengisi kelima template bawaan: semua penanda sistem terisi, QR masuk di
// posisinya, dan penanda isi (ditulis drafter) tetap ada.
func TestFillTemplateFiles(t *testing.T) {
	system := regexp.MustCompile(`(?i)###\*?(NoSurat|TglSurat|Perihal|Sifat|Lampiran|Kepada|Tembusan|Dari|JabatanPenandatangan|NamaPenandatangan|NoBadge|QRCode)\*?###`)
	values := map[string]string{phNoSurat: "001/PSP/XI/2026", phTglSurat: "5 November 2026", phPerihal: "Uji", phSifat: "Biasa",
		phLampiran: "-", phKepada: "VP Akuntansi", phTembusan: "-", phDari: "Departemen TI", phJabatanTTD: "VP TI",
		phNamaTTD: "Agus", phNoBadge: "1995021"}
	for _, name := range []string{"surat-dinas-standar", "nota-dinas", "undangan-rapat", "surat-tugas", "dokumen-kosong"} {
		src, err := os.ReadFile("../../database/seeds/templates/" + name + ".docx")
		if err != nil {
			t.Fatal(err)
		}
		out, err := fillPlaceholders(src, values, []byte("\x89PNG"), nil)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if d := docxFiles(t, out)["word/document.xml"]; strings.Count(d, "<w:drawing><wp:inline") != 1 || strings.Contains(d, `<w:jc w:val="right"/></w:pPr><w:r><w:drawing>`) {
			t.Errorf("%s: QR harus satu, di posisi penanda (bukan ditambahkan di akhir)", name)
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
		if !strings.Contains(text, "###*") {
			t.Errorf("%s: penanda isi ikut terhapus", name)
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
