package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// ActionRepository berisi query tabel letter_actions (riwayat alur surat).
type ActionRepository struct {
	q db.DBTX
}

// NewActionRepository membuat ActionRepository di atas koneksi atau transaksi.
func NewActionRepository(q db.DBTX) *ActionRepository {
	return &ActionRepository{q: q}
}

// Insert menyimpan satu baris riwayat.
func (r *ActionRepository) Insert(ctx context.Context, e model.ActionEntry) error {
	const q = `
INSERT INTO letter_actions (letter_id, actor_id, target_user_id, actor_role, action,
                            from_status, to_status, note, version_number)
VALUES (@letterId, @actorId, @targetUserId, @actorRole, @action,
        @fromStatus, @toStatus, @note, @versionNumber)`

	_, err := r.q.ExecContext(ctx, q,
		sql.Named("letterId", e.LetterID), sql.Named("actorId", e.ActorID),
		sql.Named("targetUserId", e.TargetUserID), sql.Named("actorRole", e.ActorRole),
		sql.Named("action", e.Action), sql.Named("fromStatus", e.FromStatus),
		sql.Named("toStatus", e.ToStatus), sql.Named("note", e.Note),
		sql.Named("versionNumber", e.VersionNumber))
	if err != nil {
		return fmt.Errorf("menyimpan riwayat surat: %w", err)
	}
	return nil
}

// RecordReadOnce mencatat aksi READ oleh user, kecuali ia sudah tercatat membaca sejak aksi
// transisi terakhir (transitions). Pemeriksaan dan penyimpanan dilakukan dalam satu pernyataan
// dengan kunci UPDLOCK+HOLDLOCK, sehingga dua permintaan bersamaan tidak mencatat dua kali.
func (r *ActionRepository) RecordReadOnce(ctx context.Context, letterID, userID int64, role, status string, transitions []string) error {
	param, err := jsonParam("transitions", transitions)
	if err != nil {
		return err
	}
	const q = `
INSERT INTO letter_actions (letter_id, actor_id, actor_role, action, from_status, to_status)
SELECT @letterId, @userId, @role, 'READ', @status, @status
WHERE NOT EXISTS (
    SELECT 1
    FROM letter_actions r WITH (UPDLOCK, HOLDLOCK)
    WHERE r.letter_id = @letterId AND r.actor_id = @userId AND r.action = 'READ'
      AND r.id > ISNULL((SELECT MAX(t.id) FROM letter_actions t
                         WHERE t.letter_id = @letterId
                           AND t.action IN (SELECT [value] FROM OPENJSON(@transitions))), 0)
)`
	_, err = r.q.ExecContext(ctx, q,
		sql.Named("letterId", letterID), sql.Named("userId", userID),
		sql.Named("role", role), sql.Named("status", status), param)
	if err != nil {
		return fmt.Errorf("mencatat surat dibaca: %w", err)
	}
	return nil
}

// Exists mengembalikan true bila surat pernah mendapat aksi tertentu.
func (r *ActionRepository) Exists(ctx context.Context, letterID int64, action string) (bool, error) {
	const q = `
SELECT CASE WHEN EXISTS (SELECT 1 FROM letter_actions
                         WHERE letter_id = @letterId AND action = @action AND deleted_at IS NULL)
            THEN 1 ELSE 0 END`
	var ok bool
	if err := r.q.QueryRowContext(ctx, q, sql.Named("letterId", letterID), sql.Named("action", action)).Scan(&ok); err != nil {
		return false, fmt.Errorf("memeriksa riwayat surat: %w", err)
	}
	return ok, nil
}

// LastNote mengambil catatan dari aksi tertentu yang terakhir (nil bila tidak ada).
func (r *ActionRepository) LastNote(ctx context.Context, letterID int64, action string) (*string, error) {
	const q = `
SELECT TOP 1 note FROM letter_actions
WHERE letter_id = @letterId AND action = @action AND deleted_at IS NULL
ORDER BY id DESC`
	var note *string
	err := r.q.QueryRowContext(ctx, q, sql.Named("letterId", letterID), sql.Named("action", action)).Scan(&note)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("mengambil catatan terakhir: %w", err)
	}
	return note, nil
}

// History mengambil seluruh riwayat surat, urut dari yang paling lama.
func (r *ActionRepository) History(ctx context.Context, letterID int64) ([]model.HistoryRow, error) {
	const q = `
SELECT la.action, la.actor_role, a.id, a.name, ap.name,
       t.id, t.name, tp.name,
       la.from_status, la.to_status, la.note, la.version_number, la.created_at
FROM letter_actions la
JOIN users a           ON a.id = la.actor_id
JOIN positions ap      ON ap.id = a.position_id
LEFT JOIN users t      ON t.id = la.target_user_id
LEFT JOIN positions tp ON tp.id = t.position_id
WHERE la.letter_id = @letterId AND la.deleted_at IS NULL
ORDER BY la.created_at, la.id`

	rows, err := r.q.QueryContext(ctx, q, sql.Named("letterId", letterID))
	if err != nil {
		return nil, fmt.Errorf("mengambil riwayat surat: %w", err)
	}
	defer rows.Close()

	out := []model.HistoryRow{}
	for rows.Next() {
		var (
			h                  model.HistoryRow
			targetID           sql.NullInt64
			targetName, target sql.NullString
		)
		if err := rows.Scan(&h.Action, &h.ActorRole, &h.Actor.ID, &h.Actor.Name, &h.Actor.PositionName,
			&targetID, &targetName, &target,
			&h.FromStatus, &h.ToStatus, &h.Note, &h.VersionNumber, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("membaca riwayat surat: %w", err)
		}
		if targetID.Valid {
			h.Target = &model.UserLabel{ID: targetID.Int64, Name: targetName.String, PositionName: target.String}
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca riwayat surat: %w", err)
	}
	return out, nil
}
