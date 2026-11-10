package organisasi

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Jabatan dikelola admin perusahaan (ORG_MANAGE). Posisi (level) wajib diisi: posisi pejabat
// (DIREKTUR, SVP, VP, AVP) memberi hak approve dan disposisi kepada pemegangnya. "Pimpinan unit"
// hanya menentukan siapa yang menerima surat yang ditujukan ke unit, maksimal satu per unit.

var (
	errPositionUnitInactive = &apperr.InvalidStateError{Message: "Unit jabatan tidak aktif"}
	errPositionHeadTaken    = &apperr.InvalidStateError{Message: "Unit ini sudah punya jabatan pimpinan unit yang aktif"}
	errPositionNameTaken    = &apperr.InvalidStateError{Message: "Nama jabatan sudah dipakai di unit ini"}
	errPositionHasUsers     = &apperr.InvalidStateError{Message: "Jabatan masih dipegang user aktif. Pindahkan atau nonaktifkan user-nya dulu."}
)

// positionData adalah isian jabatan yang sudah divalidasi.
type positionData struct {
	UnitID  int64
	Name    string
	LevelID int64
	IsHead  bool
}

// Positions mengambil jabatan perusahaan user.
func (s *AdminService) Positions(ctx context.Context, user *auth.CurrentUser, lq ListQuery) ([]AdminPosition, int, error) {
	f, err := filterOf(user.Company.Code, lq)
	if err != nil {
		return nil, 0, err
	}
	return s.reader.ListAdminPositions(ctx, f)
}

// Levels mengambil posisi (lookup POSITION_LEVEL) perusahaan user, untuk pilihan di form jabatan.
func (s *AdminService) Levels(ctx context.Context, user *auth.CurrentUser) ([]LevelRef, error) {
	return s.reader.ListLevels(ctx, user.Company.Code)
}

// CreatePosition menambah jabatan.
func (s *AdminService) CreatePosition(ctx context.Context, user *auth.CurrentUser, in PositionInput) (*AdminPosition, error) {
	d, err := parsePosition(in)
	if err != nil {
		return nil, err
	}
	var id int64
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := s.repo(q)
		if err := s.checkPosition(ctx, repo, user.Company.Code, 0, d, true); err != nil {
			return err
		}
		if id, err = repo.InsertPosition(ctx, d.UnitID, d.Name, d.LevelID, d.IsHead); err != nil {
			return err
		}
		return s.record(ctx, q, user, user.Company.Code, audit.EntityPosition, id, audit.ActionPositionCreate, nil, in)
	})
	if err != nil {
		return nil, err
	}
	return s.reader.FindAdminPosition(ctx, user.Company.Code, id, false)
}

// UpdatePosition mengubah jabatan. Perubahan posisi langsung mengubah hak pemegangnya.
func (s *AdminService) UpdatePosition(ctx context.Context, user *auth.CurrentUser, rawID string, in PositionInput) (*AdminPosition, error) {
	id, err := parsePathID(rawID, errPositionNotFound)
	if err != nil {
		return nil, err
	}
	d, err := parsePosition(in)
	if err != nil {
		return nil, err
	}
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := s.repo(q)
		old, err := s.lockPosition(ctx, repo, user.Company.Code, id)
		if err != nil {
			return err
		}
		if err := s.checkPosition(ctx, repo, user.Company.Code, id, d, old.IsActive); err != nil {
			return err
		}
		if err := repo.UpdatePosition(ctx, id, d.UnitID, d.Name, d.LevelID, d.IsHead); err != nil {
			return err
		}
		return s.record(ctx, q, user, user.Company.Code, audit.EntityPosition, id, audit.ActionPositionUpdate,
			positionInputOf(old), in)
	})
	if err != nil {
		return nil, err
	}
	return s.reader.FindAdminPosition(ctx, user.Company.Code, id, false)
}

// SetPositionStatus mengaktifkan/menonaktifkan jabatan. Nonaktif ditolak bila masih dipegang user
// aktif; aktif dicek ulang (unit aktif, nama dan pimpinan unit tidak bentrok).
func (s *AdminService) SetPositionStatus(ctx context.Context, user *auth.CurrentUser, rawID string, in StatusInput) (*AdminPosition, error) {
	id, err := parsePathID(rawID, errPositionNotFound)
	if err != nil {
		return nil, err
	}
	if in.Active == nil {
		return nil, apperr.Invalid("active wajib diisi (true atau false)")
	}
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := s.repo(q)
		p, err := s.lockPosition(ctx, repo, user.Company.Code, id)
		if err != nil || p.IsActive == *in.Active {
			return err
		}
		if *in.Active {
			d := positionData{UnitID: p.UnitID, Name: p.Name, IsHead: p.IsUnitHead}
			if p.Level != nil {
				d.LevelID = p.Level.ID
			}
			if err := s.checkPosition(ctx, repo, user.Company.Code, id, d, true); err != nil {
				return err
			}
		} else if p.ActiveUsers > 0 {
			return errPositionHasUsers
		}
		if err := repo.SetPositionActive(ctx, id, *in.Active); err != nil {
			return err
		}
		return s.record(ctx, q, user, user.Company.Code, audit.EntityPosition, id, audit.ActionPositionStatus,
			map[string]bool{"isActive": p.IsActive}, map[string]bool{"isActive": *in.Active})
	})
	if err != nil {
		return nil, err
	}
	return s.reader.FindAdminPosition(ctx, user.Company.Code, id, false)
}

func (s *AdminService) lockPosition(ctx context.Context, repo *Repository, companyCode string, id int64) (*AdminPosition, error) {
	p, err := repo.FindAdminPosition(ctx, companyCode, id, true)
	if errors.Is(err, db.ErrNotFound) {
		return nil, errPositionNotFound
	}
	return p, err
}

// checkPosition memeriksa unit (perusahaan sama, aktif), posisi (milik perusahaan), dan untuk
// jabatan aktif: nama unik di unitnya serta maksimal satu pimpinan unit (exceptID 0 = jabatan baru).
func (s *AdminService) checkPosition(ctx context.Context, repo *Repository, companyCode string, exceptID int64, d positionData, active bool) error {
	unit, err := repo.FindAdminUnit(ctx, companyCode, d.UnitID, false)
	if errors.Is(err, db.ErrNotFound) {
		return apperr.Invalid("Unit jabatan tidak ditemukan")
	}
	if err != nil {
		return err
	}
	if !unit.IsActive {
		return errPositionUnitInactive
	}
	if _, err := repo.FindLevel(ctx, companyCode, d.LevelID, ""); errors.Is(err, db.ErrNotFound) {
		return apperr.Invalid("Posisi tidak ditemukan")
	} else if err != nil {
		return err
	}
	if !active {
		return nil
	}
	taken, err := repo.PositionNameTaken(ctx, d.UnitID, d.Name, exceptID)
	if err != nil {
		return err
	}
	if taken {
		return errPositionNameTaken
	}
	if d.IsHead {
		other, err := repo.OtherActiveHead(ctx, d.UnitID, exceptID)
		if err != nil {
			return err
		}
		if other {
			return errPositionHeadTaken
		}
	}
	return nil
}

// parsePosition merapikan dan memvalidasi isian jabatan.
func parsePosition(in PositionInput) (positionData, error) {
	d := positionData{Name: strings.Join(strings.Fields(in.Name), " "), IsHead: in.IsUnitHead}
	if d.Name == "" || utf8.RuneCountInString(d.Name) > maxPositionNameRunes {
		return d, apperr.Invalid("Nama jabatan wajib diisi, maksimal 150 karakter")
	}
	var err error
	if d.UnitID, err = parseRequiredID(in.UnitID, "Unit"); err != nil {
		return d, err
	}
	if d.LevelID, err = parseRequiredID(in.LevelID, "Posisi"); err != nil {
		return d, err
	}
	return d, nil
}

func positionInputOf(p *AdminPosition) PositionInput {
	in := PositionInput{UnitID: strconv.FormatInt(p.UnitID, 10), Name: p.Name, IsUnitHead: p.IsUnitHead}
	if p.Level != nil {
		in.LevelID = strconv.FormatInt(p.Level.ID, 10)
	}
	return in
}
