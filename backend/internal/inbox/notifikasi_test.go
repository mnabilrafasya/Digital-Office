package inbox

import (
	"context"
	"slices"
	"testing"
	"time"

	"office.backend/internal/notifikasi"
)

type fakeNotifier struct{ events []notifikasi.Event }

func (f *fakeNotifier) Notify(e notifikasi.Event) { f.events = append(f.events, e) }

func withNotifier(svc *Service) *fakeNotifier {
	n := &fakeNotifier{}
	svc.repos.Notify = n
	return n
}

func TestForwardNotifications(t *testing.T) {
	svc := newTestService(itemOf(15, StatusRead), &fakeDirectory{heads: map[int64]int64{7: 42}}, time.UTC)
	n := withNotifier(svc)
	if _, err := svc.Dispose(context.Background(), pimpinan, DispositionInput{
		ItemID: "7", UserIDs: []string{"40", "41"}, UnitIDs: []string{"7"}, Note: ptr("Mohon ditindaklanjuti"),
	}); err != nil {
		t.Fatal(err)
	}
	if len(n.events) != 1 {
		t.Fatalf("pemberitahuan = %+v, want 1", n.events)
	}
	e := n.events[0]
	if e.Kind != notifikasi.EventDisposed || !slices.Equal(e.Recipients, []int64{40, 41, 42}) ||
		e.LetterID != 500 || e.ActorID != 15 || e.ActorName != "Sri" || *e.Note != "Mohon ditindaklanjuti" {
		t.Errorf("DISPOSISI salah: %+v", e)
	}

	svc = newTestService(itemOf(30, StatusUnread), &fakeDirectory{}, time.UTC)
	n = withNotifier(svc)
	if _, err := svc.Redirect(context.Background(), staf, RedirectInput{ItemID: "7", UserID: "40"}); err != nil {
		t.Fatal(err)
	}
	if len(n.events) != 1 || n.events[0].Kind != notifikasi.EventRedirected || !slices.Equal(n.events[0].Recipients, []int64{40}) {
		t.Errorf("PENGALIHAN salah: %+v", n.events)
	}
}
