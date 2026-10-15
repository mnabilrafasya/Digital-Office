package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
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
	err := r.db.QueryRowContext(ctx, q, sql.Named("employeeNumber", employeeNumber)).
		Scan(&rec.UserID, &rec.PasswordHash, &rec.IsActive, &rec.RoleIsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("mencari user untuk login: %w", err)
	}
	return &rec, nil
}

// FindActiveByID memuat user aktif (role aktif) beserta jabatan, unit, perusahaan, dan permission-nya
func (r *UserRepository) FindActiveByID(ctx context.Context, userID int64) (*model.CurrentUser, error) {
	const q = `
SELECT u.id, u.employee_number, u.name, u.email,
       p.id, p.name, p.is_unit_head,
       un.id, un.code, un.name,
       c.code, c.name,
       r.code, r.name, r.id
FROM users u
JOIN positions p  ON p.id = u.position_id
JOIN units un     ON un.id = p.unit_id
JOIN companies c  ON c.code = un.company_code
JOIN roles r      ON r.id = u.role_id
WHERE u.id = @userId
  AND u.is_active = 1 AND u.deleted_at IS NULL
  AND r.is_active = 1 AND r.deleted_at IS NULL`

	var u model.CurrentUser
	err := r.db.QueryRowContext(ctx, q, sql.Named("userId", userID)).Scan(
		&u.ID, &u.EmployeeNumber, &u.Name, &u.Email,
		&u.Position.ID, &u.Position.Name, &u.Position.IsUnitHead,
		&u.Unit.ID, &u.Unit.Code, &u.Unit.Name,
		&u.Company.Code, &u.Company.Name,
		&u.Role.Code, &u.Role.Name, &u.Role.ID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("memuat user: %w", err)
	}

	const qPerm = `
SELECT code FROM permissions
WHERE role_id = @roleId AND deleted_at IS NULL
ORDER BY code`

	rows, err := r.db.QueryContext(ctx, qPerm, sql.Named("roleId", u.Role.ID))
	if err != nil {
		return nil, fmt.Errorf("memuat permission: %w", err)
	}
	defer rows.Close()

	u.Permissions = []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("membaca permission: %w", err)
		}
		u.Permissions = append(u.Permissions, code)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca permission: %w", err)
	}
	return &u, nil
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

	rows, err := r.db.QueryContext(ctx, q, args...)
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
