package surat

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Penanda naskah ditulis di template sebagai ###*Nama*### (format lama ###Nama### juga dikenali).
// Saat surat terbit, hanya 4 penanda sistem yang diisi otomatis: NoSurat, TglSurat, Sifat, dan
// QRCode. Penanda lain (Kepada, Perihal, Poin1, dan seterusnya) dibiarkan apa adanya karena ditulis
// sendiri oleh drafter. Penanda QRCode wajib ada di naskah (checkQRPlaceholder); QR hanya dipasang di
// posisi penandanya, di bagian naskah mana pun (isi, tabel, header, footer).
// Word sering memecah satu penanda ke beberapa potongan teks (<w:t>), misalnya karena riwayat
// ketikan atau format, jadi pencarian dilakukan pada gabungan teks satu paragraf.

// Nama penanda sistem (huruf kecil; pencocokan tidak membedakan huruf besar).
const (
	phNoSurat     = "nosurat"
	phTglSurat    = "tglsurat"
	phSifat       = "sifat"
	phQRCode      = "qrcode"
	qrSizeEMU     = 1080000 // 3 cm (1 cm = 360000 EMU)
	qrMediaName   = "qr-persetujuan"
	qrRelID       = "rIdQrPersetujuan"
	qrDocPrIDBase = 90001
)

// placeholderPattern menemukan ###*Nama*### atau ###Nama###.
var placeholderPattern = regexp.MustCompile(`###\*?([A-Za-z][A-Za-z0-9]*)\*?###`)

// textNodePattern menemukan teks run (<w:t>) dan akhir paragraf (</w:p>). <w:tab/>, <w:tbl>,
// <w:tc> dan sejenisnya tidak cocok karena setelah "<w:t" wajib spasi atau ">"; <w:t .../>
// (kosong) juga tidak cocok karena atributnya tidak boleh diakhiri "/".
var textNodePattern = regexp.MustCompile(`(?s)<w:t(\s[^>]*[^/>])?>(.*?)</w:t>|</w:p>`)

// pngContentType mengecek apakah [Content_Types].xml sudah mengenal ekstensi png.
var pngContentType = regexp.MustCompile(`(?i)Extension="png"`)

// bulanIndonesia dipakai untuk tanggal surat, contoh "5 November 2026".
var bulanIndonesia = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli",
	"Agustus", "September", "Oktober", "November", "Desember"}

// letterPlaceholders menyusun isi penanda sistem (selain QR) dari detail surat yang sudah disetujui.
func letterPlaceholders(d *LetterDetail) map[string]string {
	v := map[string]string{}
	if d.LetterNumber != nil {
		v[phNoSurat] = *d.LetterNumber
	}
	if d.Confidentiality != nil {
		v[phSifat] = d.Confidentiality.Name
	}
	if d.Approval != nil {
		v[phTglSurat] = tanggalIndonesia(d.Approval.ApprovedAt)
	}
	return v
}

// tanggalIndonesia menulis tanggal tanpa jam, contoh "5 November 2026".
func tanggalIndonesia(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), bulanIndonesia[t.Month()-1], t.Year())
}

// fillPlaceholders mengisi penanda sistem di naskah .docx: teks dari values (kunci huruf kecil)
// dan gambar QR di setiap posisi ###*QRCode*### (qrPNG nil = penanda QR dikosongkan, untuk surat
// eksternal). File lain di dalam .docx tidak diubah. Watermark
// tidak ditanam di sini, tetapi ditempel ke PDF resmi (watermark.go).
func fillPlaceholders(docx []byte, values map[string]string, qrPNG []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		return nil, fmt.Errorf("membuka naskah .docx: %w", err)
	}
	files := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("membaca naskah .docx: %w", err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("membaca naskah .docx: %w", err)
		}
		files[f.Name] = data
	}

	qrMedia := uniqueMediaFile(files, qrMediaName)
	docPrID := qrDocPrIDBase
	var qrParts []string // bagian naskah yang memuat QR (butuh relasi ke gambar)
	var newQR func() string
	if qrPNG != nil { // tanpa gambar QR (surat eksternal): penanda QR dikosongkan
		newQR = func() string {
			docPrID++
			return qrDrawingXML(docPrID)
		}
	}
	for _, f := range zr.File {
		if !docxTextPart.MatchString(f.Name) {
			continue
		}
		filled, placed := fillPart(files[f.Name], values, newQR)
		files[f.Name] = filled
		if placed {
			qrParts = append(qrParts, f.Name)
		}
	}

	added := map[string]bool{}
	if len(qrParts) > 0 {
		files[qrMedia] = qrPNG
		added[qrMedia] = true
		files["[Content_Types].xml"] = ensurePNGContentType(files["[Content_Types].xml"])
		for _, part := range qrParts {
			rels := relsPath(part)
			if _, ok := files[rels]; !ok {
				added[rels] = true
			}
			files[rels] = addRelationship(files[rels], qrRelID, imageRelType, strings.TrimPrefix(qrMedia, "word/"))
		}
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	write := func(name string, method uint16, modified time.Time) error {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: method, Modified: modified})
		if err != nil {
			return err
		}
		_, err = w.Write(files[name])
		return err
	}
	for _, f := range zr.File {
		if err := write(f.Name, f.Method, f.Modified); err != nil {
			return nil, fmt.Errorf("menulis naskah .docx: %w", err)
		}
	}
	extra := make([]string, 0, len(added))
	for name := range added {
		extra = append(extra, name)
	}
	sort.Strings(extra)
	for _, name := range extra {
		if err := write(name, zip.Deflate, time.Now()); err != nil {
			return nil, fmt.Errorf("menulis naskah .docx: %w", err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("menulis naskah .docx: %w", err)
	}
	return buf.Bytes(), nil
}

// textNode adalah satu <w:t> di XML: posisi isinya dan teksnya (sudah di-decode).
type textNode struct {
	openStart, contentStart, contentEnd, closeEnd int
	text                                          string
	edits                                         []nodeEdit
}

// nodeEdit mengganti text[start:end] dengan XML mentah (bukan teks).
type nodeEdit struct {
	start, end int
	xml        string
}

// fillPart mengisi penanda di satu bagian XML naskah. newQR menghasilkan XML gambar QR untuk
// setiap kemunculan penanda QR (nil = penanda QR dikosongkan). placed=true bila ada QR yang dipasang.
func fillPart(data []byte, values map[string]string, newQR func() string) (out []byte, placed bool) {
	src := string(data)
	var (
		nodes  []*textNode
		joined strings.Builder
		starts []int // posisi awal teks node di joined
	)
	for _, m := range textNodePattern.FindAllStringSubmatchIndex(src, -1) {
		if m[2] < 0 && m[4] < 0 { // </w:p>: batas paragraf, penanda tidak boleh melintasinya
			joined.WriteByte('\n')
			continue
		}
		n := &textNode{openStart: m[0], contentStart: m[4], contentEnd: m[5], closeEnd: m[1],
			text: html.UnescapeString(src[m[4]:m[5]])}
		starts = append(starts, joined.Len())
		joined.WriteString(n.text)
		nodes = append(nodes, n)
	}
	if len(nodes) == 0 {
		return data, false
	}

	text := joined.String()
	nodeAt := func(pos int) int { return sort.Search(len(starts), func(i int) bool { return starts[i] > pos }) - 1 }
	for _, m := range placeholderPattern.FindAllStringSubmatchIndex(text, -1) {
		name := strings.ToLower(text[m[2]:m[3]])
		var repl string
		if name == phQRCode {
			if newQR == nil {
				repl = "" // tanpa QR: penanda dikosongkan supaya tidak tercetak
			} else {
				repl = `</w:t>` + newQR() + `<w:t xml:space="preserve">`
				placed = true
			}
		} else {
			value, ok := values[name]
			if !ok {
				continue // penanda isi (ditulis drafter) dibiarkan
			}
			repl = textXML(value)
		}
		first, last := nodeAt(m[0]), nodeAt(m[1]-1)
		for i := first; i <= last; i++ {
			n := nodes[i]
			start, end := 0, len(n.text)
			if i == first {
				start = m[0] - starts[i]
			}
			if i == last {
				end = m[1] - starts[i]
			}
			e := nodeEdit{start: start, end: end}
			if i == first {
				e.xml = repl
			}
			n.edits = append(n.edits, e)
		}
	}

	var b strings.Builder
	prev, changed := 0, false
	for _, n := range nodes {
		if len(n.edits) == 0 {
			continue
		}
		changed = true
		b.WriteString(src[prev:n.openStart])
		b.WriteString(`<w:t xml:space="preserve">`)
		pos := 0
		for _, e := range n.edits {
			b.WriteString(escapeXML(n.text[pos:e.start]))
			b.WriteString(e.xml)
			pos = e.end
		}
		b.WriteString(escapeXML(n.text[pos:]))
		b.WriteString(`</w:t>`)
		prev = n.closeEnd
	}
	if !changed {
		return data, placed
	}
	b.WriteString(src[prev:])
	return []byte(b.String()), placed
}

// hasQRPlaceholder mengecek apakah naskah .docx memuat penanda ###*QRCode*### di bagian teks mana
// pun (isi, tabel, header, footer, catatan kaki). Penanda harus utuh dalam satu paragraf.
func hasQRPlaceholder(docx []byte) (bool, error) {
	zr, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		return false, fmt.Errorf("membuka naskah .docx: %w", err)
	}
	for _, f := range zr.File {
		if !docxTextPart.MatchString(f.Name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return false, fmt.Errorf("membaca naskah .docx: %w", err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return false, fmt.Errorf("membaca naskah .docx: %w", err)
		}
		for _, m := range placeholderPattern.FindAllStringSubmatch(partText(string(data)), -1) {
			if strings.EqualFold(m[1], phQRCode) {
				return true, nil
			}
		}
	}
	return false, nil
}

// partText menggabungkan teks satu bagian XML naskah per paragraf (dipisah baris baru), sama
// dengan cara fillPart mencari penanda.
func partText(src string) string {
	var b strings.Builder
	for _, m := range textNodePattern.FindAllStringSubmatchIndex(src, -1) {
		if m[2] < 0 && m[4] < 0 { // </w:p>
			b.WriteByte('\n')
			continue
		}
		b.WriteString(html.UnescapeString(src[m[4]:m[5]]))
	}
	return b.String()
}

// textXML menulis nilai teks untuk diletakkan di dalam <w:t>; baris baru menjadi <w:br/>.
func textXML(value string) string {
	lines := strings.Split(value, "\n")
	for i, l := range lines {
		lines[i] = escapeXML(l)
	}
	return strings.Join(lines, `</w:t><w:br/><w:t xml:space="preserve">`)
}

func escapeXML(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s)) // strings.Builder tidak pernah gagal menulis
	return b.String()
}

// qrDrawingXML adalah gambar QR inline 3x3 cm. Namespace ditulis di elemennya sendiri supaya
// tetap valid walaupun root dokumen tidak mendeklarasikannya.
func qrDrawingXML(docPrID int) string {
	size := strconv.Itoa(qrSizeEMU)
	id := strconv.Itoa(docPrID)
	return `<w:drawing><wp:inline xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" distT="0" distB="0" distL="0" distR="0">` +
		`<wp:extent cx="` + size + `" cy="` + size + `"/><wp:docPr id="` + id + `" name="QR Persetujuan ` + id + `"/>` +
		`<a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:nvPicPr><pic:cNvPr id="` + id + `" name="` + qrMediaName + `.png"/><pic:cNvPicPr/></pic:nvPicPr>` +
		`<pic:blipFill><a:blip xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" r:embed="` + qrRelID + `"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>` +
		`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="` + size + `" cy="` + size + `"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr>` +
		`</pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing>`
}

// uniqueMediaFile memilih nama file gambar (word/media/<base>.png) yang belum dipakai di dalam .docx.
func uniqueMediaFile(files map[string][]byte, base string) string {
	name := "word/media/" + base + ".png"
	for i := 2; files[name] != nil; i++ {
		name = fmt.Sprintf("word/media/%s-%d.png", base, i)
	}
	return name
}

// relsPath adalah file relasi milik sebuah bagian naskah, contoh word/_rels/header1.xml.rels.
func relsPath(part string) string {
	return path.Join(path.Dir(part), "_rels", path.Base(part)+".rels")
}

// ensurePNGContentType memastikan [Content_Types].xml mengenal ekstensi png.
func ensurePNGContentType(ct []byte) []byte {
	s := string(ct)
	if pngContentType.MatchString(s) {
		return ct
	}
	return addToTypes(ct, `<Default Extension="png" ContentType="image/png"/>`)
}

// addToTypes menambah satu elemen ke [Content_Types].xml, juga bila ditulis ringkas (<Types/>).
func addToTypes(ct []byte, elem string) []byte {
	s := string(ct)
	if i := strings.LastIndex(s, "</Types>"); i >= 0 {
		return []byte(s[:i] + elem + s[i:])
	}
	if i := strings.LastIndex(s, "/>"); i >= 0 && strings.Contains(s, "<Types") {
		return []byte(s[:i] + ">" + elem + "</Types>" + s[i+2:])
	}
	return ct
}

// imageRelType adalah jenis relasi gambar di file .rels.
const imageRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"

// addRelationship menambah relasi id ke file .rels sebuah bagian naskah (dibuat bila belum ada).
// target relatif terhadap folder word/ (semua bagian naskah ada di sana).
func addRelationship(rels []byte, id, relType, target string) []byte {
	rel := `<Relationship Id="` + id + `" Type="` + relType + `" Target="` + target + `"/>`
	if len(rels) == 0 {
		return []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + rel + `</Relationships>`)
	}
	s := string(rels)
	if strings.Contains(s, `Id="`+id+`"`) {
		return rels
	}
	return []byte(strings.Replace(s, "</Relationships>", rel+"</Relationships>", 1))
}
