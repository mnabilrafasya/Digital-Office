package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"office.backend/internal/platform/db"
)

// maskedTitleSQL bernilai 1 bila perihal surat l harus disamarkan untuk @viewerId: surat Terbatas
// atau Rahasia, dan pembaca bukan drafter maupun peserta surat itu.
const maskedTitleSQL = `CASE WHEN cf.code IN ('TERBATAS', 'RAHASIA') AND l.drafter_id <> @viewerId
     AND NOT EXISTS (SELECT 1 FROM letter_participants mp
                     WHERE mp.letter_id = l.id AND mp.user_id = @viewerId AND mp.deleted_at IS NULL)
     THEN 1 ELSE 0 END`

// LogFilter adalah kriteria audit log. Nilai nol berarti tidak disaring.
type LogFilter struct {
	CompanyCode string
	ViewerID    int64 // untuk menyamarkan perihal surat Terbatas/Rahasia
	From, To    time.Time
	ActorID     int64
	Actions     []string
	EntityType  string
	Search      string // dicari di nomor surat, dan perihal yang tidak disamarkan
	LetterUUID  string
	Offset      int
	Limit       int // 0 = tanpa batas (ekspor)
}

func (f LogFilter) where() (string, []any) {
	var sb strings.Builder
	sb.WriteString("\nWHERE a.company_code = @companyCode")
	args := []any{sql.Named("companyCode", f.CompanyCode), sql.Named("viewerId", f.ViewerID)}
	if !f.From.IsZero() {
		sb.WriteString("\n  AND a.created_at >= @from")
		args = append(args, sql.Named("from", f.From))
	}
	if !f.To.IsZero() {
		sb.WriteString("\n  AND a.created_at < @to")
		args = append(args, sql.Named("to", f.To))
	}
	if f.ActorID != 0 {
		sb.WriteString("\n  AND a.actor_id = @actorId")
		args = append(args, sql.Named("actorId", f.ActorID))
	}
	if len(f.Actions) > 0 {
		actions, _ := json.Marshal(f.Actions) // []string selalu berhasil di-marshal
		sb.WriteString("\n  AND a.action IN (SELECT [value] FROM OPENJSON(@actions))")
		args = append(args, sql.Named("actions", string(actions)))
	}
	if f.EntityType != "" {
		sb.WriteString("\n  AND a.entity_type = @entityType")
		args = append(args, sql.Named("entityType", f.EntityType))
	}
	if f.LetterUUID != "" {
		sb.WriteString("\n  AND a.entity_type = 'LETTER' AND a.entity_uuid = @letterUuid")
		args = append(args, sql.Named("letterUuid", f.LetterUUID))
	}
	if f.Search != "" {
		// Perihal yang disamarkan tidak ikut dicari, supaya isinya tidak bisa ditebak lewat pencarian
		sb.WriteString("\n  AND (l.letter_number LIKE @pattern ESCAPE '\\' OR (l.title LIKE @pattern ESCAPE '\\' AND " +
			maskedTitleSQL + " = 0))")
		args = append(args, sql.Named("pattern", db.ContainsPattern(f.Search)))
	}
	return sb.String(), args
}

const logColumns = `
SELECT a.id, a.created_at, a.action, a.entity_type, CONVERT(char(36), a.entity_uuid),
       a.actor_id, a.actor_name, a.ip_address, a.request_id, a.snapshot,
       l.id, l.letter_number, l.title, ` + maskedTitleSQL + `,
       COUNT(*) OVER ()
FROM audit_logs a
LEFT JOIN letters l    ON a.entity_type = 'LETTER' AND l.id = a.entity_id
LEFT JOIN lookups cf   ON cf.id = l.confidentiality_level_id`

// ListLogs mengambil audit log terbaru lebih dulu beserta jumlah totalnya. Limit 0 mengambil
// semua baris yang cocok (untuk ekspor).
func (r *Repository) ListLogs(ctx context.Context, f LogFilter) ([]LogEntry, int, error) {
	where, args := f.where()
	q := logColumns + where + "\nORDER BY a.created_at DESC, a.id DESC"
	if f.Limit > 0 {
		q += "\nOFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY"
		args = append(args, sql.Named("offset", f.Offset), sql.Named("limit", f.Limit))
	}
	rows, err := r.q.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil audit log: %w", err)
	}
	defer rows.Close()

	out := []LogEntry{}
	total := 0
	for rows.Next() {
		e, n, err := scanLog(rows)
		if err != nil {
			return nil, 0, err
		}
		total = n
		out = append(out, *e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca audit log: %w", err)
	}

	// Halaman di luar jangkauan tidak membawa COUNT(*) OVER (); hitung terpisah supaya total tetap benar
	if len(out) == 0 && f.Offset > 0 {
		where, args := f.where()
		q := `SELECT COUNT(*) FROM audit_logs a
LEFT JOIN letters l  ON a.entity_type = 'LETTER' AND l.id = a.entity_id
LEFT JOIN lookups cf ON cf.id = l.confidentiality_level_id` + where
		if err := r.q.QueryRowContext(ctx, q, args...).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("menghitung audit log: %w", err)
		}
	}
	return out, total, nil
}

// FindLog mengambil satu baris audit log milik companyCode.
func (r *Repository) FindLog(ctx context.Context, companyCode string, viewerID, id int64) (*LogEntry, error) {
	f := LogFilter{CompanyCode: companyCode, ViewerID: viewerID}
	where, args := f.where()
	row := r.q.QueryRowContext(ctx, logColumns+where+"\n  AND a.id = @id", append(args, sql.Named("id", id))...)
	e, _, err := scanLog(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	return e, err
}

func scanLog(row interface{ Scan(...any) error }) (*LogEntry, int, error) {
	var (
		e                       LogEntry
		entityUUID, ip, reqID   sql.NullString
		snapshot, number, title sql.NullString
		actorID, letterID       sql.NullInt64
		masked, total           int
	)
	err := row.Scan(&e.ID, &e.At, &e.Action, &e.EntityType, &entityUUID,
		&actorID, &e.Actor.Name, &ip, &reqID, &snapshot,
		&letterID, &number, &title, &masked, &total)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, err
	}
	if err != nil {
		return nil, 0, fmt.Errorf("membaca audit log: %w", err)
	}
	if entityUUID.Valid {
		v := strings.ToLower(entityUUID.String)
		e.EntityID = &v
	}
	if actorID.Valid {
		e.Actor.ID = &actorID.Int64
	}
	e.IPAddress, e.RequestID = nullable(ip), nullable(reqID)
	if snapshot.Valid {
		e.Snapshot = json.RawMessage(snapshot.String)
	}
	if letterID.Valid {
		e.Letter = &LogLetter{Number: nullable(number), Title: nullable(title), Masked: masked == 1}
		if e.Letter.Masked {
			e.Letter.Title = nil
		}
	}
	return &e, total, nil
}

func nullable(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}
