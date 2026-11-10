package surat

import (
	"bytes"
	"fmt"
	"image/png"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"office.backend/internal/platform/watermark"
)

// Watermark naskah resmi: teks watermark perusahaan (companies.watermark_text, diatur admin
// perusahaan) diagonal, abu-abu transparan, di belakang isi surat pada setiap halaman. Disisipkan ke
// naskah terisi (resmi.go) sebagai gambar mengambang di header, jadi tampil sama di editor DevExpress
// dan ikut ke PDF resmi tanpa perubahan front-end. Hanya surat terbit yang punya naskah terisi, jadi
// draf tidak pernah ber-watermark. Gambarnya dibuat paket platform/watermark.

// watermarkConfidentiality adalah sifat surat (kode lookup CONFIDENTIALITY) yang diberi watermark.
var watermarkConfidentiality = map[string]bool{"BIASA": true, "TERBATAS": true, "RAHASIA": true}

// Ukuran dan letak watermark di naskah.
const (
	watermarkWidthEMU   = 5760000 // lebar gambar di halaman: 16 cm
	watermarkRelID      = "rIdWatermark"
	watermarkHeaderRel  = "rIdWatermarkHeader"
	watermarkHeaderPart = "word/header-watermark.xml"
	watermarkDocPrBase  = 90500
	emuPerTwip          = 635
)

// Ukuran halaman A4 dalam twip, dipakai bila naskah tidak menyebut ukurannya.
const (
	a4WidthTwip  = 11906
	a4HeightTwip = 16838
)

// watermarkFor mengembalikan gambar watermark untuk surat d, atau nil bila perusahaannya tidak
// memakai watermark atau sifat suratnya tidak diberi watermark.
func watermarkFor(d *LetterDetail) ([]byte, error) {
	if d.WatermarkText == nil || d.Confidentiality == nil || !watermarkConfidentiality[d.Confidentiality.Code] {
		return nil, nil
	}
	text := watermark.Normalize(*d.WatermarkText)
	if text == "" {
		return nil, nil
	}
	return watermark.PNG(text)
}

// headerPartPattern adalah bagian header naskah .docx.
var headerPartPattern = regexp.MustCompile(`^word/header\d*\.xml$`)

// pageSizePattern membaca ukuran halaman (twip) dari <w:pgSz>.
var pageSizePattern = regexp.MustCompile(`<w:pgSz\b[^>]*>`)

// addWatermark menyisipkan gambar watermark ke setiap header naskah (dibuatkan header bila belum
// ada), sebagai gambar di belakang teks yang berada di tengah halaman. Mengembalikan nama file
// baru di dalam .docx.
func addWatermark(files map[string][]byte, pngData []byte) ([]string, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("membaca gambar watermark: %w", err)
	}
	pageW, pageH := pageSizeTwip(files[mainDocumentPart])
	width := int64(watermarkWidthEMU)
	height := width * int64(cfg.Height) / int64(cfg.Width)
	offX := (int64(pageW)*emuPerTwip - width) / 2
	offY := (int64(pageH)*emuPerTwip - height) / 2

	var added []string
	media := uniqueMediaFile(files, "watermark")
	files[media] = pngData
	added = append(added, media)
	files["[Content_Types].xml"] = ensurePNGContentType(files["[Content_Types].xml"])

	var headers []string
	for name := range files {
		if headerPartPattern.MatchString(name) {
			headers = append(headers, name)
		}
	}
	sort.Strings(headers)
	if len(headers) == 0 {
		created, err := createWatermarkHeader(files)
		if err != nil {
			return nil, err
		}
		headers = []string{watermarkHeaderPart}
		added = append(added, created...)
	}

	for i, part := range headers {
		drawing := watermarkDrawingXML(watermarkDocPrBase+i, width, height, offX, offY)
		doc, ok := appendHeaderParagraph(files[part], drawing)
		if !ok {
			return nil, fmt.Errorf("header %s tidak valid", part)
		}
		files[part] = doc
		rels := relsPath(part)
		if _, exists := files[rels]; !exists {
			added = append(added, rels)
		}
		files[rels] = addRelationship(files[rels], watermarkRelID, imageRelType, strings.TrimPrefix(media, "word/"))
	}
	return added, nil
}

// pageSizeTwip membaca ukuran halaman terakhir naskah (A4 bila tidak tertulis).
func pageSizeTwip(doc []byte) (w, h int) {
	w, h = a4WidthTwip, a4HeightTwip
	tags := pageSizePattern.FindAll(doc, -1)
	if len(tags) == 0 {
		return w, h
	}
	last := string(tags[len(tags)-1])
	if v := attrInt(last, "w:w"); v > 0 {
		w = v
	}
	if v := attrInt(last, "w:h"); v > 0 {
		h = v
	}
	return w, h
}

// attrInt membaca nilai atribut angka dari sebuah tag XML (0 bila tidak ada).
func attrInt(tag, name string) int {
	m := regexp.MustCompile(`\s` + regexp.QuoteMeta(name) + `="(\d+)"`).FindStringSubmatch(tag)
	if m == nil {
		return 0
	}
	v, _ := strconv.Atoi(m[1])
	return v
}

// appendHeaderParagraph menambah paragraf berisi gambar watermark di akhir header. Tinggi barisnya
// dibuat 1 pt supaya isi header (kop) dan isi surat tidak bergeser.
func appendHeaderParagraph(hdr []byte, drawing string) ([]byte, bool) {
	s := string(hdr)
	end := strings.LastIndex(s, "</w:hdr>")
	if end < 0 {
		return hdr, false
	}
	p := `<w:p><w:pPr><w:spacing w:before="0" w:after="0" w:line="20" w:lineRule="exact"/></w:pPr><w:r>` +
		drawing + `</w:r></w:p>`
	return []byte(s[:end] + p + s[end:]), true
}

// createWatermarkHeader membuat header baru untuk naskah yang belum punya header, lalu memasangnya
// sebagai header default di setiap bagian (section) naskah yang belum punya header default.
// Mengembalikan nama file baru di dalam .docx.
func createWatermarkHeader(files map[string][]byte) ([]string, error) {
	doc := string(files[mainDocumentPart])
	if doc == "" {
		return nil, fmt.Errorf("naskah tidak punya %s", mainDocumentPart)
	}
	created := []string{watermarkHeaderPart}
	files[watermarkHeaderPart] = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
		`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"></w:hdr>`)

	ref := `<w:headerReference xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" w:type="default" r:id="` + watermarkHeaderRel + `"/>`
	sect := regexp.MustCompile(`(?s)<w:sectPr\b[^>]*?(/>|>(.*?)</w:sectPr>)`)
	doc = sect.ReplaceAllStringFunc(doc, func(tag string) string {
		if strings.Contains(tag, `w:type="default"`) && strings.Contains(tag, "w:headerReference") {
			return tag
		}
		if strings.HasSuffix(tag, "/>") && !strings.Contains(tag, "</w:sectPr>") {
			return strings.TrimSuffix(tag, "/>") + ">" + ref + "</w:sectPr>"
		}
		open := strings.Index(tag, ">") + 1
		return tag[:open] + ref + tag[open:]
	})
	if !strings.Contains(doc, watermarkHeaderRel) {
		// Naskah tanpa pengaturan halaman: tambahkan di akhir body
		end := strings.LastIndex(doc, "</w:body>")
		if end < 0 {
			return nil, fmt.Errorf("naskah tidak punya </w:body>")
		}
		doc = doc[:end] + `<w:sectPr>` + ref + `</w:sectPr>` + doc[end:]
	}
	files[mainDocumentPart] = []byte(doc)

	rels := relsPath(mainDocumentPart)
	if _, exists := files[rels]; !exists {
		created = append(created, rels)
	}
	files[rels] = addRelationship(files[rels], watermarkHeaderRel,
		"http://schemas.openxmlformats.org/officeDocument/2006/relationships/header", strings.TrimPrefix(watermarkHeaderPart, "word/"))
	files["[Content_Types].xml"] = addToTypes(files["[Content_Types].xml"], `<Override PartName="/`+watermarkHeaderPart+
		`" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>`)
	return created, nil
}

// watermarkDrawingXML adalah gambar mengambang di belakang teks, diletakkan relatif terhadap
// halaman (bukan paragraf), jadi posisinya sama di setiap halaman.
func watermarkDrawingXML(docPrID int, width, height, offX, offY int64) string {
	id := strconv.Itoa(docPrID)
	cx, cy := strconv.FormatInt(width, 10), strconv.FormatInt(height, 10)
	return `<w:drawing><wp:anchor xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
		`distT="0" distB="0" distL="0" distR="0" simplePos="0" relativeHeight="0" behindDoc="1" locked="1" layoutInCell="0" allowOverlap="1">` +
		`<wp:simplePos x="0" y="0"/>` +
		`<wp:positionH relativeFrom="page"><wp:posOffset>` + strconv.FormatInt(offX, 10) + `</wp:posOffset></wp:positionH>` +
		`<wp:positionV relativeFrom="page"><wp:posOffset>` + strconv.FormatInt(offY, 10) + `</wp:posOffset></wp:positionV>` +
		`<wp:extent cx="` + cx + `" cy="` + cy + `"/><wp:effectExtent l="0" t="0" r="0" b="0"/><wp:wrapNone/>` +
		`<wp:docPr id="` + id + `" name="Watermark ` + id + `"/><wp:cNvGraphicFramePr/>` +
		`<a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:nvPicPr><pic:cNvPr id="` + id + `" name="watermark.png"/><pic:cNvPicPr/></pic:nvPicPr>` +
		`<pic:blipFill><a:blip xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" r:embed="` + watermarkRelID + `"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>` +
		`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="` + cx + `" cy="` + cy + `"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr>` +
		`</pic:pic></a:graphicData></a:graphic></wp:anchor></w:drawing>`
}
