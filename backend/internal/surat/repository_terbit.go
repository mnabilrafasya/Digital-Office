package surat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// NextNumber mengambil nomor urut berikutnya untuk unit pengirim surat pada tahun tertentu,
// beserta kode unit dan format nomor perusahaan. Rentang nomor unit-tahun dikunci (UPDLOCK,
// HOLDLOCK) sampai transaksi selesai, jadi dua surat yang terbit bersamaan tidak mendapat nomor
// yang sama. Wajib dipanggil di dalam transaksi.
func (r *Repository) NextNumber(ctx context.Context, letterID int64, year int) (int, string, string, error) {
	const q = `
SELECT un.code, ISNULL(c.letter_number_format, ''),
       ISNULL((SELECT MAX(x.number_seq) FROM letters x WITH (UPDLOCK, HOLDLOCK)
               WHERE x.sender_unit_id = l.sender_unit_id AND x.number_year = @year
                 AND x.number_seq IS NOT NULL), 0) + 1
FROM letters l
JOIN units un     ON un.id = l.sender_unit_id
JOIN companies c  ON c.code = l.company_code
WHERE l.id = @id`
	var (
		seq              int
		unitCode, format string
	)
	if err := r.q.QueryRowContext(ctx, q, sql.Named("id", letterID), sql.Named("year", year)).Scan(&unitCode, &format, &seq); err != nil {
		return 0, "", "", fmt.Errorf("mengambil nomor urut surat: %w", err)
	}
	return seq, unitCode, format, nil
}

// unitHead adalah pasangan unit penerima dan pimpinannya, dikirim sebagai JSON ke OPENJSON.
type unitHead struct {
	UnitID int64 `json:"unit"`
	HeadID int64 `json:"head"`
}

// Publish menulis nomor, tanggal, dan penandatangan surat, lalu mencatat pimpinan yang menerima
// tiap unit penerima. Status tidak diubah di sini (itu tugas ApplyTransition).
func (r *Repository) Publish(ctx context.Context, letterID int64, p Publication) error {
	const update = `
UPDATE letters
SET letter_number = @number, number_seq = @seq, number_year = @year,
    approved_at = @approvedAt, signed_by_id = @signerId, updated_at = SYSDATETIMEOFFSET()
WHERE id = @id AND status = 'IN_APPROVAL' AND letter_number IS NULL AND deleted_at IS NULL`
	res, err := r.q.ExecContext(ctx, update,
		sql.Named("id", letterID),
		sql.Named("number", p.Number),
		sql.Named("seq", p.Seq),
		sql.Named("year", p.Year),
		sql.Named("approvedAt", p.ApprovedAt),
		sql.Named("signerId", p.SignerID),
	)
	if err != nil {
		return fmt.Errorf("memberi nomor surat: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("memberi nomor surat: %w", err)
	}
	if n != 1 { // pengaman tambahan selain kunci baris; seharusnya tidak terjadi
		return errors.New("memberi nomor surat: surat tidak lagi menunggu persetujuan atau sudah bernomor")
	}

	if len(p.UnitHeads) == 0 {
		return nil
	}
	heads := make([]unitHead, 0, len(p.UnitHeads))
	for unitID, headID := range p.UnitHeads {
		heads = append(heads, unitHead{UnitID: unitID, HeadID: headID})
	}
	raw, _ := json.Marshal(heads) // struct angka selalu berhasil di-marshal
	const setHeads = `
UPDATE r
SET resolved_head_id = h.head_id, updated_at = SYSDATETIMEOFFSET()
FROM letter_recipients r
JOIN OPENJSON(@heads) WITH (unit_id BIGINT '$.unit', head_id BIGINT '$.head') h ON h.unit_id = r.recipient_unit_id
WHERE r.letter_id = @id AND r.deleted_at IS NULL`
	if _, err := r.q.ExecContext(ctx, setHeads, sql.Named("id", letterID), sql.Named("heads", string(raw))); err != nil {
		return fmt.Errorf("mencatat pimpinan penerima: %w", err)
	}
	return nil
}
