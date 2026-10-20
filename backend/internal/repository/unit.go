package repository

import (
	"context"
	"database/sql"
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
