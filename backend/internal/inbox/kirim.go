package inbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// delivery adalah satu penerima langsung, dikirim sebagai JSON ke OPENJSON.
type delivery struct {
	UserID        int64  `json:"user"`
	RecipientType string `json:"type"`
}

// DeliverDirect memasukkan surat terbit ke inbox penerima langsung dengan status UNREAD: to untuk
// Kepada, cc untuk Tembusan. Satu user hanya boleh muncul sekali (dijaga pemanggil). Dipanggil alur
// surat di dalam transaksi terbit.
func (r *Repository) DeliverDirect(ctx context.Context, letterID int64, to, cc []int64) error {
	items := make([]delivery, 0, len(to)+len(cc))
	for _, id := range to {
		items = append(items, delivery{UserID: id, RecipientType: "TO"})
	}
	for _, id := range cc {
		items = append(items, delivery{UserID: id, RecipientType: "CC"})
	}
	if len(items) == 0 {
		return nil
	}
	raw, _ := json.Marshal(items) // struct sederhana selalu berhasil di-marshal
	const q = `
INSERT INTO inbox_items (letter_id, user_id, source_type, recipient_type, status)
SELECT @letterId, j.user_id, 'DIRECT', j.recipient_type, 'UNREAD'
FROM OPENJSON(@items) WITH (user_id BIGINT '$.user', recipient_type VARCHAR(10) '$.type') j`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("letterId", letterID), sql.Named("items", string(raw))); err != nil {
		return fmt.Errorf("mengirim surat ke inbox: %w", err)
	}
	return nil
}

// DeliverIncoming memasukkan surat masuk eksternal ke inbox pengunggahnya (DIRECT, langsung
// DISPOSED: ia mencatat sekaligus mendisposisikan) lalu mendisposisikannya ke setiap penerima
// (FORWARD, UNREAD). Dipanggil alur surat di dalam transaksi pencatatan surat masuk.
func (r *Repository) DeliverIncoming(ctx context.Context, letterID, fromUserID int64, fromName, fromPosition string,
	recipients []int64, note *string) error {
	const source = `
INSERT INTO inbox_items (letter_id, user_id, source_type, recipient_type, status, read_at)
OUTPUT inserted.id
VALUES (@letterId, @userId, 'DIRECT', 'TO', 'DISPOSED', SYSDATETIMEOFFSET())`
	var itemID int64
	if err := r.q.QueryRowContext(ctx, source, sql.Named("letterId", letterID), sql.Named("userId", fromUserID)).Scan(&itemID); err != nil {
		return fmt.Errorf("mencatat surat masuk di inbox pengunggah: %w", err)
	}
	for _, id := range recipients {
		if err := r.AddForward(ctx, NewForward{
			LetterID: letterID, SourceItemID: itemID, Type: ForwardDisposition,
			FromUserID: fromUserID, FromName: fromName, FromPosition: fromPosition,
			TargetType: TargetUser, TargetID: id, RecipientID: id, Note: note,
		}); err != nil {
			return err
		}
	}
	return nil
}
