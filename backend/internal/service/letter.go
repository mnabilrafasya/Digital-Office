package service

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/workflow"
)

// ActionEdit adalah tombol simpan/ubah draf di allowedActions (aksi alur lain dari package workflow).
const ActionEdit = "EDIT"

// LetterStore adalah query surat yang dibutuhkan LetterService.
type LetterStore interface {
	// Draf
	Create(ctx context.Context, n repository.NewLetter) (string, error)
	UpdateDraft(ctx context.Context, letterID int64, d *model.LetterDraft) error
	LoadDraft(ctx context.Context, letterID int64) (*model.LetterDraft, error)

	// Akses dan alur persetujuan
	LockByUUID(ctx context.Context, companyCode, uuid string) (*model.LetterLock, error)
	FindByUUID(ctx context.Context, companyCode, uuid string) (*model.LetterLock, error)
	IsParticipant(ctx context.Context, letterID, userID int64) (bool, error)
	Participants(ctx context.Context, letterID int64) (reviewers []int64, approverID int64, err error)
	ApplyTransition(ctx context.Context, letterID, actorID int64, d workflow.Decision, note *string) error
	RecordRead(ctx context.Context, letterID, userID int64, role string) error

	// Tampilan
	FindDetail(ctx context.Context, companyCode, uuid string) (*model.LetterDetail, error)
	List(ctx context.Context, f repository.LetterFilter) ([]model.LetterListItem, int, error)
	Actions(ctx context.Context, letterID int64) ([]model.LetterAction, error)
}

// LetterRefs adalah query untuk memeriksa data yang dirujuk surat dan daftar alias.
type LetterRefs interface {
	LookupCategories(ctx context.Context, companyCode string, ids []int64) (map[int64]string, error)
	ActiveUnitIDs(ctx context.Context, companyCode string, ids []int64) (map[int64]bool, error)
	ActiveUserIDs(ctx context.Context, companyCode string, ids []int64) (map[int64]bool, error)
	EligibleUserIDs(ctx context.Context, companyCode, permissionCode string, ids []int64) (map[int64]bool, error)
	TemplateLetterType(ctx context.Context, companyCode string, templateID int64) (*int64, error)
	UnreachableRecipients(ctx context.Context, companyCode string, unitIDs, aliasIDs []int64) (units, aliases []string, err error)
	ListAliases(ctx context.Context, companyCode string) ([]model.Alias, error)
}

// AuditStore mencatat jejak audit.
type AuditStore interface {
	Record(ctx context.Context, e repository.AuditEntry) error
}

// LetterRepos adalah repository yang dibutuhkan LetterService. Letters dan Audit dibuat di atas
// koneksi atau transaksi; Refs hanya membaca dan dipakai di luar transaksi.
type LetterRepos struct {
	Letters func(q db.DBTX) LetterStore
	Audit   func(q db.DBTX) AuditStore
	Refs    LetterRefs
}

// LetterService mengelola surat: draf, alur persetujuan, daftar, dan riwayat. Semua surat
// otomatis disaring ke perusahaan user yang login.
type LetterService struct {
	tx     db.TxRunner
	repos  LetterRepos
	reader LetterStore // repository surat di luar transaksi (baca saja)
	loc    *time.Location
}

// NewLetterService membuat LetterService. conn dipakai untuk query baca di luar transaksi;
// loc adalah zona waktu yang dikirim ke klien.
func NewLetterService(tx db.TxRunner, conn db.DBTX, repos LetterRepos, loc *time.Location) *LetterService {
	return &LetterService{tx: tx, repos: repos, reader: repos.Letters(conn), loc: loc}
}

// Aliases mengambil alias penerima (level jabatan) yang punya minimal 1 anggota.
func (s *LetterService) Aliases(ctx context.Context, user *model.CurrentUser) ([]model.Alias, error) {
	return s.repos.Refs.ListAliases(ctx, user.Company.Code)
}

// Create membuat draf surat baru. Unit pengirim dan perusahaan diambil dari user yang login.
func (s *LetterService) Create(ctx context.Context, user *model.CurrentUser, in LetterInput) (*model.LetterDetail, error) {
	d, err := parseDraft(in, user.ID)
	if err != nil {
		return nil, err
	}
	if err := s.checkRefs(ctx, user.Company.Code, d); err != nil {
		return nil, err
	}

	var id string
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		var err error
		id, err = s.repos.Letters(q).Create(ctx, repository.NewLetter{
			CompanyCode:  user.Company.Code,
			SenderUnitID: user.Unit.ID,
			DrafterID:    user.ID,
			Draft:        d,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, user, id)
}

// Get mengambil detail surat untuk drafter dan peserta (selain itu ErrLetterNotFound). Kalau yang
// membuka adalah penerima tugas saat ini, aksi READ dicatat (sekali sejak transisi terakhir).
func (s *LetterService) Get(ctx context.Context, user *model.CurrentUser, letterID string) (*model.LetterDetail, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	d, err := s.detail(ctx, user, id)
	if err != nil {
		return nil, err
	}
	if role := workflow.AssigneeRole(d.Status); role != "" && d.CurrentAssignee != nil && d.CurrentAssignee.ID == user.ID {
		err := s.tx.InTx(ctx, func(q db.DBTX) error {
			return s.repos.Letters(q).RecordRead(ctx, d.InternalID, user.ID, role)
		})
		if err != nil {
			return nil, err
		}
	}
	return d, nil
}

// Update mengganti seluruh isi draf. ifMatch wajib sama dengan versi terbaru (ETag), supaya dua
// tab yang mengedit surat yang sama tidak saling menimpa.
func (s *LetterService) Update(ctx context.Context, user *model.CurrentUser, letterID string, ifMatch []byte, in LetterInput) (*model.LetterDetail, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	d, err := parseDraft(in, user.ID)
	if err != nil {
		return nil, err
	}

	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		letters := s.repos.Letters(q)
		l, err := letters.LockByUUID(ctx, user.Company.Code, id)
		if err != nil {
			return notFoundAs(err)
		}
		if l.DrafterID != user.ID {
			isParticipant, err := letters.IsParticipant(ctx, l.ID, user.ID)
			if err != nil {
				return err
			}
			if isParticipant {
				return ErrForbidden
			}
			return ErrLetterNotFound
		}
		if !isEditable(l.Status) {
			return &InvalidStateError{Message: "Surat sedang diproses dan tidak bisa diubah. Muat ulang halaman untuk melihat status terbaru."}
		}
		if !bytes.Equal(ifMatch, l.RowVersion) {
			return &VersionConflictError{RowVersion: l.RowVersion}
		}
		// Referensi diperiksa setelah akses, supaya user tanpa akses selalu mendapat 404
		if err := s.checkRefs(ctx, user.Company.Code, d); err != nil {
			return err
		}
		// Surat yang dikembalikan ke drafter pernah lengkap; CHECK ck_letters_complete_after_draft
		// juga mewajibkannya, jadi aturannya sama dengan submit.
		if l.Status == model.StatusReturnedToDrafter {
			if err := checkComplete(d); err != nil {
				return err
			}
		}
		return letters.UpdateDraft(ctx, l.ID, d)
	})
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, user, id)
}

// detail memuat detail surat (uuid sudah baku) dan memeriksa akses, tanpa mencatat READ.
func (s *LetterService) detail(ctx context.Context, user *model.CurrentUser, id string) (*model.LetterDetail, error) {
	d, err := s.reader.FindDetail(ctx, user.Company.Code, id)
	if err != nil {
		return nil, notFoundAs(err)
	}
	if !d.IsParticipant(user.ID) {
		return nil, ErrLetterNotFound
	}
	s.present(d, user)
	return d, nil
}

// present melengkapi detail untuk user yang melihat: keadaan tiap tahap, catatan pengembalian,
// tombol yang boleh ditampilkan, dan zona waktu.
func (s *LetterService) present(d *model.LetterDetail, user *model.CurrentUser) {
	state := detailState(d)
	for i := range d.Steps {
		d.Steps[i].State = workflow.StepState(state, d.Steps[i].Role, d.Steps[i].Order)
	}
	if d.Status != model.StatusReturnedToDrafter && d.Status != model.StatusReturnedToReviewer {
		d.LastReturnNote = nil
	}

	d.AllowedActions = []string{}
	if d.Drafter.ID == user.ID && isEditable(d.Status) {
		d.AllowedActions = append(d.AllowedActions, ActionEdit)
	}
	for _, a := range workflow.AllowedActions(state, user.ID) {
		// Tombol disembunyikan bila permission peran yang dibutuhkan sudah dicabut
		if perm := rolePermission(actorRoleFor(a, d.Status)); perm == "" || user.HasPermission(perm) {
			d.AllowedActions = append(d.AllowedActions, string(a))
		}
	}

	d.CreatedAt = d.CreatedAt.In(s.loc)
	d.UpdatedAt = d.UpdatedAt.In(s.loc)
	if d.LetterDate != nil {
		t := d.LetterDate.In(s.loc)
		d.LetterDate = &t
	}
}

// detailState menyusun state workflow dari detail surat.
func detailState(d *model.LetterDetail) workflow.State {
	s := workflow.State{Status: d.Status, DrafterID: d.Drafter.ID, ForwardedByID: d.ForwardedByID}
	if d.CurrentAssignee != nil {
		s.AssigneeID = d.CurrentAssignee.ID
	}
	for _, step := range d.Steps { // repository mengurutkan reviewer sesuai step
		if step.Role == model.RoleApprover {
			s.ApproverID = step.User.ID
		} else {
			s.Reviewers = append(s.Reviewers, step.User.ID)
		}
	}
	return s
}

// actorRoleFor adalah peran pelaku aksi a pada status tertentu.
func actorRoleFor(a workflow.Action, status string) string {
	if a == workflow.Submit {
		return model.RoleDrafter
	}
	return workflow.AssigneeRole(status)
}

// rolePermission adalah permission yang wajib dimiliki untuk beraksi sebagai peran tertentu.
// Drafter tidak butuh permission tambahan (cukup sebagai pembuat surat).
func rolePermission(role string) string {
	switch role {
	case model.RoleReviewer:
		return permission.LetterReview
	case model.RoleApprover:
		return permission.LetterApprove
	}
	return ""
}

// isEditable: isi surat hanya bisa diubah drafter saat masih draf atau dikembalikan kepadanya.
func isEditable(status string) bool {
	return status == model.StatusDraft || status == model.StatusReturnedToDrafter
}

// notFoundAs mengubah repository.ErrNotFound menjadi ErrLetterNotFound; error lain diteruskan.
func notFoundAs(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrLetterNotFound
	}
	return err
}

// parseLetterID memvalidasi uuid surat dari URL dan mengubahnya ke bentuk baku (huruf kecil).
func parseLetterID(raw string) (string, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", false
	}
	return id.String(), true
}
