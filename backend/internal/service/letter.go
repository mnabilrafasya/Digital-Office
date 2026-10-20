package service

import (
	"bytes"
	"context"
	"errors"
	"log"
	"slices"
	"time"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/storage"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/workflow"
)

// LetterService menangani draf surat, alur persetujuan, serta lampiran dan versi dokumen.
type LetterService struct {
	tx       db.TxRunner
	repos    LetterReposFactory
	read     LetterRepos
	files    FileStore
	capacity CapacityChecker
	uploads  UploadTokenService
	cfg      LetterConfig
	now      func() time.Time
}

// NewLetterService membuat LetterService.
func NewLetterService(d LetterDeps) *LetterService {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Config.Location == nil {
		d.Config.Location = time.UTC
	}
	return &LetterService{
		tx: d.Tx, repos: d.Repos, read: d.Read, files: d.Files, capacity: d.Capacity,
		uploads: d.Uploads, cfg: d.Config, now: d.Now,
	}
}

// LetterResult adalah detail surat terbaru beserta ROWVERSION-nya (untuk header ETag).
type LetterResult struct {
	Detail     *model.LetterDetail
	RowVersion []byte
}

// letterCtx adalah surat yang sudah lolos pemeriksaan akses beserta pesertanya.
type letterCtx struct {
	letter *model.Letter
	parts  []model.Participant
}

func (lc *letterCtx) state() workflow.State {
	l := lc.letter
	reviewers, approver := splitParticipants(lc.parts)
	return workflow.State{
		Status: workflow.Status(l.Status), DrafterID: l.DrafterID,
		AssigneeID: deref(l.AssigneeID), ForwardedByID: deref(l.ForwardedByID),
		Reviewers: reviewers, ApproverID: approver,
	}
}

func (lc *letterCtx) isDrafter(u *model.CurrentUser) bool { return lc.letter.DrafterID == u.ID }

func isNotFound(err error) bool { return errors.Is(err, repository.ErrNotFound) }

// loadVisible mengambil surat yang boleh dilihat user: drafter atau peserta (reviewer/approver),
// di perusahaan user. Selain itu ErrNotFound, supaya keberadaan surat tidak bocor.
// lock=true mengunci baris surat (wajib di dalam transaksi).
func loadVisible(ctx context.Context, r LetterRepos, user *model.CurrentUser, uuid string, lock bool) (*letterCtx, error) {
	get := r.Letters.Get
	if lock {
		get = r.Letters.GetForUpdate
	}
	l, err := get(ctx, user.Company.Code, uuid)
	if isNotFound(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	parts, err := r.Participants.List(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	lc := &letterCtx{letter: l, parts: parts}
	if !lc.isDrafter(user) && !slices.ContainsFunc(parts, func(p model.Participant) bool { return p.UserID == user.ID }) {
		return nil, ErrNotFound
	}
	return lc, nil
}

// requireEditable memastikan user adalah drafter dan surat masih bisa diedit.
func requireEditable(lc *letterCtx, user *model.CurrentUser) error {
	if !lc.isDrafter(user) {
		return ErrForbidden
	}
	if !workflow.IsEditable(workflow.Status(lc.letter.Status)) {
		return conflict("Surat tidak dapat diubah karena sedang diproses atau sudah terkirim. Muat ulang halaman untuk melihat status terbaru.")
	}
	return nil
}

func checkVersion(lc *letterCtx, ifMatch []byte) error {
	if !bytes.Equal(ifMatch, lc.letter.RowVersion) {
		return &VersionConflictError{Current: lc.letter.RowVersion}
	}
	return nil
}

// Create membuat draf baru milik user (unit pengirim = unit user saat ini).
func (s *LetterService) Create(ctx context.Context, user *model.CurrentUser, in LetterInput) (*LetterResult, error) {
	d, err := normalizeInput(in)
	if err != nil {
		return nil, err
	}

	var uuid string
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		r := s.repos(q)
		if err := validateDraft(ctx, r, user, d, user.ID, false); err != nil {
			return err
		}
		id, newUUID, err := r.Letters.Create(ctx, model.NewLetter{
			CompanyCode: user.Company.Code, SenderUnitID: user.Unit.ID, DrafterID: user.ID, Content: d.content,
		})
		if err != nil {
			return err
		}
		uuid = newUUID
		if err := r.Participants.Insert(ctx, id, d.participants()); err != nil {
			return err
		}
		return r.Recipients.Insert(ctx, id, d.recipients())
	})
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, user, uuid, false)
}

// Update mengganti seluruh isi draf (PUT). Wajib If-Match. Di status RETURNED_TO_DRAFTER isi
// harus lengkap, karena CHECK ck_letters_complete_after_draft berlaku di luar status DRAFT.
func (s *LetterService) Update(ctx context.Context, user *model.CurrentUser, uuid string, ifMatch []byte, in LetterInput) (*LetterResult, error) {
	d, err := normalizeInput(in)
	if err != nil {
		return nil, err
	}

	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		r := s.repos(q)
		lc, err := loadVisible(ctx, r, user, uuid, true)
		if err != nil {
			return err
		}
		if err := requireEditable(lc, user); err != nil {
			return err
		}
		if err := checkVersion(lc, ifMatch); err != nil {
			return err
		}
		strict := lc.letter.Status == string(workflow.StatusReturnedToDrafter)
		if err := validateDraft(ctx, r, user, d, lc.letter.DrafterID, strict); err != nil {
			return err
		}
		return saveDraft(ctx, r, lc, d)
	})
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, user, uuid, false)
}

// saveDraft menyimpan isi surat serta peserta dan penerima secara diff (baris yang tidak berubah
// dibiarkan), sehingga simpan otomatis tidak menumpuk baris sampah.
func saveDraft(ctx context.Context, r LetterRepos, lc *letterCtx, d draft) error {
	id := lc.letter.ID
	if err := r.Letters.UpdateContent(ctx, id, d.content); err != nil {
		return err
	}

	delParts, addParts := diffParticipants(lc.parts, d.participants())
	if err := r.Participants.SoftDelete(ctx, delParts); err != nil {
		return err
	}
	if err := r.Participants.Insert(ctx, id, addParts); err != nil {
		return err
	}

	existing, err := r.Recipients.List(ctx, id)
	if err != nil {
		return err
	}
	delRecips, addRecips := diffRecipients(existing, d.recipients())
	if err := r.Recipients.SoftDelete(ctx, delRecips); err != nil {
		return err
	}
	return r.Recipients.Insert(ctx, id, addRecips)
}

// Delete menghapus (soft delete) draf yang belum pernah dikirim. Wajib If-Match.
// File lampiran dan versinya dihapus dari storage setelah commit (best effort).
func (s *LetterService) Delete(ctx context.Context, user *model.CurrentUser, uuid string, ifMatch []byte) error {
	var objectKeys []string
	err := s.tx.InTx(ctx, func(q db.DBTX) error {
		r := s.repos(q)
		lc, err := loadVisible(ctx, r, user, uuid, true)
		if err != nil {
			return err
		}
		if !lc.isDrafter(user) {
			return ErrForbidden
		}
		submitted, err := r.Actions.Exists(ctx, lc.letter.ID, string(workflow.ActionSubmit))
		if err != nil {
			return err
		}
		if lc.letter.Status != string(workflow.StatusDraft) || submitted {
			return conflict("Hanya draf yang belum pernah dikirim yang dapat dihapus.")
		}
		if err := checkVersion(lc, ifMatch); err != nil {
			return err
		}
		if err := r.Letters.SoftDelete(ctx, lc.letter.ID); err != nil {
			return err
		}

		attachments, err := parseAttachments(lc.letter.AttachmentsJSON)
		if err != nil {
			return err
		}
		for _, a := range attachments {
			objectKeys = append(objectKeys, storage.AttachmentKey(uuid, a.ID))
		}
		versionKeys, err := r.Versions.ObjectKeys(ctx, lc.letter.ID)
		if err != nil {
			return err
		}
		for _, k := range versionKeys {
			objectKeys = append(objectKeys, storage.VersionKey(uuid, k))
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.removeObjects(ctx, objectKeys)
	return nil
}

// removeObjects menghapus object setelah transaksi selesai; kegagalan hanya dicatat di log.
func (s *LetterService) removeObjects(ctx context.Context, keys []string) {
	for _, k := range keys {
		if err := s.files.Remove(context.WithoutCancel(ctx), k); err != nil {
			log.Printf("PERINGATAN: object %s gagal dihapus dari storage: %v", k, err)
		}
	}
}

// Get mengambil detail surat. Bila user adalah assignee yang belum membaca sejak transisi
// terakhir, aksi READ dicatat (baris letters tidak berubah, jadi ETag tetap).
func (s *LetterService) Get(ctx context.Context, user *model.CurrentUser, uuid string) (*LetterResult, error) {
	return s.detail(ctx, user, uuid, true)
}

func (s *LetterService) detail(ctx context.Context, user *model.CurrentUser, uuid string, recordRead bool) (*LetterResult, error) {
	r := s.read
	lc, err := loadVisible(ctx, r, user, uuid, false)
	if err != nil {
		return nil, err
	}
	if recordRead {
		if err := s.recordRead(ctx, r, user, lc); err != nil {
			return nil, err
		}
	}
	d, err := s.buildDetail(ctx, r, user, lc)
	if err != nil {
		return nil, err
	}
	return &LetterResult{Detail: d, RowVersion: lc.letter.RowVersion}, nil
}

// transitionActions adalah aksi yang menandai perpindahan tahap (acuan "sejak transisi terakhir").
var transitionActions = []string{
	string(workflow.ActionSubmit), string(workflow.ActionApprove),
	string(workflow.ActionReturn), string(workflow.ActionSign),
}

func (s *LetterService) recordRead(ctx context.Context, r LetterRepos, user *model.CurrentUser, lc *letterCtx) error {
	l := lc.letter
	if l.AssigneeID == nil || *l.AssigneeID != user.ID {
		return nil
	}
	var role string
	switch workflow.Status(l.Status) {
	case workflow.StatusInReview, workflow.StatusReturnedToReviewer:
		role = model.RoleReviewer
	case workflow.StatusInApproval:
		role = model.RoleApprover
	case workflow.StatusReturnedToDrafter:
		role = model.RoleDrafter
	default:
		return nil
	}
	return r.Actions.RecordReadOnce(ctx, l.ID, user.ID, role, l.Status, transitionActions)
}

// formatTime menulis waktu RFC 3339 dalam zona waktu bisnis (APP_TIMEZONE).
func (s *LetterService) formatTime(t time.Time) string {
	return t.In(s.cfg.Location).Format(time.RFC3339)
}
