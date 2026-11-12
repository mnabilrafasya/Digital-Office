package delegasi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"office.backend/internal/platform/db"
)

// Repository adalah query tabel delegations.
type Repository struct {
	q db.DBTX
}

// NewRepository membuat Repository di atas koneksi atau transaksi.
func NewRepository(q db.DBTX) *Repository {
	return &Repository{q: q}
}

// delegationSelect memuat jadwal beserta nama dan jabatan pegawai, pengganti, dan pengisinya.
// @today (YYYY-MM-DD) dipakai untuk menghitung status.
const delegationSelect = `
SELECT dl.id, dl.company_code,
       u.id, u.name, up.name, g.id, g.name, gp.name,
       CONVERT(char(10), dl.start_date, 23), CONVERT(char(10), dl.end_date, 23), dl.note,
       CASE WHEN dl.end_date < @today THEN 'ENDED' WHEN dl.start_date > @today THEN 'UPCOMING' ELSE 'ACTIVE' END,
       c.id, c.name, cp.name, dl.created_at, dl.updated_at
FROM delegations dl
JOIN users u      ON u.id = dl.user_id
JOIN positions up ON up.id = u.position_id
JOIN users g      ON g.id = dl.delegate_id
JOIN positions gp ON gp.id = g.position_id
JOIN users c      ON c.id = dl.created_by
JOIN positions cp ON cp.id = c.position_id`

func scanDelegation(row interface{ Scan(...any) error }) (*Delegation, error) {
	var d Delegation
	err := row.Scan(&d.ID, &d.companyCode,
		&d.User.ID, &d.User.Name, &d.User.PositionName, &d.Delegate.ID, &d.Delegate.Name, &d.Delegate.PositionName,
		&d.From, &d.Until, &d.Note, &d.Status,
		&d.CreatedBy.ID, &d.CreatedBy.Name, &d.CreatedBy.PositionName, &d.CreatedAt, &d.UpdatedAt)
	return &d, err
}

// listFilter adalah kriteria daftar delegasi. Nilai nol berarti tidak disaring.
type listFilter struct {
	CompanyCode string
	Today       string
	OwnerID     int64    // jadwal milik user ini
	DelegateID  int64    // jadwal yang diwakili user ini
	PersonID    int64    // jadwal milik atau diwakili user ini
	Statuses    []string // ACTIVE, UPCOMING, ENDED; kosong = semua
	Offset      int
	Limit       int // 0 = semua
}

// List mengambil jadwal delegasi (yang sedang/akan berlaku lebih dulu, lalu yang terbaru) beserta
// jumlah totalnya.
func (r *Repository) List(ctx context.Context, f listFilter) ([]Delegation, int, error) {
	var sb strings.Builder
	sb.WriteString("\nWHERE dl.company_code = @companyCode AND dl.deleted_at IS NULL")
	args := []any{sql.Named("companyCode", f.CompanyCode), sql.Named("today", f.Today)}
	if f.OwnerID != 0 {
		sb.WriteString("\n  AND dl.user_id = @ownerId")
		args = append(args, sql.Named("ownerId", f.OwnerID))
	}
	if f.DelegateID != 0 {
		sb.WriteString("\n  AND dl.delegate_id = @delegateId")
		args = append(args, sql.Named("delegateId", f.DelegateID))
	}
	if f.PersonID != 0 {
		sb.WriteString("\n  AND (dl.user_id = @personId OR dl.delegate_id = @personId)")
		args = append(args, sql.Named("personId", f.PersonID))
	}
	if len(f.Statuses) > 0 {
		statuses, _ := json.Marshal(f.Statuses) // []string selalu berhasil di-marshal
		sb.WriteString(`
  AND CASE WHEN dl.end_date < @today THEN 'ENDED' WHEN dl.start_date > @today THEN 'UPCOMING' ELSE 'ACTIVE' END
      IN (SELECT [value] FROM OPENJSON(@statuses))`)
		args = append(args, sql.Named("statuses", string(statuses)))
	}
	q := strings.Replace(delegationSelect, "SELECT dl.id,", "SELECT COUNT(*) OVER (), dl.id,", 1) + sb.String() + `
ORDER BY CASE WHEN dl.end_date < @today THEN 1 ELSE 0 END, dl.start_date, dl.id`
	if f.Limit > 0 {
		q += "\nOFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY"
		args = append(args, sql.Named("offset", f.Offset), sql.Named("limit", f.Limit))
	}

	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar delegasi: %w", err)
	}
	defer rows.Close()
	out := []Delegation{}
	total := 0
	for rows.Next() {
		var d Delegation
		if err := rows.Scan(&total, &d.ID, &d.companyCode,
			&d.User.ID, &d.User.Name, &d.User.PositionName, &d.Delegate.ID, &d.Delegate.Name, &d.Delegate.PositionName,
			&d.From, &d.Until, &d.Note, &d.Status,
			&d.CreatedBy.ID, &d.CreatedBy.Name, &d.CreatedBy.PositionName, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("membaca daftar delegasi: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca daftar delegasi: %w", err)
	}
	return out, total, nil
}

// Find mengambil satu jadwal milik perusahaan (db.ErrNotFound bila tidak ada). lock mengunci
// barisnya sampai transaksi selesai.
func (r *Repository) Find(ctx context.Context, companyCode string, id int64, today string, lock bool) (*Delegation, error) {
	q := delegationSelect
	if lock {
		q = strings.Replace(q, "FROM delegations dl", "FROM delegations dl WITH (UPDLOCK, ROWLOCK)", 1)
	}
	q += "\nWHERE dl.id = @id AND dl.company_code = @companyCode AND dl.deleted_at IS NULL"
	d, err := scanDelegation(r.q.QueryRowContext(ctx, q,
		sql.Named("id", id), sql.Named("companyCode", companyCode), sql.Named("today", today)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("memuat delegasi: %w", err)
	}
	return d, nil
}

// LockCompany mengunci pengisian delegasi satu perusahaan sampai transaksi selesai, supaya dua
// pengisian bersamaan tidak lolos pemeriksaan tumpang-tindih/berantai. Wajib di dalam transaksi.
func (r *Repository) LockCompany(ctx context.Context, companyCode string) error {
	const q = `
DECLARE @result INT;
EXEC @result = sp_getapplock @Resource = @resource, @LockMode = 'Exclusive', @LockOwner = 'Transaction', @LockTimeout = 15000;
SELECT @result`
	var result int
	if err := r.q.QueryRowContext(ctx, q, sql.Named("resource", "delegations:"+companyCode)).Scan(&result); err != nil {
		return fmt.Errorf("mengunci delegasi perusahaan: %w", err)
	}
	if result < 0 {
		return fmt.Errorf("mengunci delegasi perusahaan: sp_getapplock %d", result)
	}
	return nil
}

// Conflict adalah jadwal lain yang bentrok dengan jadwal yang akan disimpan.
type Conflict struct {
	Kind       string // OWN (pegawai sudah punya jadwal), DELEGATE_AWAY (pengganti sedang mendelegasikan), OWNER_SUBSTITUTE (pegawai sedang menjadi pengganti)
	OwnerName  string
	TargetName string
	From       string
	Until      string
}

// FindConflict mencari satu jadwal yang beririsan dengan from..until dan melanggar aturan:
// pegawai sudah punya jadwal di periode itu, pengganti sedang mendelegasikan (berantai), atau
// pegawai sedang menjadi pengganti orang lain (berantai). exceptID dilewati (jadwal yang diubah).
func (r *Repository) FindConflict(ctx context.Context, companyCode string, p period, exceptID int64) (*Conflict, error) {
	const q = `
SELECT TOP 1 CASE WHEN dl.user_id = @userId THEN 'OWN'
                  WHEN dl.user_id = @delegateId THEN 'DELEGATE_AWAY'
                  ELSE 'OWNER_SUBSTITUTE' END,
       u.name, g.name, CONVERT(char(10), dl.start_date, 23), CONVERT(char(10), dl.end_date, 23)
FROM delegations dl
JOIN users u ON u.id = dl.user_id
JOIN users g ON g.id = dl.delegate_id
WHERE dl.company_code = @companyCode AND dl.deleted_at IS NULL AND dl.id <> @exceptId
  AND dl.start_date <= @until AND dl.end_date >= @from
  AND (dl.user_id = @userId OR dl.user_id = @delegateId OR dl.delegate_id = @userId)
ORDER BY CASE WHEN dl.user_id = @userId THEN 0 WHEN dl.user_id = @delegateId THEN 1 ELSE 2 END, dl.start_date`
	var c Conflict
	err := r.q.QueryRowContext(ctx, q,
		sql.Named("companyCode", companyCode), sql.Named("exceptId", exceptID),
		sql.Named("userId", p.UserID), sql.Named("delegateId", p.DelegateID),
		sql.Named("from", p.From), sql.Named("until", p.Until),
	).Scan(&c.Kind, &c.OwnerName, &c.TargetName, &c.From, &c.Until)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("memeriksa jadwal delegasi: %w", err)
	}
	return &c, nil
}

// ActiveUser mengambil user aktif perusahaan (user, jabatan, unit, dan role aktif), atau
// db.ErrNotFound.
func (r *Repository) ActiveUser(ctx context.Context, companyCode string, id int64) (*PersonRef, error) {
	const q = `
SELECT u.id, u.name, p.name
FROM users u
JOIN positions p ON p.id = u.position_id AND p.is_active = 1 AND p.deleted_at IS NULL
JOIN units un    ON un.id = p.unit_id   AND un.is_active = 1 AND un.deleted_at IS NULL
JOIN roles r     ON r.id = u.role_id    AND r.is_active = 1 AND r.deleted_at IS NULL
WHERE u.id = @id AND un.company_code = @companyCode AND u.is_active = 1 AND u.deleted_at IS NULL`
	var p PersonRef
	err := r.q.QueryRowContext(ctx, q, sql.Named("id", id), sql.Named("companyCode", companyCode)).
		Scan(&p.ID, &p.Name, &p.PositionName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("memeriksa user: %w", err)
	}
	return &p, nil
}

// Insert menyimpan jadwal baru dan mengembalikan ID-nya.
func (r *Repository) Insert(ctx context.Context, companyCode string, p period, createdBy int64) (int64, error) {
	const q = `
INSERT INTO delegations (company_code, user_id, delegate_id, start_date, end_date, note, created_by)
OUTPUT inserted.id
VALUES (@companyCode, @userId, @delegateId, @from, @until, @note, @createdBy)`
	var id int64
	err := r.q.QueryRowContext(ctx, q,
		sql.Named("companyCode", companyCode), sql.Named("userId", p.UserID), sql.Named("delegateId", p.DelegateID),
		sql.Named("from", p.From), sql.Named("until", p.Until), sql.Named("note", p.Note), sql.Named("createdBy", createdBy),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("menyimpan delegasi: %w", err)
	}
	return id, nil
}

// Update mengganti pengganti, periode, dan keterangan jadwal (pegawainya tetap).
func (r *Repository) Update(ctx context.Context, id int64, p period) error {
	const q = `
UPDATE delegations
SET delegate_id = @delegateId, start_date = @from, end_date = @until, note = @note, updated_at = SYSDATETIMEOFFSET()
WHERE id = @id AND deleted_at IS NULL`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", id), sql.Named("delegateId", p.DelegateID),
		sql.Named("from", p.From), sql.Named("until", p.Until), sql.Named("note", p.Note)); err != nil {
		return fmt.Errorf("mengubah delegasi: %w", err)
	}
	return nil
}

// Delete menghapus (soft-delete) jadwal. Riwayatnya tetap ada di audit log.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	const q = `UPDATE delegations SET deleted_at = SYSDATETIMEOFFSET(), updated_at = SYSDATETIMEOFFSET() WHERE id = @id AND deleted_at IS NULL`
	if _, err := r.q.ExecContext(ctx, q, sql.Named("id", id)); err != nil {
		return fmt.Errorf("menghapus delegasi: %w", err)
	}
	return nil
}
