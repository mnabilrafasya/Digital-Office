package surat

import (
	"context"
	"database/sql"
	"fmt"
)

// Peserta (dan penerima, lihat repository_penerima.go) disimpan dengan diff: baris yang tidak berubah dibiarkan, baris yang
// hilang di-soft-delete, baris baru di-insert. Autosave yang sering tidak menumpuk baris sampah.

// participant adalah satu reviewer/approver. ID 0 berarti belum tersimpan.
type participant struct {
	ID     int64  `json:"-"`
	UserID int64  `json:"userId"`
	Role   string `json:"role"`
	Step   int    `json:"step"`
}

// wantParticipants menyusun peserta dari draf: reviewer berurutan mulai step 1, approver step 1.
func wantParticipants(d *LetterDraft) []participant {
	out := make([]participant, 0, len(d.ReviewerIDs)+1)
	for i, id := range d.ReviewerIDs {
		out = append(out, participant{UserID: id, Role: RoleReviewer, Step: i + 1})
	}
	if d.ApproverID != nil {
		out = append(out, participant{UserID: *d.ApproverID, Role: RoleApprover, Step: 1})
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

// Query peserta. Banyak baris dikirim sebagai satu parameter JSON (OPENJSON).
const (
	selectParticipantsSQL = `
SELECT id, user_id, role, step_order
FROM letter_participants
WHERE letter_id = @letterId AND deleted_at IS NULL
ORDER BY role DESC, step_order`

	insertParticipantsSQL = `
INSERT INTO letter_participants (letter_id, user_id, role, step_order)
SELECT @letterId, j.userId, j.role, j.step
FROM OPENJSON(@rows) WITH (userId BIGINT, role VARCHAR(20), step INT) j`

	deleteParticipantsSQL = `
UPDATE letter_participants SET deleted_at = SYSDATETIMEOFFSET(), updated_at = SYSDATETIMEOFFSET()
WHERE letter_id = @letterId AND deleted_at IS NULL
  AND id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids))`
)

// loadParticipants membaca peserta aktif: reviewer urut step, lalu approver.
func (r *Repository) loadParticipants(ctx context.Context, letterID int64) ([]participant, error) {
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

// draftParticipants mengubah baris peserta (urut dari loadParticipants) menjadi reviewer dan approver draf.
func draftParticipants(ps []participant) LetterDraft {
	var d LetterDraft
	for _, p := range ps {
		if p.Role == RoleApprover {
			d.ApproverID = &p.UserID
		} else {
			d.ReviewerIDs = append(d.ReviewerIDs, p.UserID)
		}
	}
	return d
}

func (r *Repository) saveParticipants(ctx context.Context, letterID int64, d *LetterDraft) error {
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

// IsParticipant mengecek apakah user adalah reviewer/approver aktif di surat.
func (r *Repository) IsParticipant(ctx context.Context, letterID, userID int64) (bool, error) {
	const q = `
SELECT CASE WHEN EXISTS (
    SELECT 1 FROM letter_participants
    WHERE letter_id = @letterId AND user_id = @userId AND deleted_at IS NULL
) THEN 1 ELSE 0 END`
	var ok bool
	if err := r.q.QueryRowContext(ctx, q, sql.Named("letterId", letterID), sql.Named("userId", userID)).Scan(&ok); err != nil {
		return false, fmt.Errorf("memeriksa peserta surat: %w", err)
	}
	return ok, nil
}

// Participants mengembalikan reviewer (urut step) dan approver surat (0 bila belum dipilih).
func (r *Repository) Participants(ctx context.Context, letterID int64) (reviewers []int64, approverID int64, err error) {
	ps, err := r.loadParticipants(ctx, letterID)
	if err != nil {
		return nil, 0, err
	}
	d := draftParticipants(ps)
	if d.ApproverID != nil {
		approverID = *d.ApproverID
	}
	return d.ReviewerIDs, approverID, nil
}
