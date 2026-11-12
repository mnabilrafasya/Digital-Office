package surat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// delegableStatusesSQL adalah status yang tugasnya dikerjakan reviewer/approver (lihat delegable).
const delegableStatusesSQL = `'IN_REVIEW', 'RETURNED_TO_REVIEWER', 'IN_APPROVAL', 'APPROVED'`

// activeDelegationSQL adalah syarat delegasi dl yang berlaku pada @today dengan pengganti g yang
// masih aktif (pengganti yang dinonaktifkan membuat tugas kembali ke pemiliknya).
const activeDelegationSQL = `dl.deleted_at IS NULL AND @today BETWEEN dl.start_date AND dl.end_date
      AND g.is_active = 1 AND g.deleted_at IS NULL`

// ActiveDelegation mengambil delegasi userID yang berlaku pada day (YYYY-MM-DD), atau nil.
func (r *Repository) ActiveDelegation(ctx context.Context, companyCode string, userID int64, day string) (*Delegation, error) {
	const q = `
SELECT TOP 1 g.id, g.name, gp.name, CONVERT(char(10), dl.end_date, 23)
FROM delegations dl
JOIN users g      ON g.id = dl.delegate_id
JOIN positions gp ON gp.id = g.position_id
WHERE dl.user_id = @userId AND dl.company_code = @companyCode AND ` + activeDelegationSQL
	d := Delegation{UserID: userID}
	err := r.q.QueryRowContext(ctx, q, sql.Named("userId", userID), sql.Named("companyCode", companyCode),
		sql.Named("today", day)).Scan(&d.Delegate.ID, &d.Delegate.Name, &d.Delegate.PositionName, &d.Until)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("memeriksa delegasi: %w", err)
	}
	return &d, nil
}

// HasActed mengecek apakah userID pernah melakukan aksi alur atau menyimpan naskah pada surat.
func (r *Repository) HasActed(ctx context.Context, letterID, userID int64) (bool, error) {
	const q = `
SELECT CASE WHEN EXISTS (SELECT 1 FROM letter_actions
                         WHERE letter_id = @letterId AND actor_id = @userId AND action <> 'READ') THEN 1 ELSE 0 END`
	var acted bool
	if err := r.q.QueryRowContext(ctx, q, sql.Named("letterId", letterID), sql.Named("userId", userID)).Scan(&acted); err != nil {
		return false, fmt.Errorf("memeriksa riwayat pelaku: %w", err)
	}
	return acted, nil
}
