package audit

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Audit log untuk pemegang permission AUDIT_VIEW (dijaga router): daftar dengan filter, detail
// (dengan snapshot), log satu surat, dan ekspor CSV. Semua dibatasi ke perusahaan user.

// Batas audit log.
const (
	defaultLogDays = 7  // rentang default bila from/to tidak diisi
	maxExportDays  = 31 // ekspor CSV maksimal 31 hari per unduhan
	maxLetterLogs  = 1000
	dateLayout     = "2006-01-02"
)

var actionPattern = regexp.MustCompile(`^[A-Z_]{1,30}$`)

// LogQuery adalah parameter audit log apa adanya dari query string. Page dan PageSize sudah
// divalidasi handler.
type LogQuery struct {
	From, To   string // YYYY-MM-DD (zona waktu tampilan), To ikut dihitung
	ActorID    string
	Action     string // boleh beberapa dipisah koma
	EntityType string
	Search     string
	Page       int
	PageSize   int
}

// Logs mengambil satu halaman audit log beserta jumlah totalnya.
func (s *Service) Logs(ctx context.Context, user *auth.CurrentUser, lq LogQuery) ([]LogEntry, int, error) {
	f, err := s.logFilter(user, lq)
	if err != nil {
		return nil, 0, err
	}
	f.Offset, f.Limit = (lq.Page-1)*lq.PageSize, lq.PageSize
	items, total, err := s.repo.ListLogs(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].Snapshot = nil // snapshot hanya di detail
		items[i].At = items[i].At.In(s.loc)
	}
	return items, total, nil
}

// Log mengambil satu baris audit log beserta snapshot-nya.
func (s *Service) Log(ctx context.Context, user *auth.CurrentUser, rawID string) (*LogEntry, error) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id < 1 {
		return nil, ErrLogNotFound
	}
	e, err := s.repo.FindLog(ctx, user.Company.Code, user.ID, id)
	if errors.Is(err, db.ErrNotFound) {
		return nil, ErrLogNotFound
	}
	if err != nil {
		return nil, err
	}
	e.At = e.At.In(s.loc)
	return e, nil
}

// LetterLogs mengambil semua audit log satu surat (terbaru dulu, maksimal 1000) beserta snapshot.
func (s *Service) LetterLogs(ctx context.Context, user *auth.CurrentUser, letterID string) ([]LogEntry, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	if _, err := s.repo.findLetter(ctx, user.Company.Code, id, s.today()); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, ErrLetterNotFound
		}
		return nil, err
	}
	items, _, err := s.repo.ListLogs(ctx, LogFilter{
		CompanyCode: user.Company.Code, ViewerID: user.ID, LetterUUID: id, Limit: maxLetterLogs,
	})
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].At = items[i].At.In(s.loc)
	}
	return items, nil
}

// exportSnapshot adalah isi audit_logs.snapshot saat audit log diekspor.
type exportSnapshot struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Action string `json:"action,omitempty"`
	Search string `json:"search,omitempty"`
	Rows   int    `json:"rows"`
}

// Export menulis audit log sebagai CSV (UTF-8 dengan BOM supaya terbaca benar di Excel) lalu
// mencatat ekspor itu sendiri. Rentang wajib diisi, maksimal 31 hari.
func (s *Service) Export(ctx context.Context, user *auth.CurrentUser, lq LogQuery, w io.Writer) error {
	if lq.From == "" || lq.To == "" {
		return apperr.Invalid("Parameter from dan to wajib diisi untuk ekspor")
	}
	f, err := s.logFilter(user, lq)
	if err != nil {
		return err
	}
	if f.To.Sub(f.From) > maxExportDays*24*time.Hour {
		return apperr.Invalid(fmt.Sprintf("Rentang ekspor maksimal %d hari", maxExportDays))
	}
	items, _, err := s.repo.ListLogs(ctx, f)
	if err != nil {
		return err
	}
	if err := s.repo.Record(ctx, Entry{
		CompanyCode: user.Company.Code, EntityType: EntityAudit, Action: ActionExport,
		ActorID: user.ID, ActorName: user.Name,
		Snapshot: exportSnapshot{From: lq.From, To: lq.To, Action: lq.Action, Search: lq.Search, Rows: len(items)},
	}); err != nil {
		return err
	}

	if _, err := io.WriteString(w, "\xEF\xBB\xBF"); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	cw.Write([]string{"Waktu", "Aksi", "Pelaku", "Nomor Surat", "Perihal", "Alamat IP", "Request ID"})
	for _, e := range items {
		var number, title string
		if e.Letter != nil {
			number, title = deref(e.Letter.Number), deref(e.Letter.Title)
			if e.Letter.Masked {
				title = "(disamarkan)"
			}
		}
		cw.Write([]string{e.At.In(s.loc).Format("2006-01-02 15:04:05"), e.Action, e.Actor.Name,
			number, title, deref(e.IPAddress), deref(e.RequestID)})
	}
	cw.Flush()
	return cw.Error()
}

// logFilter mengubah parameter query menjadi filter. Tanpa from/to: 7 hari terakhir.
func (s *Service) logFilter(user *auth.CurrentUser, lq LogQuery) (LogFilter, error) {
	f := LogFilter{CompanyCode: user.Company.Code, ViewerID: user.ID, EntityType: strings.ToUpper(strings.TrimSpace(lq.EntityType))}

	today := time.Now().In(s.loc)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, s.loc)
	from, to := today.AddDate(0, 0, -(defaultLogDays-1)), today
	var err error
	if lq.From != "" {
		if from, err = time.ParseInLocation(dateLayout, lq.From, s.loc); err != nil {
			return f, apperr.Invalid("Parameter from harus berformat YYYY-MM-DD")
		}
	}
	if lq.To != "" {
		if to, err = time.ParseInLocation(dateLayout, lq.To, s.loc); err != nil {
			return f, apperr.Invalid("Parameter to harus berformat YYYY-MM-DD")
		}
	}
	if to.Before(from) {
		return f, apperr.Invalid("Parameter to tidak boleh sebelum from")
	}
	f.From, f.To = from, to.AddDate(0, 0, 1) // to ikut dihitung sampai akhir hari

	if lq.ActorID != "" {
		if f.ActorID, err = strconv.ParseInt(lq.ActorID, 10, 64); err != nil || f.ActorID < 1 {
			return f, apperr.Invalid("Parameter actorId tidak valid")
		}
	}
	for part := range strings.SplitSeq(lq.Action, ",") {
		a := strings.ToUpper(strings.TrimSpace(part))
		if a == "" {
			continue
		}
		if !actionPattern.MatchString(a) {
			return f, apperr.Invalid("Parameter action tidak valid")
		}
		f.Actions = append(f.Actions, a)
	}
	if f.Search, err = db.NormalizeSearch(lq.Search); err != nil {
		return f, err
	}
	return f, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
