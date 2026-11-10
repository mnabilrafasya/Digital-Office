package organisasi

import (
	"context"
	"strconv"
	"strings"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// AuditStore mencatat jejak audit.
type AuditStore interface {
	Record(ctx context.Context, e audit.Entry) error
}

// AdminService mengelola master data organisasi: unit, jabatan, dan user. Semua di dalam
// transaksi bersama catatan audit-nya.
type AdminService struct {
	tx     db.TxRunner
	repo   func(q db.DBTX) *Repository
	audit  func(q db.DBTX) AuditStore
	reader *Repository
}

// NewAdminService membuat AdminService. conn untuk query baca di luar transaksi.
func NewAdminService(tx db.TxRunner, conn db.DBTX, repo func(q db.DBTX) *Repository, audit func(q db.DBTX) AuditStore) *AdminService {
	return &AdminService{tx: tx, repo: repo, audit: audit, reader: repo(conn)}
}

var (
	errUnitNotFound     = &apperr.NotFoundError{Message: "Unit tidak ditemukan"}
	errPositionNotFound = &apperr.NotFoundError{Message: "Jabatan tidak ditemukan"}
	errUserNotFound     = &apperr.NotFoundError{Message: "User tidak ditemukan"}
	errCompanyNotFound  = &apperr.NotFoundError{Message: "Perusahaan tidak ditemukan"}
)

// Batas isian (sama dengan kolom tabel).
const (
	maxUnitNameRunes     = 150
	maxUnitTypeRunes     = 30
	maxPositionNameRunes = 150
	maxUserNameRunes     = 150
	maxEmailRunes        = 150
	maxEmployeeNumber    = 30
	defaultAdminPageSize = 20
)

// parseStatus memvalidasi filter status daftar.
func parseStatus(s string) (string, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s != "" && s != "ACTIVE" && s != "INACTIVE" {
		return "", apperr.Invalid("Parameter status wajib ACTIVE atau INACTIVE")
	}
	return s, nil
}

// parseOptionalID membaca ID opsional dari query string ("" = 0).
func parseOptionalID(raw, label string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("Parameter " + label + " tidak valid")
	}
	return id, nil
}

// parseRequiredID membaca ID wajib dari isian form.
func parseRequiredID(raw, label string) (int64, error) {
	id, err := parseOptionalID(raw, label)
	if err == nil && id == 0 {
		err = apperr.Invalid(label + " wajib dipilih")
	}
	if err != nil {
		return 0, apperr.Invalid(label + " wajib dipilih dan valid")
	}
	return id, nil
}

// parsePathID membaca ID dari URL; tidak valid = tidak ditemukan.
func parsePathID(raw string, notFound error) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, notFound
	}
	return id, nil
}

// filterOf memvalidasi filter daftar untuk perusahaan user.
func filterOf(companyCode string, lq ListQuery) (adminFilter, error) {
	f := adminFilter{CompanyCode: companyCode, Offset: lq.Offset, Limit: lq.Limit}
	var err error
	if f.Search, err = db.NormalizeSearch(lq.Search); err != nil {
		return f, err
	}
	if f.Status, err = parseStatus(lq.Status); err != nil {
		return f, err
	}
	if f.UnitID, err = parseOptionalID(lq.UnitID, "unitId"); err != nil {
		return f, err
	}
	if f.RoleID, err = parseOptionalID(lq.RoleID, "roleId"); err != nil {
		return f, err
	}
	if f.Limit <= 0 {
		f.Limit = defaultAdminPageSize
	}
	return f, nil
}

// record mencatat perubahan master data di audit log perusahaan.
func (s *AdminService) record(ctx context.Context, q db.DBTX, user *auth.CurrentUser, companyCode, entity string,
	id int64, action string, before, after any) error {
	return s.audit(q).Record(ctx, audit.Entry{
		CompanyCode: companyCode, EntityType: entity, EntityID: id, Action: action,
		ActorID: user.ID, ActorName: user.Name, Snapshot: adminSnapshot{Before: before, After: after},
	})
}
