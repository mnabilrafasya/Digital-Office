package surat

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// Attachments mengambil daftar lampiran surat. Wajib dipanggil di dalam transaksi, setelah
// LockByUUID, supaya daftar tidak berubah sebelum disimpan kembali.
func (r *Repository) Attachments(ctx context.Context, letterID int64) ([]Attachment, error) {
	var raw sql.NullString
	err := r.q.QueryRowContext(ctx, "SELECT attachments FROM letters WHERE id = @id", sql.Named("id", letterID)).Scan(&raw)
	if err != nil {
		return nil, fmt.Errorf("mengambil lampiran: %w", err)
	}
	return decodeAttachments(raw)
}

// SaveAttachments mengganti daftar lampiran surat (kosong = NULL) dan mengembalikan ROWVERSION
// surat terbaru.
func (r *Repository) SaveAttachments(ctx context.Context, letterID int64, items []Attachment) ([]byte, error) {
	var raw sql.NullString
	if len(items) > 0 {
		b, err := json.Marshal(items)
		if err != nil {
			return nil, fmt.Errorf("menyusun lampiran: %w", err)
		}
		raw = sql.NullString{String: string(b), Valid: true}
	}
	const q = `
UPDATE letters
SET attachments = @attachments, updated_at = SYSDATETIMEOFFSET()
OUTPUT inserted.row_ver
WHERE id = @id AND deleted_at IS NULL`
	var rowVer []byte
	if err := r.q.QueryRowContext(ctx, q, sql.Named("id", letterID), sql.Named("attachments", raw)).Scan(&rowVer); err != nil {
		return nil, fmt.Errorf("menyimpan lampiran: %w", err)
	}
	return rowVer, nil
}

// decodeAttachments membaca kolom letters.attachments (NULL = tanpa lampiran).
func decodeAttachments(raw sql.NullString) ([]Attachment, error) {
	items := []Attachment{}
	if !raw.Valid || raw.String == "" {
		return items, nil
	}
	if err := json.Unmarshal([]byte(raw.String), &items); err != nil {
		return nil, fmt.Errorf("membaca lampiran: %w", err)
	}
	return items, nil
}
