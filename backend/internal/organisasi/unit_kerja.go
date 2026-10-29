package organisasi

import (
	"context"
	"database/sql"
	"fmt"

	"office.backend/internal/platform/db"
)

// ListUnits mengambil unit aktif milik satu perusahaan beserta nama jabatan pimpinannya.
// search "" = tanpa pencarian; selain itu dicari di nama unit.
func (r *Repository) ListUnits(ctx context.Context, companyCode, search string) ([]Unit, error) {
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
		args = append(args, sql.Named("pattern", db.ContainsPattern(search)))
	}
	q += "\nORDER BY un.name"

	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil unit: %w", err)
	}
	defer rows.Close()

	out := []Unit{}
	for rows.Next() {
		var u Unit
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

// ActiveUnitIDs mengembalikan unit aktif milik companyCode di antara ids.
func (r *Repository) ActiveUnitIDs(ctx context.Context, companyCode string, ids []int64) (map[int64]bool, error) {
	if len(ids) == 0 {
		return map[int64]bool{}, nil
	}
	const q = `
SELECT id FROM units
WHERE id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids)) AND company_code = @companyCode
  AND is_active = 1 AND deleted_at IS NULL`
	return r.idSet(ctx, "memeriksa unit", q, sql.Named("ids", db.JSONIDs(ids)), sql.Named("companyCode", companyCode))
}

// UnitHeadIDs mengembalikan satu pimpinan aktif untuk setiap unit di unitIDs (unit -> user).
// Bila satu unit punya lebih dari satu pimpinan, dipilih jabatan lalu user dengan id terkecil.
// Unit tanpa pimpinan aktif tidak ada di hasil.
func (r *Repository) UnitHeadIDs(ctx context.Context, companyCode string, unitIDs []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	if len(unitIDs) == 0 {
		return out, nil
	}
	const q = `
SELECT un.id, head.user_id
FROM units un
CROSS APPLY (
    SELECT TOP 1 u.id AS user_id
    FROM positions p
    JOIN users u ON u.position_id = p.id AND u.is_active = 1 AND u.deleted_at IS NULL
    WHERE p.unit_id = un.id AND p.is_unit_head = 1 AND p.is_active = 1 AND p.deleted_at IS NULL
    ORDER BY p.id, u.id
) head
WHERE un.id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids)) AND un.company_code = @companyCode`
	pairs, err := r.idPairs(ctx, "mengambil pimpinan unit penerima", q,
		sql.Named("ids", db.JSONIDs(unitIDs)), sql.Named("companyCode", companyCode))
	if err != nil {
		return nil, err
	}
	for _, p := range pairs {
		out[p[0]] = p[1]
	}
	return out, nil
}

// UnreachableRecipients mengembalikan penerima yang saat ini tidak bisa menerima surat: nama unit
// yang tidak punya pimpinan aktif, dan label alias yang tidak punya anggota.
func (r *Repository) UnreachableRecipients(ctx context.Context, companyCode string, unitIDs, aliasIDs []int64) (units, aliases []string, err error) {
	if len(unitIDs) > 0 {
		const q = `
SELECT un.name
FROM units un
WHERE un.id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids)) AND un.company_code = @companyCode
  AND NOT EXISTS (
        SELECT 1
        FROM positions p
        JOIN users u ON u.position_id = p.id AND u.is_active = 1 AND u.deleted_at IS NULL
        WHERE p.unit_id = un.id AND p.is_unit_head = 1 AND p.is_active = 1 AND p.deleted_at IS NULL)
ORDER BY un.name`
		if units, err = r.names(ctx, q, unitIDs, companyCode); err != nil {
			return nil, nil, fmt.Errorf("memeriksa pimpinan unit penerima: %w", err)
		}
	}
	if len(aliasIDs) > 0 {
		const q = `
SELECT lv.name
FROM lookups lv
WHERE lv.id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids)) AND lv.company_code = @companyCode
  AND ` + AliasMemberCountSQL + ` = 0
ORDER BY lv.sort_order`
		names, err := r.names(ctx, q, aliasIDs, companyCode)
		if err != nil {
			return nil, nil, fmt.Errorf("memeriksa anggota alias penerima: %w", err)
		}
		for _, n := range names {
			aliases = append(aliases, AliasLabel(n))
		}
	}
	return units, aliases, nil
}
