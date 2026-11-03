// Package audit berisi jejak audit dan pelacakan surat: menulis audit_logs untuk semua fitur
// (repository.go), melacak posisi dan perjalanan satu surat (lacak.go), menampilkan audit log
// untuk pemegang AUDIT_VIEW (log.go), dan memantau surat yang sedang diproses (pantauan.go).
// Pembacaan memakai query sendiri atas tabel surat dan inbox, sehingga package ini tidak
// bergantung pada package surat atau inbox.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	chimw "github.com/go-chi/chi/v5/middleware"

	"office.backend/internal/platform/db"
	"office.backend/internal/platform/middleware"
)

// Repository menulis dan membaca jejak audit.
type Repository struct {
	q db.DBTX
}

// NewRepository membuat Repository di atas koneksi atau transaksi.
func NewRepository(q db.DBTX) *Repository {
	return &Repository{q: q}
}

// Record menyimpan satu baris audit. Dipanggil di transaksi yang sama dengan perubahan datanya.
// IP dan request ID diambil dari request yang sedang berjalan bila tidak diisi pemanggil.
// Baris audit tidak bisa diubah atau dihapus (dijaga trigger trg_audit_logs_immutable).
func (r *Repository) Record(ctx context.Context, e Entry) error {
	var snapshot sql.NullString // tanpa snapshot = NULL (CHECK ISJSON menolak teks "null")
	if e.Snapshot != nil {
		raw, err := json.Marshal(e.Snapshot)
		if err != nil {
			return fmt.Errorf("menyusun snapshot audit: %w", err)
		}
		snapshot = sql.NullString{String: string(raw), Valid: true}
	}
	if e.IPAddress == "" {
		e.IPAddress = middleware.ClientIPFrom(ctx)
	}
	if e.ActorName == "" {
		e.ActorName = SystemActor
	}
	const q = `
INSERT INTO audit_logs (company_code, entity_type, entity_id, entity_uuid, action, actor_id, actor_name,
                        snapshot, ip_address, request_id)
VALUES (@companyCode, @entityType, @entityId, @entityUuid, @action, @actorId, @actorName, @snapshot, @ip, @requestId)`
	if _, err := r.q.ExecContext(ctx, q,
		sql.Named("companyCode", e.CompanyCode),
		sql.Named("entityType", e.EntityType),
		sql.Named("entityId", e.EntityID),
		sql.Named("entityUuid", db.NullString(e.EntityUUID)),
		sql.Named("action", e.Action),
		sql.Named("actorId", db.NullID(e.ActorID)),
		sql.Named("actorName", e.ActorName),
		sql.Named("snapshot", snapshot),
		sql.Named("ip", db.NullString(e.IPAddress)),
		sql.Named("requestId", db.NullString(chimw.GetReqID(ctx))),
	); err != nil {
		return fmt.Errorf("mencatat audit: %w", err)
	}
	return nil
}
