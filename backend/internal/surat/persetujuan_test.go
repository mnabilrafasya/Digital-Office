package surat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

// Pemeran: drafter Andi (15), reviewer Fajar (9) lalu Dewi (4), approver Agus (3), orang luar (77).
var (
	andiUser     = &auth.CurrentUser{ID: 15, Name: "Andi", Company: auth.UserCompany{Code: "PSP"}, Permissions: []string{auth.LetterCreate}}
	fajarUser    = &auth.CurrentUser{ID: 9, Name: "Fajar", Company: auth.UserCompany{Code: "PSP"}, Permissions: []string{auth.LetterReview}}
	dewiUser     = &auth.CurrentUser{ID: 4, Name: "Dewi", Company: auth.UserCompany{Code: "PSP"}, Permissions: []string{auth.LetterReview}}
	agusUser     = &auth.CurrentUser{ID: 3, Name: "Agus", Company: auth.UserCompany{Code: "PSP"}, Permissions: []string{auth.LetterApprove}}
	outsiderUser = &auth.CurrentUser{ID: 77, Name: "Luar", Company: auth.UserCompany{Code: "PSP"}}
)

// letterAt menyiapkan surat lengkap (reviewer [9, 4], approver 3) pada status dan assignee tertentu.
func letterAt(status string, assignee, forwardedBy int64) *fakeLetterStore {
	d, _ := parseDraft(completeInput(), 15)
	var current *PersonRef
	if assignee != 0 {
		current = &PersonRef{ID: assignee}
	}
	return &fakeLetterStore{
		lock:       &LetterLock{ID: 1, DrafterID: 15, AssigneeID: assignee, ForwardedByID: forwardedBy, Status: status, RowVersion: testRowVersion},
		reviewers:  []int64{9, 4},
		approverID: 3,
		draft:      d,
		detail: &LetterDetail{
			ID: testLetterID, InternalID: 1, Status: status, Drafter: PersonRef{ID: 15},
			CurrentAssignee: current, ForwardedByID: forwardedBy,
			Steps: []LetterStep{
				{Role: RoleReviewer, Order: 1, User: StepUser{ID: 9}},
				{Role: RoleReviewer, Order: 2, User: StepUser{ID: 4}},
				{Role: RoleApprover, Order: 1, User: StepUser{ID: 3}},
			},
		},
	}
}

func TestSubmitRecordsTransitionAndAudit(t *testing.T) {
	store := letterAt(StatusDraft, 0, 0)
	svc, audit := newTestLetterServiceWithAudit(store, newFakeRefs())

	_, err := svc.Submit(context.Background(), andiUser, TransitionInput{LetterID: testLetterID, Note: ptr("diabaikan"), ClientIP: "10.0.0.5"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if store.applied == nil || store.applied.To != StatusInReview || store.applied.AssigneeID != 9 || store.appliedNote != nil {
		t.Errorf("transisi salah: %+v note=%v", store.applied, store.appliedNote)
	}
	if len(audit.entries) != 1 {
		t.Fatalf("audit = %d entri, want 1", len(audit.entries))
	}
	e := audit.entries[0]
	snap, _ := e.Snapshot.(transitionSnapshot)
	if e.Action != "SUBMIT" || e.EntityType != "LETTER" || e.EntityUUID != testLetterID || e.ActorID != 15 ||
		e.ActorName != "Andi" || e.IPAddress != "10.0.0.5" || snap.From != StatusDraft || snap.AssigneeID != 9 {
		t.Errorf("audit salah: %+v", e)
	}
}

func TestReturnPassesNote(t *testing.T) {
	store := letterAt(StatusInApproval, 3, 4)
	if _, err := newTestLetterService(store, newFakeRefs()).Return(context.Background(), agusUser,
		TransitionInput{LetterID: testLetterID, Note: ptr("  Perbaiki lampiran  ")}); err != nil {
		t.Fatalf("Return: %v", err)
	}
	if store.applied.To != StatusReturnedToReviewer || store.applied.AssigneeID != 4 || *store.appliedNote != "Perbaiki lampiran" {
		t.Errorf("transisi salah: %+v note=%q", store.applied, *store.appliedNote)
	}
}

func TestTransitionRejected(t *testing.T) {
	type run func(svc *Service) error
	submit := func(u *auth.CurrentUser) run {
		return func(svc *Service) error {
			_, err := svc.Submit(context.Background(), u, TransitionInput{LetterID: testLetterID})
			return err
		}
	}
	approve := func(u *auth.CurrentUser) run {
		return func(svc *Service) error {
			_, err := svc.Approve(context.Background(), u, TransitionInput{LetterID: testLetterID})
			return err
		}
	}
	returnWith := func(u *auth.CurrentUser, note *string) run {
		return func(svc *Service) error {
			_, err := svc.Return(context.Background(), u, TransitionInput{LetterID: testLetterID, Note: note})
			return err
		}
	}
	isState := func(want string) func(error) bool {
		return func(err error) bool {
			var e *apperr.InvalidStateError
			return errors.As(err, &e) && strings.Contains(e.Message, want)
		}
	}
	isValidation := func(want string) func(error) bool {
		return func(err error) bool {
			var e *apperr.ValidationError
			return errors.As(err, &e) && strings.Contains(e.Message, want)
		}
	}
	notFound := func(err error) bool { return errors.Is(err, ErrLetterNotFound) }

	tests := []struct {
		name  string
		store *fakeLetterStore
		refs  func(r *fakeRefs)
		run   run
		check func(error) bool
	}{
		{"bukan peserta", letterAt(StatusDraft, 0, 0), nil, submit(outsiderUser), notFound},
		{"surat tidak ada", &fakeLetterStore{}, nil, submit(andiUser), notFound},
		{"submit dua kali", letterAt(StatusInReview, 9, 0), nil, submit(andiUser), isState("sudah diproses")},
		{"bukan giliran reviewer", letterAt(StatusInReview, 4, 9), nil, approve(fajarUser), isState("bukan giliran")},
		{"reviewer mencoba submit", letterAt(StatusDraft, 0, 0), nil, submit(fajarUser), isState("bukan giliran")},
		{"permission review dicabut", letterAt(StatusInReview, 9, 0), nil,
			approve(&auth.CurrentUser{ID: 9, Company: auth.UserCompany{Code: "PSP"}}),
			func(err error) bool { return errors.Is(err, apperr.ErrForbidden) }},
		{"submit belum lengkap", func() *fakeLetterStore {
			s := letterAt(StatusDraft, 0, 0)
			s.draft.Title = nil
			return s
		}(), nil, submit(andiUser), isValidation("perihal")},
		{"submit unit tanpa pimpinan", letterAt(StatusDraft, 0, 0),
			func(r *fakeRefs) { r.headlessUnits = []string{"Departemen SDM"} }, submit(andiUser), isState("Departemen SDM")},
		{"submit alias tanpa anggota", letterAt(StatusDraft, 0, 0),
			func(r *fakeRefs) { r.emptyAliases = []string{"Seluruh Staf"} }, submit(andiUser), isState("Seluruh Staf")},
		{"submit reviewer sudah nonaktif", letterAt(StatusDraft, 0, 0),
			func(r *fakeRefs) { r.inactiveUsers = map[int64]bool{4: true} }, submit(andiUser), isValidation("Reviewer tidak ditemukan")},
		{"reviewer berikutnya nonaktif", letterAt(StatusInReview, 9, 0),
			func(r *fakeRefs) { r.inactiveUsers = map[int64]bool{4: true} }, approve(fajarUser), isState("Reviewer berikutnya")},
		{"drafter nonaktif saat dikembalikan", letterAt(StatusInReview, 9, 0),
			func(r *fakeRefs) { r.inactiveUsers = map[int64]bool{15: true} }, returnWith(fajarUser, ptr("revisi")), isState("Pembuat surat")},
		{"return tanpa catatan", letterAt(StatusInReview, 9, 0), nil, returnWith(fajarUser, ptr("   ")), isValidation("Catatan wajib")},
		{"catatan terlalu panjang", letterAt(StatusInReview, 9, 0), nil,
			returnWith(fajarUser, ptr(strings.Repeat("a", 1001))), isValidation("maksimal 1000")},
	}
	for _, tt := range tests {
		refs := newFakeRefs()
		if tt.refs != nil {
			tt.refs(refs)
		}
		err := tt.run(newTestLetterService(tt.store, refs))
		if !tt.check(err) {
			t.Errorf("%s: error = %v", tt.name, err)
		}
		if tt.store.applied != nil {
			t.Errorf("%s: transisi tetap diterapkan", tt.name)
		}
	}
}

func TestGetRecordsReadForAssignee(t *testing.T) {
	store := letterAt(StatusInReview, 9, 0)
	svc := newTestLetterService(store, newFakeRefs())
	ctx := context.Background()

	d, err := svc.Get(ctx, fajarUser, testLetterID)
	if err != nil || store.reads != 1 {
		t.Fatalf("assignee membuka surat: err=%v reads=%d, want 1", err, store.reads)
	}
	if got := strings.Join(d.AllowedActions, ","); got != "APPROVE,RETURN" {
		t.Errorf("allowedActions Fajar = %s, want APPROVE,RETURN", got)
	}
	if d.Steps[0].State != StepActive || d.Steps[1].State != StepPending {
		t.Errorf("state tahap salah: %+v", d.Steps)
	}

	if _, err := svc.Get(ctx, andiUser, testLetterID); err != nil || store.reads != 1 {
		t.Errorf("drafter membuka surat tidak boleh mencatat READ: err=%v reads=%d", err, store.reads)
	}
	if d, _ := svc.Get(ctx, &auth.CurrentUser{ID: 9}, testLetterID); len(d.AllowedActions) != 0 {
		t.Errorf("tanpa LETTER_REVIEW tombol harus disembunyikan: %v", d.AllowedActions)
	}
}

func TestPresentLastReturnNote(t *testing.T) {
	for _, tc := range []struct {
		status string
		want   bool
	}{{StatusReturnedToDrafter, true}, {StatusInReview, false}} {
		store := letterAt(tc.status, 15, 0)
		store.detail.LastReturnNote = ptr("revisi")
		d, err := newTestLetterService(store, newFakeRefs()).Get(context.Background(), andiUser, testLetterID)
		if err != nil || (d.LastReturnNote != nil) != tc.want {
			t.Errorf("%s: err=%v lastReturnNote=%v, want ada=%v", tc.status, err, d.LastReturnNote, tc.want)
		}
	}
}

func TestLetterList(t *testing.T) {
	ctx := context.Background()
	page := func(scope, status string) LetterListQuery {
		return LetterListQuery{Scope: scope, Status: status, Page: 2, PageSize: 10}
	}

	store := &fakeLetterStore{list: []LetterListItem{{Status: StatusInReview, AssigneeStep: 2}}}
	items, _, err := newTestLetterService(store, newFakeRefs()).List(ctx, andiUser, page("", ""))
	if err != nil || store.filter.DrafterID != 15 || store.filter.AssigneeID != 0 || store.filter.Offset != 10 {
		t.Errorf("mine: err=%v filter=%+v", err, store.filter)
	}
	if cs := items[0].CurrentStep; cs == nil || cs.Role != RoleReviewer || cs.Order != 2 {
		t.Errorf("currentStep = %+v, want REVIEWER 2", cs)
	}

	store = &fakeLetterStore{}
	_, _, err = newTestLetterService(store, newFakeRefs()).List(ctx, fajarUser, page(ScopeTasks, "in_review, SENT"))
	if err != nil || store.filter.AssigneeID != 9 || strings.Join(store.filter.Statuses, ",") != StatusInReview {
		t.Errorf("tasks: err=%v filter=%+v", err, store.filter)
	}

	store = &fakeLetterStore{}
	if items, total, err := newTestLetterService(store, newFakeRefs()).List(ctx, fajarUser, page(ScopeTasks, "DRAFT")); err != nil ||
		len(items) != 0 || total != 0 || store.filter != nil {
		t.Errorf("tasks berstatus DRAFT harus kosong tanpa query: err=%v filter=%+v", err, store.filter)
	}

	for _, q := range []LetterListQuery{page("semua", ""), page("", "SELESAI")} {
		var ve *apperr.ValidationError
		if _, _, err := newTestLetterService(&fakeLetterStore{}, newFakeRefs()).List(ctx, andiUser, q); !errors.As(err, &ve) {
			t.Errorf("%+v: error = %v, want apperr.ValidationError", q, err)
		}
	}
}

func TestHistoryAccess(t *testing.T) {
	store := letterAt(StatusInReview, 9, 0)
	store.participants = map[int64]bool{9: true}
	svc := newTestLetterService(store, newFakeRefs())
	for _, tc := range []struct {
		user *auth.CurrentUser
		ok   bool
	}{{andiUser, true}, {fajarUser, true}, {outsiderUser, false}} {
		_, err := svc.History(context.Background(), tc.user, testLetterID)
		if (err == nil) != tc.ok || (!tc.ok && !errors.Is(err, ErrLetterNotFound)) {
			t.Errorf("user %d: error = %v, want ok=%v", tc.user.ID, err, tc.ok)
		}
	}
}
