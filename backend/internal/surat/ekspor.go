package surat

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Unduhan naskah mengikuti status surat:
//
//	belum disetujui approver   .docx (versi DRAFT terbaru, atau ?version=N)   drafter dan peserta
//	sudah terbit (SENT)        PDF resmi ber-QR (versi FINAL)                 drafter, peserta, penerima inbox
//
// PDF resmi dibuat sekali, saat pertama kali diunduh: penanda ###NoSurat### di .docx terakhir
// diisi nomor surat, lalu .docx diubah ke PDF oleh
// LibreOffice (Gotenberg), QR persetujuan ditempel di pojok kanan bawah halaman terakhir, lalu
// disimpan sebagai versi FINAL. Unduhan berikutnya memakai file yang sama persis.

// PDFContentType adalah tipe file PDF resmi.
const PDFContentType = "application/pdf"

// OpenDocument membuka naskah surat untuk diunduh (version 0 = versi terbaru). Pemanggil wajib
// menutup Body.
func (s *Service) OpenDocument(ctx context.Context, user *auth.CurrentUser, letterID string, version int) (*DocumentFile, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	d, err := s.detail(ctx, user, id) // memeriksa akses, termasuk penerima inbox untuk surat terbit
	if err != nil {
		return nil, err
	}
	if d.Status == StatusSent {
		if version != 0 {
			return nil, &apperr.InvalidStateError{Message: "Surat sudah disetujui. Yang bisa diunduh hanya PDF resmi ber-QR."}
		}
		return s.finalPDF(ctx, user, d)
	}

	var v *DocumentVersion
	if version == 0 {
		v, err = s.reader.LatestVersion(ctx, d.InternalID, VersionDraft)
	} else {
		v, err = s.reader.FindVersion(ctx, d.InternalID, version)
	}
	if errors.Is(err, db.ErrNotFound) || (err == nil && v.Type != VersionDraft) {
		if version == 0 {
			return nil, &apperr.NotFoundError{Message: "Naskah surat belum diunggah"}
		}
		return nil, &apperr.NotFoundError{Message: "Versi naskah tidak ditemukan"}
	}
	if err != nil {
		return nil, err
	}
	return s.openVersion(ctx, v, DocxContentType)
}

// finalPDF mengembalikan PDF resmi surat terbit, membuatnya lebih dulu bila belum ada.
func (s *Service) finalPDF(ctx context.Context, user *auth.CurrentUser, d *LetterDetail) (*DocumentFile, error) {
	v, err := s.reader.LatestVersion(ctx, d.InternalID, VersionFinal)
	if errors.Is(err, db.ErrNotFound) {
		v, err = s.createFinalPDF(ctx, user, d)
	}
	if err != nil {
		return nil, err
	}
	return s.openVersion(ctx, v, PDFContentType)
}

// createFinalPDF membuat dan menyimpan PDF resmi. Baris surat dikunci selama pembuatan, jadi dua
// unduhan pertama yang bersamaan tidak membuat dua PDF: yang kedua menunggu lalu memakai hasil
// yang pertama.
func (s *Service) createFinalPDF(ctx context.Context, user *auth.CurrentUser, d *LetterDetail) (*DocumentVersion, error) {
	if d.Approval == nil || d.LetterNumber == nil { // tidak terjadi untuk SENT (dijaga CHECK)
		return nil, fmt.Errorf("surat %s berstatus SENT tanpa data persetujuan", d.ID)
	}
	var final *DocumentVersion
	err := s.tx.InTx(ctx, func(q db.DBTX) error {
		letters := s.repos.Letters(q)
		if _, err := letters.LockByUUID(ctx, user.Company.Code, d.ID); err != nil {
			return notFoundAs(err)
		}
		existing, err := letters.LatestVersion(ctx, d.InternalID, VersionFinal)
		if err == nil {
			final = existing // dibuat request lain selagi menunggu kunci
			return nil
		}
		if !errors.Is(err, db.ErrNotFound) {
			return err
		}

		src, err := letters.LatestVersion(ctx, d.InternalID, VersionDraft)
		if errors.Is(err, db.ErrNotFound) {
			return &apperr.InvalidStateError{Message: "Naskah surat belum diunggah, sehingga PDF resmi belum bisa dibuat."}
		}
		if err != nil {
			return err
		}
		docx, err := s.readFile(ctx, src.ObjectKey)
		if err != nil {
			return err
		}
		if docx, err = fillLetterNumber(docx, *d.LetterNumber); err != nil {
			return err
		}
		converted, err := s.repos.PDF.FromOffice(ctx, "naskah.docx", docx)
		if err != nil {
			return err
		}
		qrPNG, err := approvalQRPNG(d)
		if err != nil {
			return err
		}
		stamped, err := s.repos.PDF.StampLastPage(ctx, converted, qrPNG)
		if err != nil {
			return err
		}

		key := uuid.NewString()
		if err := s.repos.Files.PutDocument(ctx, key, stamped, PDFContentType); err != nil {
			return err
		}
		sum := sha256.Sum256(stamped)
		number, _, err := letters.AddVersion(ctx, NewVersion{
			LetterID: d.InternalID, Status: d.Status, Type: VersionFinal, ObjectKey: key,
			FileName: finalFileName(*d.LetterNumber), FileExtension: "pdf",
			FileSize: int64(len(stamped)), Checksum: hex.EncodeToString(sum[:]),
			CreatedBy: d.Approval.ApprovedBy.ID, CreatedByRole: RoleApprover, // dibuat atas nama penandatangan
		})
		if err != nil {
			return err
		}
		final, err = letters.FindVersion(ctx, d.InternalID, number)
		return err
	})
	return final, err
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

// numberPlaceholder adalah penanda nomor surat di template naskah (diisi saat surat terbit).
const numberPlaceholder = "###NoSurat###"

// docxTextPart: bagian .docx yang berisi teks naskah (isi, header, footer, catatan kaki).
var docxTextPart = regexp.MustCompile(`^word/(document|header\d*|footer\d*|footnotes|endnotes)\.xml$`)

// fillLetterNumber mengganti penanda ###NoSurat### di naskah .docx dengan nomor surat. Penanda
// yang terpecah oleh format berbeda di tengahnya (misalnya sebagian dicetak tebal) tidak
// terganti; template sebaiknya menulis penanda dengan satu format.
func fillLetterNumber(docx []byte, letterNumber string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		return nil, fmt.Errorf("membuka naskah .docx: %w", err)
	}
	var escaped bytes.Buffer
	if err := xml.EscapeText(&escaped, []byte(letterNumber)); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	zw := zip.NewWriter(&out)
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
		if docxTextPart.MatchString(f.Name) {
			data = bytes.ReplaceAll(data, []byte(numberPlaceholder), escaped.Bytes())
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f.Name, Method: f.Method, Modified: f.Modified})
		if err != nil {
			return nil, fmt.Errorf("menulis naskah .docx: %w", err)
		}
		if _, err := w.Write(data); err != nil {
			return nil, fmt.Errorf("menulis naskah .docx: %w", err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("menulis naskah .docx: %w", err)
	}
	return out.Bytes(), nil
}

// finalFileName membentuk nama file PDF resmi dari nomor surat, contoh 015-TI-PSP-X-2026.pdf.
func finalFileName(letterNumber string) string {
	return strings.NewReplacer("/", "-", `\`, "-").Replace(letterNumber) + ".pdf"
}
