package organisasi

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Unit dikelola admin perusahaan (ORG_MANAGE): semua unit, termasuk hasil inisiasi HC. Sinkron HC
// hanya menambah unit baru, jadi perubahan admin tidak tertimpa. Unit baru diberi kode acak (UUID),
// seperti kode unit HC.

// unitTypePattern adalah jenis unit: huruf besar, angka, spasi, dan garis bawah (contoh DEPARTEMEN).
var unitTypePattern = regexp.MustCompile(`^[A-Z0-9_ ]+$`)

var (
	errUnitParentInactive = &apperr.InvalidStateError{Message: "Unit induk tidak aktif"}
	errUnitCycle          = apperr.Invalid("Unit induk tidak boleh unit itu sendiri atau sub-unitnya")
	errUnitHasChildren    = &apperr.InvalidStateError{Message: "Unit masih punya sub-unit aktif. Nonaktifkan atau pindahkan sub-unitnya dulu."}
	errUnitHasUsers       = &apperr.InvalidStateError{Message: "Unit masih punya user aktif. Pindahkan atau nonaktifkan user-nya dulu."}
)

// Units mengambil semua unit perusahaan user (aktif dan nonaktif) untuk halaman master data.
func (s *AdminService) Units(ctx context.Context, user *auth.CurrentUser, lq ListQuery) ([]AdminUnit, error) {
	return s.unitsOf(ctx, user.Company.Code, lq)
}

// CompanyUnits mengambil unit aktif perusahaan lain, untuk SUPERADMIN memilih unit admin pertama.
func (s *AdminService) CompanyUnits(ctx context.Context, companyCode string, lq ListQuery) ([]AdminUnit, error) {
	companyCode = strings.ToUpper(strings.TrimSpace(companyCode))
	ok, err := s.reader.CompanyExists(ctx, companyCode)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errCompanyNotFound
	}
	lq.Status = "ACTIVE"
	return s.unitsOf(ctx, companyCode, lq)
}

func (s *AdminService) unitsOf(ctx context.Context, companyCode string, lq ListQuery) ([]AdminUnit, error) {
	f, err := filterOf(companyCode, lq)
	if err != nil {
		return nil, err
	}
	return s.reader.ListAdminUnits(ctx, f)
}

// CreateUnit menambah unit di perusahaan user.
func (s *AdminService) CreateUnit(ctx context.Context, user *auth.CurrentUser, in UnitInput) (*AdminUnit, error) {
	name, unitType, parentID, err := parseUnit(in)
	if err != nil {
		return nil, err
	}
	var id int64
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := s.repo(q)
		if err := s.checkParent(ctx, repo, user.Company.Code, 0, parentID); err != nil {
			return err
		}
		if id, err = repo.InsertUnit(ctx, user.Company.Code, uuid.NewString(), name, unitType, parentID); err != nil {
			return err
		}
		return s.record(ctx, q, user, user.Company.Code, audit.EntityUnit, id, audit.ActionUnitCreate, nil, in)
	})
	if err != nil {
		return nil, err
	}
	return s.reader.FindAdminUnit(ctx, user.Company.Code, id, false)
}

// UpdateUnit mengubah nama, jenis, dan induk unit.
func (s *AdminService) UpdateUnit(ctx context.Context, user *auth.CurrentUser, rawID string, in UnitInput) (*AdminUnit, error) {
	id, err := parsePathID(rawID, errUnitNotFound)
	if err != nil {
		return nil, err
	}
	name, unitType, parentID, err := parseUnit(in)
	if err != nil {
		return nil, err
	}
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := s.repo(q)
		old, err := s.lockUnit(ctx, repo, user.Company.Code, id)
		if err != nil {
			return err
		}
		if err := s.checkParent(ctx, repo, user.Company.Code, id, parentID); err != nil {
			return err
		}
		if err := repo.UpdateUnit(ctx, id, name, unitType, parentID); err != nil {
			return err
		}
		return s.record(ctx, q, user, user.Company.Code, audit.EntityUnit, id, audit.ActionUnitUpdate,
			UnitInput{Name: old.Name, UnitType: old.UnitType, ParentID: idString(old.ParentID)}, in)
	})
	if err != nil {
		return nil, err
	}
	return s.reader.FindAdminUnit(ctx, user.Company.Code, id, false)
}

// SetUnitStatus mengaktifkan/menonaktifkan unit. Nonaktif ditolak bila masih ada sub-unit aktif atau
// user aktif di unit itu; aktif ditolak bila induknya nonaktif.
func (s *AdminService) SetUnitStatus(ctx context.Context, user *auth.CurrentUser, rawID string, in StatusInput) (*AdminUnit, error) {
	id, err := parsePathID(rawID, errUnitNotFound)
	if err != nil {
		return nil, err
	}
	if in.Active == nil {
		return nil, apperr.Invalid("active wajib diisi (true atau false)")
	}
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := s.repo(q)
		u, err := s.lockUnit(ctx, repo, user.Company.Code, id)
		if err != nil || u.IsActive == *in.Active {
			return err
		}
		if *in.Active {
			if err := s.checkParent(ctx, repo, user.Company.Code, id, u.ParentID); err != nil {
				return err
			}
		} else {
			switch {
			case u.ActiveChildren > 0:
				return errUnitHasChildren
			case u.ActiveUsers > 0:
				return errUnitHasUsers
			}
		}
		if err := repo.SetUnitActive(ctx, id, *in.Active); err != nil {
			return err
		}
		return s.record(ctx, q, user, user.Company.Code, audit.EntityUnit, id, audit.ActionUnitStatus,
			map[string]bool{"isActive": u.IsActive}, map[string]bool{"isActive": *in.Active})
	})
	if err != nil {
		return nil, err
	}
	return s.reader.FindAdminUnit(ctx, user.Company.Code, id, false)
}

func (s *AdminService) lockUnit(ctx context.Context, repo *Repository, companyCode string, id int64) (*AdminUnit, error) {
	u, err := repo.FindAdminUnit(ctx, companyCode, id, true)
	if errors.Is(err, db.ErrNotFound) {
		return nil, errUnitNotFound
	}
	return u, err
}

// checkParent memastikan induk unit ada di perusahaan yang sama, aktif, dan bukan unit itu sendiri
// atau sub-unitnya (unitID 0 = unit baru).
func (s *AdminService) checkParent(ctx context.Context, repo *Repository, companyCode string, unitID int64, parentID *int64) error {
	if parentID == nil {
		return nil
	}
	parent, err := repo.FindAdminUnit(ctx, companyCode, *parentID, false)
	if errors.Is(err, db.ErrNotFound) {
		return apperr.Invalid("Unit induk tidak ditemukan")
	}
	if err != nil {
		return err
	}
	if !parent.IsActive {
		return errUnitParentInactive
	}
	if unitID == 0 {
		return nil
	}
	cycle, err := repo.IsAncestorOrSelf(ctx, unitID, *parentID)
	if err != nil {
		return err
	}
	if cycle {
		return errUnitCycle
	}
	return nil
}

// parseUnit merapikan dan memvalidasi isian unit.
func parseUnit(in UnitInput) (name, unitType string, parentID *int64, err error) {
	name = strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > maxUnitNameRunes {
		return "", "", nil, apperr.Invalid("Nama unit wajib diisi, maksimal 150 karakter")
	}
	unitType = strings.ToUpper(strings.Join(strings.Fields(in.UnitType), " "))
	if unitType == "" || utf8.RuneCountInString(unitType) > maxUnitTypeRunes || !unitTypePattern.MatchString(unitType) {
		return "", "", nil, apperr.Invalid("Jenis unit wajib diisi (huruf/angka, maksimal 30 karakter), contoh DEPARTEMEN")
	}
	if in.ParentID != nil && strings.TrimSpace(*in.ParentID) != "" {
		id, err := parseRequiredID(*in.ParentID, "Unit induk")
		if err != nil {
			return "", "", nil, err
		}
		parentID = &id
	}
	return name, unitType, parentID, nil
}

func idString(id *int64) *string {
	if id == nil {
		return nil
	}
	s := strconv.FormatInt(*id, 10)
	return &s
}
