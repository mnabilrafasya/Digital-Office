package service

import (
	"context"
	"time"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/workflow"
)

// fakeTx menjalankan fungsi langsung tanpa database.
type fakeTx struct{}

func (fakeTx) InTx(_ context.Context, fn func(q db.DBTX) error) error { return fn(nil) }

// fakeLetterStore adalah repository surat palsu: data diatur lewat field, pemanggilan dicatat.
type fakeLetterStore struct {
	lock         *model.LetterLock
	participants map[int64]bool // untuk IsParticipant
	reviewers    []int64
	approverID   int64
	draft        *model.LetterDraft
	detail       *model.LetterDetail
	list         []model.LetterListItem

	created     *repository.NewLetter
	updated     *model.LetterDraft
	applied     *workflow.Decision
	appliedNote *string
	reads       int
	filter      *repository.LetterFilter
}

func (f *fakeLetterStore) Create(_ context.Context, n repository.NewLetter) (string, error) {
	f.created = &n
	return testLetterID, nil
}

func (f *fakeLetterStore) UpdateDraft(_ context.Context, _ int64, d *model.LetterDraft) error {
	f.updated = d
	return nil
}

func (f *fakeLetterStore) LoadDraft(context.Context, int64) (*model.LetterDraft, error) {
	return f.draft, nil
}

func (f *fakeLetterStore) LockByUUID(context.Context, string, string) (*model.LetterLock, error) {
	if f.lock == nil {
		return nil, repository.ErrNotFound
	}
	return f.lock, nil
}

func (f *fakeLetterStore) FindByUUID(ctx context.Context, companyCode, uuid string) (*model.LetterLock, error) {
	return f.LockByUUID(ctx, companyCode, uuid)
}

func (f *fakeLetterStore) IsParticipant(_ context.Context, _, userID int64) (bool, error) {
	return f.participants[userID], nil
}

func (f *fakeLetterStore) Participants(context.Context, int64) ([]int64, int64, error) {
	return f.reviewers, f.approverID, nil
}

func (f *fakeLetterStore) ApplyTransition(_ context.Context, _, _ int64, d workflow.Decision, note *string) error {
	f.applied, f.appliedNote = &d, note
	return nil
}

func (f *fakeLetterStore) RecordRead(context.Context, int64, int64, string) error {
	f.reads++
	return nil
}

func (f *fakeLetterStore) FindDetail(context.Context, string, string) (*model.LetterDetail, error) {
	if f.detail == nil {
		return nil, repository.ErrNotFound
	}
	d := *f.detail
	d.Steps = append([]model.LetterStep(nil), f.detail.Steps...)
	return &d, nil
}

func (f *fakeLetterStore) List(_ context.Context, filter repository.LetterFilter) ([]model.LetterListItem, int, error) {
	f.filter = &filter
	return f.list, len(f.list), nil
}

func (f *fakeLetterStore) Actions(context.Context, int64) ([]model.LetterAction, error) {
	return []model.LetterAction{}, nil
}

// fakeRefs menganggap semua data valid kecuali yang diatur sebaliknya.
type fakeRefs struct {
	lookups       map[int64]string // id -> kategori
	badUnits      map[int64]bool
	inactiveUsers map[int64]bool
	reviewers     map[int64]bool // user yang punya LETTER_REVIEW
	approvers     map[int64]bool // user yang punya LETTER_APPROVE
	templateType  *int64
	noTemplate    bool
	headlessUnits []string
	emptyAliases  []string
}

func newFakeRefs() *fakeRefs {
	return &fakeRefs{
		lookups:   map[int64]string{12: "LETTER_TYPE", 40: "CLASSIFICATION", 1: "CONFIDENTIALITY", 5: "URGENCY", 60: "POSITION_LEVEL"},
		reviewers: map[int64]bool{9: true, 4: true, 15: true},
		approvers: map[int64]bool{3: true, 15: true},
	}
}

func (f *fakeRefs) LookupCategories(_ context.Context, _ string, ids []int64) (map[int64]string, error) {
	out := map[int64]string{}
	for _, id := range ids {
		if c, ok := f.lookups[id]; ok {
			out[id] = c
		}
	}
	return out, nil
}

func (f *fakeRefs) ActiveUnitIDs(_ context.Context, _ string, ids []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	for _, id := range ids {
		out[id] = !f.badUnits[id]
	}
	return out, nil
}

func (f *fakeRefs) ActiveUserIDs(_ context.Context, _ string, ids []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	for _, id := range ids {
		out[id] = !f.inactiveUsers[id]
	}
	return out, nil
}

func (f *fakeRefs) EligibleUserIDs(_ context.Context, _, perm string, ids []int64) (map[int64]bool, error) {
	pool := f.reviewers
	if perm == permission.LetterApprove {
		pool = f.approvers
	}
	out := map[int64]bool{}
	for _, id := range ids {
		out[id] = pool[id] && !f.inactiveUsers[id]
	}
	return out, nil
}

func (f *fakeRefs) TemplateLetterType(context.Context, string, int64) (*int64, error) {
	if f.noTemplate {
		return nil, repository.ErrNotFound
	}
	return f.templateType, nil
}

func (f *fakeRefs) UnreachableRecipients(context.Context, string, []int64, []int64) ([]string, []string, error) {
	return f.headlessUnits, f.emptyAliases, nil
}

func (f *fakeRefs) ListAliases(context.Context, string) ([]model.Alias, error) { return nil, nil }

// fakeAudit mencatat entri audit yang ditulis.
type fakeAudit struct{ entries []repository.AuditEntry }

func (f *fakeAudit) Record(_ context.Context, e repository.AuditEntry) error {
	f.entries = append(f.entries, e)
	return nil
}

func newTestLetterService(store *fakeLetterStore, refs *fakeRefs) *LetterService {
	svc, _ := newTestLetterServiceWithAudit(store, refs)
	return svc
}

func newTestLetterServiceWithAudit(store *fakeLetterStore, refs *fakeRefs) (*LetterService, *fakeAudit) {
	audit := &fakeAudit{}
	return NewLetterService(fakeTx{}, nil, LetterRepos{
		Letters: func(db.DBTX) LetterStore { return store },
		Audit:   func(db.DBTX) AuditStore { return audit },
		Refs:    refs,
	}, time.UTC), audit
}
