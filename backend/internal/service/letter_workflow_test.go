package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
)

// Pemeran: drafter Andi (15), reviewer Fajar (9) lalu Dewi (4), approver Agus (3), orang luar (77).
var (
	andiUser  = &model.CurrentUser{ID: 15, Name: "Andi", Company: model.UserCompany{Code: "PSP"}, Permissions: []string{permission.LetterCreate}}
	fajarUser = &model.CurrentUser{ID: 9, Name: "Fajar", Company: model.UserCompany{Code: "PSP"}, Permissions: []string{permission.LetterReview}}
	dewiUser  = &model.CurrentUser{ID: 4, Name: "Dewi", Company: model.UserCompany{Code: "PSP"}, Permissions: []string{permission.LetterReview}}
	agusUser  = &model.CurrentUser{ID: 3, Name: "Agus", Company: model.UserCompany{Code: "PSP"}, Permissions: []string{permission.LetterApprove}}
	outsider  = &model.CurrentUser{ID: 77, Name: "Luar", Company: model.UserCompany{Code: "PSP"}}
)

// letterAt menyiapkan surat lengkap (reviewer [9, 4], approver 3) pada status dan assignee tertentu.
func letterAt(status string, assignee, forwardedBy int64) *fakeLetterStore {
	d, _ := parseDraft(completeInput(), 15)
	var current *model.PersonRef
	if assignee != 0 {
		current = &model.PersonRef{ID: assignee}
	}
	return &fakeLetterStore{
		lock:       &model.LetterLock{ID: 1, DrafterID: 15, AssigneeID: assignee, ForwardedByID: forwardedBy, Status: status, RowVersion: testRowVersion},
		reviewers:  []int64{9, 4},
		approverID: 3,
		draft:      d,
		detail: &model.LetterDetail{
			ID: testLetterID, InternalID: 1, Status: status, Drafter: model.PersonRef{ID: 15},
			CurrentAssignee: current, ForwardedByID: forwardedBy,
			Steps: []model.LetterStep{
				{Role: model.RoleReviewer, Order: 1, User: model.StepUser{ID: 9}},
				{Role: model.RoleReviewer, Order: 2, User: model.StepUser{ID: 4}},
				{Role: model.RoleApprover, Order: 1, User: model.StepUser{ID: 3}},
			},
		},
	}
}

func TestSubmitRecordsTransitionAndAudit(t *testing.T) {
	store := letterAt(model.StatusDraft, 0, 0)
	svc, audit := newTestLetterServiceWithAudit(store, newFakeRefs())

	_, err := svc.Submit(context.Background(), andiUser, TransitionInput{LetterID: testLetterID, Note: ptr("diabaikan"), ClientIP: "10.0.0.5"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if store.applied == nil || store.applied.To != model.StatusInReview || store.applied.AssigneeID != 9 || store.appliedNote != nil {
		t.Errorf("transisi salah: %+v note=%v", store.applied, store.appliedNote)
	}
	if len(audit.entries) != 1 {
		t.Fatalf("audit = %d entri, want 1", len(audit.entries))
	}
	e := audit.entries[0]
	snap, _ := e.Snapshot.(transitionSnapshot)
	if e.Action != "SUBMIT" || e.EntityType != "LETTER" || e.EntityUUID != testLetterID || e.ActorID != 15 ||
		e.ActorName != "Andi" || e.IPAddress != "10.0.0.5" || snap.From != model.StatusDraft || snap.AssigneeID != 9 {
		t.Errorf("audit salah: %+v", e)
	}
}

func TestReturnPassesNote(t *testing.T) {
	store := letterAt(model.StatusInApproval, 3, 4)
	if _, err := newTestLetterService(store, newFakeRefs()).Return(context.Background(), agusUser,
		TransitionInput{LetterID: testLetterID, Note: ptr("  Perbaiki lampiran  ")}); err != nil {
		t.Fatalf("Return: %v", err)
	}
	if store.applied.To != model.StatusReturnedToReviewer || store.applied.AssigneeID != 4 || *store.appliedNote != "Perbaiki lampiran" {
		t.Errorf("transisi salah: %+v note=%q", store.applied, *store.appliedNote)
	}
}

func TestTransitionRejected(t *testing.T) {
	type run func(svc *LetterService) error
	submit := func(u *model.CurrentUser) run {
		return func(svc *LetterService) error {
			_, err := svc.Submit(context.Background(), u, TransitionInput{LetterID: testLetterID})
			return err
		}
	}
	approve := func(u *model.CurrentUser) run {
		return func(svc *LetterService) error {
			_, err := svc.Approve(context.Background(), u, TransitionInput{LetterID: testLetterID})
			return err
		}
	}
	returnWith := func(u *model.CurrentUser, note *string) run {
		return func(svc *LetterService) error {
			_, err := svc.Return(context.Background(), u, TransitionInput{LetterID: testLetterID, Note: note})
			return err
		}
	}
	isState := func(want string) func(error) bool {
		return func(err error) bool {
			var e *InvalidStateError
			return errors.As(err, &e) && strings.Contains(e.Message, want)
		}
	}
	isValidation := func(want string) func(error) bool {
		return func(err error) bool {
			var e *ValidationError
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
		{"bukan peserta", letterAt(model.StatusDraft, 0, 0), nil, submit(outsider), notFound},
		{"surat tidak ada", &fakeLetterStore{}, nil, submit(andiUser), notFound},
		{"submit dua kali", letterAt(model.StatusInReview, 9, 0), nil, submit(andiUser), isState("sudah diproses")},
		{"bukan giliran reviewer", letterAt(model.StatusInReview, 4, 9), nil, approve(fajarUser), isState("bukan giliran")},
		{"reviewer mencoba submit", letterAt(model.StatusDraft, 0, 0), nil, submit(fajarUser), isState("bukan giliran")},
		{"permission review dicabut", letterAt(model.StatusInReview, 9, 0), nil,
			approve(&model.CurrentUser{ID: 9, Company: model.UserCompany{Code: "PSP"}}),
			func(err error) bool { return errors.Is(err, ErrForbidden) }},
		{"submit belum lengkap", func() *fakeLetterStore {
			s := letterAt(model.StatusDraft, 0, 0)
			s.draft.Title = nil
			return s
		}(), nil, submit(andiUser), isValidation("perihal")},
		{"submit unit tanpa pimpinan", letterAt(model.StatusDraft, 0, 0),
			func(r *fakeRefs) { r.headlessUnits = []string{"Departemen SDM"} }, submit(andiUser), isState("Departemen SDM")},
		{"submit alias tanpa anggota", letterAt(model.StatusDraft, 0, 0),
			func(r *fakeRefs) { r.emptyAliases = []string{"Seluruh Staf"} }, submit(andiUser), isState("Seluruh Staf")},
		{"submit reviewer sudah nonaktif", letterAt(model.StatusDraft, 0, 0),
			func(r *fakeRefs) { r.inactiveUsers = map[int64]bool{4: true} }, submit(andiUser), isValidation("Reviewer tidak ditemukan")},
		{"reviewer berikutnya nonaktif", letterAt(model.StatusInReview, 9, 0),
			func(r *fakeRefs) { r.inactiveUsers = map[int64]bool{4: true} }, approve(fajarUser), isState("Reviewer berikutnya")},
		{"drafter nonaktif saat dikembalikan", letterAt(model.StatusInReview, 9, 0),
			func(r *fakeRefs) { r.inactiveUsers = map[int64]bool{15: true} }, returnWith(fajarUser, ptr("revisi")), isState("Pembuat surat")},
		{"return tanpa catatan", letterAt(model.StatusInReview, 9, 0), nil, returnWith(fajarUser, ptr("   ")), isValidation("Catatan wajib")},
		{"catatan terlalu panjang", letterAt(model.StatusInReview, 9, 0), nil,
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
	store := letterAt(model.StatusInReview, 9, 0)
	svc := newTestLetterService(store, newFakeRefs())
	ctx := context.Background()

	d, err := svc.Get(ctx, fajarUser, testLetterID)
	if err != nil || store.reads != 1 {
		t.Fatalf("assignee membuka surat: err=%v reads=%d, want 1", err, store.reads)
	}
	if got := strings.Join(d.AllowedActions, ","); got != "APPROVE,RETURN" {
		t.Errorf("allowedActions Fajar = %s, want APPROVE,RETURN", got)
	}
	if d.Steps[0].State != model.StepActive || d.Steps[1].State != model.StepPending {
		t.Errorf("state tahap salah: %+v", d.Steps)
	}

	if _, err := svc.Get(ctx, andiUser, testLetterID); err != nil || store.reads != 1 {
		t.Errorf("drafter membuka surat tidak boleh mencatat READ: err=%v reads=%d", err, store.reads)
	}
	if d, _ := svc.Get(ctx, &model.CurrentUser{ID: 9}, testLetterID); len(d.AllowedActions) != 0 {
		t.Errorf("tanpa LETTER_REVIEW tombol harus disembunyikan: %v", d.AllowedActions)
	}
}

func TestPresentLastReturnNote(t *testing.T) {
	for _, tc := range []struct {
		status string
		want   bool
	}{{model.StatusReturnedToDrafter, true}, {model.StatusInReview, false}} {
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

	store := &fakeLetterStore{list: []model.LetterListItem{{Status: model.StatusInReview, AssigneeStep: 2}}}
	items, _, err := newTestLetterService(store, newFakeRefs()).List(ctx, andiUser, page("", ""))
	if err != nil || store.filter.DrafterID != 15 || store.filter.AssigneeID != 0 || store.filter.Offset != 10 {
		t.Errorf("mine: err=%v filter=%+v", err, store.filter)
	}
	if cs := items[0].CurrentStep; cs == nil || cs.Role != model.RoleReviewer || cs.Order != 2 {
		t.Errorf("currentStep = %+v, want REVIEWER 2", cs)
	}

	store = &fakeLetterStore{}
	_, _, err = newTestLetterService(store, newFakeRefs()).List(ctx, fajarUser, page(ScopeTasks, "in_review, SENT"))
	if err != nil || store.filter.AssigneeID != 9 || strings.Join(store.filter.Statuses, ",") != model.StatusInReview {
		t.Errorf("tasks: err=%v filter=%+v", err, store.filter)
	}

	store = &fakeLetterStore{}
	if items, total, err := newTestLetterService(store, newFakeRefs()).List(ctx, fajarUser, page(ScopeTasks, "DRAFT")); err != nil ||
		len(items) != 0 || total != 0 || store.filter != nil {
		t.Errorf("tasks berstatus DRAFT harus kosong tanpa query: err=%v filter=%+v", err, store.filter)
	}

	for _, q := range []LetterListQuery{page("semua", ""), page("", "SELESAI")} {
		var ve *ValidationError
		if _, _, err := newTestLetterService(&fakeLetterStore{}, newFakeRefs()).List(ctx, andiUser, q); !errors.As(err, &ve) {
			t.Errorf("%+v: error = %v, want ValidationError", q, err)
		}
	}
}

func TestHistoryAccess(t *testing.T) {
	store := letterAt(model.StatusInReview, 9, 0)
	store.participants = map[int64]bool{9: true}
	svc := newTestLetterService(store, newFakeRefs())
	for _, tc := range []struct {
		user *model.CurrentUser
		ok   bool
	}{{andiUser, true}, {fajarUser, true}, {outsider, false}} {
		_, err := svc.History(context.Background(), tc.user, testLetterID)
		if (err == nil) != tc.ok || (!tc.ok && !errors.Is(err, ErrLetterNotFound)) {
			t.Errorf("user %d: error = %v, want ok=%v", tc.user.ID, err, tc.ok)
		}
	}
}
