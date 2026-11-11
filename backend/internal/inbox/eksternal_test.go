package inbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"office.backend/internal/masterdata"
	"office.backend/internal/platform/apperr"
)

// Arsip surat eksternal keluar di inbox: hanya dibaca, tidak bisa didisposisi atau dialihkan.
func TestOutgoingArchiveReadOnly(t *testing.T) {
	repo := itemOf(15, StatusUnread)
	repo.item.Letter.Direction = masterdata.DirectionOutgoing
	svc := newTestService(repo, &fakeDirectory{}, time.UTC)
	ctx := context.Background()

	d, err := svc.Get(ctx, pimpinan, "7")
	if err != nil || len(d.AllowedActions) != 0 {
		t.Fatalf("arsip keluar: err=%v actions=%v", err, d.AllowedActions)
	}
	var state *apperr.InvalidStateError
	if _, err := svc.Redirect(ctx, pimpinan, RedirectInput{ItemID: "7", UserID: "40"}); !errors.As(err, &state) {
		t.Errorf("alihkan arsip keluar: %v", err)
	}
	if _, err := svc.Dispose(ctx, pimpinan, DispositionInput{ItemID: "7", UserIDs: []string{"40"}}); !errors.As(err, &state) {
		t.Errorf("disposisi arsip keluar: %v", err)
	}

	// Surat masuk eksternal tetap bisa diteruskan seperti surat biasa
	repo = itemOf(15, StatusUnread)
	repo.item.Letter.Direction = masterdata.DirectionIncoming
	if d, err := newTestService(repo, &fakeDirectory{}, time.UTC).Get(ctx, pimpinan, "7"); err != nil ||
		len(d.AllowedActions) != 2 {
		t.Errorf("surat masuk eksternal: err=%v actions=%v", err, d.AllowedActions)
	}
}

func TestListCategory(t *testing.T) {
	repo := &fakeRepo{}
	svc := newTestService(repo, &fakeDirectory{}, time.UTC)
	ctx := context.Background()
	if _, err := svc.List(ctx, pimpinan, ListQuery{Category: " external ", Page: 1, PageSize: 10}); err != nil || repo.got.Category != CategoryExternal {
		t.Errorf("kategori eksternal: err=%v filter=%+v", err, repo.got)
	}
	var invalid *apperr.ValidationError
	if _, err := svc.List(ctx, pimpinan, ListQuery{Category: "LUAR", Page: 1, PageSize: 10}); !errors.As(err, &invalid) {
		t.Errorf("kategori tidak dikenal: %v", err)
	}
}
