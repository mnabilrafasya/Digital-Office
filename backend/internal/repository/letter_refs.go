package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// aliasMemberCountSQL menghitung anggota alias level jabatan lv (tabel lookups beralias lv):
// user aktif yang jabatannya pimpinan unit dengan level lv, di unit aktif perusahaan yang sama.
const aliasMemberCountSQL = `(
    SELECT COUNT(*)
    FROM users au
    JOIN positions ap ON ap.id = au.position_id AND ap.is_unit_head = 1
                     AND ap.is_active = 1 AND ap.deleted_at IS NULL
    JOIN units aun    ON aun.id = ap.unit_id AND aun.is_active = 1 AND aun.deleted_at IS NULL
    WHERE ap.level_id = lv.id AND aun.company_code = lv.company_code
      AND au.is_active = 1 AND au.deleted_at IS NULL)`

// LetterRefRepository berisi query untuk memeriksa data yang dirujuk surat (lookup, unit,
// template, reviewer/approver) dan daftar alias penerima.
type LetterRefRepository struct {
	q db.DBTX
}

// NewLetterRefRepository membuat LetterRefRepository di atas koneksi atau transaksi.
func NewLetterRefRepository(q db.DBTX) *LetterRefRepository {
	return &LetterRefRepository{q: q}
}

// LookupCategories mengembalikan kategori setiap lookup aktif (global atau milik companyCode)
// di antara ids. ID yang tidak aktif/tidak ada tidak muncul di hasil.
func (r *LetterRefRepository) LookupCategories(ctx context.Context, companyCode string, ids []int64) (map[int64]string, error) {
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
	rows, err := r.q.QueryContext(ctx, q, sql.Named("ids", jsonIDs(ids)), sql.Named("companyCode", companyCode))
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

// ActiveUnitIDs mengembalikan unit aktif milik companyCode di antara ids.
func (r *LetterRefRepository) ActiveUnitIDs(ctx context.Context, companyCode string, ids []int64) (map[int64]bool, error) {
	if len(ids) == 0 {
		return map[int64]bool{}, nil
	}
	const q = `
SELECT id FROM units
WHERE id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids)) AND company_code = @companyCode
  AND is_active = 1 AND deleted_at IS NULL`
	return r.idSet(ctx, "memeriksa unit", q, sql.Named("ids", jsonIDs(ids)), sql.Named("companyCode", companyCode))
}

// EligibleUserIDs mengembalikan user di antara ids yang memenuhi syarat kandidat (eligibleUserSQL)
// untuk permission tertentu.
func (r *LetterRefRepository) EligibleUserIDs(ctx context.Context, companyCode, permissionCode string, ids []int64) (map[int64]bool, error) {
	if len(ids) == 0 {
		return map[int64]bool{}, nil
	}
	const q = "SELECT u.id" + eligibleUserSQL + `
  AND u.id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids))`
	return r.idSet(ctx, "memeriksa reviewer/approver", q,
		sql.Named("ids", jsonIDs(ids)),
		sql.Named("companyCode", companyCode),
		sql.Named("permissionCode", permissionCode),
	)
}

// ActiveUserIDs mengembalikan user aktif (activeUserSQL) di antara ids, tanpa syarat permission.
func (r *LetterRefRepository) ActiveUserIDs(ctx context.Context, companyCode string, ids []int64) (map[int64]bool, error) {
	if len(ids) == 0 {
		return map[int64]bool{}, nil
	}
	const q = "SELECT u.id" + activeUserSQL + `
  AND u.id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids))`
	return r.idSet(ctx, "memeriksa user aktif", q, sql.Named("ids", jsonIDs(ids)), sql.Named("companyCode", companyCode))
}

// UnreachableRecipients mengembalikan penerima yang saat ini tidak bisa menerima surat: nama unit
// yang tidak punya pimpinan aktif, dan label alias yang tidak punya anggota.
func (r *LetterRefRepository) UnreachableRecipients(ctx context.Context, companyCode string, unitIDs, aliasIDs []int64) (units, aliases []string, err error) {
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
  AND ` + aliasMemberCountSQL + ` = 0
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

// TemplateLetterType mengembalikan jenis surat template aktif milik companyCode
// (nil = template berlaku untuk semua jenis). ErrNotFound bila template tidak ada/tidak aktif.
func (r *LetterRefRepository) TemplateLetterType(ctx context.Context, companyCode string, templateID int64) (*int64, error) {
	const q = `
SELECT letter_type_id FROM letter_templates
WHERE id = @id AND company_code = @companyCode AND is_active = 1 AND deleted_at IS NULL`
	var typeID *int64
	err := r.q.QueryRowContext(ctx, q, sql.Named("id", templateID), sql.Named("companyCode", companyCode)).Scan(&typeID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("memeriksa template: %w", err)
	}
	return typeID, nil
}

// ListAliases mengambil alias (level jabatan perusahaan) yang punya minimal 1 anggota.
func (r *LetterRefRepository) ListAliases(ctx context.Context, companyCode string) ([]model.Alias, error) {
	q := `
SELECT id, code, name, member_count
FROM (
    SELECT lv.id, lv.code, lv.name, lv.sort_order, ` + aliasMemberCountSQL + ` AS member_count
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

	out := []model.Alias{}
	for rows.Next() {
		var (
			a    model.Alias
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

func (r *LetterRefRepository) idSet(ctx context.Context, what, q string, args ...any) (map[int64]bool, error) {
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

func (r *LetterRefRepository) names(ctx context.Context, q string, ids []int64, companyCode string) ([]string, error) {
	rows, err := r.q.QueryContext(ctx, q, sql.Named("ids", jsonIDs(ids)), sql.Named("companyCode", companyCode))
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
