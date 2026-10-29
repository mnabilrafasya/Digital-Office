// Package pdf membuat PDF: dokumen Office (.docx) diubah ke PDF oleh LibreOffice yang berjalan di
// Gotenberg (container Docker), lalu gambar (QR persetujuan) ditempel dengan pdfcpu.
package pdf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"office.backend/internal/platform/apperr"
)

// Batas konversi.
const (
	convertTimeout = 60 * time.Second
	maxPDFBytes    = 50 << 20 // PDF hasil konversi maksimal 50 MB
)

// stampDesc menaruh gambar di pojok kanan bawah halaman, lebarnya 15% lebar halaman (sekitar
// 3 cm di A4), 36 pt (1,27 cm) dari tepi kanan dan bawah, di atas isi halaman. Nama parameter
// ditulis lengkap: singkatan seperti "sc" ambigu di pdfcpu v0.16.
const stampDesc = "position:br, offset:-36 36, scalefactor:0.15 rel, rotation:0"

// unavailableMessage adalah pesan untuk pengguna saat Gotenberg tidak bisa dihubungi.
const unavailableMessage = "Layanan pembuat PDF sedang tidak tersedia. Coba beberapa saat lagi."

// Maker membuat PDF lewat Gotenberg di baseURL (contoh http://localhost:3001).
type Maker struct {
	baseURL string
	client  *http.Client
}

// NewMaker membuat Maker.
func NewMaker(baseURL string) *Maker {
	return &Maker{baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: convertTimeout}}
}

// FromOffice mengubah dokumen Office (contoh .docx) menjadi PDF dengan LibreOffice. fileName
// wajib berekstensi sesuai isinya, karena LibreOffice menentukan format dari ekstensi.
func (m *Maker) FromOffice(ctx context.Context, fileName string, data []byte) ([]byte, error) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("files", fileName)
	if err != nil {
		return nil, fmt.Errorf("menyiapkan konversi PDF: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("menyiapkan konversi PDF: %w", err)
	}
	if err := form.Close(); err != nil {
		return nil, fmt.Errorf("menyiapkan konversi PDF: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/forms/libreoffice/convert", &body)
	if err != nil {
		return nil, fmt.Errorf("menyiapkan konversi PDF: %w", err)
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	res, err := m.client.Do(req)
	if err != nil {
		return nil, &apperr.UnavailableError{Message: unavailableMessage, Cause: err}
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		cause := fmt.Errorf("Gotenberg HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(detail)))
		if res.StatusCode >= 500 {
			return nil, &apperr.UnavailableError{Message: unavailableMessage, Cause: cause}
		}
		return nil, fmt.Errorf("mengubah naskah ke PDF: %w", cause) // naskah tidak bisa dibaca LibreOffice
	}
	out, err := io.ReadAll(io.LimitReader(res.Body, maxPDFBytes+1))
	if err != nil {
		return nil, &apperr.UnavailableError{Message: unavailableMessage, Cause: err}
	}
	if len(out) > maxPDFBytes {
		return nil, fmt.Errorf("PDF hasil konversi lebih dari %d MB", maxPDFBytes>>20)
	}
	return out, nil
}

// StampLastPage menempel gambar (PNG) di pojok kanan bawah halaman terakhir PDF.
func (m *Maker) StampLastPage(ctx context.Context, pdf, png []byte) ([]byte, error) {
	conf := model.NewDefaultConfiguration()
	pages, err := api.PageCount(ctx, bytes.NewReader(pdf), conf)
	if err != nil {
		return nil, fmt.Errorf("membaca PDF: %w", err)
	}
	wm, err := api.ImageWatermarkForReader(ctx, bytes.NewReader(png), stampDesc, true, false, types.POINTS)
	if err != nil {
		return nil, fmt.Errorf("menyiapkan QR untuk PDF: %w", err)
	}
	var out bytes.Buffer
	if err := api.AddWatermarks(ctx, bytes.NewReader(pdf), &out, []string{fmt.Sprint(pages)}, wm, conf); err != nil {
		return nil, fmt.Errorf("menempel QR ke PDF: %w", err)
	}
	return out.Bytes(), nil
}
