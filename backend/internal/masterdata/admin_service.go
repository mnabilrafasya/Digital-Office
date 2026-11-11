package masterdata

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

var (
	errLookupNotFound  = &apperr.NotFoundError{Message: "Data referensi tidak ditemukan"}
	errLookupCodeTaken = &apperr.InvalidStateError{Message: "Kode sudah dipakai pada kategori ini"}
	lookupCodePattern  = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_.-]{0,29}$`)
)

// AdminService mengelola referensi persuratan (LETTER_TYPE dan CLASSIFICATION) khusus perusahaan.
type AdminService struct {
	tx   db.TxRunner
	conn db.DBTX
}

// NewAdminService membuat AdminService.
func NewAdminService(tx db.TxRunner, conn db.DBTX) *AdminService {
	return &AdminService{tx: tx, conn: conn}
}

// Hanya jenis surat dan klasifikasi yang dapat dikelola; enum sifat, kecepatan, dan status tetap terkunci.
func editableCategory(category string) error {
	if category != "LETTER_TYPE" && category != "CLASSIFICATION" {
		return apperr.Invalid("Kategori wajib LETTER_TYPE atau CLASSIFICATION")
	}
	return nil
}

func parseLookup(in LookupInput, create bool) (LookupInput, error) {
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	in.Note = strings.TrimSpace(in.Note)

	if create && !lookupCodePattern.MatchString(in.Code) {
		return in, apperr.Invalid("Kode wajib 1–30 huruf besar, angka, titik, garis bawah, atau tanda hubung")
	}
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 100 {
		return in, apperr.Invalid("Nama wajib diisi, maksimal 100 karakter")
	}
	if utf8.RuneCountInString(in.Note) > 2000 {
		return in, apperr.Invalid("Keterangan maksimal 2000 karakter")
	}
	if in.SortOrder < 0 || in.SortOrder > 9999 {
		return in, apperr.Invalid("Urutan wajib antara 0 dan 9999")
	}
	return in, nil
}

func manageableLookup(v *AdminLookup) error {
	if v.ReadOnly {
		return &apperr.InvalidStateError{
			Message: "Referensi standar sistem tidak dapat diubah. Tambahkan data khusus perusahaan bila diperlukan.",
		}
	}
	return nil
}

// List mengambil daftar lookup standar global beserta data khusus perusahaan user.
func (s *AdminService) List(ctx context.Context, user *auth.CurrentUser, category, status, search string) ([]AdminLookup, error) {
	if !user.HasPermission(auth.OrgManage) {
		return nil, apperr.ErrForbidden
	}
	if err := editableCategory(category); err != nil {
		return nil, err
	}
	if status != "" && status != "ACTIVE" && status != "INACTIVE" {
		return nil, apperr.Invalid("Status wajib ACTIVE atau INACTIVE")
	}
	search, err := db.NormalizeSearch(search)
	if err != nil {
		return nil, err
	}
	return (AdminRepository{s.conn}).List(ctx, user.Company.Code, category, status, search)
}

// Save menambah atau memperbarui lookup perusahaan user.
func (s *AdminService) Save(ctx context.Context, user *auth.CurrentUser, category, rawID string, in LookupInput) (*AdminLookup, error) {
	if !user.HasPermission(auth.OrgManage) {
		return nil, apperr.ErrForbidden
	}
	if err := editableCategory(category); err != nil {
		return nil, err
	}
	create := rawID == ""
	in, err := parseLookup(in, create)
	if err != nil {
		return nil, err
	}

	var id int64
	if !create {
		id, err = strconv.ParseInt(rawID, 10, 64)
		if err != nil || id <= 0 {
			return nil, errLookupNotFound
		}
	}

	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := AdminRepository{q}
		var before *AdminLookup
		action := "LOOKUP_CREATE"

		if create {
			var e error
			id, e = repo.Insert(ctx, user.Company.Code, category, in)
			if e != nil {
				return e
			}
		} else {
			old, e := repo.Find(ctx, user.Company.Code, category, id, true)
			if e != nil {
				return e
			}
			if e = manageableLookup(old); e != nil {
				return e
			}
			before = old
			if e = repo.Update(ctx, user.Company.Code, id, in); e != nil {
				return e
			}
			action = "LOOKUP_UPDATE"
		}

		after, e := repo.Find(ctx, user.Company.Code, category, id, false)
		if e != nil {
			return e
		}
		return recordLookup(ctx, q, user, category, id, action, before, after)
	})
	if err != nil {
		return nil, err
	}
	return (AdminRepository{s.conn}).Find(ctx, user.Company.Code, category, id, false)
}

// SetStatus mengaktifkan atau menonaktifkan lookup perusahaan user.
func (s *AdminService) SetStatus(ctx context.Context, user *auth.CurrentUser, category, rawID string, active *bool) (*AdminLookup, error) {
	if !user.HasPermission(auth.OrgManage) {
		return nil, apperr.ErrForbidden
	}
	if err := editableCategory(category); err != nil {
		return nil, err
	}
	if active == nil {
		return nil, apperr.Invalid("active wajib diisi")
	}
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return nil, errLookupNotFound
	}

	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := AdminRepository{q}
		old, e := repo.Find(ctx, user.Company.Code, category, id, true)
		if e != nil {
			return e
		}
		if e = manageableLookup(old); e != nil {
			return e
		}
		if old.IsActive == *active {
			return nil
		}
		if e = repo.Status(ctx, user.Company.Code, id, *active); e != nil {
			return e
		}
		after := *old
		after.IsActive = *active
		return recordLookup(ctx, q, user, category, id, "LOOKUP_STATUS", old, &after)
	})
	if err != nil {
		return nil, err
	}
	return (AdminRepository{s.conn}).Find(ctx, user.Company.Code, category, id, false)
}

// Delete menghapus (soft-delete) lookup khusus perusahaan user.
func (s *AdminService) Delete(ctx context.Context, user *auth.CurrentUser, category, rawID string) error {
	if !user.HasPermission(auth.OrgManage) {
		return apperr.ErrForbidden
	}
	if err := editableCategory(category); err != nil {
		return err
	}
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return errLookupNotFound
	}

	return s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := AdminRepository{q}
		old, e := repo.Find(ctx, user.Company.Code, category, id, true)
		if e != nil {
			return e
		}
		if e = manageableLookup(old); e != nil {
			return e
		}
		if e = repo.Delete(ctx, user.Company.Code, id); e != nil {
			return e
		}
		return recordLookup(ctx, q, user, category, id, "LOOKUP_DELETE", old, nil)
	})
}

func recordLookup(ctx context.Context, q db.DBTX, user *auth.CurrentUser, category string, id int64, action string, before, after *AdminLookup) error {
	return audit.NewRepository(q).Record(ctx, audit.Entry{
		CompanyCode: user.Company.Code,
		EntityType:  "LOOKUP",
		EntityID:    id,
		Action:      action,
		ActorID:     user.ID,
		ActorName:   user.Name,
		Snapshot: struct {
			Category string       `json:"category"`
			Before   *AdminLookup `json:"before,omitempty"`
			After    *AdminLookup `json:"after,omitempty"`
		}{
			Category: category,
			Before:   before,
			After:    after,
		},
	})
}
