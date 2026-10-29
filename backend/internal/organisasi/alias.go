package organisasi

import (
	"context"
	"database/sql"
	"fmt"

	"office.backend/internal/platform/db"
)

// AliasMemberCountSQL menghitung anggota alias level jabatan lv (tabel lookups beralias lv):
// user aktif yang jabatannya pimpinan unit dengan level lv, di unit aktif perusahaan yang sama.
// Diekspor supaya query surat (detail penerima) memakai aturan yang persis sama.
const AliasMemberCountSQL = `(
    SELECT COUNT(*)
    FROM users au
    JOIN positions ap ON ap.id = au.position_id AND ap.is_unit_head = 1
                     AND ap.is_active = 1 AND ap.deleted_at IS NULL
    JOIN units aun    ON aun.id = ap.unit_id AND aun.is_active = 1 AND aun.deleted_at IS NULL
    WHERE ap.level_id = lv.id AND aun.company_code = lv.company_code
      AND au.is_active = 1 AND au.deleted_at IS NULL)`

// AliasMemberIDs mengembalikan anggota setiap alias di aliasIDs (level jabatan -> user), dengan
// aturan yang sama dengan AliasMemberCountSQL.
func (r *Repository) AliasMemberIDs(ctx context.Context, companyCode string, aliasIDs []int64) (map[int64][]int64, error) {
	out := map[int64][]int64{}
	if len(aliasIDs) == 0 {
		return out, nil
	}
	const q = `
SELECT lv.id, au.id
FROM lookups lv
JOIN positions ap ON ap.level_id = lv.id AND ap.is_unit_head = 1
                 AND ap.is_active = 1 AND ap.deleted_at IS NULL
JOIN units aun    ON aun.id = ap.unit_id AND aun.is_active = 1 AND aun.deleted_at IS NULL
                 AND aun.company_code = lv.company_code
JOIN users au     ON au.position_id = ap.id AND au.is_active = 1 AND au.deleted_at IS NULL
WHERE lv.id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids)) AND lv.company_code = @companyCode
ORDER BY lv.id, au.id`
	pairs, err := r.idPairs(ctx, "mengambil anggota alias penerima", q,
		sql.Named("ids", db.JSONIDs(aliasIDs)), sql.Named("companyCode", companyCode))
	if err != nil {
		return nil, err
	}
	for _, p := range pairs {
		out[p[0]] = append(out[p[0]], p[1])
	}
	return out, nil
}

// ListAliases mengambil alias (level jabatan perusahaan) yang punya minimal 1 anggota.
func (r *Repository) ListAliases(ctx context.Context, companyCode string) ([]Alias, error) {
	q := `
SELECT id, code, name, member_count
FROM (
    SELECT lv.id, lv.code, lv.name, lv.sort_order, ` + AliasMemberCountSQL + ` AS member_count
    FROM lookups lv
    WHERE lv.category = 'POSITION_LEVEL' AND lv.company_code = @companyCode
      AND lv.is_active = 1 AND lv.deleted_at IS NULL
) a
WHERE member_count > 0
ORDER BY sort_order, name`
	rows, err := r.q.QueryContext(ctx, q, sql.Named("companyCode", companyCode))
	if err != nil {
		return nil, fmt.Errorf("mengambil alias: %w", err)
	}
	defer rows.Close()

	out := []Alias{}
	for rows.Next() {
		var (
			a    Alias
			name string
		)
		if err := rows.Scan(&a.LevelID, &a.Code, &name, &a.MemberCount); err != nil {
			return nil, fmt.Errorf("membaca alias: %w", err)
		}
		a.Label = AliasLabel(name)
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca alias: %w", err)
	}
	return out, nil
}

// AliasLabel membentuk label alias dari nama level jabatan, contoh "VP" -> "Seluruh VP".
func AliasLabel(levelName string) string { return "Seluruh " + levelName }
