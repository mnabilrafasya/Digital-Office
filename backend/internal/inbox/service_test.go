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

func TestList(t *testing.T) {
	wib := time.FixedZone("WIB", 7*3600)
	received := time.Date(2026, 10, 1, 2, 0, 0, 0, time.UTC)
	repo := &fakeRepo{
		counts: Counts{StatusUnread: 3, StatusRead: 2},
		items:  []Item{{ID: 1, ReceivedAt: received, Letter: Letter{LetterDate: &received}}},
	}
	svc := newTestService(repo, &fakeDirectory{}, wib)
	user := &auth.CurrentUser{ID: 8, Company: auth.UserCompany{Code: "PSP"}}
	ctx := context.Background()

	page, err := svc.List(ctx, user, ListQuery{Status: " unread ,UNREAD", Search: "  rapat ", Page: 2, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	f := repo.got
	if f.UserID != 8 || f.CompanyCode != "PSP" || f.Search != "rapat" || f.Offset != 20 || f.Limit != 20 ||
		strings.Join(f.Statuses, ",") != StatusUnread {
		t.Errorf("filter = %+v", f)
	}
	// Total hanya status yang diminta; counts selalu lengkap empat status
	if page.Total != 3 || len(page.Counts) != 4 || page.Counts[StatusRead] != 2 || page.Counts[StatusDisposed] != 0 {
		t.Errorf("total = %d, counts = %v", page.Total, page.Counts)
	}
	if repo.listCalls != 0 || page.Items == nil || len(page.Items) != 0 {
		t.Errorf("halaman di luar jangkauan: listCalls = %d, items = %v", repo.listCalls, page.Items)
	}

	page, err = svc.List(ctx, user, ListQuery{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 5 || len(repo.got.Statuses) != 0 || repo.listCalls != 1 {
		t.Errorf("tanpa status: total = %d, filter = %+v", page.Total, repo.got)
	}
	it := page.Items[0]
	if it.ReceivedAt.Location() != wib || it.Letter.LetterDate.Location() != wib || it.ReadAt != nil {
		t.Errorf("zona waktu: %v, %v, %v", it.ReceivedAt, it.Letter.LetterDate, it.ReadAt)
	}

	invalid := []ListQuery{
		{Status: "SENT"},
		{Status: "UNREAD,X"},
		{Search: strings.Repeat("a", 101)},
	}
	for _, q := range invalid {
		q.Page, q.PageSize = 1, 20
		var ve *apperr.ValidationError
		if _, err := svc.List(ctx, user, q); !errors.As(err, &ve) {
			t.Errorf("%+v: err = %v, ingin ValidationError", q, err)
		}
	}
}
