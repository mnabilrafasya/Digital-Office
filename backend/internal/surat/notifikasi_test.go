package surat

import (
	"context"
	"slices"
	"testing"

	"office.backend/internal/notifikasi"
)

// fakeNotifier mencatat pemberitahuan yang diantre.
type fakeNotifier struct{ events []notifikasi.Event }

func (f *fakeNotifier) Notify(e notifikasi.Event) { f.events = append(f.events, e) }

func withNotifier(svc *Service) *fakeNotifier {
	n := &fakeNotifier{}
	svc.repos.Notify = n
	return n
}

func TestTransitionNotifications(t *testing.T) {
	ctx := context.Background()
	in := TransitionInput{LetterID: testLetterID}
	type want struct {
		kind       string
		recipients []int64
		note       string
	}
	cases := []struct {
		name  string
		store *fakeLetterStore
		run   func(*Service) error
		want  []want
	}{
		{"ajukan: reviewer pertama", letterAt(StatusDraft, 0, 0),
			func(s *Service) error { _, err := s.Submit(ctx, andiUser, in); return err },
			[]want{{notifikasi.EventReviewRequested, []int64{9}, ""}}},
		{"reviewer 1 setuju: reviewer 2", letterAt(StatusInReview, 9, 0),
			func(s *Service) error { _, err := s.Approve(ctx, fajarUser, in); return err },
			[]want{{notifikasi.EventReviewRequested, []int64{4}, ""}}},
		{"reviewer terakhir setuju: approver", letterAt(StatusInReview, 4, 9),
			func(s *Service) error { _, err := s.Approve(ctx, dewiUser, in); return err },
			[]want{{notifikasi.EventApprovalRequested, []int64{3}, ""}}},
		{"approver kembalikan: reviewer", letterAt(StatusInApproval, 3, 4),
			func(s *Service) error {
				_, err := s.Return(ctx, agusUser, TransitionInput{LetterID: testLetterID, Note: ptr("Perbaiki lampiran")})
				return err
			},
			[]want{{notifikasi.EventReturned, []int64{4}, "Perbaiki lampiran"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestLetterService(tc.store, newFakeRefs())
			n := withNotifier(svc)
			if err := tc.run(svc); err != nil {
				t.Fatal(err)
			}
			if len(n.events) != len(tc.want) {
				t.Fatalf("pemberitahuan = %+v, want %d", n.events, len(tc.want))
			}
			for i, w := range tc.want {
				e := n.events[i]
				note := ""
				if e.Note != nil {
					note = *e.Note
				}
				if e.Kind != w.kind || !slices.Equal(e.Recipients, w.recipients) || note != w.note || e.LetterID != 1 {
					t.Errorf("pemberitahuan %d = %+v, want %+v", i, e, w)
				}
			}
		})
	}
}

func TestPublishNotifications(t *testing.T) {
	store := letterAt(StatusInApproval, 3, 4)
	refs := newFakeRefs()
	refs.unitHeads = map[int64]int64{7: 50, 9: 15} // pimpinan unit Tembusan = drafter sendiri
	refs.aliasMembers = map[int64][]int64{60: {52}}
	svc := newTestLetterService(store, refs)
	n := withNotifier(svc)

	if _, err := svc.Approve(context.Background(), agusUser, TransitionInput{LetterID: testLetterID}); err != nil {
		t.Fatal(err)
	}
	if len(n.events) != 2 {
		t.Fatalf("pemberitahuan = %+v, want 2", n.events)
	}
	if e := n.events[0]; e.Kind != notifikasi.EventPublished || !slices.Equal(e.Recipients, []int64{15}) || e.ActorID != 3 {
		t.Errorf("TERBIT salah: %+v", e)
	}
	// Drafter (15) tidak mendapat pesan "surat masuk" lagi
	if e := n.events[1]; e.Kind != notifikasi.EventReceived || !slices.Equal(e.Recipients, []int64{50, 52}) {
		t.Errorf("SURAT_MASUK salah: %+v", e)
	}
}

func TestNoNotificationWhenTransitionFails(t *testing.T) {
	store := letterAt(StatusInReview, 9, 0)
	svc := newTestLetterService(store, newFakeRefs())
	n := withNotifier(svc)
	// Dewi bukan giliran: aksi ditolak, tidak ada pesan
	if _, err := svc.Approve(context.Background(), dewiUser, TransitionInput{LetterID: testLetterID}); err == nil {
		t.Fatal("Approve berhasil padahal bukan giliran Dewi")
	}
	if len(n.events) != 0 {
		t.Errorf("pemberitahuan terkirim padahal aksi gagal: %+v", n.events)
	}
}
