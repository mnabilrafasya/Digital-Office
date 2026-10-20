package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
)

// AuditRepository menulis jejak audit (audit_logs).
type AuditRepository struct {
	q db.DBTX
}

// NewAuditRepository membuat AuditRepository di atas koneksi atau transaksi.
func NewAuditRepository(q db.DBTX) *AuditRepository {
	return &AuditRepository{q: q}
}

// AuditEntry adalah satu kejadian yang dicatat. Snapshot di-encode menjadi JSON kecil.
type AuditEntry struct {
	CompanyCode string
	EntityType  string
	EntityID    int64
	EntityUUID  string
	Action      string
	ActorID     int64
	ActorName   string // disalin, supaya jejak tetap terbaca walau nama user berubah
	IPAddress   string
	Snapshot    any
}

// Record menyimpan satu baris audit. Dipanggil di transaksi yang sama dengan perubahan datanya.
func (r *AuditRepository) Record(ctx context.Context, e AuditEntry) error {
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
		sql.Named("entityUuid", nullString(e.EntityUUID)),
		sql.Named("action", e.Action),
		sql.Named("actorId", nullID(e.ActorID)),
		sql.Named("actorName", e.ActorName),
		sql.Named("snapshot", string(snapshot)),
		sql.Named("ip", nullString(e.IPAddress)),
	); err != nil {
		return fmt.Errorf("mencatat audit: %w", err)
	}
	return nil
}

// nullString mengubah "" menjadi NULL untuk parameter SQL.
func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
