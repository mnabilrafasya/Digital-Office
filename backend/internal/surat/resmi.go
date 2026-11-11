package surat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
	"office.backend/internal/platform/pdf"
)

// PDF resmi dibuat oleh editor DevExpress di front-end, supaya PDF sama persis dengan tampilan di
// layar (mesin tata letak dan font yang sama). Satu surat punya satu naskah resmi dan satu PDF resmi
// yang sama untuk semua orang (drafter, peserta, penerima, penerima disposisi):
//
//  1. Setelah surat terbit, front-end mengambil naskah terisi (FilledDocument): penanda sistem
//     sudah diisi dan QR sudah di posisinya (penanda.go). Naskah ini juga yang ditampilkan dalam
//     mode baca, tanpa watermark.
//  2. Bila PDF resmi belum ada (OfficialStatus), editor mengekspor naskah tersebut ke PDF dan
//     mengunggahnya (SaveFinalPDF). Backend menempel watermark perusahaan di setiap halaman
//     (watermark.go), lalu menyimpannya sebagai versi FINAL. PDF pertama yang diterima dipakai dan
//     tidak bisa diganti; naskah terisinya disimpan bersamanya (filledObjectKey).
//  3. Unduhan dan cetak memakai PDF resmi yang tersimpan (OpenDocument), jadi selalu ber-watermark.

// MaxFinalPDFBytes adalah batas ukuran PDF resmi yang diunggah editor.
const MaxFinalPDFBytes = MaxUploadBytes

// pdfSignature adalah awal setiap file PDF.
var pdfSignature = []byte("%PDF-")

// FinalPDFInput adalah PDF resmi hasil ekspor editor.
type FinalPDFInput struct {
	LetterID string
	Data     []byte
	ClientIP string // dicatat di audit
}

// OfficialStatus adalah keadaan naskah resmi surat terbit.
type OfficialStatus struct {
	PDFReady bool    `json:"pdfReady"` // PDF resmi sudah tersimpan
	CopyFor  *string `json:"copyFor"`  // selalu null (fitur salinan per penerima sudah dihapus)
}

// filledObjectKey adalah nama objek MinIO (bucket documents) untuk naskah terisi yang disimpan
// bersama PDF resmi. Nama "salinan/.../induk.docx" dipertahankan agar naskah lama tetap terbaca.
func filledObjectKey(letterUUID string) string {
	return "salinan/" + letterUUID + "/induk.docx"
}

// finalPDFSnapshot adalah isi audit_logs.snapshot saat PDF resmi disimpan.
type finalPDFSnapshot struct {
	VersionNumber int    `json:"versionNumber"`
	FileSize      int64  `json:"fileSize"`
	Pages         int    `json:"pages"`
	Checksum      string `json:"checksum"`
	Generator     string `json:"generator"`           // pembuat PDF, saat ini selalu DEVEXPRESS
	Watermark     string `json:"watermark,omitempty"` // teks watermark yang ditempel; kosong = tanpa
}

// FilledDocument membuka naskah terisi (.docx) surat terbit, untuk ditampilkan editor dalam mode
// baca dan diekspor menjadi PDF resmi. Bila PDF resmi sudah ada, naskah yang disimpan bersamanya
// yang dikirim. Tidak dicatat sebagai unduhan. Pemanggil wajib menutup Body.
func (s *Service) FilledDocument(ctx context.Context, user *auth.CurrentUser, letterID string) (*DocumentFile, error) {
	d, err := s.sentDetail(ctx, user, letterID)
	if err != nil {
		return nil, err
	}
	docx, err := s.officialDocx(ctx, d)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSuffix(finalFileName(*d.LetterNumber), ".pdf") + ".docx"
	return &DocumentFile{Name: name, ContentType: DocxContentType, Size: int64(len(docx)),
		Body: io.NopCloser(bytes.NewReader(docx))}, nil
}

// OfficialStatus mengembalikan keadaan naskah resmi surat terbit: apakah PDF resminya sudah ada
// (bila belum, editor front-end membuatnya).
func (s *Service) OfficialStatus(ctx context.Context, user *auth.CurrentUser, letterID string) (*OfficialStatus, error) {
	d, err := s.sentDetail(ctx, user, letterID)
	if err != nil {
		return nil, err
	}
	_, err = s.finalVersion(ctx, d.InternalID)
	if errors.Is(err, errFinalPDFNotReady) {
		return &OfficialStatus{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &OfficialStatus{PDFReady: true}, nil
}

// officialDocx mengambil naskah terisi yang tersimpan bila ada, selain itu disusun dari naskah
// .docx terakhir.
func (s *Service) officialDocx(ctx context.Context, d *LetterDetail) ([]byte, error) {
	key := filledObjectKey(d.ID)
	stored, err := s.repos.Files.DocumentExists(ctx, key)
	if err != nil {
		return nil, err
	}
	if stored {
		return s.readFile(ctx, key)
	}
	return s.fillOfficial(ctx, d)
}

// fillOfficial menyusun naskah terisi dari naskah .docx terakhir surat.
func (s *Service) fillOfficial(ctx context.Context, d *LetterDetail) ([]byte, error) {
	src, err := s.reader.LatestVersion(ctx, d.InternalID, VersionDraft)
	if errors.Is(err, db.ErrNotFound) {
		return nil, &apperr.InvalidStateError{Message: "Naskah surat belum diunggah, sehingga naskah resmi belum bisa ditampilkan."}
	}
	if err != nil {
		return nil, err
	}
	docx, err := s.readFile(ctx, src.ObjectKey)
	if err != nil {
		return nil, err
	}
	qrPNG, err := approvalQRPNG(d)
	if err != nil {
		return nil, err
	}
	return fillPlaceholders(docx, letterPlaceholders(d), qrPNG)
}

// SaveFinalPDF menyimpan PDF resmi hasil ekspor editor. Yang boleh: siapa pun yang boleh membuka
// surat. PDF pertama yang diterima dipakai; bila sudah ada, created=false (unggahan ganda dari dua
// tab tidak dianggap error). Watermark perusahaan ditempel sebelum disimpan; naskah terisinya ikut
// disimpan.
func (s *Service) SaveFinalPDF(ctx context.Context, user *auth.CurrentUser, in FinalPDFInput) (v *DocumentVersion, created bool, err error) {
	d, err := s.sentDetail(ctx, user, in.LetterID)
	if err != nil {
		return nil, false, err
	}
	pages, err := parseFinalPDF(ctx, in.Data)
	if err != nil {
		return nil, false, err
	}
	data, wmText, err := addPDFWatermark(ctx, d, in.Data)
	if err != nil {
		return nil, false, err
	}
	sum := sha256.Sum256(data)
	snap := finalPDFSnapshot{FileSize: int64(len(data)), Pages: pages, Checksum: hex.EncodeToString(sum[:]),
		Generator: "DEVEXPRESS", Watermark: wmText}

	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		letters := s.repos.Letters(q)
		// Kunci baris surat: unggahan bersamaan menunggu, lalu memakai hasil pertama
		if _, err := letters.LockByUUID(ctx, user.Company.Code, d.ID); err != nil {
			return notFoundAs(err)
		}
		if _, err := letters.LatestVersion(ctx, d.InternalID, VersionFinal); err == nil {
			return nil // PDF resmi sudah ada
		} else if !errors.Is(err, db.ErrNotFound) {
			return err
		}
		docx, err := s.fillOfficial(ctx, d)
		if err != nil {
			return err
		}
		if err := s.repos.Files.PutDocument(ctx, filledObjectKey(d.ID), docx, DocxContentType); err != nil {
			return err
		}
		key := uuid.NewString()
		if err := s.repos.Files.PutDocument(ctx, key, data, PDFContentType); err != nil {
			return err
		}
		if snap.VersionNumber, _, err = letters.AddVersion(ctx, NewVersion{
			LetterID: d.InternalID, Status: d.Status, Type: VersionFinal, ObjectKey: key,
			FileName: finalFileName(*d.LetterNumber), FileExtension: "pdf",
			FileSize: snap.FileSize, Checksum: snap.Checksum,
			CreatedBy: d.Approval.ApprovedBy.ID, CreatedByRole: RoleApprover, // atas nama penandatangan
		}); err != nil {
			return err
		}
		created = true
		e := letterEntry(user, d.InternalID, d.ID, audit.ActionFinalPDF, snap)
		e.IPAddress = in.ClientIP
		return s.repos.Audit(q).Record(ctx, e)
	})
	if err != nil {
		return nil, false, err
	}
	if v, err = s.finalVersion(ctx, d.InternalID); err != nil {
		return nil, false, err
	}
	v.CreatedAt = v.CreatedAt.In(s.loc)
	return v, created, nil
}

// sentDetail memuat detail surat yang sudah terbit dan memeriksa aksesnya (termasuk penerima inbox).
func (s *Service) sentDetail(ctx context.Context, user *auth.CurrentUser, letterID string) (*LetterDetail, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	d, err := s.detail(ctx, user, id)
	if err != nil {
		return nil, err
	}
	if d.Status != StatusSent {
		return nil, &apperr.InvalidStateError{Message: "Naskah resmi baru tersedia setelah surat disetujui approver."}
	}
	if d.Approval == nil || d.LetterNumber == nil { // tidak terjadi untuk SENT (dijaga CHECK)
		return nil, fmt.Errorf("surat %s berstatus SENT tanpa data persetujuan", d.ID)
	}
	return d, nil
}

// parseFinalPDF memvalidasi PDF resmi: tidak kosong, maksimal MaxFinalPDFBytes, dan strukturnya
// PDF yang valid dengan minimal satu halaman. Mengembalikan jumlah halaman.
func parseFinalPDF(ctx context.Context, data []byte) (int, error) {
	switch {
	case len(data) == 0:
		return 0, apperr.Invalid("File PDF resmi kosong")
	case len(data) > MaxFinalPDFBytes:
		return 0, apperr.Invalid(fmt.Sprintf("Ukuran PDF resmi maksimal %d MB", MaxFinalPDFBytes>>20))
	case !bytes.HasPrefix(data, pdfSignature):
		return 0, apperr.Invalid("File bukan PDF yang valid")
	}
	pages, err := pdf.PageCount(ctx, data)
	if err != nil || pages < 1 {
		return 0, apperr.Invalid("File bukan PDF yang valid")
	}
	return pages, nil
}
