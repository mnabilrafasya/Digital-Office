package audit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"office.backend/internal/platform/db"
)

// letterCore adalah data surat yang dibutuhkan pelacakan.
type letterCore struct {
	ID              int64
	UUID            string
	Status          string
	Number, Title   *string
	Confidentiality string // kode lookup CONFIDENTIALITY, "" bila belum diisi
	Drafter         PersonRef
	Holder          *PersonRef     // current_assignee
	HolderStep      *int           // step reviewer milik Holder
	HolderDelegate  *DelegationRef // Holder sedang mendelegasikan tugas review/approval-nya
	Since           time.Time      // waktu perpindahan terakhir (atau dibuat, bila belum pernah berpindah)
	ApprovedAt      *time.Time
	CreatedAt       time.Time
}

// lastMoveSQL adalah waktu perpindahan terakhir surat l: aksi alur terakhir selain READ/SAVE, atau
// waktu surat dibuat bila belum pernah berpindah.
const lastMoveSQL = `COALESCE((SELECT MAX(x.created_at) FROM letter_actions x
          WHERE x.letter_id = l.id AND x.action NOT IN ('READ', 'SAVE')), l.created_at)`

// holderStepSQL adalah step pemegang surat l di perannya: approver saat IN_APPROVAL/APPROVED,
// selain itu reviewer (NULL bila pemegangnya bukan peserta di peran itu).
const holderStepSQL = `(SELECT TOP 1 lp.step_order FROM letter_participants lp
          WHERE lp.letter_id = l.id AND lp.user_id = l.current_assignee_id
            AND lp.role = CASE WHEN l.status IN ('IN_APPROVAL', 'APPROVED') THEN 'APPROVER' ELSE 'REVIEWER' END
            AND lp.deleted_at IS NULL)`

// findLetter memuat data pelacakan satu surat milik companyCode (db.ErrNotFound bila tidak ada).
// today (YYYY-MM-DD) menentukan delegasi pemegang surat yang sedang berlaku.
func (r *Repository) findLetter(ctx context.Context, companyCode, uuid, today string) (*letterCore, error) {
	q := `
SELECT l.id, CONVERT(char(36), l.uuid), l.status, l.letter_number, l.title, ISNULL(cf.code, ''),
       d.id, d.name, dp.name, a.id, a.name, ap.name, ` + holderStepSQL + `,
       ` + lastMoveSQL + `, l.approved_at, l.created_at,
       dg.delegate_id, dg.delegate_name, dg.delegate_position, dg.until
FROM letters l
JOIN users d           ON d.id = l.drafter_id
JOIN positions dp      ON dp.id = d.position_id
LEFT JOIN lookups cf   ON cf.id = l.confidentiality_level_id
LEFT JOIN users a      ON a.id = l.current_assignee_id
LEFT JOIN positions ap ON ap.id = a.position_id
OUTER APPLY (SELECT TOP 1 g.id AS delegate_id, g.name AS delegate_name, gp.name AS delegate_position,
                    CONVERT(char(10), dl.end_date, 23) AS until
             FROM delegations dl
             JOIN users g      ON g.id = dl.delegate_id
             JOIN positions gp ON gp.id = g.position_id
             WHERE dl.user_id = l.current_assignee_id
               AND l.status IN ('IN_REVIEW', 'RETURNED_TO_REVIEWER', 'IN_APPROVAL', 'APPROVED')
               AND dl.deleted_at IS NULL AND @today BETWEEN dl.start_date AND dl.end_date
               AND g.is_active = 1 AND g.deleted_at IS NULL) dg
WHERE l.uuid = @uuid AND l.company_code = @companyCode AND l.deleted_at IS NULL`
	var (
		c                     letterCore
		holderID              sql.NullInt64
		holderName, holderPos sql.NullString
		holderStep            sql.NullInt32
		approvedAt            sql.NullTime
		delegateID            sql.NullInt64
		delegateName          sql.NullString
		delegatePos, until    sql.NullString
	)
	err := r.q.QueryRowContext(ctx, q, sql.Named("uuid", uuid), sql.Named("companyCode", companyCode),
		sql.Named("today", today)).Scan(
		&c.ID, &c.UUID, &c.Status, &c.Number, &c.Title, &c.Confidentiality,
		&c.Drafter.ID, &c.Drafter.Name, &c.Drafter.PositionName, &holderID, &holderName, &holderPos, &holderStep,
		&c.Since, &approvedAt, &c.CreatedAt, &delegateID, &delegateName, &delegatePos, &until)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("memuat surat untuk pelacakan: %w", err)
	}
	c.UUID = strings.ToLower(c.UUID)
	if holderID.Valid {
		c.Holder = &PersonRef{ID: holderID.Int64, Name: holderName.String, PositionName: holderPos.String}
	}
	if holderStep.Valid {
		step := int(holderStep.Int32)
		c.HolderStep = &step
	}
	if approvedAt.Valid {
		c.ApprovedAt = &approvedAt.Time
	}
	if delegateID.Valid {
		c.HolderDelegate = &DelegationRef{
			Delegate: PersonRef{ID: delegateID.Int64, Name: delegateName.String, PositionName: delegatePos.String},
			Until:    until.String,
		}
	}
	return &c, nil
}

// involvement menentukan hubungan user dengan surat: peserta (reviewer/approver, atau pernah
// bertindak atas surat sebagai pengganti delegasi) dan penerima inbox.
func (r *Repository) involvement(ctx context.Context, letterID, userID int64) (participant, recipient bool, err error) {
	const q = `
SELECT CASE WHEN EXISTS (SELECT 1 FROM letter_participants
                         WHERE letter_id = @letterId AND user_id = @userId AND deleted_at IS NULL)
              OR EXISTS (SELECT 1 FROM letter_actions
                         WHERE letter_id = @letterId AND actor_id = @userId AND action <> 'READ') THEN 1 ELSE 0 END,
       CASE WHEN EXISTS (SELECT 1 FROM inbox_items
                         WHERE letter_id = @letterId AND user_id = @userId) THEN 1 ELSE 0 END`
	err = r.q.QueryRowContext(ctx, q, sql.Named("letterId", letterID), sql.Named("userId", userID)).Scan(&participant, &recipient)
	if err != nil {
		return false, false, fmt.Errorf("memeriksa akses pelacakan: %w", err)
	}
	return participant, recipient, nil
}

// letterAction adalah satu baris riwayat alur surat (letter_actions).
type letterAction struct {
	Action, ActorRole string
	Actor             PersonRef
	Target            *PersonRef
	ToStatus          string
	Note              *string
	VersionNumber     *int
	At                time.Time
}

// actions mengambil riwayat alur surat, urut dari yang paling lama.
func (r *Repository) actions(ctx context.Context, letterID int64) ([]letterAction, error) {
	const q = `
SELECT a.action, a.actor_role, u.id, u.name, p.name, t.id, t.name, tp.name,
       a.to_status, a.note, a.version_number, a.created_at
FROM letter_actions a
JOIN users u           ON u.id = a.actor_id
JOIN positions p       ON p.id = u.position_id
LEFT JOIN users t      ON t.id = a.target_user_id
LEFT JOIN positions tp ON tp.id = t.position_id
WHERE a.letter_id = @letterId
ORDER BY a.id`
	rows, err := r.q.QueryContext(ctx, q, sql.Named("letterId", letterID))
	if err != nil {
		return nil, fmt.Errorf("mengambil riwayat surat: %w", err)
	}
	defer rows.Close()

	var out []letterAction
	for rows.Next() {
		var (
			a                  letterAction
			targetID           sql.NullInt64
			targetName, tgtPos sql.NullString
			version            sql.NullInt32
		)
		if err := rows.Scan(&a.Action, &a.ActorRole, &a.Actor.ID, &a.Actor.Name, &a.Actor.PositionName,
			&targetID, &targetName, &tgtPos, &a.ToStatus, &a.Note, &version, &a.At); err != nil {
			return nil, fmt.Errorf("membaca riwayat surat: %w", err)
		}
		if targetID.Valid {
			a.Target = &PersonRef{ID: targetID.Int64, Name: targetName.String, PositionName: tgtPos.String}
		}
		if version.Valid {
			v := int(version.Int32)
			a.VersionNumber = &v
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca riwayat surat: %w", err)
	}
	return out, nil
}

// inboxRow adalah satu item inbox surat beserta asal penerusannya.
type inboxRow struct {
	Node     DistributionNode
	OwnerID  int64
	ParentID int64 // item inbox sumber penerusan; 0 = penerima langsung
}

// distribution mengambil semua item inbox satu surat, urut dari yang paling lama.
func (r *Repository) distribution(ctx context.Context, letterID int64) ([]inboxRow, error) {
	const q = `
SELECT i.id, i.user_id, u.name, p.name, i.source_type, i.recipient_type, i.status, i.created_at, i.read_at,
       fw.type, ISNULL(fw.source_inbox_item_id, 0), fw.note, fw.from_user_id, fw.from_user_name, fw.from_position
FROM inbox_items i
JOIN users u                 ON u.id = i.user_id
JOIN positions p             ON p.id = u.position_id
LEFT JOIN letter_forwards fw ON fw.id = i.forward_id
WHERE i.letter_id = @letterId
ORDER BY i.id`
	rows, err := r.q.QueryContext(ctx, q, sql.Named("letterId", letterID))
	if err != nil {
		return nil, fmt.Errorf("mengambil distribusi surat: %w", err)
	}
	defer rows.Close()

	var out []inboxRow
	for rows.Next() {
		var (
			row                   inboxRow
			source                string
			recipientType, fwType sql.NullString
			readAt                sql.NullTime
			fromID                sql.NullInt64
			fromName, fromPos     sql.NullString
		)
		n := &row.Node
		if err := rows.Scan(&n.InboxItemID, &row.OwnerID, &n.Recipient.Name, &n.Recipient.PositionName,
			&source, &recipientType, &n.Status, &n.ReceivedAt, &readAt,
			&fwType, &row.ParentID, &n.Note, &fromID, &fromName, &fromPos); err != nil {
			return nil, fmt.Errorf("membaca distribusi surat: %w", err)
		}
		n.Recipient.ID = row.OwnerID
		n.Via = recipientType.String
		if source == "FORWARD" {
			n.Via = fwType.String
		}
		if readAt.Valid {
			n.ReadAt = &readAt.Time
		}
		if fromID.Valid {
			n.From = &PersonRef{ID: fromID.Int64, Name: fromName.String, PositionName: fromPos.String}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca distribusi surat: %w", err)
	}
	return out, nil
}
