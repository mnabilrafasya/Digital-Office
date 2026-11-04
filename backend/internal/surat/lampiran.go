package surat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Lampiran surat: file di MinIO (bucket documents) dan daftarnya (JSON) di letters.attachments.
// Hanya drafter yang boleh menambah dan menghapus, selama surat masih draf atau dikembalikan
// kepadanya. Yang boleh mengunduh sama dengan yang boleh membuka surat (drafter, peserta, dan
// penerima inbox surat terbit).

// Batas lampiran (mengikuti form front-end).
const (
	MaxAttachmentBytes = 25 << 20 // 25 MB per file
	MaxAttachments     = 20       // per surat
)

// oleSignature adalah awal file Office lama (.doc).
var oleSignature = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}

// attachmentTypes adalah format lampiran yang diterima: ekstensi -> tipe file dan tanda awal isinya.
// Tipe file ditentukan server dari ekstensi, bukan dari header yang dikirim browser.
var attachmentTypes = map[string]struct {
	contentType string
	signature   []byte
}{
	".pdf":  {PDFContentType, []byte("%PDF-")},
	".docx": {DocxContentType, zipSignature},
	".xlsx": {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", zipSignature},
	".pptx": {"application/vnd.openxmlformats-officedocument.presentationml.presentation", zipSignature},
	".doc":  {"application/msword", oleSignature},
}

// AttachmentInput adalah file lampiran yang diunggah.
type AttachmentInput struct {
	LetterID string
	FileName string
	Data     []byte
	ClientIP string // dicatat di audit
}

// attachmentSnapshot adalah isi kolom audit_logs.snapshot saat lampiran ditambah, dihapus, atau diunduh.
type attachmentSnapshot struct {
	Type         string `json:"type,omitempty"` // ATTACHMENT saat diunduh (membedakan dari unduhan naskah)
	AttachmentID string `json:"attachmentId"`
	FileName     string `json:"fileName"`
	FileSize     int64  `json:"fileSize,omitempty"`
	Checksum     string `json:"checksum,omitempty"`
}

// AddAttachment menyimpan lampiran baru. Mengembalikan lampiran dan ROWVERSION surat terbaru
// (ETag ikut berubah, sama seperti saat naskah disimpan).
func (s *Service) AddAttachment(ctx context.Context, user *auth.CurrentUser, in AttachmentInput) (*Attachment, []byte, error) {
	id, ok := parseLetterID(in.LetterID)
	if !ok {
		return nil, nil, ErrLetterNotFound
	}
	name, contentType, err := parseAttachment(in.FileName, in.Data)
	if err != nil {
		return nil, nil, err
	}
	sum := sha256.Sum256(in.Data)

	var (
		a      Attachment
		rowVer []byte
	)
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		letters := s.repos.Letters(q)
		l, err := s.lockForAttachments(ctx, letters, user, id)
		if err != nil {
			return err
		}
		items, err := letters.Attachments(ctx, l.ID)
		if err != nil {
			return err
		}
		if len(items) >= MaxAttachments {
			return apperr.Invalid(fmt.Sprintf("Lampiran maksimal %d file per surat", MaxAttachments))
		}

		// Diulang saat deadlock: objek lama tertinggal tanpa dirujuk daftar lampiran
		a = Attachment{
			ID: uuid.NewString(), FileName: name, Size: int64(len(in.Data)), ContentType: contentType,
			Checksum: hex.EncodeToString(sum[:]), UploadedAt: time.Now().UTC(),
			UploadedBy: UserRef{ID: user.ID, Name: user.Name},
		}
		if err := s.repos.Files.PutDocument(ctx, a.ID, in.Data, contentType); err != nil {
			return err
		}
		if rowVer, err = letters.SaveAttachments(ctx, l.ID, append(items, a)); err != nil {
			return err
		}
		e := letterEntry(user, l.ID, id, audit.ActionAttachmentAdd,
			attachmentSnapshot{AttachmentID: a.ID, FileName: a.FileName, FileSize: a.Size, Checksum: a.Checksum})
		e.IPAddress = in.ClientIP
		return s.repos.Audit(q).Record(ctx, e)
	})
	if err != nil {
		return nil, nil, err
	}
	a.UploadedAt = a.UploadedAt.In(s.loc)
	return &a, rowVer, nil
}

// RemoveAttachment menghapus satu lampiran. Mengembalikan ROWVERSION surat terbaru. File di MinIO
// dihapus setelah transaksi berhasil; kalau gagal hanya dicatat di log (tidak lagi dirujuk).
func (s *Service) RemoveAttachment(ctx context.Context, user *auth.CurrentUser, letterID, attachmentID string) ([]byte, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	attID, ok := parseAttachmentID(attachmentID)
	if !ok {
		return nil, errAttachmentNotFound
	}

	var rowVer []byte
	err := s.tx.InTx(ctx, func(q db.DBTX) error {
		letters := s.repos.Letters(q)
		l, err := s.lockForAttachments(ctx, letters, user, id)
		if err != nil {
			return err
		}
		items, err := letters.Attachments(ctx, l.ID)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(items, func(a Attachment) bool { return a.ID == attID })
		if i < 0 {
			return errAttachmentNotFound
		}
		removed := items[i]
		if rowVer, err = letters.SaveAttachments(ctx, l.ID, slices.Delete(items, i, i+1)); err != nil {
			return err
		}
		return s.repos.Audit(q).Record(ctx, letterEntry(user, l.ID, id, audit.ActionAttachmentDelete,
			attachmentSnapshot{AttachmentID: removed.ID, FileName: removed.FileName, FileSize: removed.Size, Checksum: removed.Checksum}))
	})
	if err != nil {
		return nil, err
	}
	if err := s.repos.Files.RemoveDocument(ctx, attID); err != nil {
		log.Printf("menghapus file lampiran %s: %v", attID, err)
	}
	return rowVer, nil
}

// OpenAttachment membuka file lampiran untuk diunduh. Pemanggil wajib menutup Body.
func (s *Service) OpenAttachment(ctx context.Context, user *auth.CurrentUser, letterID, attachmentID string) (*DocumentFile, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	d, err := s.detail(ctx, user, id) // memeriksa akses, termasuk penerima inbox untuk surat terbit
	if err != nil {
		return nil, err
	}
	attID, ok := parseAttachmentID(attachmentID)
	if !ok {
		return nil, errAttachmentNotFound
	}
	i := slices.IndexFunc(d.Attachments, func(a Attachment) bool { return a.ID == attID })
	if i < 0 {
		return nil, errAttachmentNotFound
	}
	a := d.Attachments[i]

	body, size, err := s.repos.Files.OpenDocument(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	err = s.audit.Record(ctx, letterEntry(user, d.InternalID, d.ID, audit.ActionDownload,
		attachmentSnapshot{Type: "ATTACHMENT", AttachmentID: a.ID, FileName: a.FileName}))
	if err != nil {
		body.Close()
		return nil, err
	}
	return &DocumentFile{Name: a.FileName, ContentType: a.ContentType, Size: size, Body: body}, nil
}

// lockForAttachments mengunci surat dan memastikan user adalah drafter dan surat masih bisa
// diubah. Peserta lain mendapat 403, user lain 404 (keberadaan surat tidak bocor).
func (s *Service) lockForAttachments(ctx context.Context, letters Store, user *auth.CurrentUser, id string) (*LetterLock, error) {
	l, err := letters.LockByUUID(ctx, user.Company.Code, id)
	if err != nil {
		return nil, notFoundAs(err)
	}
	if l.DrafterID != user.ID {
		isParticipant, err := letters.IsParticipant(ctx, l.ID, user.ID)
		if err != nil {
			return nil, err
		}
		if isParticipant {
			return nil, apperr.ErrForbidden
		}
		return nil, ErrLetterNotFound
	}
	if !isEditable(l.Status) {
		return nil, &apperr.InvalidStateError{Message: "Lampiran hanya bisa diubah pembuat surat saat surat masih draf atau dikembalikan kepadanya."}
	}
	return l, nil
}

// parseAttachment memvalidasi file lampiran: format yang diterima (dicek dari ekstensi dan
// isinya), tidak kosong, maksimal MaxAttachmentBytes. Mengembalikan nama file dan tipe filenya.
func parseAttachment(fileName string, data []byte) (name, contentType string, err error) {
	name = baseFileName(fileName)
	kind, ok := attachmentTypes[strings.ToLower(path.Ext(name))]
	if !ok {
		return "", "", apperr.Invalid("Lampiran wajib berupa file PDF, DOC, DOCX, XLSX, atau PPTX")
	}
	if _, err := optionalText(&name, maxFileNameLength, "Nama file"); err != nil {
		return "", "", err
	}
	switch {
	case len(data) == 0:
		return "", "", apperr.Invalid("File lampiran kosong")
	case len(data) > MaxAttachmentBytes:
		return "", "", apperr.Invalid(fmt.Sprintf("Ukuran lampiran maksimal %d MB", MaxAttachmentBytes>>20))
	case !bytes.HasPrefix(data, kind.signature):
		return "", "", apperr.Invalid("Isi file lampiran tidak sesuai dengan ekstensinya")
	}
	return name, kind.contentType, nil
}

// parseAttachmentID memvalidasi ID lampiran (uuid) dari URL dan mengubahnya ke huruf kecil.
func parseAttachmentID(raw string) (string, bool) {
	return parseLetterID(raw)
}
