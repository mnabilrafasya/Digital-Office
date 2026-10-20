package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// AuditRepository berisi query tabel audit_logs.
type AuditRepository struct {
	q db.DBTX
}

// NewAuditRepository membuat AuditRepository di atas koneksi atau transaksi.
func NewAuditRepository(q db.DBTX) *AuditRepository {
	return &AuditRepository{q: q}
}

// Insert menyimpan satu jejak audit.
func (r *AuditRepository) Insert(ctx context.Context, e model.AuditEntry) error {
	const q = `
INSERT INTO audit_logs (company_code, entity_type, entity_id, entity_uuid, action,
                        actor_id, actor_name, snapshot, ip_address)
VALUES (@companyCode, @entityType, @entityId, @entityUuid, @action,
        @actorId, @actorName, @snapshot, NULLIF(@ipAddress, ''))`

	_, err := r.q.ExecContext(ctx, q,
		sql.Named("companyCode", e.CompanyCode), sql.Named("entityType", e.EntityType),
		sql.Named("entityId", e.EntityID), sql.Named("entityUuid", e.EntityUUID),
		sql.Named("action", e.Action), sql.Named("actorId", e.ActorID),
		sql.Named("actorName", e.ActorName), sql.Named("snapshot", e.Snapshot),
		sql.Named("ipAddress", e.IPAddress))
	if err != nil {
		return fmt.Errorf("menyimpan audit log: %w", err)
	}
	return nil
}
