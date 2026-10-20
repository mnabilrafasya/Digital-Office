package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// FindDetail memuat detail surat milik companyCode (pengecekan peserta adalah tugas service). Data
// dibaca dengan 3 query: surat, peserta, penerima. ETag diambil dari query pertama, jadi kalau ada
// perubahan di sela-selanya klien hanya akan mendapat 412 saat menyimpan (aman, tidak menimpa).
// LastReturnNote berisi catatan RETURN terakhir apa pun statusnya; service yang menyaringnya.
func (r *LetterRepository) FindDetail(ctx context.Context, companyCode, uuid string) (*model.LetterDetail, error) {
	d, err := r.findLetter(ctx, companyCode, uuid)
	if err != nil {
		return nil, err
	}
	if d.Steps, err = r.findSteps(ctx, d.InternalID); err != nil {
		return nil, err
	}
	if d.Recipients, err = r.findRecipients(ctx, d.InternalID); err != nil {
		return nil, err
	}
	d.Attachments = []model.Attachment{}
	d.AllowedActions = []string{}
	return d, nil
}

// nullRef menampung kolom id/kode/nama dari LEFT JOIN yang bisa NULL.
type nullRef struct {
	ID   sql.NullInt64
	Code sql.NullString
	Name sql.NullString
}

func (n nullRef) lookup() *model.LookupRef {
	if !n.ID.Valid {
		return nil
	}
	return &model.LookupRef{ID: n.ID.Int64, Code: n.Code.String, Name: n.Name.String}
}

func (r *LetterRepository) findLetter(ctx context.Context, companyCode, uuid string) (*model.LetterDetail, error) {
	const q = `
SELECT l.id, CONVERT(char(36), l.uuid), l.row_ver, l.status,
       l.letter_number, l.approved_at, l.title, l.summary, l.special_note,
       lt.id, lt.code, lt.name, cl.id, cl.code, cl.name,
       cf.id, cf.code, cf.name, ug.id, ug.code, ug.name,
       tp.id, tp.name,
       su.id, su.code, COALESCE(l.sender_unit_name, su.name),
       d.id, d.name, dp.name,
       a.id, a.name, ap.name, ISNULL(l.forwarded_by_id, 0),
       (SELECT TOP 1 ra.note FROM letter_actions ra
        WHERE ra.letter_id = l.id AND ra.action = 'RETURN' AND ra.deleted_at IS NULL
        ORDER BY ra.id DESC),
       l.current_version_number, l.created_at, l.updated_at
FROM letters l
JOIN units su               ON su.id = l.sender_unit_id
JOIN users d                ON d.id = l.drafter_id
JOIN positions dp           ON dp.id = d.position_id
LEFT JOIN lookups lt        ON lt.id = l.letter_type_id
LEFT JOIN lookups cl        ON cl.id = l.classification_id
LEFT JOIN lookups cf        ON cf.id = l.confidentiality_level_id
LEFT JOIN lookups ug        ON ug.id = l.urgency_level_id
LEFT JOIN letter_templates tp ON tp.id = l.template_id
LEFT JOIN users a           ON a.id = l.current_assignee_id
LEFT JOIN positions ap      ON ap.id = a.position_id
WHERE l.uuid = @uuid AND l.company_code = @companyCode AND l.deleted_at IS NULL`

	var (
		d                                model.LetterDetail
		letterType, class, conf, urgency nullRef
		templateID, assigneeID           sql.NullInt64
		templateName                     sql.NullString
		assigneeName, assigneePos        sql.NullString
	)
	err := r.q.QueryRowContext(ctx, q, sql.Named("uuid", uuid), sql.Named("companyCode", companyCode)).Scan(
		&d.InternalID, &d.ID, &d.RowVersion, &d.Status,
		&d.LetterNumber, &d.LetterDate, &d.Title, &d.Summary, &d.SpecialNote,
		&letterType.ID, &letterType.Code, &letterType.Name, &class.ID, &class.Code, &class.Name,
		&conf.ID, &conf.Code, &conf.Name, &urgency.ID, &urgency.Code, &urgency.Name,
		&templateID, &templateName,
		&d.SenderUnit.ID, &d.SenderUnit.Code, &d.SenderUnit.Name,
		&d.Drafter.ID, &d.Drafter.Name, &d.Drafter.PositionName,
		&assigneeID, &assigneeName, &assigneePos, &d.ForwardedByID, &d.LastReturnNote,
		&d.CurrentVersion, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("memuat surat: %w", err)
	}

	d.ID = strings.ToLower(d.ID)
	d.LetterType, d.Classification = letterType.lookup(), class.lookup()
	d.Confidentiality, d.Urgency = conf.lookup(), urgency.lookup()
	if templateID.Valid {
		d.Template = &model.TemplateRef{ID: templateID.Int64, Name: templateName.String}
	}
	if assigneeID.Valid {
		d.CurrentAssignee = &model.PersonRef{ID: assigneeID.Int64, Name: assigneeName.String, PositionName: assigneePos.String}
	}
	return &d, nil
}

func (r *LetterRepository) findSteps(ctx context.Context, letterID int64) ([]model.LetterStep, error) {
	const q = `
SELECT lp.role, lp.step_order, u.id, u.name, p.name, un.name
FROM letter_participants lp
JOIN users u      ON u.id = lp.user_id
JOIN positions p  ON p.id = u.position_id
JOIN units un     ON un.id = p.unit_id
WHERE lp.letter_id = @letterId AND lp.deleted_at IS NULL
ORDER BY CASE lp.role WHEN 'REVIEWER' THEN 0 ELSE 1 END, lp.step_order`

	rows, err := r.q.QueryContext(ctx, q, sql.Named("letterId", letterID))
	if err != nil {
		return nil, fmt.Errorf("mengambil peserta surat: %w", err)
	}
	defer rows.Close()

	out := []model.LetterStep{}
	for rows.Next() {
		s := model.LetterStep{State: model.StepPending}
		if err := rows.Scan(&s.Role, &s.Order, &s.User.ID, &s.User.Name, &s.User.PositionName, &s.User.UnitName); err != nil {
			return nil, fmt.Errorf("membaca peserta surat: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca peserta surat: %w", err)
	}
	return out, nil
}

func (r *LetterRepository) findRecipients(ctx context.Context, letterID int64) (model.LetterRecipients, error) {
	q := `
SELECT lr.recipient_type,
       un.id, un.code, un.name,
       h.id, h.name, hp.name,
       lv.id, lv.code, lv.name, ` + aliasMemberCountSQL + `
FROM letter_recipients lr
LEFT JOIN units un      ON un.id = lr.recipient_unit_id
LEFT JOIN users h       ON h.id = lr.resolved_head_id
LEFT JOIN positions hp  ON hp.id = h.position_id
LEFT JOIN lookups lv    ON lv.id = lr.alias_level_id
WHERE lr.letter_id = @letterId AND lr.deleted_at IS NULL
ORDER BY lr.sort_order, lr.id`

	out := model.LetterRecipients{To: []model.LetterRecipient{}, CC: []model.LetterRecipient{}}
	rows, err := r.q.QueryContext(ctx, q, sql.Named("letterId", letterID))
	if err != nil {
		return out, fmt.Errorf("mengambil penerima surat: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			typ           string
			unit, alias   nullRef
			headID        sql.NullInt64
			headName, pos sql.NullString
			memberCount   int
		)
		if err := rows.Scan(&typ, &unit.ID, &unit.Code, &unit.Name, &headID, &headName, &pos,
			&alias.ID, &alias.Code, &alias.Name, &memberCount); err != nil {
			return out, fmt.Errorf("membaca penerima surat: %w", err)
		}
		var rc model.LetterRecipient
		if alias.ID.Valid {
			rc.Alias = &model.Alias{LevelID: alias.ID.Int64, Code: alias.Code.String,
				Label: AliasLabel(alias.Name.String), MemberCount: memberCount}
		} else {
			rc.Unit = &model.UnitRef{ID: unit.ID.Int64, Code: unit.Code.String, Name: unit.Name.String}
			if headID.Valid {
				rc.ResolvedHead = &model.PersonRef{ID: headID.Int64, Name: headName.String, PositionName: pos.String}
			}
		}
		if typ == model.RecipientTo {
			out.To = append(out.To, rc)
		} else {
			out.CC = append(out.CC, rc)
		}
	}
	if err := rows.Err(); err != nil {
		return out, fmt.Errorf("membaca penerima surat: %w", err)
	}
	return out, nil
}
