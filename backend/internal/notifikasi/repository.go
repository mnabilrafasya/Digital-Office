package notifikasi

import (
	"context"
	"database/sql"
	"fmt"

	"office.backend/internal/platform/db"
)

// Repository membaca data untuk pesan dari tabel yang sudah ada (letters, users, units, lookups).
type Repository struct {
	q db.DBTX
}

// NewRepository membuat Repository.
func NewRepository(q db.DBTX) *Repository {
	return &Repository{q: q}
}

// letterInfo membaca nomor, perihal, sifat, unit pengirim, dan nama drafter surat.
func (r *Repository) letterInfo(ctx context.Context, letterID int64) (letterInfo, error) {
	const q = `
SELECT l.letter_number, l.title, ISNULL(cf.code, ''), COALESCE(l.sender_unit_name, su.name), d.name
FROM letters l
JOIN units su        ON su.id = l.sender_unit_id
JOIN users d         ON d.id = l.drafter_id
LEFT JOIN lookups cf ON cf.id = l.confidentiality_level_id
WHERE l.id = @id`
	var l letterInfo
	err := r.q.QueryRowContext(ctx, q, sql.Named("id", letterID)).
		Scan(&l.Number, &l.Title, &l.Confidentiality, &l.SenderUnit, &l.DrafterName)
	if err != nil {
		return l, fmt.Errorf("membaca surat untuk pemberitahuan: %w", err)
	}
	return l, nil
}

// employeeNumbers mengembalikan NIK user aktif di antara ids (urut sesuai ids).
func (r *Repository) employeeNumbers(ctx context.Context, ids []int64) ([]string, error) {
	const q = `
SELECT u.id, u.employee_number
FROM users u
WHERE u.id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids)) AND u.is_active = 1 AND u.deleted_at IS NULL`
	rows, err := r.q.QueryContext(ctx, q, sql.Named("ids", db.JSONIDs(ids)))
	if err != nil {
		return nil, fmt.Errorf("membaca NIK penerima: %w", err)
	}
	defer rows.Close()
	nik := map[int64]string{}
	for rows.Next() {
		var id int64
		var n string
		if err := rows.Scan(&id, &n); err != nil {
			return nil, fmt.Errorf("membaca NIK penerima: %w", err)
		}
		nik[id] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var out []string
	for _, id := range ids {
		if n, ok := nik[id]; ok {
			out = append(out, n)
		}
	}
	return out, nil
}

// activeEmployee mengecek NIK pegawai aktif milik companyCode (untuk pesan uji).
func (r *Repository) activeEmployee(ctx context.Context, companyCode, employeeNumber string) (bool, error) {
	const q = `
SELECT CASE WHEN EXISTS (
    SELECT 1 FROM users u
    JOIN positions p ON p.id = u.position_id
    JOIN units un    ON un.id = p.unit_id
    WHERE u.employee_number = @nik AND un.company_code = @companyCode AND u.is_active = 1 AND u.deleted_at IS NULL
) THEN 1 ELSE 0 END`
	var ok bool
	if err := r.q.QueryRowContext(ctx, q, sql.Named("nik", employeeNumber), sql.Named("companyCode", companyCode)).Scan(&ok); err != nil {
		return false, fmt.Errorf("memeriksa pegawai: %w", err)
	}
	return ok, nil
}
