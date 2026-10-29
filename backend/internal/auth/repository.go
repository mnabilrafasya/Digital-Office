package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"office.backend/internal/platform/db"
)

// Repository berisi query tabel users untuk login, memuat user yang login, dan logout.
type Repository struct {
	q db.DBTX
}

// NewRepository membuat Repository di atas koneksi atau transaksi.
func NewRepository(q db.DBTX) *Repository {
	return &Repository{q: q}
}

// FindLoginByEmployeeNumber mengambil data untuk memeriksa login berdasarkan NIK
func (r *Repository) FindLoginByEmployeeNumber(ctx context.Context, employeeNumber string) (*LoginRecord, error) {
	const q = `
SELECT u.id, u.password_hash, u.is_active,
       CASE WHEN r.is_active = 1 AND r.deleted_at IS NULL THEN 1 ELSE 0 END
FROM users u
JOIN roles r ON r.id = u.role_id
WHERE u.employee_number = @employeeNumber AND u.deleted_at IS NULL`

	var rec LoginRecord
	err := r.q.QueryRowContext(ctx, q, sql.Named("employeeNumber", employeeNumber)).
		Scan(&rec.UserID, &rec.PasswordHash, &rec.IsActive, &rec.RoleIsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("mencari user untuk login: %w", err)
	}
	return &rec, nil
}

// FindActiveByID memuat user aktif (role aktif) beserta jabatan, level jabatan, unit, perusahaan,
// permission, dan token_version dalam satu query (dipanggil di setiap request yang wajib login).
func (r *Repository) FindActiveByID(ctx context.Context, userID int64) (*CurrentUser, error) {
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
		u                    CurrentUser
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
		return nil, db.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("memuat user: %w", err)
	}

	if levelCode.Valid {
		u.Position.Level = &PositionLevel{Code: levelCode.String, Name: levelName.String}
	}
	u.Permissions = []string{}
	if perms.Valid && perms.String != "" {
		u.Permissions = strings.Split(perms.String, ",")
	}
	return &u, nil
}

// IncrementTokenVersion menaikkan token_version sehingga semua access token lama user tidak berlaku.
func (r *Repository) IncrementTokenVersion(ctx context.Context, userID int64) error {
	const q = `
UPDATE users
SET token_version = token_version + 1, updated_at = SYSDATETIMEOFFSET()
WHERE id = @userId AND deleted_at IS NULL`

	res, err := r.q.ExecContext(ctx, q, sql.Named("userId", userID))
	if err != nil {
		return fmt.Errorf("menaikkan token_version: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return db.ErrNotFound
	}
	return nil
}
