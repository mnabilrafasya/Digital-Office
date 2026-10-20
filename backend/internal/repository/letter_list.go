package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// LetterFilter adalah kriteria daftar surat. Nilai nol berarti tidak disaring.
type LetterFilter struct {
	CompanyCode string
	DrafterID   int64
	AssigneeID  int64
	Statuses    []string
	Search      string // dicari di perihal dan nomor surat
	Offset      int
	Limit       int
}

// where menyusun klausa WHERE dari potongan SQL tetap; nilai filter selalu lewat parameter.
func (f LetterFilter) where() (string, []any) {
	var sb strings.Builder
	sb.WriteString("\nWHERE l.company_code = @companyCode AND l.deleted_at IS NULL")
	args := []any{sql.Named("companyCode", f.CompanyCode)}
	if f.DrafterID != 0 {
		sb.WriteString("\n  AND l.drafter_id = @drafterId")
		args = append(args, sql.Named("drafterId", f.DrafterID))
	}
	if f.AssigneeID != 0 {
		sb.WriteString("\n  AND l.current_assignee_id = @assigneeId")
		args = append(args, sql.Named("assigneeId", f.AssigneeID))
	}
	if len(f.Statuses) > 0 {
		statuses, _ := json.Marshal(f.Statuses) // []string selalu berhasil di-marshal
		sb.WriteString("\n  AND l.status IN (SELECT [value] FROM OPENJSON(@statuses))")
		args = append(args, sql.Named("statuses", string(statuses)))
	}
	if f.Search != "" {
		sb.WriteString("\n  AND (l.title LIKE @pattern ESCAPE '\\' OR l.letter_number LIKE @pattern ESCAPE '\\')")
		args = append(args, sql.Named("pattern", ContainsPattern(f.Search)))
	}
	return sb.String(), args
}

// List mengambil satu halaman surat (terbaru diubah lebih dulu) beserta jumlah totalnya.
func (r *LetterRepository) List(ctx context.Context, f LetterFilter) ([]model.LetterListItem, int, error) {
	where, args := f.where()
	q := `
SELECT CONVERT(char(36), l.uuid), l.letter_number, l.title, l.status,
       lt.id, lt.code, lt.name, cf.id, cf.code, cf.name, ug.id, ug.code, ug.name,
       su.id, su.code, COALESCE(l.sender_unit_name, su.name),
       a.id, a.name, ap.name,
       ISNULL((SELECT TOP 1 lp.step_order FROM letter_participants lp
               WHERE lp.letter_id = l.id AND lp.user_id = l.current_assignee_id
                 AND lp.role = 'REVIEWER' AND lp.deleted_at IS NULL), 0),
       l.created_at, l.updated_at,
       COUNT(*) OVER ()
FROM letters l
JOIN units su          ON su.id = l.sender_unit_id
LEFT JOIN lookups lt   ON lt.id = l.letter_type_id
LEFT JOIN lookups cf   ON cf.id = l.confidentiality_level_id
LEFT JOIN lookups ug   ON ug.id = l.urgency_level_id
LEFT JOIN users a      ON a.id = l.current_assignee_id
LEFT JOIN positions ap ON ap.id = a.position_id` + where + `
ORDER BY l.updated_at DESC, l.id DESC
OFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY`
	args = append(args, sql.Named("offset", f.Offset), sql.Named("limit", f.Limit))

	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar surat: %w", err)
	}
	defer rows.Close()

	items := []model.LetterListItem{}
	total := 0
	for rows.Next() {
		var (
			it                        model.LetterListItem
			letterType, conf, urgency nullRef
			assigneeID                sql.NullInt64
			assigneeName, assigneePos sql.NullString
		)
		if err := rows.Scan(&it.ID, &it.LetterNumber, &it.Title, &it.Status,
			&letterType.ID, &letterType.Code, &letterType.Name, &conf.ID, &conf.Code, &conf.Name,
			&urgency.ID, &urgency.Code, &urgency.Name,
			&it.SenderUnit.ID, &it.SenderUnit.Code, &it.SenderUnit.Name,
			&assigneeID, &assigneeName, &assigneePos, &it.AssigneeStep,
			&it.CreatedAt, &it.UpdatedAt, &total,
		); err != nil {
			return nil, 0, fmt.Errorf("membaca daftar surat: %w", err)
		}
		it.ID = strings.ToLower(it.ID)
		it.LetterType, it.Confidentiality, it.Urgency = letterType.lookup(), conf.lookup(), urgency.lookup()
		if assigneeID.Valid {
			it.CurrentAssignee = &model.PersonRef{ID: assigneeID.Int64, Name: assigneeName.String, PositionName: assigneePos.String}
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca daftar surat: %w", err)
	}

	// Halaman di luar jangkauan tidak membawa COUNT(*) OVER (); hitung terpisah supaya total tetap benar
	if len(items) == 0 && f.Offset > 0 {
		where, args := f.where()
		if err := r.q.QueryRowContext(ctx, "SELECT COUNT(*) FROM letters l"+where, args...).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("menghitung surat: %w", err)
		}
	}
	return items, total, nil
}

// Actions mengambil riwayat aksi surat, urut dari yang paling lama.
func (r *LetterRepository) Actions(ctx context.Context, letterID int64) ([]model.LetterAction, error) {
	const q = `
SELECT a.action, a.actor_role, u.id, u.name, p.name,
       t.id, t.name, tp.name,
       a.from_status, a.to_status, a.note, a.version_number, a.created_at
FROM letter_actions a
JOIN users u           ON u.id = a.actor_id
JOIN positions p       ON p.id = u.position_id
LEFT JOIN users t      ON t.id = a.target_user_id
LEFT JOIN positions tp ON tp.id = t.position_id
WHERE a.letter_id = @letterId AND a.deleted_at IS NULL
ORDER BY a.id`

	rows, err := r.q.QueryContext(ctx, q, sql.Named("letterId", letterID))
	if err != nil {
		return nil, fmt.Errorf("mengambil riwayat surat: %w", err)
	}
	defer rows.Close()

	out := []model.LetterAction{}
	for rows.Next() {
		var (
			a                  model.LetterAction
			targetID           sql.NullInt64
			targetName, tgtPos sql.NullString
		)
		if err := rows.Scan(&a.Action, &a.ActorRole, &a.Actor.ID, &a.Actor.Name, &a.Actor.PositionName,
			&targetID, &targetName, &tgtPos,
			&a.FromStatus, &a.ToStatus, &a.Note, &a.VersionNumber, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("membaca riwayat surat: %w", err)
		}
		if targetID.Valid {
			a.Target = &model.PersonRef{ID: targetID.Int64, Name: targetName.String, PositionName: tgtPos.String}
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca riwayat surat: %w", err)
	}
	return out, nil
}
