package masterdata

import (
	"context"
	"database/sql"
	"errors"

	"office.backend/internal/platform/db"
)

// AdminLookup adalah satu baris lookup untuk halaman master data.
type AdminLookup struct {
	ID        int64   `json:"id,string"`
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Note      *string `json:"note"`
	SortOrder int     `json:"sortOrder"`
	IsActive  bool    `json:"isActive"`
	ReadOnly  bool    `json:"readOnly"`
}

// LookupInput adalah isi form tambah/ubah lookup perusahaan.
type LookupInput struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Note      string `json:"note"`
	SortOrder int    `json:"sortOrder"`
}

// AdminRepository mengelola query tabel lookups untuk halaman master data.
type AdminRepository struct {
	q db.DBTX
}

const adminColumns = `id, code, name, note, sort_order, is_active, CAST(CASE WHEN company_code IS NULL THEN 1 ELSE 0 END AS BIT)`

// List mengambil daftar lookup global dan milik perusahaan tertentu (aktif maupun nonaktif).
func (r AdminRepository) List(ctx context.Context, company, category, status, search string) ([]AdminLookup, error) {
	const q = `
SELECT ` + adminColumns + `
FROM lookups
WHERE deleted_at IS NULL
  AND category = @category
  AND (company_code IS NULL OR company_code = @company)
  AND (@status = '' OR is_active = CASE WHEN @status = 'ACTIVE' THEN 1 ELSE 0 END)
  AND (@search = '' OR CHARINDEX(@search, code) > 0 OR CHARINDEX(@search, name) > 0)
ORDER BY sort_order, name, id`

	rows, err := r.q.QueryContext(ctx, q,
		sql.Named("company", company),
		sql.Named("category", category),
		sql.Named("status", status),
		sql.Named("search", search),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AdminLookup{}
	for rows.Next() {
		var v AdminLookup
		if err := rows.Scan(&v.ID, &v.Code, &v.Name, &v.Note, &v.SortOrder, &v.IsActive, &v.ReadOnly); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Find mengambil satu lookup berdasarkan ID dan kategori.
func (r AdminRepository) Find(ctx context.Context, company, category string, id int64, lock bool) (*AdminLookup, error) {
	hint := ""
	if lock {
		hint = " WITH (UPDLOCK, HOLDLOCK)"
	}
	q := `
SELECT ` + adminColumns + `
FROM lookups` + hint + `
WHERE id = @id
  AND category = @category
  AND deleted_at IS NULL
  AND (company_code IS NULL OR company_code = @company)`

	var v AdminLookup
	err := r.q.QueryRowContext(ctx, q,
		sql.Named("id", id),
		sql.Named("company", company),
		sql.Named("category", category),
	).Scan(&v.ID, &v.Code, &v.Name, &v.Note, &v.SortOrder, &v.IsActive, &v.ReadOnly)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errLookupNotFound
	}
	return &v, err
}

// Insert menambahkan lookup baru khusus perusahaan.
func (r AdminRepository) Insert(ctx context.Context, company, category string, in LookupInput) (int64, error) {
	const checkSQL = `
SELECT COUNT(*)
FROM lookups WITH (UPDLOCK, HOLDLOCK)
WHERE category = @category
  AND code = @code
  AND deleted_at IS NULL
  AND (company_code IS NULL OR company_code = @company)`

	var count int
	err := r.q.QueryRowContext(ctx, checkSQL,
		sql.Named("company", company),
		sql.Named("category", category),
		sql.Named("code", in.Code),
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return 0, errLookupCodeTaken
	}

	const insertSQL = `
INSERT INTO lookups (company_code, category, code, name, note, sort_order)
OUTPUT INSERTED.id
VALUES (@company, @category, @code, @name, NULLIF(@note, ''), @sort)`

	var id int64
	err = r.q.QueryRowContext(ctx, insertSQL,
		sql.Named("company", company),
		sql.Named("category", category),
		sql.Named("code", in.Code),
		sql.Named("name", in.Name),
		sql.Named("note", in.Note),
		sql.Named("sort", in.SortOrder),
	).Scan(&id)
	return id, err
}

// Update memperbarui nama, keterangan, dan urutan tampilan lookup perusahaan.
func (r AdminRepository) Update(ctx context.Context, company string, id int64, in LookupInput) error {
	const q = `
UPDATE lookups
SET name = @name,
    note = NULLIF(@note, ''),
    sort_order = @sort,
    updated_at = SYSDATETIMEOFFSET()
WHERE id = @id AND company_code = @company AND deleted_at IS NULL`

	_, err := r.q.ExecContext(ctx, q,
		sql.Named("company", company),
		sql.Named("id", id),
		sql.Named("name", in.Name),
		sql.Named("note", in.Note),
		sql.Named("sort", in.SortOrder),
	)
	return err
}

// Status mengubah status aktif/nonaktif lookup perusahaan.
func (r AdminRepository) Status(ctx context.Context, company string, id int64, active bool) error {
	const q = `
UPDATE lookups
SET is_active = @active,
    updated_at = SYSDATETIMEOFFSET()
WHERE id = @id AND company_code = @company AND deleted_at IS NULL`

	_, err := r.q.ExecContext(ctx, q,
		sql.Named("company", company),
		sql.Named("id", id),
		sql.Named("active", active),
	)
	return err
}

// Delete menghapus (soft-delete) lookup khusus perusahaan sehingga kode dapat dipakai kembali.
func (r AdminRepository) Delete(ctx context.Context, company string, id int64) error {
	const q = `
UPDATE lookups
SET is_active = 0,
    deleted_at = SYSDATETIMEOFFSET(),
    updated_at = SYSDATETIMEOFFSET()
WHERE id = @id AND company_code = @company AND deleted_at IS NULL`

	_, err := r.q.ExecContext(ctx, q,
		sql.Named("company", company),
		sql.Named("id", id),
	)
	return err
}
