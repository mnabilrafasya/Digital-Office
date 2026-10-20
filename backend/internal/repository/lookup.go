package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// LookupRepository berisi query tabel lookups.
type LookupRepository struct {
	q db.DBTX
}

// NewLookupRepository membuat LookupRepository di atas koneksi atau transaksi.
func NewLookupRepository(q db.DBTX) *LookupRepository {
	return &LookupRepository{q: q}
}

// List mengambil lookup aktif satu kategori: milik semua perusahaan (company_code NULL)
// ditambah milik perusahaan tertentu
func (r *LookupRepository) List(ctx context.Context, companyCode, category string) ([]model.Lookup, error) {
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

	out := []model.Lookup{}
	for rows.Next() {
		var l model.Lookup
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
