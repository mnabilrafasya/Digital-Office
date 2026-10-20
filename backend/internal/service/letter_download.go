package service

import (
	"context"
	"errors"
	"slices"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/storage"
)

var (
	errAttachmentNotFound = &notFoundError{"Lampiran tidak ditemukan"}
	errVersionNotFound    = &notFoundError{"Versi dokumen tidak ditemukan"}
)

// notFoundError adalah 404 dengan pesan selain "Surat tidak ditemukan".
type notFoundError struct{ msg string }

func (e *notFoundError) Error() string { return e.msg }

// AttachmentDownload membuat URL unduh lampiran (berlaku DOWNLOAD_URL_TTL).
func (s *LetterService) AttachmentDownload(ctx context.Context, user *model.CurrentUser, letterUUID, attachmentID string) (*model.DownloadLink, error) {
	lc, err := loadVisible(ctx, s.read, user, letterUUID, false)
	if err != nil {
		return nil, err
	}
	list, err := parseAttachments(lc.letter.AttachmentsJSON)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(list, func(a model.Attachment) bool { return a.ID == attachmentID })
	if i < 0 {
		return nil, errAttachmentNotFound
	}
	return s.downloadLink(ctx, storage.AttachmentKey(letterUUID, attachmentID), list[i].FileName)
}

// Versions mengambil daftar versi dokumen surat.
func (s *LetterService) Versions(ctx context.Context, user *model.CurrentUser, letterUUID string) ([]model.VersionView, error) {
	lc, err := loadVisible(ctx, s.read, user, letterUUID, false)
	if err != nil {
		return nil, err
	}
	rows, err := s.read.Versions.List(ctx, lc.letter.ID)
	if err != nil {
		return nil, err
	}
	out := make([]model.VersionView, 0, len(rows))
	for _, v := range rows {
		out = append(out, s.versionView(v))
	}
	return out, nil
}

// VersionDownload membuat URL unduh satu versi dokumen.
func (s *LetterService) VersionDownload(ctx context.Context, user *model.CurrentUser, letterUUID string, number int) (*model.DownloadLink, error) {
	lc, err := loadVisible(ctx, s.read, user, letterUUID, false)
	if err != nil {
		return nil, err
	}
	v, err := s.read.Versions.GetByNumber(ctx, lc.letter.ID, number)
	if isNotFound(err) {
		return nil, errVersionNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.downloadLink(ctx, storage.VersionKey(letterUUID, v.ObjectKey), v.FileName)
}

func (s *LetterService) downloadLink(ctx context.Context, key, fileName string) (*model.DownloadLink, error) {
	url, exp, err := s.files.PresignDownload(ctx, key, fileName, s.cfg.DownloadURLTTL)
	if err != nil {
		return nil, err
	}
	return &model.DownloadLink{URL: url, FileName: fileName, ExpiresAt: s.formatTime(exp)}, nil
}

// IsNotFound mengembalikan true untuk semua error "tidak ditemukan" dari service, beserta pesannya.
func IsNotFound(err error) (string, bool) {
	var nf *notFoundError
	if errors.As(err, &nf) {
		return nf.msg, true
	}
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound.Error(), true
	}
	return "", false
}
