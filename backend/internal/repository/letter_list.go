package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// listWhere menyusun WHERE daftar surat. Potongan SQL berupa konstanta; nilai filter selalu parameter.
func listWhere(f model.LetterListFilter) (string, []any, error) {
	where := `
WHERE l.company_code = @companyCode AND l.deleted_at IS NULL`
	args := []any{sql.Named("companyCode", f.CompanyCode)}

	if f.DrafterID != 0 {
		where += "\n  AND l.drafter_id = @drafterId"
		args = append(args, sql.Named("drafterId", f.DrafterID))
	}
	if f.AssigneeID != 0 {
		p, err := jsonParam("taskStatuses", f.TaskStatuses)
		if err != nil {
			return "", nil, err
		}
		where += "\n  AND l.current_assignee_id = @assigneeId AND l.status IN (SELECT [value] FROM OPENJSON(@taskStatuses))"
		args = append(args, sql.Named("assigneeId", f.AssigneeID), p)
	}
	if len(f.Statuses) > 0 {
		p, err := jsonParam("statuses", f.Statuses)
		if err != nil {
			return "", nil, err
		}
		where += "\n  AND l.status IN (SELECT [value] FROM OPENJSON(@statuses))"
		args = append(args, p)
	}
	if f.Search != "" {
		where += "\n  AND (l.title LIKE @pattern ESCAPE '\\' OR l.letter_number LIKE @pattern ESCAPE '\\')"
		args = append(args, sql.Named("pattern", ContainsPattern(f.Search)))
	}
	return where, args, nil
}

// List mengambil satu halaman daftar surat beserta jumlah total yang cocok dengan filter.
func (r *LetterRepository) List(ctx context.Context, f model.LetterListFilter) ([]model.LetterListRow, int, error) {
	where, args, err := listWhere(f)
	if err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.q.QueryRowContext(ctx, "SELECT COUNT(*) FROM letters l"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("menghitung surat: %w", err)
	}
	if total == 0 {
		return []model.LetterListRow{}, 0, nil
	}

	q := `
SELECT LOWER(CONVERT(VARCHAR(36), l.uuid)), l.letter_number, l.title, l.status,
       lt.id, lt.code, lt.name, cf.id, cf.code, cf.name, ug.id, ug.code, ug.name,
       su.id, su.code, su.name,
       a.id, a.name, ap.name,
       rs.step_order,
       l.created_at, l.updated_at
FROM letters l
JOIN units su             ON su.id = l.sender_unit_id
LEFT JOIN lookups lt      ON lt.id = l.letter_type_id
LEFT JOIN lookups cf      ON cf.id = l.confidentiality_level_id
LEFT JOIN lookups ug      ON ug.id = l.urgency_level_id
LEFT JOIN users a         ON a.id = l.current_assignee_id
LEFT JOIN positions ap    ON ap.id = a.position_id
OUTER APPLY (
    SELECT TOP 1 lp.step_order
    FROM letter_participants lp
    WHERE lp.letter_id = l.id AND lp.user_id = l.current_assignee_id
      AND lp.role = 'REVIEWER' AND lp.deleted_at IS NULL
) rs` + where + `
ORDER BY l.updated_at DESC, l.id DESC
OFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY`
	args = append(args, sql.Named("offset", f.Offset), sql.Named("limit", f.Limit))

	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar surat: %w", err)
	}
	defer rows.Close()

	out := []model.LetterListRow{}
	for rows.Next() {
		var (
			row                      model.LetterListRow
			lt, cf, ug               nullLookup
			assigneeID               sql.NullInt64
			assigneeName, assigneePo sql.NullString
			step                     sql.NullInt32
		)
		if err := rows.Scan(&row.UUID, &row.LetterNumber, &row.Title, &row.Status,
			&lt.id, &lt.code, &lt.name, &cf.id, &cf.code, &cf.name, &ug.id, &ug.code, &ug.name,
			&row.SenderUnit.ID, &row.SenderUnit.Code, &row.SenderUnit.Name,
			&assigneeID, &assigneeName, &assigneePo,
			&step, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("membaca daftar surat: %w", err)
		}
		row.LetterType, row.Confidentiality, row.Urgency = lt.label(), cf.label(), ug.label()
		if assigneeID.Valid {
			row.Assignee = &model.UserLabel{ID: assigneeID.Int64, Name: assigneeName.String, PositionName: assigneePo.String}
		}
		if step.Valid {
			s := int(step.Int32)
			row.StepOrder = &s
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca daftar surat: %w", err)
	}
	return out, total, nil
}

type nullLookup struct {
	id         sql.NullInt64
	code, name sql.NullString
}

func (n nullLookup) label() *model.LookupLabel {
	if !n.id.Valid {
		return nil
	}
	return &model.LookupLabel{ID: n.id.Int64, Code: n.code.String, Name: n.name.String}
}
