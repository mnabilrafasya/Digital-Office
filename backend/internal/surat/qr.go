package surat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"rsc.io/qr"

	"office.backend/internal/auth"
	"office.backend/internal/masterdata"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// qrScale adalah ukuran satu titik QR dalam piksel PNG.
const qrScale = 6

// ApprovalQR membuat gambar PNG QR persetujuan surat yang sudah disetujui approver. Isinya teks
// biasa (lihat approvalQRText). Akses sama dengan detail surat (termasuk penerima inbox).
func (s *Service) ApprovalQR(ctx context.Context, user *auth.CurrentUser, letterID string) ([]byte, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	d, err := s.detail(ctx, user, id)
	if err != nil {
		return nil, err
	}
	if d.Direction != masterdata.DirectionInternal {
		return nil, &apperr.InvalidStateError{Message: "Surat eksternal ditandatangani basah dan tidak memakai QR persetujuan"}
	}
	if d.Approval == nil {
		return nil, &apperr.InvalidStateError{Message: "QR persetujuan baru tersedia setelah surat disetujui approver"}
	}
	return approvalQRPNG(d)
}

// Penanda QR wajib ada di naskah surat internal yang sudah diunggah, karena QR persetujuan hanya
// dipasang di posisi penandanya (penanda.go). Surat tanpa naskah dan surat eksternal (ditandatangani
// basah) tidak diperiksa.
var (
	errNoQRPlaceholderSubmit = apperr.Invalid("Naskah belum memuat penanda QR Code (###*QRCode*###). Tambahkan penanda di posisi tanda tangan sebelum mengajukan.")
	errNoQRPlaceholderSign   = &apperr.InvalidStateError{Message: "Naskah tidak memuat penanda QR Code (###*QRCode*###). Kembalikan surat ke drafter untuk diperbaiki."}
)

// checkQRPlaceholder memastikan naskah .docx terakhir surat memuat penanda QR; tidak ada = errNoQR.
// Surat yang belum punya naskah lolos.
func (s *Service) checkQRPlaceholder(ctx context.Context, letters Store, letterID int64, errNoQR error) error {
	v, err := letters.LatestVersion(ctx, letterID, VersionDraft)
	if errors.Is(err, db.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	docx, err := s.readFile(ctx, v.ObjectKey)
	if err != nil {
		return err
	}
	ok, err := hasQRPlaceholder(docx)
	if err != nil {
		return apperr.Invalid("Naskah surat bukan dokumen Word (.docx) yang valid")
	}
	if !ok {
		return errNoQR
	}
	return nil
}

// approvalQRPNG membuat gambar PNG QR persetujuan dari detail surat yang sudah disetujui.
func approvalQRPNG(d *LetterDetail) ([]byte, error) {
	code, err := qr.Encode(approvalQRText(d), qr.M)
	if err != nil {
		return nil, fmt.Errorf("membuat QR persetujuan: %w", err)
	}
	code.Scale = qrScale
	return code.PNG(), nil
}

// approvalQRText adalah isi QR: nomor surat, siapa yang menyetujui, dan kapan (zona waktu
// tampilan, misalnya WIB). Detail sudah melewati present, jadi waktunya sudah di zona tersebut.
func approvalQRText(d *LetterDetail) string {
	a := d.Approval
	lines := []string{"Persetujuan Surat - Digital Office"}
	if d.LetterNumber != nil {
		lines = append(lines, "Nomor: "+*d.LetterNumber)
	}
	lines = append(lines,
		"Disetujui oleh: "+a.ApprovedBy.Name,
		"Jabatan: "+a.ApprovedBy.PositionName,
		"Waktu: "+a.ApprovedAt.Format("02-01-2006 15:04:05 MST"),
	)
	return strings.Join(lines, "\n")
}
