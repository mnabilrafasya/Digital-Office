package surat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Unduhan naskah mengikuti status surat:
//
//	belum disetujui approver   .docx (versi DRAFT terbaru, atau ?version=N)   drafter dan peserta
//	sudah terbit (SENT)        PDF resmi ber-QR (versi FINAL)                 drafter, peserta, penerima inbox
//
// PDF resmi dibuat oleh editor DevExpress di front-end dari naskah terisi, supaya sama persis dengan
// tampilan di layar, lalu disimpan sekali sebagai versi FINAL (lihat resmi.go). Unduhan berikutnya
// memakai file yang sama persis.

// PDFContentType adalah tipe file PDF resmi.
const PDFContentType = "application/pdf"

// OpenDocument membuka naskah surat untuk diunduh (version 0 = versi terbaru); unduhannya dicatat
// di audit. Pemanggil wajib menutup Body.
func (s *Service) OpenDocument(ctx context.Context, user *auth.CurrentUser, letterID string, version int) (*DocumentFile, error) {
	return s.openDocument(ctx, user, letterID, version, true)
}

// ViewDocument sama dengan OpenDocument, tetapi untuk ditampilkan di layar (misalnya PDF resmi di
// halaman detail surat), sehingga tidak dicatat sebagai DOWNLOAD. Pembukaan surat Rahasia sudah
// dicatat saat detailnya dibuka (VIEW).
func (s *Service) ViewDocument(ctx context.Context, user *auth.CurrentUser, letterID string, version int) (*DocumentFile, error) {
	return s.openDocument(ctx, user, letterID, version, false)
}

func (s *Service) openDocument(ctx context.Context, user *auth.CurrentUser, letterID string, version int, recordDownload bool) (*DocumentFile, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	d, err := s.detail(ctx, user, id) // memeriksa akses, termasuk penerima inbox untuk surat terbit
	if err != nil {
		return nil, err
	}
	var (
		v           *DocumentVersion
		contentType = DocxContentType
	)
	if d.Status == StatusSent {
		if version != 0 {
			return nil, &apperr.InvalidStateError{Message: "Surat sudah disetujui. Yang bisa diunduh hanya PDF resmi ber-QR."}
		}
		if v, err = s.finalVersion(ctx, d.InternalID); err != nil {
			return nil, err
		}
		contentType = PDFContentType
	} else {
		if v, err = s.draftVersion(ctx, d.InternalID, version); err != nil {
			return nil, err
		}
	}

	f, err := s.openVersion(ctx, v, contentType)
	if err != nil {
		return nil, err
	}
	if !recordDownload {
		return f, nil
	}
	err = s.audit.Record(ctx, letterEntry(user, d.InternalID, d.ID, audit.ActionDownload,
		downloadSnapshot{VersionNumber: v.VersionNumber, Type: v.Type}))
	if err != nil {
		f.Body.Close()
		return nil, err
	}
	return f, nil
}

// downloadSnapshot adalah isi audit_logs.snapshot saat dokumen surat diunduh.
type downloadSnapshot struct {
	VersionNumber int    `json:"versionNumber"`
	Type          string `json:"type"` // DRAFT (.docx) atau FINAL (PDF resmi)
}

// draftVersion mengambil versi naskah .docx (version 0 = terbaru).
func (s *Service) draftVersion(ctx context.Context, letterID int64, version int) (*DocumentVersion, error) {
	var (
		v   *DocumentVersion
		err error
	)
	if version == 0 {
		v, err = s.reader.LatestVersion(ctx, letterID, VersionDraft)
	} else {
		v, err = s.reader.FindVersion(ctx, letterID, version)
	}
	if errors.Is(err, db.ErrNotFound) || (err == nil && v.Type != VersionDraft) {
		if version == 0 {
			return nil, &apperr.NotFoundError{Message: "Naskah surat belum diunggah"}
		}
		return nil, &apperr.NotFoundError{Message: "Versi naskah tidak ditemukan"}
	}
	return v, err
}

// finalVersion mengambil PDF resmi surat terbit. Belum ada berarti belum disiapkan front-end
// (lihat resmi.go).
func (s *Service) finalVersion(ctx context.Context, letterID int64) (*DocumentVersion, error) {
	v, err := s.reader.LatestVersion(ctx, letterID, VersionFinal)
	if errors.Is(err, db.ErrNotFound) {
		return nil, errFinalPDFNotReady
	}
	return v, err
}

// openVersion membuka file satu versi naskah dari storage.
func (s *Service) openVersion(ctx context.Context, v *DocumentVersion, contentType string) (*DocumentFile, error) {
	body, size, err := s.repos.Files.OpenDocument(ctx, v.ObjectKey)
	if err != nil {
		return nil, err
	}
	return &DocumentFile{Name: v.FileName, ContentType: contentType, Size: size, Body: body}, nil
}

// readFile membaca seluruh isi file naskah (maksimal MaxDocumentBytes).
func (s *Service) readFile(ctx context.Context, objectKey string) ([]byte, error) {
	body, _, err := s.repos.Files.OpenDocument(ctx, objectKey)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, MaxDocumentBytes+1))
	if err != nil {
		return nil, fmt.Errorf("membaca naskah: %w", err)
	}
	return data, nil
}

// docxTextPart: bagian .docx yang berisi teks naskah (isi, header, footer, catatan kaki).
var docxTextPart = regexp.MustCompile(`^word/(document|header\d*|footer\d*|footnotes|endnotes)\.xml$`)

// finalFileName membentuk nama file PDF resmi dari nomor surat, contoh 015-TI-PSP-X-2026.pdf.
func finalFileName(letterNumber string) string {
	return strings.NewReplacer("/", "-", `\`, "-").Replace(letterNumber) + ".pdf"
}
