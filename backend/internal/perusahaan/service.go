package perusahaan

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
	"office.backend/internal/platform/watermark"
	"office.backend/internal/surat"
)

// Store adalah query perusahaan yang dibutuhkan Service.
type Store interface {
	ListCompanies(ctx context.Context, f CompanyQuery) ([]Company, int, error)
	FindCompany(ctx context.Context, code string, lock bool) (*Company, error)
	CodeTaken(ctx context.Context, code string) (bool, error)
	InsertCompany(ctx context.Context, in CompanyInput) error
	UpdateIdentity(ctx context.Context, code, name string, city *string) error
	SetActive(ctx context.Context, code string, active bool) error
	DeleteCompany(ctx context.Context, code string) error
	UpdateSettings(ctx context.Context, code, format string, watermark *string) error
}

// AuditStore mencatat jejak audit.
type AuditStore interface {
	Record(ctx context.Context, e audit.Entry) error
}

// Repos adalah repository yang dibutuhkan Service, dibuat di atas koneksi atau transaksi.
type Repos struct {
	Companies func(q db.DBTX) Store
	Audit     func(q db.DBTX) AuditStore
}

// Service mengelola perusahaan (SUPERADMIN) dan pengaturan perusahaan (ADMIN).
type Service struct {
	tx     db.TxRunner
	repos  Repos
	reader Store
	loc    *time.Location
}

// NewService membuat Service. conn untuk query baca di luar transaksi; loc untuk contoh nomor surat.
func NewService(tx db.TxRunner, conn db.DBTX, repos Repos, loc *time.Location) *Service {
	return &Service{tx: tx, repos: repos, reader: repos.Companies(conn), loc: loc}
}

// Batas isian perusahaan (sama dengan kolom tabel companies).
const (
	maxNameRunes = 150
	maxCityRunes = 50
)

// codePattern adalah kode perusahaan: huruf besar dan angka, 2-20 karakter (dipakai di nomor surat).
var codePattern = regexp.MustCompile(`^[A-Z0-9]{2,20}$`)

var (
	// ErrCompanyNotFound: kode perusahaan tidak ada.
	ErrCompanyNotFound   = &apperr.NotFoundError{Message: "Perusahaan tidak ditemukan"}
	errCodeTaken         = &apperr.InvalidStateError{Message: "Kode perusahaan sudah dipakai"}
	errOwnCompany        = &apperr.InvalidStateError{Message: "Perusahaan tempat Anda bekerja tidak bisa dinonaktifkan"}
	errOwnCompanyDelete  = &apperr.InvalidStateError{Message: "Perusahaan tempat Anda bekerja tidak bisa dihapus"}
	errCompanyHasMembers = &apperr.InvalidStateError{Message: "Perusahaan masih memiliki unit kerja atau user aktif. Hapus atau nonaktifkan dulu."}
)

// --- SUPERADMIN ---------------------------------------------------------------------------

// Companies mengambil daftar perusahaan.
func (s *Service) Companies(ctx context.Context, f CompanyQuery) ([]Company, int, error) {
	search, err := db.NormalizeSearch(f.Search)
	if err != nil {
		return nil, 0, err
	}
	f.Search = search
	f.Status = strings.ToUpper(strings.TrimSpace(f.Status))
	if f.Status != "" && f.Status != "ACTIVE" && f.Status != "INACTIVE" {
		return nil, 0, apperr.Invalid("Parameter status wajib ACTIVE atau INACTIVE")
	}
	return s.reader.ListCompanies(ctx, f)
}

// Company mengambil satu perusahaan.
func (s *Service) Company(ctx context.Context, code string) (*Company, error) {
	c, err := s.reader.FindCompany(ctx, strings.ToUpper(strings.TrimSpace(code)), false)
	if errors.Is(err, db.ErrNotFound) {
		return nil, ErrCompanyNotFound
	}
	return c, err
}

// CreateCompany menambah perusahaan beserta posisi standarnya.
func (s *Service) CreateCompany(ctx context.Context, user *auth.CurrentUser, in CompanyInput) (*Company, error) {
	in, err := parseCompany(in, true)
	if err != nil {
		return nil, err
	}
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		store := s.repos.Companies(q)
		taken, err := store.CodeTaken(ctx, in.Code)
		if err != nil {
			return err
		}
		if taken {
			return errCodeTaken
		}
		if err := store.InsertCompany(ctx, in); err != nil {
			return err
		}
		return s.record(ctx, q, user, in.Code, audit.ActionCompanyCreate, nil,
			map[string]any{"name": in.Name, "city": in.City})
	})
	if err != nil {
		return nil, err
	}
	return s.Company(ctx, in.Code)
}

// UpdateCompany mengubah nama dan kota perusahaan.
func (s *Service) UpdateCompany(ctx context.Context, user *auth.CurrentUser, code string, in CompanyInput) (*Company, error) {
	in, err := parseCompany(in, false)
	if err != nil {
		return nil, err
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		store := s.repos.Companies(q)
		old, err := store.FindCompany(ctx, code, true)
		if errors.Is(err, db.ErrNotFound) {
			return ErrCompanyNotFound
		}
		if err != nil {
			return err
		}
		if err := store.UpdateIdentity(ctx, code, in.Name, in.City); err != nil {
			return err
		}
		return s.record(ctx, q, user, code, audit.ActionCompanyUpdate,
			map[string]any{"name": old.Name, "city": old.City}, map[string]any{"name": in.Name, "city": in.City})
	})
	if err != nil {
		return nil, err
	}
	return s.Company(ctx, code)
}

// LookupCompany mengembalikan kode dan nama perusahaan untuk pengalihan konteks SUPERADMIN.
func (s *Service) LookupCompany(ctx context.Context, code string) (*auth.CompanyRef, error) {
	c, err := s.Company(ctx, code)
	if err != nil {
		return nil, err
	}
	return &auth.CompanyRef{Code: c.Code, Name: c.Name}, nil
}

// SetCompanyStatus mengaktifkan/menonaktifkan perusahaan. Perusahaan pelaku tidak bisa
// dinonaktifkan (supaya SUPERADMIN tidak mengunci dirinya sendiri).
func (s *Service) SetCompanyStatus(ctx context.Context, user *auth.CurrentUser, code string, in StatusInput) (*Company, error) {
	if in.Active == nil {
		return nil, apperr.Invalid("active wajib diisi (true atau false)")
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if !*in.Active && code == user.Company.Code {
		return nil, errOwnCompany
	}
	err := s.tx.InTx(ctx, func(q db.DBTX) error {
		store := s.repos.Companies(q)
		old, err := store.FindCompany(ctx, code, true)
		if errors.Is(err, db.ErrNotFound) {
			return ErrCompanyNotFound
		}
		if err != nil || old.IsActive == *in.Active {
			return err
		}
		if err := store.SetActive(ctx, code, *in.Active); err != nil {
			return err
		}
		return s.record(ctx, q, user, code, audit.ActionCompanyStatus,
			map[string]any{"isActive": old.IsActive}, map[string]any{"isActive": *in.Active})
	})
	if err != nil {
		return nil, err
	}
	return s.Company(ctx, code)
}

// DeleteCompany menghapus (soft-delete) perusahaan apabila bukan perusahaan tempat pelaku bekerja
// dan tidak lagi memiliki unit/user aktif.
func (s *Service) DeleteCompany(ctx context.Context, user *auth.CurrentUser, code string) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == user.Company.Code {
		return errOwnCompanyDelete
	}
	return s.tx.InTx(ctx, func(q db.DBTX) error {
		store := s.repos.Companies(q)
		old, err := store.FindCompany(ctx, code, true)
		if errors.Is(err, db.ErrNotFound) {
			return ErrCompanyNotFound
		}
		if err != nil {
			return err
		}
		if old.ActiveUnits > 0 || old.ActiveUsers > 0 {
			return errCompanyHasMembers
		}
		if err := store.DeleteCompany(ctx, code); err != nil {
			return err
		}
		return s.record(ctx, q, user, code, audit.ActionCompanyDelete,
			map[string]any{"name": old.Name, "city": old.City, "isActive": old.IsActive}, nil)
	})
}

// parseCompany merapikan dan memvalidasi isian perusahaan.
func parseCompany(in CompanyInput, withCode bool) (CompanyInput, error) {
	out := CompanyInput{Name: strings.TrimSpace(in.Name)}
	if withCode {
		out.Code = strings.ToUpper(strings.TrimSpace(in.Code))
		if !codePattern.MatchString(out.Code) {
			return out, apperr.Invalid("Kode perusahaan wajib 2-20 huruf besar atau angka, contoh PSP")
		}
	}
	if out.Name == "" || utf8.RuneCountInString(out.Name) > maxNameRunes {
		return out, apperr.Invalid("Nama perusahaan wajib diisi, maksimal 150 karakter")
	}
	if in.City != nil {
		if city := strings.TrimSpace(*in.City); city != "" {
			if utf8.RuneCountInString(city) > maxCityRunes {
				return out, apperr.Invalid("Kota maksimal 50 karakter")
			}
			out.City = &city
		}
	}
	return out, nil
}

// --- ADMIN: pengaturan perusahaannya sendiri ------------------------------------------------

// Settings mengambil pengaturan perusahaan user.
func (s *Service) Settings(ctx context.Context, user *auth.CurrentUser) (*Settings, error) {
	c, err := s.reader.FindCompany(ctx, user.Company.Code, false)
	if err != nil {
		return nil, err
	}
	return s.settingsOf(c), nil
}

func (s *Service) settingsOf(c *Company) *Settings {
	format := surat.EffectiveNumberFormat(c.LetterNumberFormat) // format yang benar-benar dipakai
	return &Settings{
		Code: c.Code, Name: c.Name, LetterNumberFormat: format, WatermarkText: c.WatermarkText,
		NumberPreview: surat.NumberPreview(format, c.Code, time.Now().In(s.loc)),
	}
}

// UpdateSettings menyimpan format nomor dan watermark perusahaan user. Watermark baru hanya berlaku
// untuk naskah resmi yang dibuat setelahnya; PDF resmi yang sudah ada tidak berubah.
func (s *Service) UpdateSettings(ctx context.Context, user *auth.CurrentUser, in SettingsInput) (*Settings, error) {
	format := strings.TrimSpace(in.LetterNumberFormat)
	if err := surat.ValidateNumberFormat(format); err != nil {
		return nil, err
	}
	text, err := parseWatermark(in.WatermarkText)
	if err != nil {
		return nil, err
	}
	var saved *Company
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		store := s.repos.Companies(q)
		old, err := store.FindCompany(ctx, user.Company.Code, true)
		if err != nil {
			return err
		}
		if err := store.UpdateSettings(ctx, old.Code, format, text); err != nil {
			return err
		}
		saved = old
		saved.LetterNumberFormat, saved.WatermarkText = format, text
		return s.record(ctx, q, user, old.Code, audit.ActionCompanySettings,
			map[string]any{"letterNumberFormat": old.LetterNumberFormat, "watermarkText": old.WatermarkText},
			map[string]any{"letterNumberFormat": format, "watermarkText": text})
	})
	if err != nil {
		return nil, err
	}
	return s.settingsOf(saved), nil
}

// NumberPreview memvalidasi format nomor lalu mengembalikan contoh nomornya (sebelum disimpan).
func (s *Service) NumberPreview(user *auth.CurrentUser, format string) (string, error) {
	format = strings.TrimSpace(format)
	if err := surat.ValidateNumberFormat(format); err != nil {
		return "", err
	}
	return surat.NumberPreview(format, user.Company.Code, time.Now().In(s.loc)), nil
}

// WatermarkPreview menggambar watermark (PNG) dari text, atau dari watermark tersimpan bila text
// nil. Sama persis dengan yang dipasang di naskah resmi.
func (s *Service) WatermarkPreview(ctx context.Context, user *auth.CurrentUser, text *string) ([]byte, error) {
	if text == nil {
		c, err := s.reader.FindCompany(ctx, user.Company.Code, false)
		if err != nil {
			return nil, err
		}
		text = c.WatermarkText
	}
	t, err := parseWatermark(text)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, &apperr.NotFoundError{Message: "Perusahaan tidak memakai watermark"}
	}
	return watermark.PNG(*t)
}

// parseWatermark merapikan teks watermark: kosong = tanpa watermark (nil).
func parseWatermark(text *string) (*string, error) {
	if text == nil {
		return nil, nil
	}
	t := watermark.Normalize(*text)
	if t == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(t) > watermark.MaxRunes {
		return nil, apperr.Invalid("Teks watermark maksimal 100 karakter")
	}
	return &t, nil
}

// record mencatat perubahan perusahaan di audit log perusahaan tersebut.
func (s *Service) record(ctx context.Context, q db.DBTX, user *auth.CurrentUser, code, action string, before, after map[string]any) error {
	return s.repos.Audit(q).Record(ctx, audit.Entry{
		CompanyCode: code, EntityType: audit.EntityCompany, Action: action,
		ActorID: user.ID, ActorName: user.Name,
		Snapshot: companySnapshot{Code: code, Before: before, After: after},
	})
}
