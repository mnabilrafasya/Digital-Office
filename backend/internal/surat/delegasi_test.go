package surat

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

// budiUser adalah staf tanpa hak approve yang menjadi pengganti delegasi.
var budiUser = &auth.CurrentUser{ID: 20, Name: "Budi", Company: auth.UserCompany{Code: "PSP"}, Permissions: []string{auth.LetterCreate}}

// delegatedTo membuat surat yang sedang menunggu assignee, dengan assignee mendelegasikan ke Budi.
func delegatedTo(status string, assignee int64) *fakeLetterStore {
	store := letterAt(status, assignee)
	store.delegations = map[int64]*Delegation{
		assignee: {UserID: assignee, Delegate: PersonRef{ID: 20, Name: "Budi"}, Until: "2026-11-15"},
	}
	return store
}

// Pengganti (staf) menyetujui sebagai approver terakhir atas nama Agus: surat terbit dengan Budi
// sebagai penandatangan, audit mencatat atas nama siapa, dan penerima yang sedang mendelegasikan
// ditembuskan ke penggantinya.
func TestDelegateApprovesOnBehalf(t *testing.T) {
	store := delegatedTo(StatusInApproval, 3)
	refs := newFakeRefs()
	refs.unitHeads = map[int64]int64{7: 50, 9: 51}
	refs.aliasMembers = map[int64][]int64{60: {52}}
	svc, audit, inbox := newTestLetterServiceFull(store, refs)
	ctx := context.Background()

	d, err := svc.Get(ctx, budiUser, testLetterID)
	if err != nil || d.OnBehalfOf == nil || d.OnBehalfOf.ID != 3 || d.DelegatedTo == nil ||
		d.DelegatedTo.Delegate.ID != 20 || !slices.Contains(d.AllowedActions, string(Approve)) {
		t.Fatalf("detail pengganti: err=%v detail=%+v", err, d)
	}
	if d, err := svc.Get(ctx, agusUser, testLetterID); err != nil || len(d.AllowedActions) != 0 || d.OnBehalfOf != nil {
		t.Errorf("detail pemilik tugas: err=%v actions=%v", err, d.AllowedActions)
	}

	if _, err := svc.Approve(ctx, budiUser, TransitionInput{LetterID: testLetterID}); err != nil {
		t.Fatalf("Approve pengganti: %v", err)
	}
	if store.applied == nil || store.applied.Action != Sign || store.appliedActor != 20 {
		t.Errorf("transisi = %+v oleh %d, want SIGN oleh Budi", store.applied, store.appliedActor)
	}
	if p := store.published; p == nil || p.SignerID != 20 {
		t.Errorf("penandatangan = %+v, want Budi", store.published)
	}
	sign := audit.entries[len(audit.entries)-1]
	if snap, _ := sign.Snapshot.(transitionSnapshot); sign.ActorID != 20 || snap.OnBehalfOfID != 3 {
		t.Errorf("audit SIGN = %+v", sign)
	}
	if !slices.Equal(inbox.copied, []int64{50, 52, 51}) {
		t.Errorf("tembusan pengganti diminta untuk %v, want semua penerima [50 52 51]", inbox.copied)
	}
}

// Selama delegasi, pemilik tugas tidak bisa bertindak; orang lain tetap bukan siapa-siapa.
func TestDelegatorCannotAct(t *testing.T) {
	store := delegatedTo(StatusInReview, 9)
	svc := newTestLetterService(store, newFakeRefs())
	ctx := context.Background()

	var state *apperr.InvalidStateError
	_, err := svc.Approve(ctx, fajarUser, TransitionInput{LetterID: testLetterID})
	if !errors.As(err, &state) || !strings.Contains(state.Message, "didelegasikan ke Budi sampai 15 November 2026") {
		t.Errorf("pemilik tugas bertindak: %v", err)
	}
	if store.applied != nil {
		t.Errorf("transisi tetap dijalankan: %+v", store.applied)
	}
	if _, err := svc.Return(ctx, budiUser, TransitionInput{LetterID: testLetterID, Note: ptr("Lengkapi data")}); err != nil {
		t.Fatalf("pengganti mengembalikan: %v", err)
	}
	if d := store.applied; d.To != StatusReturnedToDrafter || store.appliedActor != 20 || d.ReturnedByReviewer.UserID != 9 {
		t.Errorf("pengembalian pengganti = %+v oleh %d", d, store.appliedActor)
	}

	other := &auth.CurrentUser{ID: 77, Name: "Lain", Company: auth.UserCompany{Code: "PSP"}}
	if _, err := svc.Get(ctx, other, testLetterID); !errors.Is(err, ErrLetterNotFound) {
		t.Errorf("orang lain membuka surat: %v", err)
	}
}

// Pengganti tetap bisa membuka surat yang pernah ia kerjakan setelah delegasi selesai.
func TestDelegateKeepsAccessAfterward(t *testing.T) {
	store := letterAt(StatusSent, 0)
	svc := newTestLetterService(store, newFakeRefs())
	if _, err := svc.Get(context.Background(), budiUser, testLetterID); !errors.Is(err, ErrLetterNotFound) {
		t.Fatalf("sebelum pernah bertindak: %v", err)
	}
	store.acted = map[int64]bool{20: true}
	if _, err := svc.Get(context.Background(), budiUser, testLetterID); err != nil {
		t.Errorf("setelah pernah bertindak: %v", err)
	}
}

// Kewenangan pengganti mengikuti orang yang diwakili: bila approver tidak lagi berwenang, ditolak.
func TestDelegateUsesOwnerAuthority(t *testing.T) {
	store := delegatedTo(StatusInApproval, 3)
	refs := newFakeRefs()
	refs.approvers = map[int64]bool{} // Agus tidak lagi berwenang approve
	var state *apperr.InvalidStateError
	if _, err := newTestLetterService(store, refs).Approve(context.Background(), budiUser, TransitionInput{LetterID: testLetterID}); !errors.As(err, &state) {
		t.Errorf("pengganti tanpa kewenangan pemilik: %v", err)
	}
}

func TestMarkOnBehalf(t *testing.T) {
	andi, fajar, agus, budi := PersonRef{ID: 15}, PersonRef{ID: 9}, PersonRef{ID: 3}, PersonRef{ID: 20}
	acts := []LetterAction{
		{Action: "SUBMIT", Actor: andi, Target: &fajar},
		{Action: "READ", Actor: budi},
		{Action: "SAVE", Actor: budi},
		{Action: "APPROVE", Actor: budi, Target: &agus},
		{Action: "SIGN", Actor: agus},
	}
	markOnBehalf(acts)
	if acts[0].OnBehalfOf != nil || acts[1].OnBehalfOf != nil || acts[4].OnBehalfOf != nil {
		t.Errorf("aksi biasa ditandai pengganti: %+v", acts)
	}
	if acts[2].OnBehalfOf == nil || acts[2].OnBehalfOf.ID != 9 || acts[3].OnBehalfOf == nil || acts[3].OnBehalfOf.ID != 9 {
		t.Errorf("aksi pengganti tidak ditandai: %+v", acts)
	}
}

func TestApprovalQROnBehalf(t *testing.T) {
	d := &LetterDetail{LetterNumber: ptr("001/PSP/XI/2026"), Approval: &Approval{
		ApprovedBy: PersonRef{Name: "Budi", PositionName: "Staf"},
		OnBehalfOf: &PersonRef{Name: "Agus", PositionName: "VP SDM"},
	}}
	text := approvalQRText(d)
	if !strings.Contains(text, "Disetujui oleh: Budi a.n. Agus\nJabatan: VP SDM") {
		t.Errorf("teks QR = %q", text)
	}
}
