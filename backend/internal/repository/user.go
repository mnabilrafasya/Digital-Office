package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// UserRepository berisi query tabel users.
type UserRepository struct {
	q db.DBTX
}

// NewUserRepository membuat UserRepository di atas koneksi atau transaksi.
func NewUserRepository(q db.DBTX) *UserRepository {
	return &UserRepository{q: q}
}

// FindLoginByEmployeeNumber mengambil data untuk memeriksa login berdasarkan NIK
func (r *UserRepository) FindLoginByEmployeeNumber(ctx context.Context, employeeNumber string) (*model.LoginRecord, error) {
	const q = `
SELECT u.id, u.password_hash, u.is_active,
       CASE WHEN r.is_active = 1 AND r.deleted_at IS NULL THEN 1 ELSE 0 END
FROM users u
JOIN roles r ON r.id = u.role_id
WHERE u.employee_number = @employeeNumber AND u.deleted_at IS NULL`

	var rec model.LoginRecord
	err := r.q.QueryRowContext(ctx, q, sql.Named("employeeNumber", employeeNumber)).
		Scan(&rec.UserID, &rec.PasswordHash, &rec.IsActive, &rec.RoleIsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("mencari user untuk login: %w", err)
	}
	return &rec, nil
}

// FindActiveByID memuat user aktif (role aktif) beserta jabatan, level jabatan, unit, perusahaan,
// permission, dan token_version dalam satu query (dipanggil di setiap request yang wajib login).
func (r *UserRepository) FindActiveByID(ctx context.Context, userID int64) (*model.CurrentUser, error) {
	const q = `
SELECT u.id, u.employee_number, u.name, u.email, u.token_version,
       p.id, p.name, p.is_unit_head, lv.code, lv.name,
       un.id, un.code, un.name,
       c.code, c.name,
       r.code, r.name, r.id,
       (SELECT STRING_AGG(pm.code, ',') WITHIN GROUP (ORDER BY pm.code)
        FROM permissions pm
        WHERE pm.role_id = r.id AND pm.deleted_at IS NULL)
FROM users u
JOIN positions p     ON p.id = u.position_id
JOIN units un        ON un.id = p.unit_id
JOIN companies c     ON c.code = un.company_code
JOIN roles r         ON r.id = u.role_id
LEFT JOIN lookups lv ON lv.id = p.level_id AND lv.deleted_at IS NULL
WHERE u.id = @userId
  AND u.is_active = 1 AND u.deleted_at IS NULL
  AND r.is_active = 1 AND r.deleted_at IS NULL`

	var (
		u                    model.CurrentUser
		levelCode, levelName sql.NullString
		perms                sql.NullString
	)
	err := r.q.QueryRowContext(ctx, q, sql.Named("userId", userID)).Scan(
		&u.ID, &u.EmployeeNumber, &u.Name, &u.Email, &u.TokenVersion,
		&u.Position.ID, &u.Position.Name, &u.Position.IsUnitHead, &levelCode, &levelName,
		&u.Unit.ID, &u.Unit.Code, &u.Unit.Name,
		&u.Company.Code, &u.Company.Name,
		&u.Role.Code, &u.Role.Name, &u.Role.ID,
		&perms,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("memuat user: %w", err)
	}

	if levelCode.Valid {
		u.Position.Level = &model.PositionLevel{Code: levelCode.String, Name: levelName.String}
	}
	u.Permissions = []string{}
	if perms.Valid && perms.String != "" {
		u.Permissions = strings.Split(perms.String, ",")
	}
	return &u, nil
}

// IncrementTokenVersion menaikkan token_version sehingga semua access token lama user tidak berlaku.
func (r *UserRepository) IncrementTokenVersion(ctx context.Context, userID int64) error {
	const q = `
UPDATE users
SET token_version = token_version + 1, updated_at = SYSDATETIMEOFFSET()
WHERE id = @userId AND deleted_at IS NULL`

	res, err := r.q.ExecContext(ctx, q, sql.Named("userId", userID))
	if err != nil {
		return fmt.Errorf("menaikkan token_version: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// CandidateFilter adalah kriteria pencarian kandidat reviewer/approver
type CandidateFilter struct {
	CompanyCode    string
	PermissionCode string
	ExcludeUserID  int64  // 0 = tidak ada user yang dikecualikan
	Search         string // "" = tidak mencari; selain itu dicari di nama user dan nama jabatan
	Limit          int
}

// ListCandidates mengambil user aktif di satu perusahaan yang role-nya punya permission tertentu
func (r *UserRepository) ListCandidates(ctx context.Context, f CandidateFilter) ([]model.Candidate, error) {
	q := `
SELECT TOP (@limit)
       u.id, u.name, u.employee_number,
       p.name  AS position_name,
       un.name AS unit_name
FROM users u
JOIN positions p ON p.id = u.position_id AND p.is_active = 1 AND p.deleted_at IS NULL
JOIN units un    ON un.id = p.unit_id   AND un.is_active = 1 AND un.deleted_at IS NULL
JOIN roles r     ON r.id = u.role_id    AND r.is_active = 1 AND r.deleted_at IS NULL
WHERE un.company_code = @companyCode
  AND u.is_active = 1 AND u.deleted_at IS NULL
  AND EXISTS (
        SELECT 1 FROM permissions pm
        WHERE pm.role_id = u.role_id AND pm.code = @permissionCode AND pm.deleted_at IS NULL
  )`
	args := []any{
		sql.Named("limit", f.Limit),
		sql.Named("companyCode", f.CompanyCode),
		sql.Named("permissionCode", f.PermissionCode),
	}
	if f.ExcludeUserID != 0 {
		q += "\n  AND u.id <> @currentUserId"
		args = append(args, sql.Named("currentUserId", f.ExcludeUserID))
	}
	if f.Search != "" {
		q += "\n  AND (u.name LIKE @pattern ESCAPE '\\' OR p.name LIKE @pattern ESCAPE '\\')"
		args = append(args, sql.Named("pattern", ContainsPattern(f.Search)))
	}
	q += "\nORDER BY u.name"

	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mencari kandidat: %w", err)
	}
	defer rows.Close()

	out := []model.Candidate{}
	for rows.Next() {
		var c model.Candidate
		if err := rows.Scan(&c.ID, &c.Name, &c.EmployeeNumber, &c.PositionName, &c.UnitName); err != nil {
			return nil, fmt.Errorf("membaca kandidat: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca kandidat: %w", err)
	}
	return out, nil
}
