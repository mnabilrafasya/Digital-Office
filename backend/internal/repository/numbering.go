package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
)

// NumberingRepository berisi query penomoran surat.
type NumberingRepository struct {
	q db.DBTX
}

// NewNumberingRepository membuat NumberingRepository di atas koneksi atau transaksi.
func NewNumberingRepository(q db.DBTX) *NumberingRepository {
	return &NumberingRepository{q: q}
}

// LetterNumberFormat mengambil format nomor surat perusahaan ("" bila belum diatur).
func (r *NumberingRepository) LetterNumberFormat(ctx context.Context, companyCode string) (string, error) {
	const q = `SELECT letter_number_format FROM companies WHERE code = @companyCode AND deleted_at IS NULL`
	var format sql.NullString
	err := r.q.QueryRowContext(ctx, q, sql.Named("companyCode", companyCode)).Scan(&format)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("mengambil format nomor surat: %w", err)
	}
	return format.String, nil
}

// NextSequence mengambil nomor urut berikutnya per unit pengirim per tahun. Rentang baris dikunci
// (UPDLOCK, HOLDLOCK) sampai transaksi selesai, sehingga dua penandatanganan bersamaan di unit yang
// sama tidak mendapat nomor yang sama. Wajib dipanggil setelah baris letters dikunci.
func (r *NumberingRepository) NextSequence(ctx context.Context, unitID int64, year int) (int, error) {
	const q = `
SELECT ISNULL(MAX(number_seq), 0) + 1
FROM letters WITH (UPDLOCK, HOLDLOCK)
WHERE sender_unit_id = @unitId AND number_year = @year`
	var n int
	if err := r.q.QueryRowContext(ctx, q, sql.Named("unitId", unitID), sql.Named("year", year)).Scan(&n); err != nil {
		return 0, fmt.Errorf("mengambil nomor urut surat: %w", err)
	}
	return n, nil
}
