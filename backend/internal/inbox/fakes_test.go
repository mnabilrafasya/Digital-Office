package inbox

import (
	"context"
	"time"

	"office.backend/internal/audit"
	"office.backend/internal/platform/db"
)

// fakeTx menjalankan fungsi langsung tanpa database.
type fakeTx struct{}

func (fakeTx) InTx(_ context.Context, fn func(q db.DBTX) error) error { return fn(nil) }

// fakeRepo adalah repository inbox palsu: data diatur lewat field, pemanggilan dicatat.
type fakeRepo struct {
	counts    Counts
	items     []Item
	got       Filter
	listCalls int

	item     *Item // item yang bisa dibuka
	ownerID  int64 // pemilik item
	forwards []NewForward
	marked   bool
}

func (f *fakeRepo) List(_ context.Context, filter Filter) ([]Item, error) {
	f.got = filter
	f.listCalls++
	return f.items, nil
}

func (f *fakeRepo) Counts(_ context.Context, filter Filter) (Counts, error) {
	f.got = filter
	return f.counts, nil
}

func (f *fakeRepo) FindItem(_ context.Context, _ string, userID, itemID int64) (*Item, error) {
	if f.item == nil || f.item.ID != itemID || f.ownerID != userID {
		return nil, db.ErrNotFound
	}
	it := *f.item
	return &it, nil
}

func (f *fakeRepo) Forwards(context.Context, int64) ([]Forward, error) {
	out := []Forward{}
	for _, nf := range f.forwards {
		out = append(out, Forward{Type: nf.Type, TargetType: nf.TargetType, Note: nf.Note})
	}
	return out, nil
}

func (f *fakeRepo) MarkRead(context.Context, int64) error {
	f.marked = true
	if f.item.Status == StatusUnread {
		f.item.Status = StatusRead
	}
	return nil
}

func (f *fakeRepo) LockItem(ctx context.Context, companyCode string, userID, itemID int64) (*ItemLock, error) {
	it, err := f.FindItem(ctx, companyCode, userID, itemID)
	if err != nil {
		return nil, err
	}
	return &ItemLock{ID: it.ID, LetterID: 500, LetterUUID: it.Letter.ID, Status: it.Status}, nil
}

func (f *fakeRepo) AddForward(_ context.Context, nf NewForward) error {
	f.forwards = append(f.forwards, nf)
	return nil
}

func (f *fakeRepo) SetForwarded(_ context.Context, _ int64, status string) error {
	f.item.Status = status
	return nil
}

// fakeDirectory menganggap semua user dan unit aktif kecuali yang diatur sebaliknya.
type fakeDirectory struct {
	inactiveUsers map[int64]bool
	heads         map[int64]int64 // unit -> pimpinan; unit tanpa entri tidak punya pimpinan
}

func (f *fakeDirectory) ActiveUserIDs(_ context.Context, _ string, ids []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	for _, id := range ids {
		out[id] = !f.inactiveUsers[id]
	}
	return out, nil
}

func (f *fakeDirectory) ActiveUnitIDs(_ context.Context, _ string, ids []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

func (f *fakeDirectory) UnitHeadIDs(context.Context, string, []int64) (map[int64]int64, error) {
	return f.heads, nil
}

// fakeAudit mencatat entri audit yang ditulis.
type fakeAudit struct{ entries []audit.Entry }

func (f *fakeAudit) Record(_ context.Context, e audit.Entry) error {
	f.entries = append(f.entries, e)
	return nil
}

func newTestService(repo *fakeRepo, dir *fakeDirectory, loc *time.Location) *Service {
	svc, _ := newTestServiceWithAudit(repo, dir, loc)
	return svc
}

func newTestServiceWithAudit(repo *fakeRepo, dir *fakeDirectory, loc *time.Location) (*Service, *fakeAudit) {
	a := &fakeAudit{}
	return NewService(fakeTx{}, nil, Repos{
		Items:     func(db.DBTX) Store { return repo },
		Audit:     func(db.DBTX) AuditStore { return a },
		Directory: dir,
	}, loc), a
}
