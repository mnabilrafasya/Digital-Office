package surat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"

	"github.com/google/uuid"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Naskah surat disimpan per versi: file di MinIO (bucket documents) dan datanya di tabel
// letter_versions. Selama surat belum disetujui approver, naskah berupa .docx (versi DRAFT).
// Unduhan (.docx sebelum terbit, PDF resmi ber-QR sesudahnya) ada di ekspor.go.

// Batas dan format naskah.
const (
	MaxDocumentBytes  = 10 << 20 // 10 MB
	DocxContentType   = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	maxFileNameLength = 255 // letter_versions.file_name NVARCHAR(255)
)

// zipSignature adalah awal setiap file .docx (arsip ZIP).
var zipSignature = []byte("PK\x03\x04")

// DocumentInput adalah file naskah yang diunggah.
type DocumentInput struct {
	LetterID string
	FileName string
	Data     []byte
	ClientIP string // dicatat di audit
}

// DocumentFile adalah file naskah yang siap dikirim ke klien. Pemanggil wajib menutup Body.
type DocumentFile struct {
	Name        string
	ContentType string
	Size        int64
	Body        io.ReadCloser
}

// documentSnapshot adalah isi kolom audit_logs.snapshot saat naskah disimpan.
type documentSnapshot struct {
	VersionNumber int    `json:"versionNumber"`
	FileName      string `json:"fileName"`
	FileSize      int64  `json:"fileSize"`
	Checksum      string `json:"checksum"`
}

// SaveDocument menyimpan naskah .docx sebagai versi baru. Yang boleh: drafter saat surat masih
// draf atau dikembalikan kepadanya, dan reviewer yang sedang memegang surat; approver hanya
// membaca. Mengembalikan versi baru dan ROWVERSION surat terbaru (ETag ikut berubah).
func (s *Service) SaveDocument(ctx context.Context, user *auth.CurrentUser, in DocumentInput) (*DocumentVersion, []byte, error) {
	id, ok := parseLetterID(in.LetterID)
	if !ok {
		return nil, nil, ErrLetterNotFound
	}
	name, err := parseDocument(in.FileName, in.Data)
	if err != nil {
		return nil, nil, err
	}
	sum := sha256.Sum256(in.Data)
	checksum := hex.EncodeToString(sum[:])

	var (
		letterID int64
		number   int
		rowVer   []byte
	)
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		letters := s.repos.Letters(q)
		l, err := letters.LockByUUID(ctx, user.Company.Code, id)
		if err != nil {
			return notFoundAs(err)
		}
		reviewers, approverID, err := letters.Participants(ctx, l.ID)
		if err != nil {
			return err
		}
		if !involves(l.DrafterID, approverID, reviewers, user.ID) {
			return ErrLetterNotFound
		}
		role := documentEditorRole(l, reviewers, user.ID)
		if role == "" {
			return &apperr.InvalidStateError{Message: "Naskah hanya bisa diubah pembuat surat saat draf, atau reviewer yang sedang memeriksa surat."}
		}
		if perm := rolePermission(role); perm != "" && !user.HasPermission(perm) {
			return apperr.ErrForbidden
		}

		// Diulang saat deadlock: objek lama tertinggal tanpa baris versi (tidak pernah dirujuk)
		key := uuid.NewString()
		if err := s.repos.Files.PutDocument(ctx, key, in.Data, DocxContentType); err != nil {
			return err
		}
		letterID = l.ID
		number, rowVer, err = letters.AddVersion(ctx, NewVersion{
			LetterID: l.ID, Status: l.Status, Type: VersionDraft, ObjectKey: key,
			FileName: name, FileExtension: "docx", FileSize: int64(len(in.Data)), Checksum: checksum,
			CreatedBy: user.ID, CreatedByRole: role,
		})
		if err != nil {
			return err
		}
		return s.repos.Audit(q).Record(ctx, audit.Entry{
			CompanyCode: user.Company.Code,
			EntityType:  "LETTER",
			EntityID:    l.ID,
			EntityUUID:  id,
			Action:      "SAVE",
			ActorID:     user.ID,
			ActorName:   user.Name,
			IPAddress:   in.ClientIP,
			Snapshot:    documentSnapshot{VersionNumber: number, FileName: name, FileSize: int64(len(in.Data)), Checksum: checksum},
		})
	})
	if err != nil {
		return nil, nil, err
	}
	v, err := s.reader.FindVersion(ctx, letterID, number)
	if err != nil {
		return nil, nil, err
	}
	v.CreatedAt = v.CreatedAt.In(s.loc)
	return v, rowVer, nil
}

// Versions mengambil daftar versi naskah (terbaru dulu) untuk drafter dan peserta.
func (s *Service) Versions(ctx context.Context, user *auth.CurrentUser, letterID string) ([]DocumentVersion, error) {
	l, err := s.readable(ctx, user, letterID)
	if err != nil {
		return nil, err
	}
	versions, err := s.reader.Versions(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	for i := range versions {
		versions[i].CreatedAt = versions[i].CreatedAt.In(s.loc)
	}
	return versions, nil
}

// readable memastikan surat ada dan user adalah drafter atau peserta (selain itu ErrLetterNotFound).
func (s *Service) readable(ctx context.Context, user *auth.CurrentUser, letterID string) (*LetterLock, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	l, err := s.reader.FindByUUID(ctx, user.Company.Code, id)
	if err != nil {
		return nil, notFoundAs(err)
	}
	if l.DrafterID != user.ID {
		isParticipant, err := s.reader.IsParticipant(ctx, l.ID, user.ID)
		if err != nil {
			return nil, err
		}
		if !isParticipant {
			return nil, ErrLetterNotFound
		}
	}
	return l, nil
}

// involves: user adalah drafter, reviewer, atau approver surat.
func involves(drafterID, approverID int64, reviewers []int64, userID int64) bool {
	return userID == drafterID || userID == approverID || slices.Contains(reviewers, userID)
}

// documentEditorRole adalah peran user yang boleh mengubah naskah saat ini, atau "" bila tidak boleh.
func documentEditorRole(l *LetterLock, reviewers []int64, userID int64) string {
	switch {
	case l.DrafterID == userID && isEditable(l.Status):
		return RoleDrafter
	case (l.Status == StatusInReview || l.Status == StatusReturnedToReviewer) &&
		l.AssigneeID == userID && slices.Contains(reviewers, userID):
		return RoleReviewer
	}
	return ""
}

// parseDocument memvalidasi file naskah: .docx (arsip ZIP), tidak kosong, maksimal
// MaxDocumentBytes. Mengembalikan nama file tanpa path.
func parseDocument(fileName string, data []byte) (string, error) {
	name := path.Base(strings.ReplaceAll(strings.TrimSpace(fileName), "\\", "/"))
	if name == "." || name == "/" {
		name = ""
	}
	if !strings.EqualFold(path.Ext(name), ".docx") {
		return "", apperr.Invalid("Naskah wajib berupa file .docx")
	}
	if _, err := optionalText(&name, maxFileNameLength, "Nama file"); err != nil {
		return "", err
	}
	switch {
	case len(data) == 0:
		return "", apperr.Invalid("File naskah kosong")
	case len(data) > MaxDocumentBytes:
		return "", apperr.Invalid(fmt.Sprintf("Ukuran naskah maksimal %d MB", MaxDocumentBytes>>20))
	case !bytes.HasPrefix(data, zipSignature):
		return "", apperr.Invalid("File bukan dokumen .docx yang valid")
	}
	return name, nil
}
