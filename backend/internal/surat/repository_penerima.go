package surat

import (
	"context"
	"database/sql"
	"fmt"
)

// Penerima disimpan dengan diff seperti peserta (lihat repository_peserta.go).

// recipient adalah satu penerima. Tepat satu dari UnitID/AliasID bukan 0. ID 0 = belum tersimpan.
type recipient struct {
	ID      int64  `json:"id"`
	Type    string `json:"type"`
	UnitID  int64  `json:"unitId"`
	AliasID int64  `json:"aliasId"`
	Sort    int    `json:"sort"`
}

func (p recipient) key() string {
	if p.UnitID != 0 {
		return fmt.Sprintf("U%d", p.UnitID)
	}
	return fmt.Sprintf("A%d", p.AliasID)
}

// planRecipients membandingkan penerima tersimpan dengan susunan baru berdasarkan unit/alias.
// Penerima yang tetap ada tetapi jenis (TO/CC) atau urutannya berubah cukup di-update.
func planRecipients(current, want []recipient) (deleteIDs []int64, update, insert []recipient) {
	byKey := map[string]recipient{}
	for _, c := range current {
		byKey[c.key()] = c
	}
	for _, w := range want {
		c, ok := byKey[w.key()]
		if !ok {
			insert = append(insert, w)
			continue
		}
		delete(byKey, w.key())
		if c.Type != w.Type || c.Sort != w.Sort {
			w.ID = c.ID
			update = append(update, w)
		}
	}
	for _, c := range current {
		if _, gone := byKey[c.key()]; gone {
			deleteIDs = append(deleteIDs, c.ID)
		}
	}
	return deleteIDs, update, insert
}

// Query penerima. Banyak baris dikirim sebagai satu parameter JSON (OPENJSON).
const (
	selectRecipientsSQL = `
SELECT id, recipient_type, ISNULL(recipient_unit_id, 0), ISNULL(alias_level_id, 0), sort_order
FROM letter_recipients
WHERE letter_id = @letterId AND deleted_at IS NULL
ORDER BY sort_order, id`

	insertRecipientsSQL = `
INSERT INTO letter_recipients (letter_id, recipient_type, recipient_unit_id, alias_level_id, sort_order)
SELECT @letterId, j.type, NULLIF(j.unitId, 0), NULLIF(j.aliasId, 0), j.sort
FROM OPENJSON(@rows) WITH (type VARCHAR(10), unitId BIGINT, aliasId BIGINT, sort INT) j`

	updateRecipientsSQL = `
UPDATE lr SET recipient_type = j.type, sort_order = j.sort, updated_at = SYSDATETIMEOFFSET()
FROM letter_recipients lr
JOIN OPENJSON(@rows) WITH (id BIGINT, type VARCHAR(10), sort INT) j ON j.id = lr.id
WHERE lr.letter_id = @letterId AND lr.deleted_at IS NULL`

	deleteRecipientsSQL = `
UPDATE letter_recipients SET deleted_at = SYSDATETIMEOFFSET(), updated_at = SYSDATETIMEOFFSET()
WHERE letter_id = @letterId AND deleted_at IS NULL
  AND id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids))`
)

// loadRecipients membaca penerima aktif sesuai urutan simpan.
func (r *Repository) loadRecipients(ctx context.Context, letterID int64) ([]recipient, error) {
	rows, err := r.q.QueryContext(ctx, selectRecipientsSQL, sql.Named("letterId", letterID))
	if err != nil {
		return nil, fmt.Errorf("mengambil penerima surat: %w", err)
	}
	defer rows.Close()
	var out []recipient
	for rows.Next() {
		var p recipient
		if err := rows.Scan(&p.ID, &p.Type, &p.UnitID, &p.AliasID, &p.Sort); err != nil {
			return nil, fmt.Errorf("membaca penerima surat: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca penerima surat: %w", err)
	}
	return out, nil
}

func (r *Repository) saveRecipients(ctx context.Context, letterID int64, refs []RecipientRef) error {
	current, err := r.loadRecipients(ctx, letterID)
	if err != nil {
		return err
	}

	want := make([]recipient, len(refs))
	for i, ref := range refs {
		want[i] = recipient{Type: ref.Type, UnitID: ref.UnitID, AliasID: ref.AliasLevelID, Sort: i}
	}
	deleteIDs, update, insert := planRecipients(current, want)
	if len(deleteIDs) > 0 {
		if err := r.execRows(ctx, deleteRecipientsSQL, letterID, "ids", deleteIDs); err != nil {
			return fmt.Errorf("menghapus penerima surat: %w", err)
		}
	}
	if len(update) > 0 {
		if err := r.execRows(ctx, updateRecipientsSQL, letterID, "rows", update); err != nil {
			return fmt.Errorf("mengubah penerima surat: %w", err)
		}
	}
	if len(insert) > 0 {
		if err := r.execRows(ctx, insertRecipientsSQL, letterID, "rows", insert); err != nil {
			return fmt.Errorf("menyimpan penerima surat: %w", err)
		}
	}
	return nil
}
