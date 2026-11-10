package organisasi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"office.backend/internal/auth"
	"office.backend/internal/platform/db"
)

// Query master data organisasi (kelola_*.go). Semua query disaring ke satu perusahaan.

// adminFilter adalah filter daftar yang sudah divalidasi.
type adminFilter struct {
	CompanyCode string
	Search      string
	Status      string // "" | ACTIVE | INACTIVE
	UnitID      int64  // 0 = semua
	RoleID      int64  // 0 = semua
	Offset      int
	Limit       int
}

// statusSQL adalah filter aktif/nonaktif untuk kolom is_active tertentu.
func statusSQL(column string) string {
	return "\n  AND (@status = '' OR (@status = 'ACTIVE' AND " + column + " = 1) OR (@status = 'INACTIVE' AND " + column + " = 0))"
}

func notFoundOr(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return db.ErrNotFound
	}
	return err
}

// --- Unit -----------------------------------------------------------------------------------

const adminUnitSelect = `
SELECT un.id, un.code, un.name, un.unit_type, un.parent_id, pu.name, un.is_active, hp.name, hu.name,
       (SELECT COUNT(*) FROM positions p WHERE p.unit_id = un.id AND p.is_active = 1 AND p.deleted_at IS NULL),
       (SELECT COUNT(*) FROM users u JOIN positions p ON p.id = u.position_id
        WHERE p.unit_id = un.id AND u.is_active = 1 AND u.deleted_at IS NULL),
       (SELECT COUNT(*) FROM units c WHERE c.parent_id = un.id AND c.is_active = 1 AND c.deleted_at IS NULL)
FROM units un%s
LEFT JOIN units pu ON pu.id = un.parent_id
OUTER APPLY (SELECT TOP 1 p.id, p.name FROM positions p
             WHERE p.unit_id = un.id AND p.is_unit_head = 1 AND p.is_active = 1 AND p.deleted_at IS NULL
             ORDER BY p.id) hp
OUTER APPLY (SELECT TOP 1 u.name FROM users u
             WHERE u.position_id = hp.id AND u.is_active = 1 AND u.deleted_at IS NULL
             ORDER BY u.id) hu
WHERE un.company_code = @companyCode AND un.deleted_at IS NULL`

func scanAdminUnit(scan func(...any) error) (*AdminUnit, error) {
	var u AdminUnit
	err := scan(&u.ID, &u.Code, &u.Name, &u.UnitType, &u.ParentID, &u.ParentName, &u.IsActive, &u.HeadPosition,
		&u.HeadName, &u.ActivePositions, &u.ActiveUsers, &u.ActiveChildren)
	return &u, err
}

// ListAdminUnits mengambil semua unit perusahaan (urut nama), opsional dicari dan disaring status.
func (r *Repository) ListAdminUnits(ctx context.Context, f adminFilter) ([]AdminUnit, error) {
	q := fmt.Sprintf(adminUnitSelect, "") + statusSQL("un.is_active")
	args := []any{sql.Named("companyCode", f.CompanyCode), sql.Named("status", f.Status)}
	if f.Search != "" {
		q += "\n  AND (un.name LIKE @pattern ESCAPE '\\' OR un.unit_type LIKE @pattern ESCAPE '\\')"
		args = append(args, sql.Named("pattern", db.ContainsPattern(f.Search)))
	}
	q += "\nORDER BY un.name, un.id"
	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil unit: %w", err)
	}
	defer rows.Close()
	out := []AdminUnit{}
	for rows.Next() {
		u, err := scanAdminUnit(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("membaca unit: %w", err)
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

// FindAdminUnit mengambil satu unit perusahaan; lock=true mengunci barisnya sampai transaksi selesai.
func (r *Repository) FindAdminUnit(ctx context.Context, companyCode string, id int64, lock bool) (*AdminUnit, error) {
	hint := ""
	if lock {
		hint = " WITH (UPDLOCK, ROWLOCK)"
	}
	q := fmt.Sprintf(adminUnitSelect, hint) + "\n  AND un.id = @id"
	u, err := scanAdminUnit(r.q.QueryRowContext(ctx, q, sql.Named("companyCode", companyCode), sql.Named("id", id)).Scan)
	if err != nil {
		return nil, notFoundOr(err)
	}
	return u, nil
}

// InsertUnit menambah unit dan mengembalikan ID-nya.
func (r *Repository) InsertUnit(ctx context.Context, companyCode, code, name, unitType string, parentID *int64) (int64, error) {
	const q = `
INSERT INTO units (company_code, parent_id, unit_type, code, name)
OUTPUT inserted.id
VALUES (@companyCode, @parentId, @unitType, @code, @name)`
	var id int64
	err := r.q.QueryRowContext(ctx, q, sql.Named("companyCode", companyCode), sql.Named("parentId", parentID),
		sql.Named("unitType", unitType), sql.Named("code", code), sql.Named("name", name)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("menambah unit: %w", err)
	}
	return id, nil
}

// UpdateUnit mengubah nama, jenis, dan induk unit.
func (r *Repository) UpdateUnit(ctx context.Context, id int64, name, unitType string, parentID *int64) error {
	const q = `UPDATE units SET name = @name, unit_type = @unitType, parent_id = @parentId, updated_at = SYSDATETIMEOFFSET() WHERE id = @id`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", id), sql.Named("name", name), sql.Named("unitType", unitType),
		sql.Named("parentId", parentID)); err != nil {
		return fmt.Errorf("mengubah unit: %w", err)
	}
	return nil
}

// SetUnitActive mengaktifkan/menonaktifkan unit.
func (r *Repository) SetUnitActive(ctx context.Context, id int64, active bool) error {
	const q = `UPDATE units SET is_active = @active, updated_at = SYSDATETIMEOFFSET() WHERE id = @id`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", id), sql.Named("active", active)); err != nil {
		return fmt.Errorf("mengubah status unit: %w", err)
	}
	return nil
}

// IsAncestorOrSelf mengecek apakah unitID adalah candidateID sendiri atau salah satu induknya ke
// atas. Dipakai mencegah induk unit membentuk lingkaran.
func (r *Repository) IsAncestorOrSelf(ctx context.Context, unitID, candidateID int64) (bool, error) {
	const q = `
WITH chain AS (
    SELECT id, parent_id FROM units WHERE id = @candidate
    UNION ALL
    SELECT u.id, u.parent_id FROM units u JOIN chain c ON u.id = c.parent_id
)
SELECT CASE WHEN EXISTS (SELECT 1 FROM chain WHERE id = @unit) THEN 1 ELSE 0 END`
	var found bool
	if err := r.q.QueryRowContext(ctx, q, sql.Named("candidate", candidateID), sql.Named("unit", unitID)).Scan(&found); err != nil {
		return false, fmt.Errorf("memeriksa induk unit: %w", err)
	}
	return found, nil
}

// --- Jabatan --------------------------------------------------------------------------------

// holderSeparator memisahkan nama pemegang jabatan di STRING_AGG (karakter kontrol, tidak ada di nama).
const holderSeparator = "\x1f"

const adminPositionSelect = `
SELECT p.id, p.name, p.unit_id, un.name, lv.id, lv.code, lv.name, p.is_unit_head, p.is_active,
       (SELECT COUNT(*) FROM users u WHERE u.position_id = p.id AND u.is_active = 1 AND u.deleted_at IS NULL),
       (SELECT STRING_AGG(u.name, CHAR(31)) WITHIN GROUP (ORDER BY u.name) FROM users u
        WHERE u.position_id = p.id AND u.is_active = 1 AND u.deleted_at IS NULL)%s
FROM positions p%s
JOIN units un        ON un.id = p.unit_id
LEFT JOIN lookups lv ON lv.id = p.level_id
WHERE un.company_code = @companyCode AND p.deleted_at IS NULL`

func scanAdminPosition(scan func(...any) error, extra ...any) (*AdminPosition, error) {
	var (
		p                  AdminPosition
		lvID               sql.NullInt64
		lvCode, lvName, hs sql.NullString
	)
	dest := append([]any{&p.ID, &p.Name, &p.UnitID, &p.UnitName, &lvID, &lvCode, &lvName, &p.IsUnitHead,
		&p.IsActive, &p.ActiveUsers, &hs}, extra...)
	if err := scan(dest...); err != nil {
		return nil, err
	}
	if lvID.Valid {
		p.Level = &LevelRef{ID: lvID.Int64, Code: lvCode.String, Name: lvName.String}
		p.IsOfficial = auth.IsOfficialLevel(lvCode.String)
	}
	p.Holders = []string{}
	if hs.Valid && hs.String != "" {
		p.Holders = strings.Split(hs.String, holderSeparator)
	}
	return &p, nil
}

// ListAdminPositions mengambil jabatan perusahaan (urut unit lalu nama) beserta jumlah totalnya.
func (r *Repository) ListAdminPositions(ctx context.Context, f adminFilter) ([]AdminPosition, int, error) {
	q := fmt.Sprintf(adminPositionSelect, ",\n       COUNT(*) OVER ()", "") + statusSQL("p.is_active") +
		"\n  AND (@unitId = 0 OR p.unit_id = @unitId)"
	args := []any{sql.Named("companyCode", f.CompanyCode), sql.Named("status", f.Status), sql.Named("unitId", f.UnitID),
		sql.Named("offset", f.Offset), sql.Named("limit", f.Limit)}
	if f.Search != "" {
		q += "\n  AND (p.name LIKE @pattern ESCAPE '\\' OR un.name LIKE @pattern ESCAPE '\\')"
		args = append(args, sql.Named("pattern", db.ContainsPattern(f.Search)))
	}
	q += "\nORDER BY un.name, p.name, p.id\nOFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY"
	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil jabatan: %w", err)
	}
	defer rows.Close()
	out, total := []AdminPosition{}, 0
	for rows.Next() {
		p, err := scanAdminPosition(rows.Scan, &total)
		if err != nil {
			return nil, 0, fmt.Errorf("membaca jabatan: %w", err)
		}
		out = append(out, *p)
	}
	return out, total, rows.Err()
}

// FindAdminPosition mengambil satu jabatan perusahaan; lock=true mengunci barisnya.
func (r *Repository) FindAdminPosition(ctx context.Context, companyCode string, id int64, lock bool) (*AdminPosition, error) {
	hint := ""
	if lock {
		hint = " WITH (UPDLOCK, ROWLOCK)"
	}
	q := fmt.Sprintf(adminPositionSelect, "", hint) + "\n  AND p.id = @id"
	p, err := scanAdminPosition(r.q.QueryRowContext(ctx, q, sql.Named("companyCode", companyCode), sql.Named("id", id)).Scan)
	if err != nil {
		return nil, notFoundOr(err)
	}
	return p, nil
}

// FindLevel mengambil posisi (lookup POSITION_LEVEL) milik perusahaan.
func (r *Repository) FindLevel(ctx context.Context, companyCode string, id int64, code string) (*LevelRef, error) {
	const q = `
SELECT id, code, name FROM lookups
WHERE category = 'POSITION_LEVEL' AND company_code = @companyCode AND deleted_at IS NULL
  AND ((@id <> 0 AND id = @id) OR (@id = 0 AND code = @code))`
	var lv LevelRef
	err := r.q.QueryRowContext(ctx, q, sql.Named("companyCode", companyCode), sql.Named("id", id), sql.Named("code", code)).
		Scan(&lv.ID, &lv.Code, &lv.Name)
	if err != nil {
		return nil, notFoundOr(err)
	}
	return &lv, nil
}

// ListLevels mengambil posisi (lookup POSITION_LEVEL) perusahaan, urut tingkatnya.
func (r *Repository) ListLevels(ctx context.Context, companyCode string) ([]LevelRef, error) {
	const q = `
SELECT id, code, name FROM lookups
WHERE category = 'POSITION_LEVEL' AND company_code = @companyCode AND deleted_at IS NULL
ORDER BY sort_order, name`
	rows, err := r.q.QueryContext(ctx, q, sql.Named("companyCode", companyCode))
	if err != nil {
		return nil, fmt.Errorf("mengambil posisi: %w", err)
	}
	defer rows.Close()
	out := []LevelRef{}
	for rows.Next() {
		var lv LevelRef
		if err := rows.Scan(&lv.ID, &lv.Code, &lv.Name); err != nil {
			return nil, fmt.Errorf("membaca posisi: %w", err)
		}
		out = append(out, lv)
	}
	return out, rows.Err()
}

// OtherActiveHead mengecek apakah unit sudah punya jabatan pimpinan aktif selain exceptID.
func (r *Repository) OtherActiveHead(ctx context.Context, unitID, exceptID int64) (bool, error) {
	const q = `
SELECT CASE WHEN EXISTS (
    SELECT 1 FROM positions WITH (UPDLOCK, HOLDLOCK)
    WHERE unit_id = @unitId AND id <> @exceptId AND is_unit_head = 1 AND is_active = 1 AND deleted_at IS NULL
) THEN 1 ELSE 0 END`
	var found bool
	if err := r.q.QueryRowContext(ctx, q, sql.Named("unitId", unitID), sql.Named("exceptId", exceptID)).Scan(&found); err != nil {
		return false, fmt.Errorf("memeriksa pimpinan unit: %w", err)
	}
	return found, nil
}

// PositionNameTaken mengecek nama jabatan aktif yang sama di satu unit (selain exceptID).
func (r *Repository) PositionNameTaken(ctx context.Context, unitID int64, name string, exceptID int64) (bool, error) {
	const q = `
SELECT CASE WHEN EXISTS (
    SELECT 1 FROM positions WITH (UPDLOCK, HOLDLOCK)
    WHERE unit_id = @unitId AND name = @name AND id <> @exceptId AND is_active = 1 AND deleted_at IS NULL
) THEN 1 ELSE 0 END`
	var found bool
	if err := r.q.QueryRowContext(ctx, q, sql.Named("unitId", unitID), sql.Named("name", name), sql.Named("exceptId", exceptID)).Scan(&found); err != nil {
		return false, fmt.Errorf("memeriksa nama jabatan: %w", err)
	}
	return found, nil
}

// InsertPosition menambah jabatan dan mengembalikan ID-nya.
func (r *Repository) InsertPosition(ctx context.Context, unitID int64, name string, levelID int64, isHead bool) (int64, error) {
	const q = `
INSERT INTO positions (unit_id, name, is_unit_head, level_id)
OUTPUT inserted.id
VALUES (@unitId, @name, @isHead, @levelId)`
	var id int64
	if err := r.q.QueryRowContext(ctx, q, sql.Named("unitId", unitID), sql.Named("name", name),
		sql.Named("isHead", isHead), sql.Named("levelId", levelID)).Scan(&id); err != nil {
		return 0, fmt.Errorf("menambah jabatan: %w", err)
	}
	return id, nil
}

// UpdatePosition mengubah jabatan.
func (r *Repository) UpdatePosition(ctx context.Context, id, unitID int64, name string, levelID int64, isHead bool) error {
	const q = `
UPDATE positions
SET unit_id = @unitId, name = @name, level_id = @levelId, is_unit_head = @isHead, updated_at = SYSDATETIMEOFFSET()
WHERE id = @id`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", id), sql.Named("unitId", unitID), sql.Named("name", name),
		sql.Named("levelId", levelID), sql.Named("isHead", isHead)); err != nil {
		return fmt.Errorf("mengubah jabatan: %w", err)
	}
	return nil
}

// SetPositionActive mengaktifkan/menonaktifkan jabatan.
func (r *Repository) SetPositionActive(ctx context.Context, id int64, active bool) error {
	const q = `UPDATE positions SET is_active = @active, updated_at = SYSDATETIMEOFFSET() WHERE id = @id`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", id), sql.Named("active", active)); err != nil {
		return fmt.Errorf("mengubah status jabatan: %w", err)
	}
	return nil
}

// --- User -----------------------------------------------------------------------------------

const adminUserSelect = `
SELECT u.id, u.employee_number, u.name, u.email, u.is_active,
       CASE WHEN u.password_hash IS NULL THEN 0 ELSE 1 END,
       r.id, r.code, r.name, p.id, p.name, lv.id, lv.code, lv.name, p.is_unit_head,
       un.id, un.name, u.created_at, u.updated_at%s
FROM users u%s
JOIN roles r         ON r.id = u.role_id
JOIN positions p     ON p.id = u.position_id
JOIN units un        ON un.id = p.unit_id
LEFT JOIN lookups lv ON lv.id = p.level_id
WHERE un.company_code = @companyCode AND u.deleted_at IS NULL`

func scanAdminUser(scan func(...any) error, extra ...any) (*AdminUser, error) {
	var (
		u              AdminUser
		lvID           sql.NullInt64
		lvCode, lvName sql.NullString
	)
	dest := append([]any{&u.ID, &u.EmployeeNumber, &u.Name, &u.Email, &u.IsActive, &u.HasPassword,
		&u.Role.ID, &u.Role.Code, &u.Role.Name, &u.Position.ID, &u.Position.Name, &lvID, &lvCode, &lvName,
		&u.Position.IsUnitHead, &u.Unit.ID, &u.Unit.Name, &u.CreatedAt, &u.UpdatedAt}, extra...)
	if err := scan(dest...); err != nil {
		return nil, err
	}
	if lvID.Valid {
		u.Position.Level = &LevelRef{ID: lvID.Int64, Code: lvCode.String, Name: lvName.String}
		u.Position.IsOfficial = auth.IsOfficialLevel(lvCode.String)
	}
	return &u, nil
}

// ListAdminUsers mengambil user perusahaan (urut nama) beserta jumlah totalnya.
func (r *Repository) ListAdminUsers(ctx context.Context, f adminFilter) ([]AdminUser, int, error) {
	q := fmt.Sprintf(adminUserSelect, ",\n       COUNT(*) OVER ()", "") + statusSQL("u.is_active") +
		"\n  AND (@unitId = 0 OR un.id = @unitId)\n  AND (@roleId = 0 OR r.id = @roleId)"
	args := []any{sql.Named("companyCode", f.CompanyCode), sql.Named("status", f.Status), sql.Named("unitId", f.UnitID),
		sql.Named("roleId", f.RoleID), sql.Named("offset", f.Offset), sql.Named("limit", f.Limit)}
	if f.Search != "" {
		q += "\n  AND (u.name LIKE @pattern ESCAPE '\\' OR u.employee_number LIKE @pattern ESCAPE '\\'" +
			" OR u.email LIKE @pattern ESCAPE '\\' OR p.name LIKE @pattern ESCAPE '\\')"
		args = append(args, sql.Named("pattern", db.ContainsPattern(f.Search)))
	}
	q += "\nORDER BY u.name, u.id\nOFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY"
	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil user: %w", err)
	}
	defer rows.Close()
	out, total := []AdminUser{}, 0
	for rows.Next() {
		u, err := scanAdminUser(rows.Scan, &total)
		if err != nil {
			return nil, 0, fmt.Errorf("membaca user: %w", err)
		}
		out = append(out, *u)
	}
	return out, total, rows.Err()
}

// FindAdminUser mengambil satu user perusahaan; lock=true mengunci barisnya.
func (r *Repository) FindAdminUser(ctx context.Context, companyCode string, id int64, lock bool) (*AdminUser, error) {
	hint := ""
	if lock {
		hint = " WITH (UPDLOCK, ROWLOCK)"
	}
	q := fmt.Sprintf(adminUserSelect, "", hint) + "\n  AND u.id = @id"
	u, err := scanAdminUser(r.q.QueryRowContext(ctx, q, sql.Named("companyCode", companyCode), sql.Named("id", id)).Scan)
	if err != nil {
		return nil, notFoundOr(err)
	}
	return u, nil
}

// EmployeeNumberTaken mengecek NIK yang sudah dipakai user lain (di semua perusahaan, karena NIK
// dipakai untuk login).
func (r *Repository) EmployeeNumberTaken(ctx context.Context, nik string, exceptID int64) (bool, error) {
	const q = `
SELECT CASE WHEN EXISTS (
    SELECT 1 FROM users WITH (UPDLOCK, HOLDLOCK)
    WHERE employee_number = @nik AND id <> @exceptId AND deleted_at IS NULL
) THEN 1 ELSE 0 END`
	var found bool
	if err := r.q.QueryRowContext(ctx, q, sql.Named("nik", nik), sql.Named("exceptId", exceptID)).Scan(&found); err != nil {
		return false, fmt.Errorf("memeriksa NIK: %w", err)
	}
	return found, nil
}

// FindRole mengambil role aktif berdasarkan ID, atau berdasarkan kode bila id = 0.
func (r *Repository) FindRole(ctx context.Context, id int64, code string) (*RoleRef, error) {
	const q = `
SELECT id, code, name FROM roles
WHERE is_active = 1 AND deleted_at IS NULL AND ((@id <> 0 AND id = @id) OR (@id = 0 AND code = @code))`
	var role RoleRef
	if err := r.q.QueryRowContext(ctx, q, sql.Named("id", id), sql.Named("code", code)).Scan(&role.ID, &role.Code, &role.Name); err != nil {
		return nil, notFoundOr(err)
	}
	return &role, nil
}

// ListRoles mengambil semua role aktif (urut nama).
func (r *Repository) ListRoles(ctx context.Context) ([]RoleRef, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT id, code, name FROM roles WHERE is_active = 1 AND deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("mengambil role: %w", err)
	}
	defer rows.Close()
	out := []RoleRef{}
	for rows.Next() {
		var role RoleRef
		if err := rows.Scan(&role.ID, &role.Code, &role.Name); err != nil {
			return nil, fmt.Errorf("membaca role: %w", err)
		}
		out = append(out, role)
	}
	return out, rows.Err()
}

// CountOtherActiveAdmins menghitung user aktif ber-role ADMIN di perusahaan selain exceptID.
func (r *Repository) CountOtherActiveAdmins(ctx context.Context, companyCode string, exceptID int64) (int, error) {
	const q = `
SELECT COUNT(*)
FROM users u WITH (UPDLOCK, HOLDLOCK)
JOIN roles r     ON r.id = u.role_id AND r.code = @admin
JOIN positions p ON p.id = u.position_id
JOIN units un    ON un.id = p.unit_id
WHERE un.company_code = @companyCode AND u.id <> @exceptId AND u.is_active = 1 AND u.deleted_at IS NULL`
	var n int
	if err := r.q.QueryRowContext(ctx, q, sql.Named("admin", auth.RoleAdmin), sql.Named("companyCode", companyCode),
		sql.Named("exceptId", exceptID)).Scan(&n); err != nil {
		return 0, fmt.Errorf("menghitung admin: %w", err)
	}
	return n, nil
}

// InsertUser menambah user dan mengembalikan ID-nya.
func (r *Repository) InsertUser(ctx context.Context, u userRecord, passwordHash string) (int64, error) {
	const q = `
INSERT INTO users (position_id, role_id, employee_number, name, email, password_hash)
OUTPUT inserted.id
VALUES (@positionId, @roleId, @nik, @name, @email, @hash)`
	var id int64
	if err := r.q.QueryRowContext(ctx, q, sql.Named("positionId", u.PositionID), sql.Named("roleId", u.RoleID),
		sql.Named("nik", u.EmployeeNumber), sql.Named("name", u.Name), sql.Named("email", u.Email),
		sql.Named("hash", passwordHash)).Scan(&id); err != nil {
		return 0, fmt.Errorf("menambah user: %w", err)
	}
	return id, nil
}

// UpdateUser mengubah data user. logout=true menaikkan token_version (semua sesi user berakhir).
func (r *Repository) UpdateUser(ctx context.Context, id int64, u userRecord, logout bool) error {
	const q = `
UPDATE users
SET employee_number = @nik, name = @name, email = @email, position_id = @positionId, role_id = @roleId,
    token_version = token_version + CASE WHEN @logout = 1 THEN 1 ELSE 0 END, updated_at = SYSDATETIMEOFFSET()
WHERE id = @id`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", id), sql.Named("nik", u.EmployeeNumber), sql.Named("name", u.Name),
		sql.Named("email", u.Email), sql.Named("positionId", u.PositionID), sql.Named("roleId", u.RoleID),
		sql.Named("logout", logout)); err != nil {
		return fmt.Errorf("mengubah user: %w", err)
	}
	return nil
}

// SetUserActive mengaktifkan/menonaktifkan user; selalu mengakhiri semua sesinya.
func (r *Repository) SetUserActive(ctx context.Context, id int64, active bool) error {
	const q = `
UPDATE users SET is_active = @active, token_version = token_version + 1, updated_at = SYSDATETIMEOFFSET()
WHERE id = @id`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", id), sql.Named("active", active)); err != nil {
		return fmt.Errorf("mengubah status user: %w", err)
	}
	return nil
}

// SetPassword mengganti password user dan mengakhiri semua sesinya.
func (r *Repository) SetPassword(ctx context.Context, id int64, hash string) error {
	const q = `
UPDATE users SET password_hash = @hash, token_version = token_version + 1, updated_at = SYSDATETIMEOFFSET()
WHERE id = @id`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", id), sql.Named("hash", hash)); err != nil {
		return fmt.Errorf("mengganti password: %w", err)
	}
	return nil
}

// CompanyExists mengecek perusahaan (aktif maupun nonaktif) yang belum dihapus.
func (r *Repository) CompanyExists(ctx context.Context, companyCode string) (bool, error) {
	var found bool
	err := r.q.QueryRowContext(ctx,
		`SELECT CASE WHEN EXISTS (SELECT 1 FROM companies WHERE code = @code AND deleted_at IS NULL) THEN 1 ELSE 0 END`,
		sql.Named("code", companyCode)).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("memeriksa perusahaan: %w", err)
	}
	return found, nil
}
