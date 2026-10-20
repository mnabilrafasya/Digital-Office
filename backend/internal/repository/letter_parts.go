package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// Peserta dan penerima disimpan dengan diff: baris yang tidak berubah dibiarkan, baris yang
// hilang di-soft-delete, baris baru di-insert. Autosave yang sering tidak menumpuk baris sampah.

// participant adalah satu reviewer/approver. ID 0 berarti belum tersimpan.
type participant struct {
	ID     int64  `json:"-"`
	UserID int64  `json:"userId"`
	Role   string `json:"role"`
	Step   int    `json:"step"`
}

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

// wantParticipants menyusun peserta dari draf: reviewer berurutan mulai step 1, approver step 1.
func wantParticipants(d *model.LetterDraft) []participant {
	out := make([]participant, 0, len(d.ReviewerIDs)+1)
	for i, id := range d.ReviewerIDs {
		out = append(out, participant{UserID: id, Role: model.RoleReviewer, Step: i + 1})
	}
	if d.ApproverID != nil {
		out = append(out, participant{UserID: *d.ApproverID, Role: model.RoleApprover, Step: 1})
	}
	return out
}

// planParticipants membandingkan peserta tersimpan dengan susunan baru. Baris yang user, peran,
// dan step-nya sama dibiarkan; sisanya dihapus lalu diganti. Karena penghapusan dijalankan
// sebelum insert, perubahan urutan reviewer aman terhadap unique index (letter_id, role, step_order).
func planParticipants(current, want []participant) (deleteIDs []int64, insert []participant) {
	type key struct {
		userID int64
		role   string
		step   int
	}
	kept := map[key]bool{}
	wanted := map[key]bool{}
	for _, w := range want {
		wanted[key{w.UserID, w.Role, w.Step}] = true
	}
	for _, c := range current {
		k := key{c.UserID, c.Role, c.Step}
		if wanted[k] && !kept[k] {
			kept[k] = true
			continue
		}
		deleteIDs = append(deleteIDs, c.ID)
	}
	for _, w := range want {
		if !kept[key{w.UserID, w.Role, w.Step}] {
			insert = append(insert, w)
		}
	}
	return deleteIDs, insert
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

// Query simpan peserta dan penerima. Banyak baris dikirim sebagai satu parameter JSON (OPENJSON).
const (
	selectParticipantsSQL = `
SELECT id, user_id, role, step_order
FROM letter_participants
WHERE letter_id = @letterId AND deleted_at IS NULL
ORDER BY role DESC, step_order` // REVIEWER dulu, lalu APPROVER

	insertParticipantsSQL = `
INSERT INTO letter_participants (letter_id, user_id, role, step_order)
SELECT @letterId, j.userId, j.role, j.step
FROM OPENJSON(@rows) WITH (userId BIGINT, role VARCHAR(20), step INT) j`

	deleteParticipantsSQL = `
UPDATE letter_participants SET deleted_at = SYSDATETIMEOFFSET(), updated_at = SYSDATETIMEOFFSET()
WHERE letter_id = @letterId AND deleted_at IS NULL
  AND id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids))`

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

// loadParticipants membaca peserta aktif: reviewer urut step, lalu approver.
func (r *LetterRepository) loadParticipants(ctx context.Context, letterID int64) ([]participant, error) {
	rows, err := r.q.QueryContext(ctx, selectParticipantsSQL, sql.Named("letterId", letterID))
	if err != nil {
		return nil, fmt.Errorf("mengambil peserta surat: %w", err)
	}
	defer rows.Close()
	var out []participant
	for rows.Next() {
		var p participant
		if err := rows.Scan(&p.ID, &p.UserID, &p.Role, &p.Step); err != nil {
			return nil, fmt.Errorf("membaca peserta surat: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca peserta surat: %w", err)
	}
	return out, nil
}

// loadRecipients membaca penerima aktif sesuai urutan simpan.
func (r *LetterRepository) loadRecipients(ctx context.Context, letterID int64) ([]recipient, error) {
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

// draftParticipants mengubah baris peserta (urut dari loadParticipants) menjadi reviewer dan approver draf.
func draftParticipants(ps []participant) model.LetterDraft {
	var d model.LetterDraft
	for _, p := range ps {
		if p.Role == model.RoleApprover {
			d.ApproverID = &p.UserID
		} else {
			d.ReviewerIDs = append(d.ReviewerIDs, p.UserID)
		}
	}
	return d
}

func (r *LetterRepository) saveParticipants(ctx context.Context, letterID int64, d *model.LetterDraft) error {
	current, err := r.loadParticipants(ctx, letterID)
	if err != nil {
		return err
	}
	deleteIDs, insert := planParticipants(current, wantParticipants(d))
	if len(deleteIDs) > 0 {
		if err := r.execRows(ctx, deleteParticipantsSQL, letterID, "ids", deleteIDs); err != nil {
			return fmt.Errorf("menghapus peserta surat: %w", err)
		}
	}
	if len(insert) > 0 {
		if err := r.execRows(ctx, insertParticipantsSQL, letterID, "rows", insert); err != nil {
			return fmt.Errorf("menyimpan peserta surat: %w", err)
		}
	}
	return nil
}

func (r *LetterRepository) saveRecipients(ctx context.Context, letterID int64, refs []model.RecipientRef) error {
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

// execRows menjalankan query dengan @letterId dan satu parameter JSON berisi banyak baris/ID.
func (r *LetterRepository) execRows(ctx context.Context, q string, letterID int64, param string, rows any) error {
	b, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	_, err = r.q.ExecContext(ctx, q, sql.Named("letterId", letterID), sql.Named(param, string(b)))
	return err
}
