package perusahaan

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"office.backend/internal/platform/db"
)

// Repository berisi query tabel companies.
type Repository struct {
	q db.DBTX
}

// NewRepository membuat Repository di atas koneksi atau transaksi.
func NewRepository(q db.DBTX) *Repository {
	return &Repository{q: q}
}

// standardLevels adalah posisi (lookup POSITION_LEVEL) yang dibuat untuk setiap perusahaan baru,
// sama dengan seed reference 004_position_levels.sql.
var standardLevels = []struct {
	Code, Name, Note string
	Order            int
}{
	{"DIREKTUR", "Direktur", "Direksi", 1},
	{"SVP", "SVP", "Senior Vice President", 2},
	{"VP", "VP", "Vice President", 3},
	{"AVP", "AVP", "Assistant Vice President", 4},
	{"STAF", "Staf", "Pelaksana / staf", 5},
}

const companyColumns = `
SELECT c.code, c.name, c.city, ISNULL(c.letter_number_format, ''), c.watermark_text, c.is_active, c.created_at,
       (SELECT COUNT(*) FROM units un
        WHERE un.company_code = c.code AND un.is_active = 1 AND un.deleted_at IS NULL),
       (SELECT COUNT(*) FROM users u
        JOIN positions p ON p.id = u.position_id
        JOIN units un    ON un.id = p.unit_id
        WHERE un.company_code = c.code AND u.is_active = 1 AND u.deleted_at IS NULL),
       (SELECT COUNT(*) FROM letters l WHERE l.company_code = c.code AND l.deleted_at IS NULL)`

func scanCompany(scan func(...any) error, extra ...any) (*Company, error) {
	var c Company
	dest := append([]any{&c.Code, &c.Name, &c.City, &c.LetterNumberFormat, &c.WatermarkText, &c.IsActive,
		&c.CreatedAt, &c.ActiveUnits, &c.ActiveUsers, &c.Letters}, extra...)
	if err := scan(dest...); err != nil {
		return nil, err
	}
	return &c, nil
}

// ListCompanies mengambil perusahaan (urut kode) beserta jumlah totalnya.
func (r *Repository) ListCompanies(ctx context.Context, f CompanyQuery) ([]Company, int, error) {
	q := companyColumns + `, COUNT(*) OVER ()
FROM companies c
WHERE c.deleted_at IS NULL
  AND (@status = '' OR (@status = 'ACTIVE' AND c.is_active = 1) OR (@status = 'INACTIVE' AND c.is_active = 0))`
	args := []any{sql.Named("status", f.Status), sql.Named("offset", f.Offset), sql.Named("limit", f.Limit)}
	if f.Search != "" {
		q += "\n  AND (c.code LIKE @pattern ESCAPE '\\' OR c.name LIKE @pattern ESCAPE '\\')"
		args = append(args, sql.Named("pattern", db.ContainsPattern(f.Search)))
	}
	q += "\nORDER BY c.code\nOFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY"

	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil perusahaan: %w", err)
	}
	defer rows.Close()
	out, total := []Company{}, 0
	for rows.Next() {
		c, err := scanCompany(rows.Scan, &total)
		if err != nil {
			return nil, 0, fmt.Errorf("membaca perusahaan: %w", err)
		}
		out = append(out, *c)
	}
	return out, total, rows.Err()
}

// FindCompany mengambil satu perusahaan (db.ErrNotFound bila tidak ada). lock=true mengunci
// barisnya sampai transaksi selesai.
func (r *Repository) FindCompany(ctx context.Context, code string, lock bool) (*Company, error) {
	hint := ""
	if lock {
		hint = " WITH (UPDLOCK, ROWLOCK)"
	}
	q := companyColumns + "\nFROM companies c" + hint + "\nWHERE c.code = @code AND c.deleted_at IS NULL"
	c, err := scanCompany(r.q.QueryRowContext(ctx, q, sql.Named("code", code)).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("mengambil perusahaan: %w", err)
	}
	return c, nil
}

// CodeTaken mengecek apakah kode perusahaan sedang dipakai perusahaan yang belum dihapus.
func (r *Repository) CodeTaken(ctx context.Context, code string) (bool, error) {
	var taken bool
	err := r.q.QueryRowContext(ctx,
		`SELECT CASE WHEN EXISTS (SELECT 1 FROM companies WITH (UPDLOCK, HOLDLOCK) WHERE code = @code AND deleted_at IS NULL) THEN 1 ELSE 0 END`,
		sql.Named("code", code)).Scan(&taken)
	if err != nil {
		return false, fmt.Errorf("memeriksa kode perusahaan: %w", err)
	}
	return taken, nil
}

// InsertCompany menambah perusahaan beserta posisi standarnya (lookup POSITION_LEVEL). Format nomor
// dan watermark dibiarkan kosong: format bawaan dipakai, admin perusahaan yang mengaturnya.
// Apabila kode pernah dihapus (deleted_at IS NOT NULL), baris perusahaan diaktifkan kembali.
func (r *Repository) InsertCompany(ctx context.Context, in CompanyInput) error {
	const upsertCompany = `
IF EXISTS (SELECT 1 FROM companies WHERE code = @code AND deleted_at IS NOT NULL)
    UPDATE companies
    SET name = @name, city = @city, is_active = 1, deleted_at = NULL, updated_at = SYSDATETIMEOFFSET()
    WHERE code = @code
ELSE
    INSERT INTO companies (code, name, city) VALUES (@code, @name, @city)`
	if _, err := r.q.ExecContext(ctx, upsertCompany,
		sql.Named("code", in.Code), sql.Named("name", in.Name), sql.Named("city", in.City)); err != nil {
		return fmt.Errorf("menambah perusahaan: %w", err)
	}
	for _, lv := range standardLevels {
		const q = `
IF NOT EXISTS (SELECT 1 FROM lookups WHERE company_code = @company AND category = 'POSITION_LEVEL' AND code = @code AND deleted_at IS NULL)
    INSERT INTO lookups (company_code, category, code, name, sort_order, note)
    VALUES (@company, 'POSITION_LEVEL', @code, @name, @order, @note)`
		if _, err := r.q.ExecContext(ctx, q, sql.Named("company", in.Code), sql.Named("code", lv.Code),
			sql.Named("name", lv.Name), sql.Named("order", lv.Order), sql.Named("note", lv.Note)); err != nil {
			return fmt.Errorf("menambah posisi %s: %w", lv.Code, err)
		}
	}
	return nil
}

// DeleteCompany menghapus (soft-delete) perusahaan.
func (r *Repository) DeleteCompany(ctx context.Context, code string) error {
	const q = `
UPDATE companies
SET is_active = 0, deleted_at = SYSDATETIMEOFFSET(), updated_at = SYSDATETIMEOFFSET()
WHERE code = @code AND deleted_at IS NULL`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("code", code)); err != nil {
		return fmt.Errorf("menghapus perusahaan: %w", err)
	}
	return nil
}

// UpdateIdentity mengubah nama dan kota perusahaan.
func (r *Repository) UpdateIdentity(ctx context.Context, code, name string, city *string) error {
	const q = `UPDATE companies SET name = @name, city = @city, updated_at = SYSDATETIMEOFFSET() WHERE code = @code`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("code", code), sql.Named("name", name), sql.Named("city", city)); err != nil {
		return fmt.Errorf("mengubah perusahaan: %w", err)
	}
	return nil
}

// SetActive mengaktifkan/menonaktifkan perusahaan.
func (r *Repository) SetActive(ctx context.Context, code string, active bool) error {
	const q = `UPDATE companies SET is_active = @active, updated_at = SYSDATETIMEOFFSET() WHERE code = @code`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("code", code), sql.Named("active", active)); err != nil {
		return fmt.Errorf("mengubah status perusahaan: %w", err)
	}
	return nil
}

// UpdateSettings menyimpan format nomor dan watermark perusahaan.
func (r *Repository) UpdateSettings(ctx context.Context, code, format string, watermark *string) error {
	const q = `
UPDATE companies
SET letter_number_format = @format, watermark_text = @watermark, updated_at = SYSDATETIMEOFFSET()
WHERE code = @code`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("code", code), sql.Named("format", format),
		sql.Named("watermark", watermark)); err != nil {
		return fmt.Errorf("menyimpan pengaturan perusahaan: %w", err)
	}
	return nil
}
