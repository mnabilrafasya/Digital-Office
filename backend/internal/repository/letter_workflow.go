package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/workflow"
)

// ApplyTransition menerapkan keputusan workflow ke surat lalu mencatat 1 baris letter_actions.
// Status lama ikut menjadi syarat UPDATE sebagai pengaman tambahan selain kunci baris.
// Wajib dipanggil di dalam transaksi, setelah LockByUUID.
func (r *LetterRepository) ApplyTransition(ctx context.Context, letterID, actorID int64, d workflow.Decision, note *string) error {
	const update = `
UPDATE letters
SET status = @to,
    current_assignee_id = @assigneeId,
    forwarded_by_id = CASE WHEN @setForwardedBy = 1 THEN @forwardedBy ELSE forwarded_by_id END,
    returned_by_reviewer_id = CASE WHEN @setReturnedByReviewer = 1 THEN @returnedByReviewer ELSE returned_by_reviewer_id END,
    returned_by_approver_id = CASE WHEN @setReturnedByApprover = 1 THEN @returnedByApprover ELSE returned_by_approver_id END,
    sender_unit_name = CASE WHEN @snapshotSender = 1
                            THEN (SELECT un.name FROM units un WHERE un.id = letters.sender_unit_id)
                            ELSE sender_unit_name END,
    updated_at = SYSDATETIMEOFFSET()
OUTPUT inserted.current_version_number
WHERE id = @id AND status = @from AND deleted_at IS NULL`

	var version int
	err := r.q.QueryRowContext(ctx, update,
		sql.Named("id", letterID),
		sql.Named("from", d.From),
		sql.Named("to", d.To),
		sql.Named("assigneeId", nullID(d.AssigneeID)),
		sql.Named("setForwardedBy", d.ForwardedBy.Changed),
		sql.Named("forwardedBy", nullID(d.ForwardedBy.UserID)),
		sql.Named("setReturnedByReviewer", d.ReturnedByReviewer.Changed),
		sql.Named("returnedByReviewer", nullID(d.ReturnedByReviewer.UserID)),
		sql.Named("setReturnedByApprover", d.ReturnedByApprover.Changed),
		sql.Named("returnedByApprover", nullID(d.ReturnedByApprover.UserID)),
		sql.Named("snapshotSender", d.SnapshotSenderUnit),
	).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("status surat berubah saat transisi %s dari %s", d.Action, d.From)
	}
	if err != nil {
		return fmt.Errorf("mengubah status surat: %w", err)
	}

	const insert = `
INSERT INTO letter_actions (letter_id, actor_id, target_user_id, actor_role, action, from_status, to_status, note, version_number)
VALUES (@letterId, @actorId, @targetId, @actorRole, @action, @from, @to, @note, @version)`
	if _, err := r.q.ExecContext(ctx, insert,
		sql.Named("letterId", letterID),
		sql.Named("actorId", actorID),
		sql.Named("targetId", nullID(d.AssigneeID)),
		sql.Named("actorRole", d.ActorRole),
		sql.Named("action", string(d.Action)),
		sql.Named("from", d.From),
		sql.Named("to", d.To),
		sql.Named("note", note),
		sql.Named("version", version),
	); err != nil {
		return fmt.Errorf("mencatat riwayat surat: %w", err)
	}
	return nil
}

// RecordRead mencatat aksi READ saat penerima tugas membuka surat, maksimal sekali sejak transisi
// terakhir. Baris letters dikunci lebih dulu (urutan kunci sama dengan transisi, jadi tidak bisa
// deadlock dengannya), sehingga dua pembukaan bersamaan tidak mencatat READ ganda. Baris letters
// tidak diubah. Wajib dipanggil di dalam transaksi.
func (r *LetterRepository) RecordRead(ctx context.Context, letterID, userID int64, role string) error {
	const lock = `
SELECT 1 FROM letters WITH (UPDLOCK, ROWLOCK)
WHERE id = @letterId AND current_assignee_id = @userId AND deleted_at IS NULL`
	var locked int
	err := r.q.QueryRowContext(ctx, lock, sql.Named("letterId", letterID), sql.Named("userId", userID)).Scan(&locked)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // surat sudah berpindah tangan, tidak perlu dicatat
	}
	if err != nil {
		return fmt.Errorf("mengunci surat untuk mencatat READ: %w", err)
	}

	const insert = `
INSERT INTO letter_actions (letter_id, actor_id, actor_role, action, from_status, to_status, version_number)
SELECT l.id, @userId, @role, 'READ', l.status, l.status, l.current_version_number
FROM letters l
WHERE l.id = @letterId
  AND NOT EXISTS (
        SELECT 1 FROM letter_actions a
        WHERE a.letter_id = l.id AND a.actor_id = @userId AND a.action = 'READ' AND a.deleted_at IS NULL
          AND a.id > ISNULL((SELECT MAX(t.id) FROM letter_actions t
                             WHERE t.letter_id = l.id AND t.action NOT IN ('READ', 'SAVE')), 0))`
	if _, err := r.q.ExecContext(ctx, insert,
		sql.Named("letterId", letterID), sql.Named("userId", userID), sql.Named("role", role)); err != nil {
		return fmt.Errorf("mencatat READ: %w", err)
	}
	return nil
}
