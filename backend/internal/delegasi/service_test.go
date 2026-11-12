package delegasi

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

var (
	agus  = &auth.CurrentUser{ID: 3, Name: "Agus", Company: auth.UserCompany{Code: "PSP"}}
	admin = &auth.CurrentUser{ID: 13, Name: "Rudi", Company: auth.UserCompany{Code: "PSP"}, Permissions: []string{auth.UserManage}}
)

type fakeTx struct{}

func (fakeTx) InTx(_ context.Context, fn func(q db.DBTX) error) error { return fn(nil) }

type fakeAudit struct{ entries []audit.Entry }

func (f *fakeAudit) Record(_ context.Context, e audit.Entry) error {
	f.entries = append(f.entries, e)
	return nil
}

// fakeStore menyimpan jadwal di memori; konflik disimulasikan lewat field conflict.
type fakeStore struct {
	rows     map[int64]*Delegation
	nextID   int64
	active   map[int64]bool // user aktif
	conflict *Conflict
	locked   bool
	gotP     period
}

func newFakeStore() *fakeStore {
	return &fakeStore{rows: map[int64]*Delegation{}, nextID: 1, active: map[int64]bool{3: true, 20: true, 9: true}}
}

func (f *fakeStore) List(_ context.Context, fl listFilter) ([]Delegation, int, error) {
	out := []Delegation{}
	for _, d := range f.rows {
		if (fl.OwnerID == 0 || d.User.ID == fl.OwnerID) && (fl.DelegateID == 0 || d.Delegate.ID == fl.DelegateID) {
			out = append(out, *d)
		}
	}
	return out, len(out), nil
}

func (f *fakeStore) Find(_ context.Context, _ string, id int64, _ string, _ bool) (*Delegation, error) {
	d, ok := f.rows[id]
	if !ok {
		return nil, db.ErrNotFound
	}
	c := *d
	return &c, nil
}

func (f *fakeStore) LockCompany(context.Context, string) error { f.locked = true; return nil }

func (f *fakeStore) FindConflict(_ context.Context, _ string, p period, _ int64) (*Conflict, error) {
	f.gotP = p
	return f.conflict, nil
}

func (f *fakeStore) ActiveUser(_ context.Context, _ string, id int64) (*PersonRef, error) {
	if !f.active[id] {
		return nil, db.ErrNotFound
	}
	return &PersonRef{ID: id}, nil
}

func (f *fakeStore) Insert(_ context.Context, _ string, p period, createdBy int64) (int64, error) {
	id := f.nextID
	f.nextID++
	f.rows[id] = &Delegation{ID: id, User: PersonRef{ID: p.UserID}, Delegate: PersonRef{ID: p.DelegateID},
		From: p.From, Until: p.Until, Note: p.Note, Status: StatusUpcoming, CreatedBy: PersonRef{ID: createdBy}}
	return id, nil
}

func (f *fakeStore) Update(_ context.Context, id int64, p period) error {
	d := f.rows[id]
	d.Delegate, d.From, d.Until, d.Note = PersonRef{ID: p.DelegateID}, p.From, p.Until, p.Note
	return nil
}

func (f *fakeStore) Delete(_ context.Context, id int64) error {
	delete(f.rows, id)
	return nil
}

// newTestService memakai "hari ini" 12 November 2026.
func newTestService(st *fakeStore) (*Service, *fakeAudit) {
	au := &fakeAudit{}
	svc := NewService(fakeTx{}, nil, func(db.DBTX) Store { return st }, func(db.DBTX) AuditStore { return au }, time.UTC)
	svc.now = func() time.Time { return time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC) }
	return svc, au
}

func in(sub, from, until string) Input {
	note := "  Cuti tahunan  "
	return Input{SubstituteID: sub, From: from, Until: until, Note: &note}
}

func TestCreateMine(t *testing.T) {
	st := newFakeStore()
	svc, au := newTestService(st)
	d, err := svc.CreateMine(context.Background(), agus, in("20", "2026-11-12", "2026-11-15"))
	if err != nil {
		t.Fatalf("CreateMine: %v", err)
	}
	if d.User.ID != 3 || d.Delegate.ID != 20 || d.From != "2026-11-12" || d.Until != "2026-11-15" ||
		d.Note == nil || *d.Note != "Cuti tahunan" || d.CreatedBy.ID != 3 || !st.locked {
		t.Errorf("delegasi = %+v", d)
	}
	if len(au.entries) != 1 || au.entries[0].Action != audit.ActionDelegationCreate || au.entries[0].EntityType != audit.EntityDelegation {
		t.Errorf("audit = %+v", au.entries)
	}
}

func TestCreateValidation(t *testing.T) {
	cases := map[string]Input{
		"pengganti diri sendiri":   in("3", "2026-11-12", "2026-11-15"),
		"pengganti kosong":         in("", "2026-11-12", "2026-11-15"),
		"format tanggal":           in("20", "12-11-2026", "2026-11-15"),
		"mulai setelah selesai":    in("20", "2026-11-16", "2026-11-15"),
		"selesai sebelum hari ini": in("20", "2026-11-01", "2026-11-11"),
		"keterangan panjang": {SubstituteID: "20", From: "2026-11-12", Until: "2026-11-15",
			Note: ptr(strings.Repeat("x", 201))},
		"pengganti tidak aktif": in("44", "2026-11-12", "2026-11-15"),
	}
	for name, input := range cases {
		svc, _ := newTestService(newFakeStore())
		var invalid *apperr.ValidationError
		if _, err := svc.CreateMine(context.Background(), agus, input); !errors.As(err, &invalid) {
			t.Errorf("%s: err = %v, want 400", name, err)
		}
	}
	// Hari ini boleh menjadi hari terakhir
	svc, _ := newTestService(newFakeStore())
	if _, err := svc.CreateMine(context.Background(), agus, in("20", "2026-11-12", "2026-11-12")); err != nil {
		t.Errorf("delegasi satu hari (hari ini): %v", err)
	}
}

func TestCreateConflicts(t *testing.T) {
	for kind, want := range map[string]string{
		"OWN":              "sudah punya jadwal delegasi",
		"DELEGATE_AWAY":    "tidak bisa menjadi pengganti",
		"OWNER_SUBSTITUTE": "supaya delegasi tidak berantai",
	} {
		st := newFakeStore()
		st.conflict = &Conflict{Kind: kind, OwnerName: "X", TargetName: "Y", From: "2026-11-13", Until: "2026-11-14"}
		svc, _ := newTestService(st)
		var state *apperr.InvalidStateError
		if _, err := svc.CreateMine(context.Background(), agus, in("20", "2026-11-12", "2026-11-15")); !errors.As(err, &state) ||
			!strings.Contains(state.Message, want) {
			t.Errorf("%s: err = %v", kind, err)
		}
		if len(st.rows) != 0 {
			t.Errorf("%s: jadwal tetap disimpan", kind)
		}
	}
}

func TestOwnershipAndEnded(t *testing.T) {
	st := newFakeStore()
	st.rows[5] = &Delegation{ID: 5, User: PersonRef{ID: 9}, Delegate: PersonRef{ID: 20}, Status: StatusActive}
	st.rows[6] = &Delegation{ID: 6, User: PersonRef{ID: 3}, Delegate: PersonRef{ID: 20}, Status: StatusEnded}
	svc, _ := newTestService(st)
	ctx := context.Background()

	var nf *apperr.NotFoundError
	if err := svc.DeleteMine(ctx, agus, "5"); !errors.As(err, &nf) {
		t.Errorf("hapus delegasi orang lain: %v", err)
	}
	var state *apperr.InvalidStateError
	if _, err := svc.UpdateMine(ctx, agus, "6", in("20", "2026-11-12", "2026-11-15")); !errors.As(err, &state) {
		t.Errorf("ubah delegasi yang sudah selesai: %v", err)
	}
	// Admin boleh mengurus delegasi pegawai mana pun; pegawai biasa tidak bisa memakai endpoint admin
	if _, err := svc.Update(ctx, admin, "5", in("3", "2026-11-12", "2026-11-20")); err != nil {
		t.Errorf("admin ubah delegasi: %v", err)
	}
	if st.gotP.UserID != 9 || st.gotP.DelegateID != 3 {
		t.Errorf("pegawai jadwal berubah saat diubah: %+v", st.gotP)
	}
	if _, _, err := svc.List(ctx, agus, ListQuery{}); !errors.Is(err, apperr.ErrForbidden) {
		t.Errorf("pegawai membuka daftar admin: %v", err)
	}
	if _, err := svc.Create(ctx, admin, Input{UserID: "9", SubstituteID: "20", From: "2026-12-01", Until: "2026-12-03"}); err != nil {
		t.Errorf("admin buat delegasi: %v", err)
	}
	var invalid *apperr.ValidationError
	if _, _, err := svc.List(ctx, admin, ListQuery{Status: "LAMA"}); !errors.As(err, &invalid) {
		t.Errorf("filter status tidak dikenal: %v", err)
	}
}

func ptr(s string) *string { return &s }
