package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// ParticipantRepository berisi query tabel letter_participants.
type ParticipantRepository struct {
	q db.DBTX
}

// NewParticipantRepository membuat ParticipantRepository di atas koneksi atau transaksi.
func NewParticipantRepository(q db.DBTX) *ParticipantRepository {
	return &ParticipantRepository{q: q}
}

// List mengambil peserta aktif surat, urut reviewer berdasarkan step lalu approver.
func (r *ParticipantRepository) List(ctx context.Context, letterID int64) ([]model.Participant, error) {
	const q = `
SELECT id, user_id, role, step_order
FROM letter_participants
WHERE letter_id = @letterId AND deleted_at IS NULL
ORDER BY CASE role WHEN 'REVIEWER' THEN 0 ELSE 1 END, step_order`

	rows, err := r.q.QueryContext(ctx, q, sql.Named("letterId", letterID))
	if err != nil {
		return nil, fmt.Errorf("mengambil peserta surat: %w", err)
	}
	defer rows.Close()

	out := []model.Participant{}
	for rows.Next() {
		var p model.Participant
		if err := rows.Scan(&p.ID, &p.UserID, &p.Role, &p.StepOrder); err != nil {
			return nil, fmt.Errorf("membaca peserta surat: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca peserta surat: %w", err)
	}
	return out, nil
}

// SoftDelete menandai peserta dengan ID tertentu sebagai terhapus.
func (r *ParticipantRepository) SoftDelete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	q := `
UPDATE letter_participants SET deleted_at = SYSDATETIMEOFFSET(), updated_at = SYSDATETIMEOFFSET()
WHERE deleted_at IS NULL AND id IN ` + idsIn("ids")
	if _, err := r.q.ExecContext(ctx, q, idsParam("ids", ids)); err != nil {
		return fmt.Errorf("menghapus peserta surat: %w", err)
	}
	return nil
}

type participantJSON struct {
	UserID int64  `json:"u"`
	Role   string `json:"r"`
	Step   int    `json:"s"`
}

// Insert menyimpan beberapa peserta sekaligus.
func (r *ParticipantRepository) Insert(ctx context.Context, letterID int64, items []model.NewParticipant) error {
	if len(items) == 0 {
		return nil
	}
	rows := make([]participantJSON, len(items))
	for i, p := range items {
		rows[i] = participantJSON{UserID: p.UserID, Role: p.Role, Step: p.StepOrder}
	}
	param, err := jsonParam("items", rows)
	if err != nil {
		return err
	}

	const q = `
INSERT INTO letter_participants (letter_id, user_id, role, step_order)
SELECT @letterId, j.u, j.r, j.s
FROM OPENJSON(@items) WITH (u BIGINT '$.u', r VARCHAR(20) '$.r', s INT '$.s') j`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("letterId", letterID), param); err != nil {
		return fmt.Errorf("menyimpan peserta surat: %w", err)
	}
	return nil
}
