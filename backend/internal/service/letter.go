package service

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
)

// LetterStore adalah query surat yang dibutuhkan LetterService.
type LetterStore interface {
	Create(ctx context.Context, n repository.NewLetter) (string, error)
	LockByUUID(ctx context.Context, companyCode, uuid string) (*model.LetterLock, error)
	IsParticipant(ctx context.Context, letterID, userID int64) (bool, error)
	UpdateDraft(ctx context.Context, letterID int64, d *model.LetterDraft) error
	FindDetail(ctx context.Context, companyCode, uuid string) (*model.LetterDetail, error)
}

// LetterRefs adalah query untuk memeriksa data yang dirujuk surat dan daftar alias.
type LetterRefs interface {
	LookupCategories(ctx context.Context, companyCode string, ids []int64) (map[int64]string, error)
	ActiveUnitIDs(ctx context.Context, companyCode string, ids []int64) (map[int64]bool, error)
	EligibleUserIDs(ctx context.Context, companyCode, permissionCode string, ids []int64) (map[int64]bool, error)
	TemplateLetterType(ctx context.Context, companyCode string, templateID int64) (*int64, error)
	ListAliases(ctx context.Context, companyCode string) ([]model.Alias, error)
}

// LetterService mengelola draf surat. Semua surat otomatis disaring ke perusahaan user yang login.
type LetterService struct {
	tx       db.TxRunner
	newStore func(q db.DBTX) LetterStore // repository di atas transaksi
	reader   LetterStore                 // repository di luar transaksi (baca saja)
	refs     LetterRefs
	loc      *time.Location
}

// NewLetterService membuat LetterService. newStore membuat repository surat di atas koneksi atau
// transaksi; conn dipakai untuk query baca. loc adalah zona waktu yang dikirim ke klien.
func NewLetterService(tx db.TxRunner, conn db.DBTX, newStore func(q db.DBTX) LetterStore, refs LetterRefs, loc *time.Location) *LetterService {
	return &LetterService{tx: tx, newStore: newStore, reader: newStore(conn), refs: refs, loc: loc}
}

// Aliases mengambil alias penerima (level jabatan) yang punya minimal 1 anggota.
func (s *LetterService) Aliases(ctx context.Context, user *model.CurrentUser) ([]model.Alias, error) {
	return s.refs.ListAliases(ctx, user.Company.Code)
}

// Create membuat draf surat baru. Unit pengirim dan perusahaan diambil dari user yang login.
func (s *LetterService) Create(ctx context.Context, user *model.CurrentUser, in LetterInput) (*model.LetterDetail, error) {
	d, err := s.prepareDraft(ctx, user, in)
	if err != nil {
		return nil, err
	}
	var id string
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		id, err = s.newStore(q).Create(ctx, repository.NewLetter{
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
	return s.Get(ctx, user, id)
}

// Get mengambil detail surat. Hanya drafter dan peserta (reviewer/approver) yang boleh melihat;
// selain itu dijawab ErrLetterNotFound.
func (s *LetterService) Get(ctx context.Context, user *model.CurrentUser, letterID string) (*model.LetterDetail, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	d, err := s.reader.FindDetail(ctx, user.Company.Code, id)
	if errors.Is(err, repository.ErrNotFound) || (err == nil && !d.IsParticipant(user.ID)) {
		return nil, ErrLetterNotFound
	}
	if err != nil {
		return nil, err
	}
	s.decorate(d, user)
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
		store := s.newStore(q)
		l, err := store.LockByUUID(ctx, user.Company.Code, id)
		if errors.Is(err, repository.ErrNotFound) {
			return ErrLetterNotFound
		}
		if err != nil {
			return err
		}
		if l.DrafterID != user.ID {
			isParticipant, err := store.IsParticipant(ctx, l.ID, user.ID)
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
		return store.UpdateDraft(ctx, l.ID, d)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, user, id)
}

// prepareDraft memeriksa format input lalu memastikan semua data yang dirujuk valid.
func (s *LetterService) prepareDraft(ctx context.Context, user *model.CurrentUser, in LetterInput) (*model.LetterDraft, error) {
	d, err := parseDraft(in, user.ID)
	if err != nil {
		return nil, err
	}
	if err := s.checkRefs(ctx, user.Company.Code, d); err != nil {
		return nil, err
	}
	return d, nil
}

// decorate melengkapi detail untuk user yang melihat: zona waktu dan tombol yang boleh ditampilkan.
func (s *LetterService) decorate(d *model.LetterDetail, user *model.CurrentUser) {
	d.CreatedAt = d.CreatedAt.In(s.loc)
	d.UpdatedAt = d.UpdatedAt.In(s.loc)
	if d.LetterDate != nil {
		t := d.LetterDate.In(s.loc)
		d.LetterDate = &t
	}
	d.AllowedActions = []string{}
	if d.Drafter.ID == user.ID && isEditable(d.Status) {
		d.AllowedActions = append(d.AllowedActions, ActionEdit)
	}
}

// ActionEdit adalah tombol simpan/ubah draf di allowedActions.
const ActionEdit = "EDIT"

// isEditable: isi surat hanya bisa diubah drafter saat masih draf atau dikembalikan kepadanya.
func isEditable(status string) bool {
	return status == model.StatusDraft || status == model.StatusReturnedToDrafter
}

// parseLetterID memvalidasi uuid surat dari URL dan mengubahnya ke bentuk baku (huruf kecil).
func parseLetterID(raw string) (string, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", false
	}
	return id.String(), true
}
