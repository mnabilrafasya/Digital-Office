package inbox

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

// Pemeran: pimpinan Sri (15, INBOX_DISPOSE) dan staf Budi (30, tanpa permission).
var (
	pimpinan = &auth.CurrentUser{ID: 15, Name: "Sri", Position: auth.UserPosition{Name: "VP SDM"},
		Company: auth.UserCompany{Code: "PSP"}, Permissions: []string{auth.InboxDispose}}
	staf = &auth.CurrentUser{ID: 30, Name: "Budi", Company: auth.UserCompany{Code: "PSP"}}
)

// itemOf menyiapkan item inbox nomor 7 milik ownerID dengan status tertentu.
func itemOf(ownerID int64, status string) *fakeRepo {
	return &fakeRepo{ownerID: ownerID, item: &Item{ID: 7, Status: status, Letter: Letter{ID: "uuid-surat"}}}
}

func TestGetMarksRead(t *testing.T) {
	repo := itemOf(15, StatusUnread)
	svc := newTestService(repo, &fakeDirectory{}, time.UTC)
	ctx := context.Background()

	d, err := svc.Get(ctx, pimpinan, "7")
	if err != nil || !repo.marked || d.Status != StatusRead || strings.Join(d.AllowedActions, ",") != "DISPOSE,REDIRECT" {
		t.Fatalf("Get: err=%v marked=%v detail=%+v", err, repo.marked, d)
	}
	for _, tc := range []struct {
		user *auth.CurrentUser
		id   string
	}{{staf, "7"}, {pimpinan, "8"}, {pimpinan, "abc"}} {
		if _, err := svc.Get(ctx, tc.user, tc.id); !errors.Is(err, ErrItemNotFound) {
			t.Errorf("user %d item %s: error = %v, want ErrItemNotFound", tc.user.ID, tc.id, err)
		}
	}
}

func TestDispose(t *testing.T) {
	repo := itemOf(15, StatusRead)
	svc, audit := newTestServiceWithAudit(repo, &fakeDirectory{heads: map[int64]int64{7: 41}}, time.UTC)

	d, err := svc.Dispose(context.Background(), pimpinan, DispositionInput{
		ItemID: "7", UserIDs: []string{"40", "40", " 41"}, UnitIDs: []string{"7"}, Note: ptr("  Mohon ditindaklanjuti  "),
	})
	if err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if d.Status != StatusDisposed || len(d.AllowedActions) != 0 || len(repo.forwards) != 3 {
		t.Fatalf("hasil salah: status=%s actions=%v forwards=%d", d.Status, d.AllowedActions, len(repo.forwards))
	}
	f := repo.forwards
	if f[0].TargetID != 40 || f[0].RecipientID != 40 || f[1].TargetID != 41 || f[1].RecipientID != 41 ||
		f[2].TargetType != TargetUnit || f[2].TargetID != 7 || f[2].RecipientID != 0 { // pimpinan unit 7 = 41, sudah menerima
		t.Errorf("penerusan salah: %+v", f)
	}
	if f[0].Type != ForwardDisposition || f[0].FromName != "Sri" || f[0].FromPosition != "VP SDM" ||
		*f[0].Note != "Mohon ditindaklanjuti" || f[0].SourceItemID != 7 || f[0].LetterID != 500 {
		t.Errorf("isi penerusan salah: %+v", f[0])
	}
	if e := audit.entries[0]; e.Action != "DISPOSITION" || e.EntityUUID != "uuid-surat" || e.EntityID != 500 {
		t.Errorf("audit salah: %+v", e)
	}
}

func TestRedirectByStaff(t *testing.T) {
	repo := itemOf(30, StatusUnread)
	d, err := newTestService(repo, &fakeDirectory{}, time.UTC).Redirect(context.Background(), staf,
		RedirectInput{ItemID: "7", UserID: "40"})
	if err != nil || d.Status != StatusRedirected || len(repo.forwards) != 1 ||
		repo.forwards[0].Type != ForwardRedirect || repo.forwards[0].TargetType != TargetUser || repo.forwards[0].Note != nil {
		t.Errorf("Redirect: err=%v detail=%+v forwards=%+v", err, d, repo.forwards)
	}
}

func TestForwardRejected(t *testing.T) {
	dispose := func(u *auth.CurrentUser, users, units []string, note *string) func(*Service) error {
		return func(svc *Service) error {
			_, err := svc.Dispose(context.Background(), u, DispositionInput{ItemID: "7", UserIDs: users, UnitIDs: units, Note: note})
			return err
		}
	}
	redirect := func(u *auth.CurrentUser, userID string) func(*Service) error {
		return func(svc *Service) error {
			_, err := svc.Redirect(context.Background(), u, RedirectInput{ItemID: "7", UserID: userID})
			return err
		}
	}
	invalid := func(want string) func(error) bool {
		return func(err error) bool {
			var e *apperr.ValidationError
			return errors.As(err, &e) && strings.Contains(e.Message, want)
		}
	}
	isState := func(err error) bool { var e *apperr.InvalidStateError; return errors.As(err, &e) }
	notFound := func(err error) bool { return errors.Is(err, ErrItemNotFound) }
	forbidden := func(err error) bool { return errors.Is(err, apperr.ErrForbidden) }

	tests := []struct {
		name  string
		repo  *fakeRepo
		dir   *fakeDirectory
		run   func(*Service) error
		check func(error) bool
	}{
		{"bukan pemilik (sebelum validasi isi)", itemOf(99, StatusRead), nil, dispose(pimpinan, nil, nil, nil), notFound},
		{"sudah didisposisi", itemOf(15, StatusDisposed), nil, dispose(pimpinan, []string{"40"}, nil, nil), isState},
		{"sudah dialihkan", itemOf(30, StatusRedirected), nil, redirect(staf, "40"), isState},
		{"staf tidak boleh disposisi", itemOf(30, StatusRead), nil, dispose(staf, []string{"40"}, nil, nil), forbidden},
		{"tanpa tujuan", itemOf(15, StatusRead), nil, dispose(pimpinan, nil, nil, nil), invalid("minimal satu")},
		{"pengalihan tanpa tujuan", itemOf(30, StatusRead), nil, redirect(staf, " "), invalid("satu orang")},
		{"ke diri sendiri", itemOf(15, StatusRead), nil, dispose(pimpinan, []string{"15"}, nil, nil), invalid("diri sendiri")},
		{"ID tidak valid", itemOf(15, StatusRead), nil, dispose(pimpinan, []string{"x"}, nil, nil), invalid("Orang tujuan tidak valid")},
		{"user nonaktif", itemOf(15, StatusRead), &fakeDirectory{inactiveUsers: map[int64]bool{40: true}},
			dispose(pimpinan, []string{"40"}, nil, nil), invalid("tidak aktif")},
		{"unit tanpa pimpinan", itemOf(15, StatusRead), nil, dispose(pimpinan, nil, []string{"7"}, nil), invalid("pimpinan aktif")},
		{"unit yang dipimpin sendiri", itemOf(15, StatusRead), &fakeDirectory{heads: map[int64]int64{7: 15}},
			dispose(pimpinan, nil, []string{"7"}, nil), invalid("Anda pimpin")},
		{"catatan terlalu panjang", itemOf(15, StatusRead), nil,
			dispose(pimpinan, []string{"40"}, nil, ptr(strings.Repeat("a", 1001))), invalid("maksimal 1000")},
	}
	for _, tt := range tests {
		dir := tt.dir
		if dir == nil {
			dir = &fakeDirectory{}
		}
		err := tt.run(newTestService(tt.repo, dir, time.UTC))
		if !tt.check(err) {
			t.Errorf("%s: error = %v", tt.name, err)
		}
		if len(tt.repo.forwards) != 0 {
			t.Errorf("%s: penerusan tetap tersimpan", tt.name)
		}
	}
}

func ptr(s string) *string { return &s }
