package surat

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"office.backend/internal/platform/db"
)

// LetterFilter adalah kriteria daftar surat. Nilai nol berarti tidak disaring.
type LetterFilter struct {
	CompanyCode string
	DrafterID   int64
	AssigneeID  int64
	TaskUserID  int64  // tugas user ini: miliknya (kecuali sedang mendelegasikan) dan yang ia wakili
	Today       string // YYYY-MM-DD, untuk delegasi yang sedang berlaku
	Statuses    []string
	Search      string // dicari di perihal dan nomor surat
	Offset      int
	Limit       int
}

// where menyusun klausa WHERE dari potongan SQL tetap; nilai filter selalu lewat parameter.
func (f LetterFilter) where() (string, []any) {
	var sb strings.Builder
	sb.WriteString("\nWHERE l.company_code = @companyCode AND l.deleted_at IS NULL")
	args := []any{sql.Named("companyCode", f.CompanyCode), sql.Named("today", f.Today)}
	if f.DrafterID != 0 {
		sb.WriteString("\n  AND l.drafter_id = @drafterId")
		args = append(args, sql.Named("drafterId", f.DrafterID))
	}
	if f.AssigneeID != 0 {
		sb.WriteString("\n  AND l.current_assignee_id = @assigneeId")
		args = append(args, sql.Named("assigneeId", f.AssigneeID))
	}
	if f.TaskUserID != 0 {
		// Tugas milik sendiri, kecuali tahap reviewer/approver yang sedang didelegasikan; ditambah
		// tahap reviewer/approver milik orang yang sedang mendelegasikan kepadanya.
		sb.WriteString(`
  AND (
        (l.current_assignee_id = @taskUserId
         AND NOT (l.status IN (` + delegableStatusesSQL + `) AND EXISTS (
             SELECT 1 FROM delegations dl JOIN users g ON g.id = dl.delegate_id
             WHERE dl.user_id = @taskUserId AND ` + activeDelegationSQL + `)))
     OR (l.status IN (` + delegableStatusesSQL + `) AND EXISTS (
             SELECT 1 FROM delegations dl JOIN users g ON g.id = dl.delegate_id
             WHERE dl.user_id = l.current_assignee_id AND dl.delegate_id = @taskUserId AND ` + activeDelegationSQL + `))
  )`)
		args = append(args, sql.Named("taskUserId", f.TaskUserID))
	}
	if len(f.Statuses) > 0 {
		statuses, _ := json.Marshal(f.Statuses) // []string selalu berhasil di-marshal
		sb.WriteString("\n  AND l.status IN (SELECT [value] FROM OPENJSON(@statuses))")
		args = append(args, sql.Named("statuses", string(statuses)))
	}
	if f.Search != "" {
		sb.WriteString("\n  AND (l.title LIKE @pattern ESCAPE '\\' OR l.letter_number LIKE @pattern ESCAPE '\\')")
		args = append(args, sql.Named("pattern", db.ContainsPattern(f.Search)))
	}
	return sb.String(), args
}

// List mengambil satu halaman surat (terbaru diubah lebih dulu) beserta jumlah totalnya.
func (r *Repository) List(ctx context.Context, f LetterFilter) ([]LetterListItem, int, error) {
	where, args := f.where()
	q := `
SELECT CONVERT(char(36), l.uuid), l.letter_number, l.title, l.status,
       lt.id, lt.code, lt.name, cf.id, cf.code, cf.name, ug.id, ug.code, ug.name,
       su.id, su.code, COALESCE(l.sender_unit_name, su.name),
       a.id, a.name, ap.name,
       ISNULL((SELECT TOP 1 lp.step_order FROM letter_participants lp
               WHERE lp.letter_id = l.id AND lp.user_id = l.current_assignee_id
                 AND lp.role = CASE WHEN l.status IN ('IN_APPROVAL', 'APPROVED') THEN 'APPROVER' ELSE 'REVIEWER' END
                 AND lp.deleted_at IS NULL), 0),
       l.created_at, l.updated_at,
       dg.delegate_id, dg.delegate_name, dg.delegate_position, dg.until,
       COUNT(*) OVER ()
FROM letters l
JOIN units su          ON su.id = l.sender_unit_id
LEFT JOIN lookups lt   ON lt.id = l.letter_type_id
LEFT JOIN lookups cf   ON cf.id = l.confidentiality_level_id
LEFT JOIN lookups ug   ON ug.id = l.urgency_level_id
LEFT JOIN users a      ON a.id = l.current_assignee_id
LEFT JOIN positions ap ON ap.id = a.position_id
OUTER APPLY (SELECT TOP 1 g.id AS delegate_id, g.name AS delegate_name, gp.name AS delegate_position,
                    CONVERT(char(10), dl.end_date, 23) AS until
             FROM delegations dl
             JOIN users g      ON g.id = dl.delegate_id
             JOIN positions gp ON gp.id = g.position_id
             WHERE dl.user_id = l.current_assignee_id AND l.status IN (` + delegableStatusesSQL + `)
               AND ` + activeDelegationSQL + `) dg` + where + `
ORDER BY l.updated_at DESC, l.id DESC
OFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY`
	args = append(args, sql.Named("offset", f.Offset), sql.Named("limit", f.Limit))

	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar surat: %w", err)
	}
	defer rows.Close()

	items := []LetterListItem{}
	total := 0
	for rows.Next() {
		var (
			it                        LetterListItem
			letterType, conf, urgency nullRef
			assigneeID                sql.NullInt64
			assigneeName, assigneePos sql.NullString
			delegateID                sql.NullInt64
			delegateName, delegatePos sql.NullString
			delegateUntil             sql.NullString
		)
		if err := rows.Scan(&it.ID, &it.LetterNumber, &it.Title, &it.Status,
			&letterType.ID, &letterType.Code, &letterType.Name, &conf.ID, &conf.Code, &conf.Name,
			&urgency.ID, &urgency.Code, &urgency.Name,
			&it.SenderUnit.ID, &it.SenderUnit.Code, &it.SenderUnit.Name,
			&assigneeID, &assigneeName, &assigneePos, &it.AssigneeStep,
			&it.CreatedAt, &it.UpdatedAt,
			&delegateID, &delegateName, &delegatePos, &delegateUntil, &total,
		); err != nil {
			return nil, 0, fmt.Errorf("membaca daftar surat: %w", err)
		}
		it.ID = strings.ToLower(it.ID)
		it.LetterType, it.Confidentiality, it.Urgency = letterType.lookup(), conf.lookup(), urgency.lookup()
		if assigneeID.Valid {
			it.CurrentAssignee = &PersonRef{ID: assigneeID.Int64, Name: assigneeName.String, PositionName: assigneePos.String}
		}
		if delegateID.Valid {
			it.DelegatedTo = &DelegationRef{
				Delegate: PersonRef{ID: delegateID.Int64, Name: delegateName.String, PositionName: delegatePos.String},
				Until:    delegateUntil.String,
			}
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
