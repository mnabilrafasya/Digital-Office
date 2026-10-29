package inbox

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"office.backend/internal/platform/db"
)

// ItemLock adalah item inbox yang dikunci untuk diteruskan.
type ItemLock struct {
	ID         int64
	LetterID   int64
	LetterUUID string
	Status     string
}

// LockItem mengunci item inbox milik userID (UPDLOCK) sampai transaksi selesai, supaya dua
// disposisi/pengalihan bersamaan dari item yang sama tidak sama-sama berhasil. db.ErrNotFound
// bila item tidak ada atau milik orang lain. Wajib dipanggil di dalam transaksi.
func (r *Repository) LockItem(ctx context.Context, companyCode string, userID, itemID int64) (*ItemLock, error) {
	const q = `
SELECT i.id, i.letter_id, CONVERT(char(36), l.uuid), i.status
FROM inbox_items i WITH (UPDLOCK, ROWLOCK)
JOIN letters l ON l.id = i.letter_id
WHERE i.id = @itemId AND i.user_id = @userId AND i.deleted_at IS NULL
  AND l.company_code = @companyCode AND l.deleted_at IS NULL`
	var it ItemLock
	err := r.q.QueryRowContext(ctx, q, sql.Named("itemId", itemID), sql.Named("userId", userID),
		sql.Named("companyCode", companyCode)).Scan(&it.ID, &it.LetterID, &it.LetterUUID, &it.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("mengunci item inbox: %w", err)
	}
	it.LetterUUID = strings.ToLower(it.LetterUUID)
	return &it, nil
}

// MarkRead menandai item inbox UNREAD menjadi READ (item berstatus lain tidak berubah).
func (r *Repository) MarkRead(ctx context.Context, itemID int64) error {
	const q = `
UPDATE inbox_items
SET status = 'READ', read_at = SYSDATETIMEOFFSET(), updated_at = SYSDATETIMEOFFSET()
WHERE id = @id AND status = 'UNREAD' AND deleted_at IS NULL`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", itemID)); err != nil {
		return fmt.Errorf("menandai inbox dibaca: %w", err)
	}
	return nil
}

// NewForward adalah satu penerusan yang akan disimpan beserta item inbox penerimanya.
type NewForward struct {
	LetterID     int64
	SourceItemID int64
	Type         string // DISPOSITION atau REDIRECT
	FromUserID   int64
	FromName     string // snapshot nama dan jabatan pengirim
	FromPosition string
	TargetType   string // USER atau UNIT
	TargetID     int64
	RecipientID  int64 // penerima di inbox; 0 = tidak dibuatkan item (sudah menerima dari target lain)
	Note         *string
}

// AddForward menyimpan satu baris letter_forwards dan item inbox FORWARD (UNREAD) untuk penerimanya.
func (r *Repository) AddForward(ctx context.Context, f NewForward) error {
	const insert = `
INSERT INTO letter_forwards (letter_id, type, source_inbox_item_id, from_user_id, from_user_name,
                             from_position, target_type, target_user_id, target_unit_id, note)
OUTPUT inserted.id
VALUES (@letterId, @type, @sourceItemId, @fromUserId, @fromName, @fromPosition, @targetType,
        @targetUserId, @targetUnitId, @note)`
	var targetUser, targetUnit int64
	if f.TargetType == TargetUser {
		targetUser = f.TargetID
	} else {
		targetUnit = f.TargetID
	}
	var forwardID int64
	if err := r.q.QueryRowContext(ctx, insert,
		sql.Named("letterId", f.LetterID),
		sql.Named("type", f.Type),
		sql.Named("sourceItemId", f.SourceItemID),
		sql.Named("fromUserId", f.FromUserID),
		sql.Named("fromName", f.FromName),
		sql.Named("fromPosition", f.FromPosition),
		sql.Named("targetType", f.TargetType),
		sql.Named("targetUserId", db.NullID(targetUser)),
		sql.Named("targetUnitId", db.NullID(targetUnit)),
		sql.Named("note", f.Note),
	).Scan(&forwardID); err != nil {
		return fmt.Errorf("menyimpan penerusan surat: %w", err)
	}
	if f.RecipientID == 0 {
		return nil
	}

	const item = `
INSERT INTO inbox_items (letter_id, user_id, source_type, forward_id, status)
VALUES (@letterId, @userId, 'FORWARD', @forwardId, 'UNREAD')`
	if _, err := r.q.ExecContext(ctx, item, sql.Named("letterId", f.LetterID),
		sql.Named("userId", f.RecipientID), sql.Named("forwardId", forwardID)); err != nil {
		return fmt.Errorf("mengirim penerusan ke inbox: %w", err)
	}
	return nil
}

// SetForwarded mengubah status item sumber menjadi DISPOSED atau REDIRECTED (sekaligus dianggap
// sudah dibaca).
func (r *Repository) SetForwarded(ctx context.Context, itemID int64, status string) error {
	const q = `
UPDATE inbox_items
SET status = @status, read_at = ISNULL(read_at, SYSDATETIMEOFFSET()), updated_at = SYSDATETIMEOFFSET()
WHERE id = @id AND deleted_at IS NULL`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", itemID), sql.Named("status", status)); err != nil {
		return fmt.Errorf("mengubah status inbox: %w", err)
	}
	return nil
}

// Forwards mengambil penerusan yang dibuat dari satu item inbox, urut dari yang paling lama.
func (r *Repository) Forwards(ctx context.Context, itemID int64) ([]Forward, error) {
	const q = `
SELECT fw.id, fw.type, fw.target_type,
       tu.id, tu.name, tp.name, un.id, un.code, un.name,
       ru.id, ru.name, rp.name, fw.note, fw.created_at
FROM letter_forwards fw
LEFT JOIN users tu       ON tu.id = fw.target_user_id
LEFT JOIN positions tp   ON tp.id = tu.position_id
LEFT JOIN units un       ON un.id = fw.target_unit_id
LEFT JOIN inbox_items ri ON ri.forward_id = fw.id AND ri.deleted_at IS NULL
LEFT JOIN users ru       ON ru.id = ri.user_id
LEFT JOIN positions rp   ON rp.id = ru.position_id
WHERE fw.source_inbox_item_id = @itemId AND fw.deleted_at IS NULL
ORDER BY fw.id`
	rows, err := r.q.QueryContext(ctx, q, sql.Named("itemId", itemID))
	if err != nil {
		return nil, fmt.Errorf("mengambil penerusan surat: %w", err)
	}
	defer rows.Close()

	out := []Forward{}
	for rows.Next() {
		var (
			f                           Forward
			userID, unitID, recipientID sql.NullInt64
			userName, userPos           sql.NullString
			unitCode, unitName          sql.NullString
			recipientName, recipientPos sql.NullString
		)
		if err := rows.Scan(&f.ID, &f.Type, &f.TargetType,
			&userID, &userName, &userPos, &unitID, &unitCode, &unitName,
			&recipientID, &recipientName, &recipientPos, &f.Note, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("membaca penerusan surat: %w", err)
		}
		if userID.Valid {
			f.TargetUser = &PersonRef{ID: userID.Int64, Name: userName.String, PositionName: userPos.String}
		}
		if unitID.Valid {
			f.TargetUnit = &UnitRef{ID: unitID.Int64, Code: unitCode.String, Name: unitName.String}
		}
		if recipientID.Valid {
			f.Recipient = &PersonRef{ID: recipientID.Int64, Name: recipientName.String, PositionName: recipientPos.String}
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca penerusan surat: %w", err)
	}
	return out, nil
}
