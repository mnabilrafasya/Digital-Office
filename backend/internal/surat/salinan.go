package surat

import (
	"context"
	"errors"
	"strconv"

	"office.backend/internal/auth"
)

// Salinan surat terbit: satu nomor surat, tetapi setiap penerima langsung (TO) mendapat salinan
// yang "Kepada"-nya jabatan dia sendiri, termasuk anggota alias ("Seluruh VP" -> "VP Akuntansi").
// Penerima disposisi/pengalihan melihat salinan milik penerima langsung asalnya. Penerima tembusan
// dan yang bukan penerima (drafter, reviewer, approver, admin) melihat salinan induk.
//
// Setiap salinan punya naskah terisi (.docx) dan PDF resmi sendiri, dibuat sekali lalu tidak bisa
// diganti. Salinan induk: PDF di letter_versions (FINAL). Salinan penerima: objek MinIO dengan nama
// tetap (copyObjectKey), tanpa baris database tambahan. Naskah terisi disimpan bersama PDF-nya,
// supaya tampilan tetap sama dengan PDF walaupun jabatan berubah setelahnya.

// officialCopy adalah salinan yang dilihat seseorang. OwnerID 0 = salinan induk.
type officialCopy struct {
	OwnerID      int64
	PositionName string // jabatan pemilik salinan, ditulis di "Kepada"
}

func (c officialCopy) isMaster() bool { return c.OwnerID == 0 }

// OfficialStatus adalah keadaan naskah resmi surat terbit untuk orang yang membukanya.
type OfficialStatus struct {
	PDFReady bool    `json:"pdfReady"` // PDF resmi salinan ini sudah tersimpan
	CopyFor  *string `json:"copyFor"`  // jabatan pemilik salinan; null = salinan induk
}

// copyFor menentukan salinan yang dilihat user untuk surat terbit d.
func (s *Service) copyFor(ctx context.Context, d *LetterDetail, user *auth.CurrentUser) (officialCopy, error) {
	ownerID, recipientType, position, err := s.inbox.CopyOwner(ctx, d.InternalID, user.ID)
	if err != nil {
		return officialCopy{}, err
	}
	if ownerID == 0 || recipientType != RecipientTo {
		return officialCopy{}, nil
	}
	return officialCopy{OwnerID: ownerID, PositionName: position}, nil
}

// copyObjectKey adalah nama objek MinIO (bucket documents) untuk berkas salinan: ext "docx" untuk
// naskah terisi, "pdf" untuk PDF resmi salinan penerima.
func copyObjectKey(letterUUID string, c officialCopy, ext string) string {
	name := "induk"
	if !c.isMaster() {
		name = strconv.FormatInt(c.OwnerID, 10)
	}
	return "salinan/" + letterUUID + "/" + name + "." + ext
}

// OfficialStatus mengembalikan keadaan naskah resmi surat terbit untuk user: salinan milik siapa
// dan apakah PDF resminya sudah ada (bila belum, editor front-end membuatnya).
func (s *Service) OfficialStatus(ctx context.Context, user *auth.CurrentUser, letterID string) (*OfficialStatus, error) {
	d, err := s.sentDetail(ctx, user, letterID)
	if err != nil {
		return nil, err
	}
	c, err := s.copyFor(ctx, d, user)
	if err != nil {
		return nil, err
	}
	ready, err := s.copyPDFExists(ctx, d, c)
	if err != nil {
		return nil, err
	}
	st := &OfficialStatus{PDFReady: ready}
	if !c.isMaster() {
		st.CopyFor = &c.PositionName
	}
	return st, nil
}

// copyPDFExists mengecek apakah PDF resmi salinan c sudah tersimpan.
func (s *Service) copyPDFExists(ctx context.Context, d *LetterDetail, c officialCopy) (bool, error) {
	_, err := s.officialPDF(ctx, d, c)
	if errors.Is(err, errFinalPDFNotReady) {
		return false, nil
	}
	return err == nil, err
}
