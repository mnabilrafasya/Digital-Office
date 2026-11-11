package surat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"office.backend/internal/platform/db"
)

const (
	templateSelect = `
SELECT t.id, t.name, t.description, lt.id, lt.code, lt.name,
       u.id, u.name, t.created_at, t.updated_at, CONVERT(char(36), t.object_key)`
	templateJoins = `
JOIN users u         ON u.id = t.created_by
LEFT JOIN lookups lt ON lt.id = t.letter_type_id`
	templateColumns = templateSelect + `
FROM letter_templates t` + templateJoins
)

// ListTemplates mengambil template aktif milik companyCode, urut nama. letterTypeID nil = semua;
// selain itu template untuk jenis tersebut ditambah template yang berlaku untuk semua jenis.
func (r *Repository) ListTemplates(ctx context.Context, companyCode string, letterTypeID *int64) ([]Template, error) {
	q := templateColumns + `
WHERE t.company_code = @companyCode AND t.is_active = 1 AND t.deleted_at IS NULL`
	args := []any{sql.Named("companyCode", companyCode)}
	if letterTypeID != nil {
		q += "\n  AND (t.letter_type_id = @letterTypeId OR t.letter_type_id IS NULL)"
		args = append(args, sql.Named("letterTypeId", *letterTypeID))
	}
	q += "\nORDER BY t.name"

	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil template: %w", err)
	}
	defer rows.Close()

	out := []Template{}
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca template: %w", err)
	}
	return out, nil
}

// FindTemplate mengambil satu template aktif milik companyCode (db.ErrNotFound bila tidak ada).
func (r *Repository) FindTemplate(ctx context.Context, companyCode string, id int64) (*Template, error) {
	return r.findTemplate(ctx, templateColumns+`
WHERE t.id = @id AND t.company_code = @companyCode AND t.is_active = 1 AND t.deleted_at IS NULL`,
		sql.Named("id", id), sql.Named("companyCode", companyCode))
}

// LockTemplate sama dengan FindTemplate sambil mengunci barisnya untuk diubah. Wajib di dalam transaksi.
func (r *Repository) LockTemplate(ctx context.Context, companyCode string, id int64) (*Template, error) {
	return r.findTemplate(ctx, templateSelect+`
FROM letter_templates t WITH (UPDLOCK, ROWLOCK)`+templateJoins+`
WHERE t.id = @id AND t.company_code = @companyCode AND t.is_active = 1 AND t.deleted_at IS NULL`,
		sql.Named("id", id), sql.Named("companyCode", companyCode))
}

// TemplateNameTaken mengecek apakah nama sudah dipakai template lain milik companyCode
// (exceptID 0 = tidak ada yang dikecualikan). Unique index tetap menjadi penjaga terakhir.
func (r *Repository) TemplateNameTaken(ctx context.Context, companyCode, name string, exceptID int64) (bool, error) {
	const q = `
SELECT CASE WHEN EXISTS (
    SELECT 1 FROM letter_templates
    WHERE company_code = @companyCode AND name = @name AND id <> @exceptId AND deleted_at IS NULL
) THEN 1 ELSE 0 END`
	var taken bool
	err := r.q.QueryRowContext(ctx, q, sql.Named("companyCode", companyCode), sql.Named("name", name),
		sql.Named("exceptId", exceptID)).Scan(&taken)
	if err != nil {
		return false, fmt.Errorf("memeriksa nama template: %w", err)
	}
	return taken, nil
}

// InsertTemplate menyimpan template baru dan mengembalikan ID-nya.
func (r *Repository) InsertTemplate(ctx context.Context, companyCode string, d TemplateData, objectKey string, createdBy int64) (int64, error) {
	const q = `
INSERT INTO letter_templates (company_code, letter_type_id, name, description, object_key, created_by)
OUTPUT inserted.id
VALUES (@companyCode, @letterTypeId, @name, @description, @objectKey, @createdBy)`
	var id int64
	err := r.q.QueryRowContext(ctx, q,
		sql.Named("companyCode", companyCode),
		sql.Named("letterTypeId", d.LetterTypeID),
		sql.Named("name", d.Name),
		sql.Named("description", d.Description),
		sql.Named("objectKey", objectKey),
		sql.Named("createdBy", createdBy),
	).Scan(&id)
	if db.IsUniqueViolation(err) {
		return 0, errTemplateNameTaken
	}
	if err != nil {
		return 0, fmt.Errorf("menyimpan template: %w", err)
	}
	return id, nil
}

// UpdateTemplate mengganti data template; objectKey "" = file tidak diganti.
func (r *Repository) UpdateTemplate(ctx context.Context, id int64, d TemplateData, objectKey string) error {
	const q = `
UPDATE letter_templates
SET name = @name, description = @description, letter_type_id = @letterTypeId,
    object_key = COALESCE(CONVERT(uniqueidentifier, NULLIF(@objectKey, '')), object_key),
    updated_at = SYSDATETIMEOFFSET()
WHERE id = @id AND deleted_at IS NULL`
	_, err := r.q.ExecContext(ctx, q,
		sql.Named("id", id),
		sql.Named("name", d.Name),
		sql.Named("description", d.Description),
		sql.Named("letterTypeId", d.LetterTypeID),
		sql.Named("objectKey", objectKey),
	)
	if db.IsUniqueViolation(err) {
		return errTemplateNameTaken
	}
	if err != nil {
		return fmt.Errorf("mengubah template: %w", err)
	}
	return nil
}

// adminTemplateColumns adalah templateColumns ditambah status aktif dan jumlah surat yang memakainya.
const adminTemplateColumns = templateSelect + `, t.is_active,
       (SELECT COUNT(*) FROM letters l WHERE l.template_id = t.id AND l.deleted_at IS NULL)
FROM letter_templates t%s` + templateJoins

func scanAdminTemplate(scan func(...any) error) (*AdminTemplate, error) {
	var a AdminTemplate
	t, err := scanTemplate(scanFunc(func(dest ...any) error {
		return scan(append(dest, &a.IsActive, &a.UsedBy)...)
	}))
	if err != nil {
		return nil, err
	}
	a.Template = *t
	return &a, nil
}

// scanFunc menjadikan fungsi Scan sebagai penerima scanTemplate.
type scanFunc func(dest ...any) error

func (f scanFunc) Scan(dest ...any) error { return f(dest...) }

// ListAdminTemplates mengambil semua template milik companyCode (aktif dan nonaktif), urut nama.
func (r *Repository) ListAdminTemplates(ctx context.Context, companyCode, status, search string) ([]AdminTemplate, error) {
	q := fmt.Sprintf(adminTemplateColumns, "") + `
WHERE t.company_code = @companyCode AND t.deleted_at IS NULL
  AND (@status = '' OR (@status = 'ACTIVE' AND t.is_active = 1) OR (@status = 'INACTIVE' AND t.is_active = 0))`
	args := []any{sql.Named("companyCode", companyCode), sql.Named("status", status)}
	if search != "" {
		q += `
  AND (t.name LIKE @pattern ESCAPE '\' OR t.description LIKE @pattern ESCAPE '\')`
		args = append(args, sql.Named("pattern", db.ContainsPattern(search)))
	}
	q += "\nORDER BY t.name"
	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil template: %w", err)
	}
	defer rows.Close()
	out := []AdminTemplate{}
	for rows.Next() {
		a, err := scanAdminTemplate(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// FindAdminTemplate mengambil satu template milik companyCode, aktif maupun nonaktif.
// lock=true mengunci barisnya sampai transaksi selesai.
func (r *Repository) FindAdminTemplate(ctx context.Context, companyCode string, id int64, lock bool) (*AdminTemplate, error) {
	hint := ""
	if lock {
		hint = " WITH (UPDLOCK, ROWLOCK)"
	}
	q := fmt.Sprintf(adminTemplateColumns, hint) + `
WHERE t.id = @id AND t.company_code = @companyCode AND t.deleted_at IS NULL`
	a, err := scanAdminTemplate(r.q.QueryRowContext(ctx, q, sql.Named("id", id), sql.Named("companyCode", companyCode)).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	return a, err
}

// LockAdminTemplate mengunci satu template (aktif maupun nonaktif) di dalam transaksi.
func (r *Repository) LockAdminTemplate(ctx context.Context, companyCode string, id int64) (*AdminTemplate, error) {
	return r.FindAdminTemplate(ctx, companyCode, id, true)
}

// SetTemplateActive mengaktifkan/menonaktifkan template.
func (r *Repository) SetTemplateActive(ctx context.Context, id int64, active bool) error {
	const q = `UPDATE letter_templates SET is_active = @active, updated_at = SYSDATETIMEOFFSET() WHERE id = @id`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", id), sql.Named("active", active)); err != nil {
		return fmt.Errorf("mengubah status template: %w", err)
	}
	return nil
}

// DeleteTemplate menghapus (soft-delete) template surat sehingga namanya dapat dipakai kembali.
func (r *Repository) DeleteTemplate(ctx context.Context, id int64) error {
	const q = `UPDATE letter_templates SET is_active = 0, deleted_at = SYSDATETIMEOFFSET(), updated_at = SYSDATETIMEOFFSET() WHERE id = @id AND deleted_at IS NULL`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", id)); err != nil {
		return fmt.Errorf("menghapus template: %w", err)
	}
	return nil
}

func (r *Repository) findTemplate(ctx context.Context, q string, args ...any) (*Template, error) {
	t, err := scanTemplate(r.q.QueryRowContext(ctx, q, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	return t, err
}

func scanTemplate(row interface{ Scan(...any) error }) (*Template, error) {
	var (
		t          Template
		letterType nullRef
	)
	err := row.Scan(&t.ID, &t.Name, &t.Description, &letterType.ID, &letterType.Code, &letterType.Name,
		&t.CreatedBy.ID, &t.CreatedBy.Name, &t.CreatedAt, &t.UpdatedAt, &t.ObjectKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("membaca template: %w", err)
	}
	t.LetterType = letterType.lookup()
	t.FileURL = "/api/v1/templates/" + strconv.FormatInt(t.ID, 10) + "/file"
	// SQL Server menulis uuid huruf besar; nama objek di MinIO memakai huruf kecil
	t.ObjectKey = strings.ToLower(t.ObjectKey)
	return &t, nil
}
