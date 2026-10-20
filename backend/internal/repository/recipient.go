package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// RecipientRepository berisi query tabel letter_recipients dan penerjemahan penerima menjadi user.
type RecipientRepository struct {
	q db.DBTX
}

// NewRecipientRepository membuat RecipientRepository di atas koneksi atau transaksi.
func NewRecipientRepository(q db.DBTX) *RecipientRepository {
	return &RecipientRepository{q: q}
}

// List mengambil penerima aktif surat, urut TO lalu CC, lalu sort_order.
func (r *RecipientRepository) List(ctx context.Context, letterID int64) ([]model.Recipient, error) {
	const q = `
SELECT id, recipient_type, recipient_unit_id, alias_level_id, resolved_head_id, sort_order
FROM letter_recipients
WHERE letter_id = @letterId AND deleted_at IS NULL
ORDER BY CASE recipient_type WHEN 'TO' THEN 0 ELSE 1 END, sort_order, id`

	rows, err := r.q.QueryContext(ctx, q, sql.Named("letterId", letterID))
	if err != nil {
		return nil, fmt.Errorf("mengambil penerima surat: %w", err)
	}
	defer rows.Close()

	out := []model.Recipient{}
	for rows.Next() {
		var rc model.Recipient
		if err := rows.Scan(&rc.ID, &rc.Type, &rc.UnitID, &rc.AliasLevelID, &rc.ResolvedHeadID, &rc.SortOrder); err != nil {
			return nil, fmt.Errorf("membaca penerima surat: %w", err)
		}
		out = append(out, rc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca penerima surat: %w", err)
	}
	return out, nil
}

// SoftDelete menandai penerima dengan ID tertentu sebagai terhapus.
func (r *RecipientRepository) SoftDelete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	q := `
UPDATE letter_recipients SET deleted_at = SYSDATETIMEOFFSET(), updated_at = SYSDATETIMEOFFSET()
WHERE deleted_at IS NULL AND id IN ` + idsIn("ids")
	if _, err := r.q.ExecContext(ctx, q, idsParam("ids", ids)); err != nil {
		return fmt.Errorf("menghapus penerima surat: %w", err)
	}
	return nil
}

type recipientJSON struct {
	Type    string `json:"t"`
	UnitID  *int64 `json:"u"`
	AliasID *int64 `json:"a"`
	Sort    int    `json:"s"`
}

// Insert menyimpan beberapa penerima sekaligus.
func (r *RecipientRepository) Insert(ctx context.Context, letterID int64, items []model.NewRecipient) error {
	if len(items) == 0 {
		return nil
	}
	rows := make([]recipientJSON, len(items))
	for i, it := range items {
		rows[i] = recipientJSON{Type: it.Type, UnitID: it.UnitID, AliasID: it.AliasLevelID, Sort: it.SortOrder}
	}
	param, err := jsonParam("items", rows)
	if err != nil {
		return err
	}

	const q = `
INSERT INTO letter_recipients (letter_id, recipient_type, recipient_unit_id, alias_level_id, sort_order)
SELECT @letterId, j.t, j.u, j.a, j.s
FROM OPENJSON(@items) WITH (t VARCHAR(10) '$.t', u BIGINT '$.u', a BIGINT '$.a', s INT '$.s') j`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("letterId", letterID), param); err != nil {
		return fmt.Errorf("menyimpan penerima surat: %w", err)
	}
	return nil
}

type resolvedHeadJSON struct {
	ID     int64 `json:"id"`
	HeadID int64 `json:"h"`
}

// SetResolvedHeads mengisi resolved_head_id penerima unit (map: id penerima -> id user pimpinan).
func (r *RecipientRepository) SetResolvedHeads(ctx context.Context, heads map[int64]int64) error {
	if len(heads) == 0 {
		return nil
	}
	rows := make([]resolvedHeadJSON, 0, len(heads))
	for id, head := range heads {
		rows = append(rows, resolvedHeadJSON{ID: id, HeadID: head})
	}
	param, err := jsonParam("items", rows)
	if err != nil {
		return err
	}

	const q = `
UPDATE lr SET resolved_head_id = j.h, updated_at = SYSDATETIMEOFFSET()
FROM letter_recipients lr
JOIN OPENJSON(@items) WITH (id BIGINT '$.id', h BIGINT '$.h') j ON j.id = lr.id`
	if _, err := r.q.ExecContext(ctx, q, param); err != nil {
		return fmt.Errorf("menyimpan pimpinan penerima: %w", err)
	}
	return nil
}

// ResolveUnitHeads mencari user aktif yang menjabat pimpinan aktif di setiap unit (map: unit -> user).
// Unit tanpa pimpinan aktif tidak ada di hasil.
func (r *RecipientRepository) ResolveUnitHeads(ctx context.Context, companyCode string, unitIDs []int64) (map[int64]int64, error) {
	const q = `
SELECT j.unit_id, h.user_id
FROM (SELECT DISTINCT CAST([value] AS BIGINT) AS unit_id FROM OPENJSON(@ids)) j
CROSS APPLY (
    SELECT TOP 1 u.id AS user_id
    FROM positions p
    JOIN units un ON un.id = p.unit_id AND un.company_code = @companyCode
                 AND un.is_active = 1 AND un.deleted_at IS NULL
    JOIN users u  ON u.position_id = p.id AND u.is_active = 1 AND u.deleted_at IS NULL
    WHERE p.unit_id = j.unit_id AND p.is_unit_head = 1
      AND p.is_active = 1 AND p.deleted_at IS NULL
    ORDER BY u.id
) h`
	return r.pairs(ctx, q, sql.Named("companyCode", companyCode), idsParam("ids", unitIDs))
}

// AliasMembers mencari anggota setiap alias: user aktif dengan jabatan pimpinan unit aktif pada
// level tersebut di perusahaan yang sama (map: level -> daftar user).
func (r *RecipientRepository) AliasMembers(ctx context.Context, companyCode string, levelIDs []int64) (map[int64][]int64, error) {
	q := `
SELECT p.level_id, u.id
FROM positions p
JOIN units un ON un.id = p.unit_id AND un.company_code = @companyCode
             AND un.is_active = 1 AND un.deleted_at IS NULL
JOIN users u  ON u.position_id = p.id AND u.is_active = 1 AND u.deleted_at IS NULL
WHERE p.is_unit_head = 1 AND p.is_active = 1 AND p.deleted_at IS NULL
  AND p.level_id IN ` + idsIn("ids") + `
ORDER BY p.level_id, u.id`

	rows, err := r.q.QueryContext(ctx, q, sql.Named("companyCode", companyCode), idsParam("ids", levelIDs))
	if err != nil {
		return nil, fmt.Errorf("mengambil anggota alias: %w", err)
	}
	defer rows.Close()

	out := map[int64][]int64{}
	for rows.Next() {
		var level, user int64
		if err := rows.Scan(&level, &user); err != nil {
			return nil, fmt.Errorf("membaca anggota alias: %w", err)
		}
		out[level] = append(out[level], user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca anggota alias: %w", err)
	}
	return out, nil
}

func (r *RecipientRepository) pairs(ctx context.Context, q string, args ...any) (map[int64]int64, error) {
	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mencari pimpinan unit: %w", err)
	}
	defer rows.Close()

	out := map[int64]int64{}
	for rows.Next() {
		var k, v int64
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("membaca pimpinan unit: %w", err)
		}
		out[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca pimpinan unit: %w", err)
	}
	return out, nil
}
