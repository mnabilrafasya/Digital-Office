// Package inbox berisi surat masuk: surat terkirim (SENT) yang sampai ke seseorang, langsung
// sebagai penerima (TO/CC) atau lewat penerusan (disposisi/pengalihan). Baris inbox langsung
// ditulis alur surat saat surat terbit, lewat Repository.DeliverDirect (memenuhi interface
// surat.InboxStore); penerusan ditulis package ini (disposisi.go).
package inbox

import (
	"context"
	"slices"
	"strings"
	"time"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Status item inbox (sama dengan CHECK ck_inbox_items_status dan lookup INBOX_STATUS).
const (
	StatusUnread     = "UNREAD"
	StatusRead       = "READ"
	StatusDisposed   = "DISPOSED"
	StatusRedirected = "REDIRECTED"
)

// allStatuses adalah semua status inbox, berurutan seperti tab di halaman inbox.
func allStatuses() []string {
	return []string{StatusUnread, StatusRead, StatusDisposed, StatusRedirected}
}

// Store adalah query inbox yang dibutuhkan Service.
type Store interface {
	List(ctx context.Context, f Filter) ([]Item, error)
	Counts(ctx context.Context, f Filter) (Counts, error)
	FindItem(ctx context.Context, companyCode string, userID, itemID int64) (*Item, error)
	Forwards(ctx context.Context, itemID int64) ([]Forward, error)
	MarkRead(ctx context.Context, itemID int64) error

	// Penerusan (di dalam transaksi)
	LockItem(ctx context.Context, companyCode string, userID, itemID int64) (*ItemLock, error)
	AddForward(ctx context.Context, f NewForward) error
	SetForwarded(ctx context.Context, itemID int64, status string) error
}

// Directory adalah data organisasi untuk memeriksa tujuan penerusan (dipenuhi repository organisasi).
type Directory interface {
	ActiveUserIDs(ctx context.Context, companyCode string, ids []int64) (map[int64]bool, error)
	ActiveUnitIDs(ctx context.Context, companyCode string, ids []int64) (map[int64]bool, error)
	UnitHeadIDs(ctx context.Context, companyCode string, unitIDs []int64) (map[int64]int64, error)
}

// AuditStore mencatat jejak audit.
type AuditStore interface {
	Record(ctx context.Context, e audit.Entry) error
}

// Repos adalah repository yang dibutuhkan Service. Items dan Audit dibuat di atas koneksi atau
// transaksi; Directory hanya membaca di luar transaksi.
type Repos struct {
	Items     func(q db.DBTX) Store
	Audit     func(q db.DBTX) AuditStore
	Directory Directory
}

// Service menyediakan inbox milik user yang login beserta disposisi dan pengalihannya.
type Service struct {
	tx     db.TxRunner
	repos  Repos
	reader Store // repository inbox di luar transaksi
	loc    *time.Location
}

// NewService membuat Service. conn dipakai untuk query di luar transaksi; loc adalah zona waktu
// tampilan (WIB).
func NewService(tx db.TxRunner, conn db.DBTX, repos Repos, loc *time.Location) *Service {
	return &Service{tx: tx, repos: repos, reader: repos.Items(conn), loc: loc}
}

// ListQuery adalah parameter inbox. Status dan Search apa adanya dari query string; Page dan
// PageSize sudah divalidasi handler.
type ListQuery struct {
	Status   string // opsional, beberapa status dipisah koma
	Search   string
	Page     int
	PageSize int
}

// Page adalah satu halaman inbox. Total mengikuti filter status; Counts selalu berisi keempat
// status (filter pencarian tetap berlaku) supaya tab bisa menampilkan jumlahnya.
type Page struct {
	Items  []Item
	Total  int
	Counts Counts
}

// List mengambil satu halaman inbox user beserta jumlah per status.
func (s *Service) List(ctx context.Context, user *auth.CurrentUser, lq ListQuery) (Page, error) {
	statuses, err := parseStatuses(lq.Status)
	if err != nil {
		return Page{}, err
	}
	search, err := db.NormalizeSearch(lq.Search)
	if err != nil {
		return Page{}, err
	}
	f := Filter{
		CompanyCode: user.Company.Code,
		UserID:      user.ID,
		Statuses:    statuses,
		Search:      search,
		Offset:      (lq.Page - 1) * lq.PageSize,
		Limit:       lq.PageSize,
	}

	found, err := s.reader.Counts(ctx, f)
	if err != nil {
		return Page{}, err
	}
	counts, total := Counts{}, 0
	for _, st := range allStatuses() {
		counts[st] = found[st]
		if len(statuses) == 0 || slices.Contains(statuses, st) {
			total += found[st]
		}
	}

	items := []Item{}
	if f.Offset < total { // halaman di luar jangkauan tidak perlu query
		if items, err = s.reader.List(ctx, f); err != nil {
			return Page{}, err
		}
	}
	for i := range items {
		s.present(&items[i])
	}
	return Page{Items: items, Total: total, Counts: counts}, nil
}

// present mengubah waktu item ke zona waktu tampilan.
func (s *Service) present(it *Item) {
	it.ReceivedAt = it.ReceivedAt.In(s.loc)
	it.ReadAt = s.local(it.ReadAt)
	it.Letter.LetterDate = s.local(it.Letter.LetterDate)
}

func (s *Service) local(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := t.In(s.loc)
	return &v
}

// parseStatuses membaca "UNREAD,READ" menjadi daftar status unik yang valid ("" = semua).
func parseStatuses(raw string) ([]string, error) {
	var out []string
	for part := range strings.SplitSeq(raw, ",") {
		st := strings.ToUpper(strings.TrimSpace(part))
		if st == "" || slices.Contains(out, st) {
			continue
		}
		if !slices.Contains(allStatuses(), st) {
			return nil, apperr.Invalid("Parameter status wajib salah satu dari: " + strings.Join(allStatuses(), ", "))
		}
		out = append(out, st)
	}
	return out, nil
}
