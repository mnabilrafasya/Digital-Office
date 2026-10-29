package organisasi

import (
	"context"
	"database/sql"
	"fmt"

	"office.backend/internal/platform/db"
)

// Repository berisi query data organisasi: unit kerja, pegawai (kandidat reviewer/approver),
// dan alias penerima. Dipakai endpoint form surat dan validasi surat (lewat interface milik surat).
type Repository struct {
	q db.DBTX
}

// NewRepository membuat Repository di atas koneksi atau transaksi.
func NewRepository(q db.DBTX) *Repository {
	return &Repository{q: q}
}

func (r *Repository) idSet(ctx context.Context, what, q string, args ...any) (map[int64]bool, error) {
	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	defer rows.Close()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("%s: %w", what, err)
		}
		out[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	return out, nil
}

// idPairs menjalankan query dua kolom BIGINT (mis. unit -> user).
func (r *Repository) idPairs(ctx context.Context, what, q string, args ...any) ([][2]int64, error) {
	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	defer rows.Close()
	var out [][2]int64
	for rows.Next() {
		var p [2]int64
		if err := rows.Scan(&p[0], &p[1]); err != nil {
			return nil, fmt.Errorf("%s: %w", what, err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	return out, nil
}

func (r *Repository) names(ctx context.Context, q string, ids []int64, companyCode string) ([]string, error) {
	rows, err := r.q.QueryContext(ctx, q, sql.Named("ids", db.JSONIDs(ids)), sql.Named("companyCode", companyCode))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
