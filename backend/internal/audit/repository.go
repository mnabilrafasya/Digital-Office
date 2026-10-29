// Package audit menulis jejak audit (tabel audit_logs) untuk semua fitur.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"office.backend/internal/platform/db"
)

// Repository menulis jejak audit (audit_logs).
type Repository struct {
	q db.DBTX
}

// NewRepository membuat Repository di atas koneksi atau transaksi.
func NewRepository(q db.DBTX) *Repository {
	return &Repository{q: q}
}

// Record menyimpan satu baris audit. Dipanggil di transaksi yang sama dengan perubahan datanya.
func (r *Repository) Record(ctx context.Context, e Entry) error {
	snapshot, err := json.Marshal(e.Snapshot)
	if err != nil {
		return fmt.Errorf("menyusun snapshot audit: %w", err)
	}
	const q = `
INSERT INTO audit_logs (company_code, entity_type, entity_id, entity_uuid, action, actor_id, actor_name, snapshot, ip_address)
VALUES (@companyCode, @entityType, @entityId, @entityUuid, @action, @actorId, @actorName, @snapshot, @ip)`
	if _, err := r.q.ExecContext(ctx, q,
		sql.Named("companyCode", e.CompanyCode),
		sql.Named("entityType", e.EntityType),
		sql.Named("entityId", e.EntityID),
		sql.Named("entityUuid", db.NullString(e.EntityUUID)),
		sql.Named("action", e.Action),
		sql.Named("actorId", db.NullID(e.ActorID)),
		sql.Named("actorName", e.ActorName),
		sql.Named("snapshot", string(snapshot)),
		sql.Named("ip", db.NullString(e.IPAddress)),
	); err != nil {
		return fmt.Errorf("mencatat audit: %w", err)
	}
	return nil
}
