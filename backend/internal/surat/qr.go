package surat

import (
	"context"
	"fmt"
	"strings"

	"rsc.io/qr"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
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
	if d.Approval == nil {
		return nil, &apperr.InvalidStateError{Message: "QR persetujuan baru tersedia setelah surat disetujui approver"}
	}
	return approvalQRPNG(d)
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
