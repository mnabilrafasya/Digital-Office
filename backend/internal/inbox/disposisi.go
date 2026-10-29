package inbox

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Penerusan surat dari inbox:
//
//	DISPOSE   pemilik item dengan permission INBOX_DISPOSE   ke orang dan/atau unit   item -> DISPOSED
//	REDIRECT  pemilik item mana pun                          ke tepat satu orang      item -> REDIRECTED
//
// Hanya item UNREAD/READ yang bisa diteruskan, dan hanya sekali. Penerima mendapat item inbox
// FORWARD (UNREAD); disposisi ke unit diterima pimpinan unit tersebut. Penerima disposisi bisa
// mendisposisi lagi dari item miliknya (berjenjang).

// Aksi yang boleh dilakukan pada item inbox (allowedActions).
const (
	ActionDispose  = "DISPOSE"
	ActionRedirect = "REDIRECT"
)

// Jenis dan tujuan penerusan (sama dengan CHECK di letter_forwards).
const (
	ForwardDisposition = "DISPOSITION"
	ForwardRedirect    = "REDIRECT"
	TargetUser         = "USER"
	TargetUnit         = "UNIT"
)

// Batas penerusan.
const (
	maxTargets    = 20
	maxNoteLength = 1000
)

// ErrItemNotFound: item inbox tidak ada atau milik orang lain.
var ErrItemNotFound error = &apperr.NotFoundError{Message: "Surat masuk tidak ditemukan"}

// DispositionInput adalah permintaan disposisi. ID apa adanya dari body (string).
type DispositionInput struct {
	ItemID   string
	UserIDs  []string
	UnitIDs  []string
	Note     *string
	ClientIP string
}

// RedirectInput adalah permintaan pengalihan ke satu orang.
type RedirectInput struct {
	ItemID   string
	UserID   string
	Note     *string
	ClientIP string
}

// forwardSnapshot adalah isi kolom audit_logs.snapshot untuk penerusan surat.
type forwardSnapshot struct {
	InboxItemID int64   `json:"inboxItemId"`
	UserIDs     []int64 `json:"userIds,omitempty"`
	UnitIDs     []int64 `json:"unitIds,omitempty"`
}

// Get membuka satu item inbox milik user. Item UNREAD otomatis menjadi READ.
func (s *Service) Get(ctx context.Context, user *auth.CurrentUser, itemID string) (*ItemDetail, error) {
	id, ok := parseItemID(itemID)
	if !ok {
		return nil, ErrItemNotFound
	}
	it, err := s.findItem(ctx, user, id)
	if err != nil {
		return nil, err
	}
	if it.Status == StatusUnread {
		if err := s.reader.MarkRead(ctx, id); err != nil {
			return nil, err
		}
	}
	return s.detail(ctx, user, id)
}

// Dispose mendisposisikan surat ke orang dan/atau unit (wajib permission INBOX_DISPOSE).
func (s *Service) Dispose(ctx context.Context, user *auth.CurrentUser, in DispositionInput) (*ItemDetail, error) {
	return s.forward(ctx, user, in.ItemID, ForwardDisposition, in.UserIDs, in.UnitIDs, in.Note, in.ClientIP)
}

// Redirect mengalihkan surat ke satu orang (misalnya surat salah alamat).
func (s *Service) Redirect(ctx context.Context, user *auth.CurrentUser, in RedirectInput) (*ItemDetail, error) {
	var users []string
	if strings.TrimSpace(in.UserID) != "" {
		users = []string{in.UserID}
	}
	return s.forward(ctx, user, in.ItemID, ForwardRedirect, users, nil, in.Note, in.ClientIP)
}

// forward menjalankan disposisi/pengalihan. Urutan pengecekan: item milik user (404), status
// (409), permission (403), lalu isi permintaan (400), sehingga keberadaan surat tidak bocor.
func (s *Service) forward(ctx context.Context, user *auth.CurrentUser, itemID, kind string, rawUsers, rawUnits []string, rawNote *string, clientIP string) (*ItemDetail, error) {
	id, err := s.checkForwardable(ctx, user, itemID, kind)
	if err != nil {
		return nil, err
	}
	userIDs, err := parseIDs(rawUsers, "Orang tujuan")
	if err != nil {
		return nil, err
	}
	unitIDs, err := parseIDs(rawUnits, "Unit tujuan")
	if err != nil {
		return nil, err
	}
	note, err := parseNote(rawNote)
	if err != nil {
		return nil, err
	}
	switch n := len(userIDs) + len(unitIDs); {
	case n == 0 && kind == ForwardRedirect:
		return nil, apperr.Invalid("Pilih satu orang tujuan pengalihan")
	case n == 0:
		return nil, apperr.Invalid("Pilih minimal satu orang atau unit tujuan disposisi")
	case n > maxTargets:
		return nil, apperr.Invalid(fmt.Sprintf("Tujuan penerusan maksimal %d", maxTargets))
	}
	heads, err := s.checkTargets(ctx, user, userIDs, unitIDs)
	if err != nil {
		return nil, err
	}

	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		items := s.repos.Items(q)
		it, err := items.LockItem(ctx, user.Company.Code, user.ID, id)
		if err != nil {
			return notFoundAs(err)
		}
		if !forwardable(it.Status) { // diteruskan request lain di sela pengecekan (klik dua kali)
			return errAlreadyForwarded
		}

		received := map[int64]bool{} // satu orang cukup satu item inbox per permintaan
		add := func(targetType string, targetID, recipientID int64) error {
			f := NewForward{
				LetterID: it.LetterID, SourceItemID: it.ID, Type: kind,
				FromUserID: user.ID, FromName: user.Name, FromPosition: user.Position.Name,
				TargetType: targetType, TargetID: targetID, Note: note,
			}
			if !received[recipientID] {
				received[recipientID] = true
				f.RecipientID = recipientID
			}
			return items.AddForward(ctx, f)
		}
		for _, uid := range userIDs {
			if err := add(TargetUser, uid, uid); err != nil {
				return err
			}
		}
		for _, unitID := range unitIDs {
			if err := add(TargetUnit, unitID, heads[unitID]); err != nil {
				return err
			}
		}

		status := StatusDisposed
		if kind == ForwardRedirect {
			status = StatusRedirected
		}
		if err := items.SetForwarded(ctx, it.ID, status); err != nil {
			return err
		}
		return s.repos.Audit(q).Record(ctx, audit.Entry{
			CompanyCode: user.Company.Code,
			EntityType:  "LETTER",
			EntityID:    it.LetterID,
			EntityUUID:  it.LetterUUID,
			Action:      kind,
			ActorID:     user.ID,
			ActorName:   user.Name,
			IPAddress:   clientIP,
			Snapshot:    forwardSnapshot{InboxItemID: it.ID, UserIDs: userIDs, UnitIDs: unitIDs},
		})
	})
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, user, id)
}

// errAlreadyForwarded: item sudah didisposisi atau dialihkan.
var errAlreadyForwarded = &apperr.InvalidStateError{Message: "Surat ini sudah didisposisi atau dialihkan. Muat ulang halaman untuk melihat status terbaru."}

// checkForwardable memastikan item milik user, masih bisa diteruskan, dan user berwenang.
func (s *Service) checkForwardable(ctx context.Context, user *auth.CurrentUser, itemID, kind string) (int64, error) {
	id, ok := parseItemID(itemID)
	if !ok {
		return 0, ErrItemNotFound
	}
	it, err := s.findItem(ctx, user, id)
	if err != nil {
		return 0, err
	}
	if !forwardable(it.Status) {
		return 0, errAlreadyForwarded
	}
	if kind == ForwardDisposition && !user.HasPermission(auth.InboxDispose) {
		return 0, apperr.ErrForbidden
	}
	return id, nil
}

// checkTargets memastikan orang dan unit tujuan aktif di perusahaan user, bukan user sendiri, dan
// setiap unit punya pimpinan aktif. Mengembalikan pimpinan tiap unit (penerima disposisinya).
func (s *Service) checkTargets(ctx context.Context, user *auth.CurrentUser, userIDs, unitIDs []int64) (map[int64]int64, error) {
	if slices.Contains(userIDs, user.ID) {
		return nil, apperr.Invalid("Tidak bisa meneruskan surat ke diri sendiri")
	}
	users, err := s.repos.Directory.ActiveUserIDs(ctx, user.Company.Code, userIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range userIDs {
		if !users[id] {
			return nil, apperr.Invalid("Orang tujuan tidak ditemukan atau tidak aktif")
		}
	}
	units, err := s.repos.Directory.ActiveUnitIDs(ctx, user.Company.Code, unitIDs)
	if err != nil {
		return nil, err
	}
	heads, err := s.repos.Directory.UnitHeadIDs(ctx, user.Company.Code, unitIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range unitIDs {
		head, ok := heads[id]
		switch {
		case !units[id]:
			return nil, apperr.Invalid("Unit tujuan tidak ditemukan atau tidak aktif")
		case !ok:
			return nil, apperr.Invalid("Unit tujuan belum punya pimpinan aktif")
		case head == user.ID:
			return nil, apperr.Invalid("Tidak bisa mendisposisi ke unit yang Anda pimpin sendiri")
		}
	}
	return heads, nil
}

// detail memuat item beserta penerusan dan aksi yang boleh dilakukan.
func (s *Service) detail(ctx context.Context, user *auth.CurrentUser, id int64) (*ItemDetail, error) {
	it, err := s.findItem(ctx, user, id)
	if err != nil {
		return nil, err
	}
	forwards, err := s.reader.Forwards(ctx, id)
	if err != nil {
		return nil, err
	}
	s.present(it)
	for i := range forwards {
		forwards[i].CreatedAt = forwards[i].CreatedAt.In(s.loc)
	}
	return &ItemDetail{Item: *it, Forwards: forwards, AllowedActions: allowedActions(it.Status, user)}, nil
}

func (s *Service) findItem(ctx context.Context, user *auth.CurrentUser, id int64) (*Item, error) {
	it, err := s.reader.FindItem(ctx, user.Company.Code, user.ID, id)
	if err != nil {
		return nil, notFoundAs(err)
	}
	return it, nil
}

// allowedActions adalah tombol yang boleh ditampilkan untuk item berstatus status.
func allowedActions(status string, user *auth.CurrentUser) []string {
	out := []string{}
	if !forwardable(status) {
		return out
	}
	if user.HasPermission(auth.InboxDispose) {
		out = append(out, ActionDispose)
	}
	return append(out, ActionRedirect)
}

// forwardable: item yang belum didisposisi/dialihkan.
func forwardable(status string) bool {
	return status == StatusUnread || status == StatusRead
}

// notFoundAs mengubah db.ErrNotFound menjadi ErrItemNotFound; error lain diteruskan.
func notFoundAs(err error) error {
	if errors.Is(err, db.ErrNotFound) {
		return ErrItemNotFound
	}
	return err
}

// parseItemID membaca ID item inbox dari URL.
func parseItemID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	return id, err == nil && id > 0
}

// parseIDs membaca daftar ID (string) menjadi angka unik, urutan dipertahankan.
func parseIDs(raw []string, label string) ([]int64, error) {
	var out []int64
	for _, r := range raw {
		id, err := strconv.ParseInt(strings.TrimSpace(r), 10, 64)
		if err != nil || id <= 0 {
			return nil, apperr.Invalid(label + " tidak valid")
		}
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out, nil
}

// parseNote merapikan catatan opsional: di-trim, kosong menjadi nil, maksimal maxNoteLength
// (dihitung dalam satuan UTF-16 seperti NVARCHAR).
func parseNote(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	note := strings.TrimSpace(*raw)
	if note == "" {
		return nil, nil
	}
	n := 0
	for _, r := range note {
		n += utf16.RuneLen(r)
	}
	if n > maxNoteLength {
		return nil, apperr.Invalid(fmt.Sprintf("Catatan maksimal %d karakter", maxNoteLength))
	}
	return &note, nil
}
