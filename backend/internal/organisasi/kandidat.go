package organisasi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"office.backend/internal/auth"
	"office.backend/internal/platform/db"
)

// CandidateFilter adalah kriteria pencarian kandidat reviewer/approver
type CandidateFilter struct {
	CompanyCode    string
	PermissionCode string // "" = semua user aktif (calon pengganti delegasi)
	Search         string // "" = tidak mencari; selain itu dicari di nama user, jabatan, dan unit
	Limit          int
}

// activeUserSQL adalah syarat user aktif di perusahaan @companyCode: user, jabatan, unit, dan role
// semuanya aktif (user seperti ini bisa login dan menerima tugas surat).
const activeUserSQL = `
FROM users u
JOIN positions p ON p.id = u.position_id AND p.is_active = 1 AND p.deleted_at IS NULL
JOIN units un    ON un.id = p.unit_id   AND un.is_active = 1 AND un.deleted_at IS NULL
JOIN roles r     ON r.id = u.role_id    AND r.is_active = 1 AND r.deleted_at IS NULL
WHERE un.company_code = @companyCode
  AND u.is_active = 1 AND u.deleted_at IS NULL`

// eligibleUserSQL adalah syarat user yang boleh dipilih sebagai reviewer/approver: user aktif yang
// punya permission @permissionCode, dari role-nya atau dari posisinya (sama dengan hak saat login,
// lihat auth/hakakses.go). Dipakai pencarian kandidat, validasi simpan surat, dan alur persetujuan,
// sehingga yang muncul di dropdown pasti lolos validasi dan sebaliknya. Parameternya dari
// eligibleArgs.
const eligibleUserSQL = activeUserSQL + `
  AND (
        EXISTS (
            SELECT 1 FROM permissions pm
            WHERE pm.role_id = u.role_id AND pm.code = @permissionCode AND pm.deleted_at IS NULL
        )
        OR (@viaPosition = 1 AND EXISTS (
            SELECT 1 FROM lookups lv
            WHERE lv.id = p.level_id AND lv.deleted_at IS NULL
              AND lv.code IN (SELECT [value] FROM OPENJSON(@officialLevels))
        ))
  )`

// eligibleArgs adalah parameter eligibleUserSQL untuk permission tertentu.
func eligibleArgs(permissionCode string) []any {
	levels, _ := json.Marshal(auth.OfficialLevels()) // []string selalu bisa di-encode
	return []any{
		sql.Named("permissionCode", permissionCode),
		sql.Named("viaPosition", auth.GrantedByPosition(permissionCode)),
		sql.Named("officialLevels", string(levels)),
	}
}

// ListCandidates mengambil user aktif di satu perusahaan yang punya permission tertentu (role atau posisi)
func (r *Repository) ListCandidates(ctx context.Context, f CandidateFilter) ([]Candidate, error) {
	q := `
SELECT TOP (@limit)
       u.id, u.name, u.employee_number,
       p.name  AS position_name,
       un.name AS unit_name`
	args := []any{sql.Named("limit", f.Limit), sql.Named("companyCode", f.CompanyCode)}
	if f.PermissionCode == "" {
		q += activeUserSQL
	} else {
		q += eligibleUserSQL
		args = append(args, eligibleArgs(f.PermissionCode)...)
	}
	if f.Search != "" {
		q += "\n  AND (u.name LIKE @pattern ESCAPE '\\' OR p.name LIKE @pattern ESCAPE '\\' OR un.name LIKE @pattern ESCAPE '\\')"
		args = append(args, sql.Named("pattern", db.ContainsPattern(f.Search)))
	}
	q += "\nORDER BY u.name"

	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mencari kandidat: %w", err)
	}
	defer rows.Close()

	out := []Candidate{}
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.ID, &c.Name, &c.EmployeeNumber, &c.PositionName, &c.UnitName); err != nil {
			return nil, fmt.Errorf("membaca kandidat: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca kandidat: %w", err)
	}
	return out, nil
}

// EligibleUserIDs mengembalikan user di antara ids yang memenuhi syarat kandidat (eligibleUserSQL)
// untuk permission tertentu.
func (r *Repository) EligibleUserIDs(ctx context.Context, companyCode, permissionCode string, ids []int64) (map[int64]bool, error) {
	if len(ids) == 0 {
		return map[int64]bool{}, nil
	}
	const q = "SELECT u.id" + eligibleUserSQL + `
  AND u.id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids))`
	args := append([]any{sql.Named("ids", db.JSONIDs(ids)), sql.Named("companyCode", companyCode)},
		eligibleArgs(permissionCode)...)
	return r.idSet(ctx, "memeriksa reviewer/approver", q, args...)
}

// ActiveUserIDs mengembalikan user aktif (activeUserSQL) di antara ids, tanpa syarat permission.
func (r *Repository) ActiveUserIDs(ctx context.Context, companyCode string, ids []int64) (map[int64]bool, error) {
	if len(ids) == 0 {
		return map[int64]bool{}, nil
	}
	const q = "SELECT u.id" + activeUserSQL + `
  AND u.id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids))`
	return r.idSet(ctx, "memeriksa user aktif", q, sql.Named("ids", db.JSONIDs(ids)), sql.Named("companyCode", companyCode))
}
