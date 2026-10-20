package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/auth"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/storage"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/workflow"
)

const (
	maxFileNameLength = 255
	maxExtLength      = 10
	defaultFileName   = "berkas"
	defaultExtension  = "bin"
	defaultMediaType  = "application/octet-stream"
)

// uploadRole menentukan peran pengunggah (DRAFTER/REVIEWER) atau alasan penolakan:
//   - lampiran: hanya drafter, saat DRAFT atau RETURNED_TO_DRAFTER
//   - versi dokumen: drafter saat DRAFT/RETURNED_TO_DRAFTER, atau reviewer yang sedang menjadi
//     assignee saat IN_REVIEW/RETURNED_TO_REVIEWER
func uploadRole(kind string, lc *letterCtx, user *model.CurrentUser) (string, error) {
	st := workflow.Status(lc.letter.Status)
	isDrafter := lc.isDrafter(user)

	if isDrafter && workflow.IsEditable(st) {
		return model.RoleDrafter, nil
	}
	if kind == model.UploadVersion &&
		(st == workflow.StatusInReview || st == workflow.StatusReturnedToReviewer) &&
		deref(lc.letter.AssigneeID) == user.ID &&
		workflow.ReviewerStep(lc.state(), user.ID) > 0 &&
		user.HasPermission(permission.LetterReview) {
		return model.RoleReviewer, nil
	}

	isReviewer := workflow.ReviewerStep(lc.state(), user.ID) > 0
	switch {
	case kind == model.UploadAttachment && !isDrafter:
		return "", ErrForbidden
	case kind == model.UploadVersion && !isDrafter && !isReviewer:
		return "", ErrForbidden // approver tidak mengedit isi surat
	case kind == model.UploadAttachment:
		return "", conflict("Lampiran hanya dapat diubah saat surat masih draf atau dikembalikan ke pembuat.")
	default:
		return "", conflict("Versi dokumen tidak dapat diunggah pada tahap ini. Muat ulang halaman untuk melihat status terbaru.")
	}
}

// SanitizeFileName membuang path, karakter kontrol, dan karakter \ / : * ? " < > |, lalu memotong
// ke 255 karakter. Nama kosong menjadi "berkas". Nama file tidak pernah dipakai sebagai object key.
func SanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	name = path.Base(name)
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`\/:*?"<>|`, r) || r == utf8.RuneError {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "." || name == ".." {
		name = ""
	}
	if utf8.RuneCountInString(name) > maxFileNameLength {
		name = strings.TrimSpace(string([]rune(name)[:maxFileNameLength]))
	}
	if name == "" {
		return defaultFileName
	}
	return name
}

// FileExtension mengambil ekstensi huruf kecil tanpa titik (maksimal 10, kosong menjadi "bin").
// Hanya metadata; bukan pengecekan jenis file.
func FileExtension(name string) string {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))
	if utf8.RuneCountInString(ext) > maxExtLength {
		ext = string([]rune(ext)[:maxExtLength])
	}
	if ext == "" {
		return defaultExtension
	}
	return ext
}

// RequestUpload memberi izin unggah langsung ke MinIO (presigned POST) beserta upload token.
func (s *LetterService) RequestUpload(ctx context.Context, user *model.CurrentUser, letterUUID, kind, fileName string, size int64) (*model.UploadTicket, error) {
	if size < 1 {
		return nil, invalid("Ukuran file wajib diisi dan lebih dari 0 byte")
	}
	if size > s.cfg.MaxUploadBytes {
		return nil, &FileTooLargeError{MaxBytes: s.cfg.MaxUploadBytes}
	}
	name := SanitizeFileName(fileName)

	lc, err := loadVisible(ctx, s.read, user, letterUUID, false)
	if err != nil {
		return nil, err
	}
	if _, err := uploadRole(kind, lc, user); err != nil {
		return nil, err
	}
	if !s.capacity.HasRoom(ctx, size) {
		return nil, ErrStorageFull
	}

	objectID := uuid.NewString()
	up, err := s.files.PresignUpload(ctx, storage.StagingKey(objectID), s.cfg.MaxUploadBytes, s.cfg.UploadURLTTL)
	if err != nil {
		return nil, err
	}
	token, _, err := s.uploads.Issue(auth.UploadClaims{
		UserID: user.ID, LetterUUID: letterUUID, Kind: kind, ObjectID: objectID, FileName: name,
	})
	if err != nil {
		return nil, err
	}
	return &model.UploadTicket{
		UploadURL: up.URL, Method: "POST", Fields: up.Fields, UploadToken: token,
		ExpiresAt: s.formatTime(up.ExpiresAt), MaxSizeBytes: s.cfg.MaxUploadBytes,
	}, nil
}

// FileResult adalah hasil konfirmasi unggah. Created=false berarti konfirmasi ulang (idempoten).
type FileResult struct {
	Attachment *model.AttachmentView
	Version    *model.VersionView
	Created    bool
	RowVersion []byte
}

// CompleteUpload mengonfirmasi unggahan: verifikasi token, cek object staging, salin ke lokasi
// final di sisi server, hitung SHA-256, lalu catat di database. Memanggil dua kali dengan token
// yang sama mengembalikan data yang sama tanpa duplikat.
func (s *LetterService) CompleteUpload(ctx context.Context, user *model.CurrentUser, letterUUID, kind, token string) (*FileResult, error) {
	claims, err := s.uploads.Verify(token, user.ID, letterUUID, kind)
	if err != nil {
		return nil, invalid("uploadToken tidak valid atau sudah kedaluwarsa. Minta izin unggah baru.")
	}

	// Konfirmasi ulang: kembalikan data yang sudah tercatat
	if res, err := s.existingUpload(ctx, user, letterUUID, kind, claims.ObjectID); err != nil || res != nil {
		return res, err
	}
	lc, err := loadVisible(ctx, s.read, user, letterUUID, false)
	if err != nil {
		return nil, err
	}
	if _, err := uploadRole(kind, lc, user); err != nil {
		return nil, err
	}

	stagingKey := storage.StagingKey(claims.ObjectID)
	finalKey := storage.AttachmentKey(letterUUID, claims.ObjectID)
	if kind == model.UploadVersion {
		finalKey = storage.VersionKey(letterUUID, claims.ObjectID)
	}

	obj, err := s.files.Stat(ctx, stagingKey)
	if errors.Is(err, storage.ErrObjectNotFound) {
		return nil, invalid("Berkas belum diunggah ke storage atau sudah kedaluwarsa. Ulangi unggahan.")
	}
	if err != nil {
		return nil, err
	}
	if obj.Size > s.cfg.MaxUploadBytes {
		s.removeObjects(ctx, []string{stagingKey})
		return nil, &FileTooLargeError{MaxBytes: s.cfg.MaxUploadBytes}
	}
	if obj.ContentType == "" {
		obj.ContentType = defaultMediaType
	}
	if err := s.files.Copy(ctx, stagingKey, finalKey); err != nil {
		return nil, err
	}
	checksum, err := s.files.Checksum(ctx, finalKey)
	if err != nil {
		return nil, err
	}

	var res *FileResult
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		var err error
		res, err = s.recordUpload(ctx, s.repos(q), user, letterUUID, kind, claims, obj, checksum)
		return err
	})
	if err != nil {
		// Kompensasi: hapus object final, kecuali ternyata sudah tercatat oleh permintaan lain
		if again, _ := s.existingUpload(ctx, user, letterUUID, kind, claims.ObjectID); again == nil {
			s.removeObjects(ctx, []string{finalKey})
		}
		return nil, err
	}
	s.removeObjects(ctx, []string{stagingKey})

	latest, err := s.read.Letters.Get(ctx, user.Company.Code, letterUUID)
	if err != nil {
		return nil, err
	}
	res.RowVersion = latest.RowVersion
	return res, nil
}

// recordUpload mencatat unggahan dalam transaksi, dengan baris surat terkunci (UPDLOCK) sehingga
// penambahan lampiran paralel tidak saling menimpa JSON attachments.
func (s *LetterService) recordUpload(ctx context.Context, r LetterRepos, user *model.CurrentUser, letterUUID, kind string,
	claims auth.UploadClaims, obj model.StoredObject, checksum string) (*FileResult, error) {
	lc, err := loadVisible(ctx, r, user, letterUUID, true)
	if err != nil {
		return nil, err
	}
	role, err := uploadRole(kind, lc, user)
	if err != nil {
		return nil, err
	}
	l := lc.letter

	if kind == model.UploadAttachment {
		list, err := parseAttachments(l.AttachmentsJSON)
		if err != nil {
			return nil, err
		}
		if i := slices.IndexFunc(list, func(a model.Attachment) bool { return a.ID == claims.ObjectID }); i >= 0 {
			v := s.attachmentView(list[i], user)
			return &FileResult{Attachment: &v}, nil // sudah dicatat permintaan sebelumnya
		}
		a := model.Attachment{
			ID: claims.ObjectID, FileName: claims.FileName, Size: obj.Size, ContentType: obj.ContentType,
			Checksum: checksum, UploadedAt: s.now().UTC(), UploadedBy: fmt.Sprint(user.ID),
		}
		raw, err := json.Marshal(append(list, a))
		if err != nil {
			return nil, fmt.Errorf("menyusun JSON lampiran: %w", err)
		}
		js := string(raw)
		if err := r.Letters.UpdateAttachments(ctx, l.ID, &js); err != nil {
			return nil, err
		}
		v := s.attachmentView(a, user)
		return &FileResult{Attachment: &v, Created: true}, nil
	}

	if v, err := r.Versions.GetByObjectKey(ctx, l.ID, claims.ObjectID); err == nil {
		view := s.versionView(*v)
		return &FileResult{Version: &view}, nil
	} else if !isNotFound(err) {
		return nil, err
	}
	n, err := r.Letters.IncrementVersion(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	if err := r.Versions.Insert(ctx, model.NewVersion{
		LetterID: l.ID, VersionNumber: n, ObjectKey: claims.ObjectID, FileName: claims.FileName,
		FileExtension: FileExtension(claims.FileName), FileSize: obj.Size, Checksum: checksum,
		CreatedBy: user.ID, CreatedByRole: role,
	}); err != nil {
		return nil, err
	}
	if err := r.Actions.Insert(ctx, model.ActionEntry{
		LetterID: l.ID, ActorID: user.ID, ActorRole: role, Action: "SAVE",
		FromStatus: l.Status, ToStatus: l.Status, VersionNumber: &n,
	}); err != nil {
		return nil, err
	}
	v, err := r.Versions.GetByNumber(ctx, l.ID, n)
	if err != nil {
		return nil, err
	}
	view := s.versionView(*v)
	return &FileResult{Version: &view, Created: true}, nil
}

// existingUpload mengembalikan unggahan yang sudah tercatat untuk object tertentu (nil bila belum).
func (s *LetterService) existingUpload(ctx context.Context, user *model.CurrentUser, letterUUID, kind, objectID string) (*FileResult, error) {
	lc, err := loadVisible(ctx, s.read, user, letterUUID, false)
	if err != nil {
		return nil, err
	}
	l := lc.letter
	if kind == model.UploadAttachment {
		list, err := parseAttachments(l.AttachmentsJSON)
		if err != nil {
			return nil, err
		}
		for _, a := range list {
			if a.ID == objectID {
				v := s.attachmentView(a, user)
				return &FileResult{Attachment: &v, RowVersion: l.RowVersion}, nil
			}
		}
		return nil, nil
	}
	v, err := s.read.Versions.GetByObjectKey(ctx, l.ID, objectID)
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	view := s.versionView(*v)
	return &FileResult{Version: &view, RowVersion: l.RowVersion}, nil
}

func (s *LetterService) attachmentView(a model.Attachment, uploader *model.CurrentUser) model.AttachmentView {
	return model.AttachmentView{
		ID: a.ID, FileName: a.FileName, Size: a.Size, ContentType: a.ContentType, Checksum: a.Checksum,
		UploadedAt: s.formatTime(a.UploadedAt), UploadedBy: model.PersonRef{ID: uploader.ID, Name: uploader.Name},
	}
}

func (s *LetterService) versionView(v model.Version) model.VersionView {
	return model.VersionView{
		VersionNumber: v.VersionNumber, VersionType: v.VersionType, FileName: v.FileName,
		FileExtension: v.FileExtension, Size: v.FileSize, Checksum: v.Checksum,
		CreatedBy: v.CreatedBy, CreatedByRole: v.CreatedByRole, CreatedAt: s.formatTime(v.CreatedAt),
	}
}

// DeleteAttachment menghapus lampiran (drafter, saat DRAFT/RETURNED_TO_DRAFTER). File di storage
// dihapus setelah commit (best effort). Mengembalikan ROWVERSION terbaru.
func (s *LetterService) DeleteAttachment(ctx context.Context, user *model.CurrentUser, letterUUID, attachmentID string) ([]byte, error) {
	err := s.tx.InTx(ctx, func(q db.DBTX) error {
		r := s.repos(q)
		lc, err := loadVisible(ctx, r, user, letterUUID, true)
		if err != nil {
			return err
		}
		if _, err := uploadRole(model.UploadAttachment, lc, user); err != nil {
			return err
		}
		list, err := parseAttachments(lc.letter.AttachmentsJSON)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(list, func(a model.Attachment) bool { return a.ID == attachmentID })
		if i < 0 {
			return errAttachmentNotFound
		}
		list = slices.Delete(list, i, i+1)
		var js *string
		if len(list) > 0 {
			raw, err := json.Marshal(list)
			if err != nil {
				return fmt.Errorf("menyusun JSON lampiran: %w", err)
			}
			str := string(raw)
			js = &str
		}
		return r.Letters.UpdateAttachments(ctx, lc.letter.ID, js)
	})
	if err != nil {
		return nil, err
	}
	s.removeObjects(ctx, []string{storage.AttachmentKey(letterUUID, attachmentID)})

	latest, err := s.read.Letters.Get(ctx, user.Company.Code, letterUUID)
	if err != nil {
		return nil, err
	}
	return latest.RowVersion, nil
}
