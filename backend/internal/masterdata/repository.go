package masterdata

import (
	"context"
	"database/sql"
	"fmt"

	"office.backend/internal/platform/db"
)

// Repository berisi query tabel lookups.
type Repository struct {
	q db.DBTX
}

// NewRepository membuat Repository di atas koneksi atau transaksi.
func NewRepository(q db.DBTX) *Repository {
	return &Repository{q: q}
}

// List mengambil lookup aktif satu kategori: milik semua perusahaan (company_code NULL)
// ditambah milik perusahaan tertentu
func (r *Repository) List(ctx context.Context, companyCode, category string) ([]Lookup, error) {
	const q = `
SELECT id, code, name, color, note, sort_order
FROM lookups
WHERE category = @category
  AND is_active = 1 AND deleted_at IS NULL
  AND (company_code IS NULL OR company_code = @companyCode)
ORDER BY sort_order, name`

	rows, err := r.q.QueryContext(ctx, q,
		sql.Named("category", category),
		sql.Named("companyCode", companyCode),
	)
	if err != nil {
		return nil, fmt.Errorf("mengambil lookup: %w", err)
	}
	defer rows.Close()

	out := []Lookup{}
	for rows.Next() {
		var l Lookup
		if err := rows.Scan(&l.ID, &l.Code, &l.Name, &l.Color, &l.Note, &l.SortOrder); err != nil {
			return nil, fmt.Errorf("membaca lookup: %w", err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca lookup: %w", err)
	}
	return out, nil
}

// LookupCategories mengembalikan kategori setiap lookup aktif (global atau milik companyCode)
// di antara ids. ID yang tidak aktif/tidak ada tidak muncul di hasil.
func (r *Repository) LookupCategories(ctx context.Context, companyCode string, ids []int64) (map[int64]string, error) {
	out := map[int64]string{}
	if len(ids) == 0 {
		return out, nil
	}
	const q = `
SELECT id, category
FROM lookups
WHERE id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids))
  AND is_active = 1 AND deleted_at IS NULL
  AND (company_code IS NULL OR company_code = @companyCode)`
	rows, err := r.q.QueryContext(ctx, q, sql.Named("ids", db.JSONIDs(ids)), sql.Named("companyCode", companyCode))
	if err != nil {
		return nil, fmt.Errorf("memeriksa lookup: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id       int64
			category string
		)
		if err := rows.Scan(&id, &category); err != nil {
			return nil, fmt.Errorf("membaca lookup: %w", err)
		}
		out[id] = category
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca lookup: %w", err)
	}
	return out, nil
}
