package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

// Pantauan surat untuk pemegang AUDIT_VIEW: surat yang sedang diproses, siapa yang memegangnya,
// dan sudah berapa lama. Yang paling lama menunggu tampil lebih dulu.

// MonitorQuery adalah parameter pantauan apa adanya dari query string. Page dan PageSize sudah
// divalidasi handler.
type MonitorQuery struct {
	Status          string // opsional, beberapa dipisah koma; default semua status yang sedang diproses
	MinWaitingHours string // opsional; contoh 48 untuk surat yang tertahan
	Page            int
	PageSize        int
}

// Monitor mengambil satu halaman surat yang sedang diproses beserta jumlah totalnya.
func (s *Service) Monitor(ctx context.Context, user *auth.CurrentUser, mq MonitorQuery) ([]MonitoredLetter, int, error) {
	statuses := inProcessStatuses()
	if mq.Status != "" {
		statuses = nil
		for part := range strings.SplitSeq(mq.Status, ",") {
			st := strings.ToUpper(strings.TrimSpace(part))
			if st == "" || slices.Contains(statuses, st) {
				continue
			}
			if !slices.Contains(inProcessStatuses(), st) {
				return nil, 0, apperr.Invalid("Parameter status wajib salah satu dari: " + strings.Join(inProcessStatuses(), ", "))
			}
			statuses = append(statuses, st)
		}
	}
	minHours := 0
	if mq.MinWaitingHours != "" {
		n, err := strconv.Atoi(mq.MinWaitingHours)
		if err != nil || n < 0 || n > 24*365 {
			return nil, 0, apperr.Invalid("Parameter minWaitingHours harus angka 0 sampai 8760")
		}
		minHours = n
	}

	now := s.now()
	items, total, err := s.repo.monitor(ctx, monitorFilter{
		CompanyCode: user.Company.Code, ViewerID: user.ID, Statuses: statuses,
		Cutoff: now.Add(-time.Duration(minHours) * time.Hour),
		Offset: (mq.Page - 1) * mq.PageSize, Limit: mq.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		it := &items[i]
		it.WaitingHours = waitingHours(it.Since, now)
		it.Stuck = it.WaitingHours >= StuckAfterHours
		it.Since = it.Since.In(s.loc)
	}
	return items, total, nil
}

type monitorFilter struct {
	CompanyCode string
	ViewerID    int64
	Statuses    []string
	Cutoff      time.Time // hanya surat yang berpindah terakhir sebelum waktu ini
	Offset      int
	Limit       int
}

// monitor mengambil surat yang sedang diproses, urut dari yang paling lama menunggu.
func (r *Repository) monitor(ctx context.Context, f monitorFilter) ([]MonitoredLetter, int, error) {
	statuses, _ := json.Marshal(f.Statuses) // []string selalu berhasil di-marshal
	args := []any{sql.Named("companyCode", f.CompanyCode), sql.Named("viewerId", f.ViewerID),
		sql.Named("statuses", string(statuses)), sql.Named("cutoff", f.Cutoff)}
	cte := `
WITH m AS (
    SELECT CONVERT(char(36), l.uuid) AS uuid, l.title, l.status, COALESCE(l.sender_unit_name, su.name) AS unit_name,
           d.id AS drafter_id, d.name AS drafter_name, dp.name AS drafter_pos,
           a.id AS holder_id, a.name AS holder_name, ap.name AS holder_pos,
           ` + holderStepSQL + ` AS step,
           ` + lastMoveSQL + ` AS since,
           ` + maskedTitleSQL + ` AS masked
    FROM letters l
    JOIN units su          ON su.id = l.sender_unit_id
    JOIN users d           ON d.id = l.drafter_id
    JOIN positions dp      ON dp.id = d.position_id
    LEFT JOIN lookups cf   ON cf.id = l.confidentiality_level_id
    LEFT JOIN users a      ON a.id = l.current_assignee_id
    LEFT JOIN positions ap ON ap.id = a.position_id
    WHERE l.company_code = @companyCode AND l.deleted_at IS NULL
      AND l.status IN (SELECT [value] FROM OPENJSON(@statuses))
)`
	q := cte + `
SELECT uuid, title, status, unit_name, drafter_id, drafter_name, drafter_pos,
       holder_id, holder_name, holder_pos, step, since, masked, COUNT(*) OVER ()
FROM m
WHERE since <= @cutoff
ORDER BY since ASC, uuid
OFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY`
	rows, err := r.q.QueryContext(ctx, q, append(args, sql.Named("offset", f.Offset), sql.Named("limit", f.Limit))...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil pantauan surat: %w", err)
	}
	defer rows.Close()

	out := []MonitoredLetter{}
	total := 0
	for rows.Next() {
		var (
			m                     MonitoredLetter
			holderID              sql.NullInt64
			holderName, holderPos sql.NullString
			step                  sql.NullInt32
			masked                int
		)
		if err := rows.Scan(&m.LetterID, &m.Title, &m.Status, &m.SenderUnit,
			&m.Drafter.ID, &m.Drafter.Name, &m.Drafter.PositionName,
			&holderID, &holderName, &holderPos, &step, &m.Since, &masked, &total); err != nil {
			return nil, 0, fmt.Errorf("membaca pantauan surat: %w", err)
		}
		m.LetterID = strings.ToLower(m.LetterID)
		if holderID.Valid {
			m.Holder = &PersonRef{ID: holderID.Int64, Name: holderName.String, PositionName: holderPos.String}
		}
		if step.Valid {
			v := int(step.Int32)
			m.Step = &v
		}
		if masked == 1 {
			m.Title, m.TitleMasked = nil, true
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca pantauan surat: %w", err)
	}

	// Halaman di luar jangkauan tidak membawa COUNT(*) OVER (); hitung terpisah supaya total tetap benar
	if len(out) == 0 && f.Offset > 0 {
		if err := r.q.QueryRowContext(ctx, cte+"\nSELECT COUNT(*) FROM m WHERE since <= @cutoff", args...).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("menghitung pantauan surat: %w", err)
		}
	}
	return out, total, nil
}
