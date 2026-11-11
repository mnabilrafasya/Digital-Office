package surat

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// NewIncoming adalah surat masuk eksternal yang dicatat pengunggahnya.
type NewIncoming struct {
	CompanyCode  string
	SenderUnitID int64 // unit pengunggah (pencatat)
	UploaderID   int64 // dicatat sebagai drafter (pencatat surat)
	LetterTypeID int64
	Title        string
	External     ExternalInfo
}

// CreateIncoming menyimpan surat masuk eksternal (status RECEIVED) beserta riwayat RECEIVE, lalu
// mengembalikan ID dan uuid-nya. Wajib dipanggil di dalam transaksi.
func (r *Repository) CreateIncoming(ctx context.Context, n NewIncoming) (int64, string, error) {
	raw, _ := json.Marshal(n.External) // struct teks selalu berhasil di-marshal
	const q = `
INSERT INTO letters (company_code, sender_unit_id, drafter_id, letter_type_id, title, status,
                     external_info, sender_unit_name)
OUTPUT inserted.id, CONVERT(char(36), inserted.uuid)
SELECT @companyCode, @senderUnitId, @uploaderId, @letterTypeId, @title, 'RECEIVED', @externalInfo, un.name
FROM units un WHERE un.id = @senderUnitId`
	var (
		id   int64
		uuid string
	)
	if err := r.q.QueryRowContext(ctx, q,
		sql.Named("companyCode", n.CompanyCode),
		sql.Named("senderUnitId", n.SenderUnitID),
		sql.Named("uploaderId", n.UploaderID),
		sql.Named("letterTypeId", n.LetterTypeID),
		sql.Named("title", n.Title),
		sql.Named("externalInfo", string(raw)),
	).Scan(&id, &uuid); err != nil {
		return 0, "", fmt.Errorf("mencatat surat masuk: %w", err)
	}
	const action = `
INSERT INTO letter_actions (letter_id, actor_id, actor_role, action, from_status, to_status, version_number)
VALUES (@letterId, @actorId, 'DRAFTER', 'RECEIVE', NULL, 'RECEIVED', NULL)`
	if _, err := r.q.ExecContext(ctx, action, sql.Named("letterId", id), sql.Named("actorId", n.UploaderID)); err != nil {
		return 0, "", fmt.Errorf("mencatat riwayat surat masuk: %w", err)
	}
	return id, strings.ToLower(uuid), nil
}
