package organisasi

import (
	"context"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// User dikelola admin perusahaan (USER_MANAGE): tambah, ubah (termasuk mutasi jabatan dan ganti
// role), aktif/nonaktif, dan reset password. Admin bisa mengangkat admin lain. Mutasi, ganti role,
// nonaktif, dan reset password mengakhiri semua sesi user itu. Pengaman (userGuard) hanya mencegah
// perusahaan terkunci: admin tidak bisa menonaktifkan/mengganti role dirinya sendiri, admin terakhir
// tidak bisa dicabut, dan hanya SUPERADMIN yang mengurus akun SUPERADMIN.

// employeeNumberPattern adalah NIK: huruf, angka, titik, garis bawah, dan tanda hubung.
var employeeNumberPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

var (
	errNIKTaken         = &apperr.InvalidStateError{Message: "NIK sudah dipakai user lain"}
	errPositionUnusable = apperr.Invalid("Jabatan tidak ditemukan atau tidak aktif")
	errRoleNotFound     = apperr.Invalid("Role tidak ditemukan")
	errSelfStatus       = &apperr.InvalidStateError{Message: "Anda tidak bisa menonaktifkan akun Anda sendiri"}
	errSelfRole         = &apperr.InvalidStateError{Message: "Anda tidak bisa mengganti role Anda sendiri"}
	errLastAdmin        = &apperr.InvalidStateError{Message: "Ini admin terakhir di perusahaan. Angkat admin lain dulu."}
)

// userRecord adalah isian user yang sudah divalidasi.
type userRecord struct {
	EmployeeNumber string
	Name           string
	Email          string
	PositionID     int64
	RoleID         int64
}

// Users mengambil user perusahaan user yang login.
func (s *AdminService) Users(ctx context.Context, user *auth.CurrentUser, lq ListQuery) ([]AdminUser, int, error) {
	f, err := filterOf(user.Company.Code, lq)
	if err != nil {
		return nil, 0, err
	}
	return s.reader.ListAdminUsers(ctx, f)
}

// User mengambil satu user perusahaan.
func (s *AdminService) User(ctx context.Context, user *auth.CurrentUser, rawID string) (*AdminUser, error) {
	id, err := parsePathID(rawID, errUserNotFound)
	if err != nil {
		return nil, err
	}
	u, err := s.reader.FindAdminUser(ctx, user.Company.Code, id, false)
	if errors.Is(err, db.ErrNotFound) {
		return nil, errUserNotFound
	}
	return u, err
}

// Roles mengambil role yang bisa diberikan user ini (SUPERADMIN hanya untuk pemegang COMPANY_MANAGE).
func (s *AdminService) Roles(ctx context.Context, user *auth.CurrentUser) ([]RoleRef, error) {
	roles, err := s.reader.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := []RoleRef{}
	for _, r := range roles {
		if r.Code != auth.RoleSuperAdmin || user.HasPermission(auth.CompanyManage) {
			out = append(out, r)
		}
	}
	return out, nil
}

// CreateUser menambah user di perusahaan user yang login.
func (s *AdminService) CreateUser(ctx context.Context, user *auth.CurrentUser, in UserInput) (*AdminUser, error) {
	rec, err := parseUser(in)
	if err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	var id int64
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := s.repo(q)
		role, err := s.checkUser(ctx, repo, user.Company.Code, 0, rec)
		if err != nil {
			return err
		}
		if role.Code == auth.RoleSuperAdmin && !user.HasPermission(auth.CompanyManage) {
			return apperr.ErrForbidden
		}
		if id, err = repo.InsertUser(ctx, rec, hash); err != nil {
			return err
		}
		return s.record(ctx, q, user, user.Company.Code, audit.EntityUser, id, audit.ActionUserCreate, nil, userSnapshot(rec))
	})
	if err != nil {
		return nil, err
	}
	return s.reader.FindAdminUser(ctx, user.Company.Code, id, false)
}

// UpdateUser mengubah data user, termasuk mutasi jabatan dan ganti role. Mutasi dan ganti role
// mengakhiri semua sesi user itu (haknya ikut berubah). Password tidak diubah di sini.
func (s *AdminService) UpdateUser(ctx context.Context, user *auth.CurrentUser, rawID string, in UserInput) (*AdminUser, error) {
	id, err := parsePathID(rawID, errUserNotFound)
	if err != nil {
		return nil, err
	}
	rec, err := parseUser(in)
	if err != nil {
		return nil, err
	}
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := s.repo(q)
		old, err := s.lockUser(ctx, repo, user.Company.Code, id)
		if err != nil {
			return err
		}
		role, err := s.checkUser(ctx, repo, user.Company.Code, id, rec)
		if err != nil {
			return err
		}
		if err := s.guard(ctx, repo, user, old, role.Code, old.IsActive); err != nil {
			return err
		}
		logout := old.Position.ID != rec.PositionID || old.Role.ID != rec.RoleID
		if err := repo.UpdateUser(ctx, id, rec, logout); err != nil {
			return err
		}
		return s.record(ctx, q, user, user.Company.Code, audit.EntityUser, id, audit.ActionUserUpdate,
			userSnapshot(userRecord{old.EmployeeNumber, old.Name, old.Email, old.Position.ID, old.Role.ID}), userSnapshot(rec))
	})
	if err != nil {
		return nil, err
	}
	return s.reader.FindAdminUser(ctx, user.Company.Code, id, false)
}

// SetUserStatus mengaktifkan/menonaktifkan user. Admin tidak dibatasi walaupun user itu sedang
// memegang giliran surat.
func (s *AdminService) SetUserStatus(ctx context.Context, user *auth.CurrentUser, rawID string, in StatusInput) (*AdminUser, error) {
	id, err := parsePathID(rawID, errUserNotFound)
	if err != nil {
		return nil, err
	}
	if in.Active == nil {
		return nil, apperr.Invalid("active wajib diisi (true atau false)")
	}
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := s.repo(q)
		old, err := s.lockUser(ctx, repo, user.Company.Code, id)
		if err != nil || old.IsActive == *in.Active {
			return err
		}
		if err := s.guard(ctx, repo, user, old, old.Role.Code, *in.Active); err != nil {
			return err
		}
		if *in.Active {
			// Jabatan user harus masih aktif supaya akunnya bisa dipakai
			if err := checkPositionUsable(ctx, repo, user.Company.Code, old.Position.ID); err != nil {
				return &apperr.InvalidStateError{Message: "Jabatan user ini tidak aktif. Pindahkan ke jabatan aktif dulu."}
			}
		}
		if err := repo.SetUserActive(ctx, id, *in.Active); err != nil {
			return err
		}
		return s.record(ctx, q, user, user.Company.Code, audit.EntityUser, id, audit.ActionUserStatus,
			map[string]bool{"isActive": old.IsActive}, map[string]bool{"isActive": *in.Active})
	})
	if err != nil {
		return nil, err
	}
	return s.reader.FindAdminUser(ctx, user.Company.Code, id, false)
}

// ResetPassword mengganti password user dan mengakhiri semua sesinya.
func (s *AdminService) ResetPassword(ctx context.Context, user *auth.CurrentUser, rawID string, in PasswordInput) error {
	id, err := parsePathID(rawID, errUserNotFound)
	if err != nil {
		return err
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return err
	}
	return s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := s.repo(q)
		target, err := s.lockUser(ctx, repo, user.Company.Code, id)
		if err != nil {
			return err
		}
		if target.Role.Code == auth.RoleSuperAdmin && !user.HasPermission(auth.CompanyManage) {
			return apperr.ErrForbidden
		}
		if err := repo.SetPassword(ctx, id, hash); err != nil {
			return err
		}
		return s.record(ctx, q, user, user.Company.Code, audit.EntityUser, id, audit.ActionUserPassword, nil, nil)
	})
}

// CreateCompanyAdmin (SUPERADMIN) menambah admin untuk sebuah perusahaan: di unit yang sudah ada
// (UnitID) atau unit baru (UnitName, untuk perusahaan yang belum punya unit), dengan jabatan baru
// berposisi STAF. Admin itu yang kemudian mengatur isi perusahaannya.
func (s *AdminService) CreateCompanyAdmin(ctx context.Context, user *auth.CurrentUser, companyCode string, in CompanyAdminInput) (*AdminUser, error) {
	companyCode = strings.ToUpper(strings.TrimSpace(companyCode))
	rec, err := parseUser(UserInput{EmployeeNumber: in.EmployeeNumber, Name: in.Name, Email: in.Email, PositionID: "1", RoleID: "1"})
	if err != nil {
		return nil, err
	}
	posName := strings.Join(strings.Fields(in.PositionName), " ")
	if posName == "" || utf8.RuneCountInString(posName) > maxPositionNameRunes {
		return nil, apperr.Invalid("Nama jabatan admin wajib diisi, maksimal 150 karakter")
	}
	var unitID int64
	var unitName string
	switch {
	case in.UnitID != nil && strings.TrimSpace(*in.UnitID) != "":
		if unitID, err = parseRequiredID(*in.UnitID, "Unit"); err != nil {
			return nil, err
		}
	case in.UnitName != nil && strings.TrimSpace(*in.UnitName) != "":
		unitName = strings.Join(strings.Fields(*in.UnitName), " ")
		if utf8.RuneCountInString(unitName) > maxUnitNameRunes {
			return nil, apperr.Invalid("Nama unit maksimal 150 karakter")
		}
	default:
		return nil, apperr.Invalid("Pilih unit (unitId) atau isi nama unit baru (unitName)")
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	var id int64
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		repo := s.repo(q)
		ok, err := repo.CompanyExists(ctx, companyCode)
		if err != nil {
			return err
		}
		if !ok {
			return errCompanyNotFound
		}
		if unitID == 0 {
			if unitID, err = repo.InsertUnit(ctx, companyCode, uuid.NewString(), unitName, "UNIT", nil); err != nil {
				return err
			}
			if err := s.record(ctx, q, user, companyCode, audit.EntityUnit, unitID, audit.ActionUnitCreate, nil,
				UnitInput{Name: unitName, UnitType: "UNIT"}); err != nil {
				return err
			}
		} else if unit, err := repo.FindAdminUnit(ctx, companyCode, unitID, false); errors.Is(err, db.ErrNotFound) {
			return apperr.Invalid("Unit tidak ditemukan di perusahaan ini")
		} else if err != nil {
			return err
		} else if !unit.IsActive {
			return errPositionUnitInactive
		}
		staf, err := repo.FindLevel(ctx, companyCode, 0, "STAF")
		if errors.Is(err, db.ErrNotFound) {
			return &apperr.InvalidStateError{Message: "Perusahaan belum punya data posisi (STAF)"}
		}
		if err != nil {
			return err
		}
		if taken, err := repo.PositionNameTaken(ctx, unitID, posName, 0); err != nil {
			return err
		} else if taken {
			return errPositionNameTaken
		}
		if rec.PositionID, err = repo.InsertPosition(ctx, unitID, posName, staf.ID, false); err != nil {
			return err
		}
		if err := s.record(ctx, q, user, companyCode, audit.EntityPosition, rec.PositionID, audit.ActionPositionCreate, nil,
			PositionInput{Name: posName}); err != nil {
			return err
		}
		admin, err := repo.FindRole(ctx, 0, auth.RoleAdmin)
		if err != nil {
			return err
		}
		rec.RoleID = admin.ID
		if taken, err := repo.EmployeeNumberTaken(ctx, rec.EmployeeNumber, 0); err != nil {
			return err
		} else if taken {
			return errNIKTaken
		}
		if id, err = repo.InsertUser(ctx, rec, hash); err != nil {
			return err
		}
		return s.record(ctx, q, user, companyCode, audit.EntityUser, id, audit.ActionUserCreate, nil, userSnapshot(rec))
	})
	if err != nil {
		return nil, err
	}
	return s.reader.FindAdminUser(ctx, companyCode, id, false)
}

func (s *AdminService) lockUser(ctx context.Context, repo *Repository, companyCode string, id int64) (*AdminUser, error) {
	u, err := repo.FindAdminUser(ctx, companyCode, id, true)
	if errors.Is(err, db.ErrNotFound) {
		return nil, errUserNotFound
	}
	return u, err
}

// checkUser memeriksa NIK unik, jabatan aktif di perusahaan, dan role aktif (exceptID 0 = user baru).
func (s *AdminService) checkUser(ctx context.Context, repo *Repository, companyCode string, exceptID int64, rec userRecord) (*RoleRef, error) {
	taken, err := repo.EmployeeNumberTaken(ctx, rec.EmployeeNumber, exceptID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, errNIKTaken
	}
	if err := checkPositionUsable(ctx, repo, companyCode, rec.PositionID); err != nil {
		return nil, err
	}
	role, err := repo.FindRole(ctx, rec.RoleID, "")
	if errors.Is(err, db.ErrNotFound) {
		return nil, errRoleNotFound
	}
	return role, err
}

// checkPositionUsable memastikan jabatan aktif di unit aktif milik perusahaan.
func checkPositionUsable(ctx context.Context, repo *Repository, companyCode string, positionID int64) error {
	p, err := repo.FindAdminPosition(ctx, companyCode, positionID, false)
	if errors.Is(err, db.ErrNotFound) {
		return errPositionUnusable
	}
	if err != nil {
		return err
	}
	if !p.IsActive {
		return errPositionUnusable
	}
	unit, err := repo.FindAdminUnit(ctx, companyCode, p.UnitID, false)
	if err != nil {
		return err
	}
	if !unit.IsActive {
		return errPositionUnusable
	}
	return nil
}

// guard menerapkan userGuard; jumlah admin lain hanya dihitung bila dibutuhkan.
func (s *AdminService) guard(ctx context.Context, repo *Repository, actor *auth.CurrentUser, target *AdminUser, newRole string, newActive bool) error {
	others := -1
	if target.Role.Code == auth.RoleAdmin && target.IsActive && (newRole != auth.RoleAdmin || !newActive) {
		n, err := repo.CountOtherActiveAdmins(ctx, actor.Company.Code, target.ID)
		if err != nil {
			return err
		}
		others = n
	}
	return userGuard(actor, target, newRole, newActive, others)
}

// userGuard adalah pengaman perubahan user oleh admin. otherAdmins = jumlah admin aktif lain di
// perusahaan (-1 = tidak dihitung karena tidak relevan).
func userGuard(actor *auth.CurrentUser, target *AdminUser, newRole string, newActive bool, otherAdmins int) error {
	// Akun SUPERADMIN (dan pemberian role SUPERADMIN) hanya diurus pemegang COMPANY_MANAGE
	if (target.Role.Code == auth.RoleSuperAdmin || newRole == auth.RoleSuperAdmin) && !actor.HasPermission(auth.CompanyManage) {
		return apperr.ErrForbidden
	}
	if target.ID == actor.ID {
		if !newActive {
			return errSelfStatus
		}
		if newRole != target.Role.Code {
			return errSelfRole
		}
	}
	if otherAdmins == 0 {
		return errLastAdmin
	}
	return nil
}

// parseUser merapikan dan memvalidasi isian user (tanpa password).
func parseUser(in UserInput) (userRecord, error) {
	rec := userRecord{
		EmployeeNumber: strings.TrimSpace(in.EmployeeNumber),
		Name:           strings.Join(strings.Fields(in.Name), " "),
		Email:          strings.ToLower(strings.TrimSpace(in.Email)),
	}
	if rec.EmployeeNumber == "" || len(rec.EmployeeNumber) > maxEmployeeNumber || !employeeNumberPattern.MatchString(rec.EmployeeNumber) {
		return rec, apperr.Invalid("NIK wajib diisi (huruf/angka, maksimal 30 karakter)")
	}
	if rec.Name == "" || utf8.RuneCountInString(rec.Name) > maxUserNameRunes {
		return rec, apperr.Invalid("Nama wajib diisi, maksimal 150 karakter")
	}
	if addr, err := mail.ParseAddress(rec.Email); err != nil || addr.Address != rec.Email || utf8.RuneCountInString(rec.Email) > maxEmailRunes {
		return rec, apperr.Invalid("Email tidak valid")
	}
	var err error
	if rec.PositionID, err = parseRequiredID(in.PositionID, "Jabatan"); err != nil {
		return rec, err
	}
	if rec.RoleID, err = parseRequiredID(in.RoleID, "Role"); err != nil {
		return rec, err
	}
	return rec, nil
}

func userSnapshot(r userRecord) map[string]any {
	return map[string]any{"employeeNumber": r.EmployeeNumber, "name": r.Name, "email": r.Email,
		"positionId": r.PositionID, "roleId": r.RoleID}
}
