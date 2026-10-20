package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// UnitRepository berisi query tabel units.
type UnitRepository struct {
	q db.DBTX
}

// NewUnitRepository membuat UnitRepository di atas koneksi atau transaksi.
func NewUnitRepository(q db.DBTX) *UnitRepository {
	return &UnitRepository{q: q}
}

// List mengambil unit aktif milik satu perusahaan beserta nama jabatan pimpinannya.
// search "" = tanpa pencarian; selain itu dicari di nama unit.
func (r *UnitRepository) List(ctx context.Context, companyCode, search string) ([]model.Unit, error) {
	q := `
SELECT un.id, un.code, un.name, un.unit_type, un.parent_id, head.name
FROM units un
OUTER APPLY (
    SELECT TOP 1 p.name
    FROM positions p
    WHERE p.unit_id = un.id AND p.is_unit_head = 1
      AND p.is_active = 1 AND p.deleted_at IS NULL
    ORDER BY p.id
) head
WHERE un.company_code = @companyCode
  AND un.is_active = 1 AND un.deleted_at IS NULL`
	args := []any{sql.Named("companyCode", companyCode)}
	if search != "" {
		q += "\n  AND un.name LIKE @pattern ESCAPE '\\'"
		args = append(args, sql.Named("pattern", ContainsPattern(search)))
	}
	q += "\nORDER BY un.name"

	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil unit: %w", err)
	}
	defer rows.Close()

	out := []model.Unit{}
	for rows.Next() {
		var u model.Unit
		if err := rows.Scan(&u.ID, &u.Code, &u.Name, &u.UnitType, &u.ParentID, &u.HeadPositionName); err != nil {
			return nil, fmt.Errorf("membaca unit: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca unit: %w", err)
	}
	return out, nil
}

// FindRefs mengambil unit (aktif maupun tidak, belum terhapus) milik perusahaan berdasarkan ID.
func (r *UnitRepository) FindRefs(ctx context.Context, companyCode string, ids []int64) (map[int64]model.UnitRef, error) {
	q := `
SELECT id, code, name, is_active
FROM units
WHERE company_code = @companyCode AND deleted_at IS NULL
  AND id IN ` + idsIn("ids")

	rows, err := r.q.QueryContext(ctx, q, sql.Named("companyCode", companyCode), idsParam("ids", ids))
	if err != nil {
		return nil, fmt.Errorf("mengambil referensi unit: %w", err)
	}
	defer rows.Close()

	out := make(map[int64]model.UnitRef, len(ids))
	for rows.Next() {
		var u model.UnitRef
		if err := rows.Scan(&u.ID, &u.Code, &u.Name, &u.IsActive); err != nil {
			return nil, fmt.Errorf("membaca referensi unit: %w", err)
		}
		out[u.ID] = u
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca referensi unit: %w", err)
	}
	return out, nil
}

// FindTemplate mengambil template surat milik perusahaan (belum terhapus).
func (r *UnitRepository) FindTemplate(ctx context.Context, companyCode string, id int64) (*model.TemplateRef, error) {
	const q = `
SELECT id, name, is_active
FROM letter_templates
WHERE id = @id AND company_code = @companyCode AND deleted_at IS NULL`

	var t model.TemplateRef
	err := r.q.QueryRowContext(ctx, q, sql.Named("id", id), sql.Named("companyCode", companyCode)).
		Scan(&t.ID, &t.Name, &t.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("mengambil template: %w", err)
	}
	return &t, nil
}
