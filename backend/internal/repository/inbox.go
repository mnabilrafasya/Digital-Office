package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// InboxRepository berisi query tabel inbox_items.
type InboxRepository struct {
	q db.DBTX
}

// NewInboxRepository membuat InboxRepository di atas koneksi atau transaksi.
func NewInboxRepository(q db.DBTX) *InboxRepository {
	return &InboxRepository{q: q}
}

// InsertDirect menyimpan item inbox DIRECT (status UNREAD) untuk setiap penerima akhir.
func (r *InboxRepository) InsertDirect(ctx context.Context, letterID int64, entries []model.InboxEntry) error {
	if len(entries) == 0 {
		return nil
	}
	param, err := jsonParam("items", entries)
	if err != nil {
		return err
	}
	const q = `
INSERT INTO inbox_items (letter_id, user_id, source_type, recipient_type)
SELECT @letterId, j.u, 'DIRECT', j.t
FROM OPENJSON(@items) WITH (u BIGINT '$.u', t VARCHAR(10) '$.t') j`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("letterId", letterID), param); err != nil {
		return fmt.Errorf("menyimpan inbox penerima: %w", err)
	}
	return nil
}
