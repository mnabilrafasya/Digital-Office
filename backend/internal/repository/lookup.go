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
// ditambah milik perusahaan tertentu.
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

// FindRefs mengambil lookup (aktif maupun tidak, belum terhapus) berdasarkan ID,
// hanya yang global atau milik perusahaan tertentu.
func (r *LookupRepository) FindRefs(ctx context.Context, companyCode string, ids []int64) (map[int64]model.LookupRef, error) {
	q := `
SELECT id, category, code, name, is_active
FROM lookups
WHERE deleted_at IS NULL
  AND (company_code IS NULL OR company_code = @companyCode)
  AND id IN ` + idsIn("ids")

	rows, err := r.q.QueryContext(ctx, q, sql.Named("companyCode", companyCode), idsParam("ids", ids))
	if err != nil {
		return nil, fmt.Errorf("mengambil referensi lookup: %w", err)
	}
	defer rows.Close()

	out := make(map[int64]model.LookupRef, len(ids))
	for rows.Next() {
		var l model.LookupRef
		if err := rows.Scan(&l.ID, &l.Category, &l.Code, &l.Name, &l.IsActive); err != nil {
			return nil, fmt.Errorf("membaca referensi lookup: %w", err)
		}
		out[l.ID] = l
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca referensi lookup: %w", err)
	}
	return out, nil
}

// aliasSelect menghitung anggota alias: user aktif dengan jabatan pimpinan unit aktif
// pada level jabatan tersebut, di perusahaan yang sama.
const aliasSelect = `
SELECT lv.id, lv.code, lv.name, lv.is_active,
       (SELECT COUNT(*)
        FROM positions p
        JOIN units un ON un.id = p.unit_id AND un.is_active = 1 AND un.deleted_at IS NULL
        JOIN users u  ON u.position_id = p.id AND u.is_active = 1 AND u.deleted_at IS NULL
        WHERE p.level_id = lv.id AND p.is_unit_head = 1
          AND p.is_active = 1 AND p.deleted_at IS NULL
          AND un.company_code = @companyCode) AS member_count
FROM lookups lv
WHERE lv.category = 'POSITION_LEVEL'
  AND lv.company_code = @companyCode
  AND lv.deleted_at IS NULL`

// AliasRow adalah level jabatan beserta jumlah anggota aliasnya.
type AliasRow struct {
	model.Alias
	Name     string
	IsActive bool
}

// ListAliases mengambil semua level jabatan aktif milik perusahaan beserta jumlah anggotanya.
func (r *LookupRepository) ListAliases(ctx context.Context, companyCode string) ([]AliasRow, error) {
	return r.queryAliases(ctx, aliasSelect+"\n  AND lv.is_active = 1\nORDER BY lv.sort_order, lv.name",
		sql.Named("companyCode", companyCode))
}

// FindAliases mengambil level jabatan tertentu (aktif maupun tidak) milik perusahaan.
func (r *LookupRepository) FindAliases(ctx context.Context, companyCode string, ids []int64) (map[int64]AliasRow, error) {
	rows, err := r.queryAliases(ctx, aliasSelect+"\n  AND lv.id IN "+idsIn("ids"),
		sql.Named("companyCode", companyCode), idsParam("ids", ids))
	if err != nil {
		return nil, err
	}
	out := make(map[int64]AliasRow, len(rows))
	for _, a := range rows {
		out[a.LevelID] = a
	}
	return out, nil
}

func (r *LookupRepository) queryAliases(ctx context.Context, q string, args ...any) ([]AliasRow, error) {
	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil alias penerima: %w", err)
	}
	defer rows.Close()

	out := []AliasRow{}
	for rows.Next() {
		var a AliasRow
		if err := rows.Scan(&a.LevelID, &a.Code, &a.Name, &a.IsActive, &a.MemberCount); err != nil {
			return nil, fmt.Errorf("membaca alias penerima: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca alias penerima: %w", err)
	}
	return out, nil
}
