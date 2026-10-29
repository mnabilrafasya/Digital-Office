package surat

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// fakeTx menjalankan fungsi langsung tanpa database.
type fakeTx struct{}

func (fakeTx) InTx(_ context.Context, fn func(q db.DBTX) error) error { return fn(nil) }

// fakeLetterStore adalah repository surat palsu: data diatur lewat field, pemanggilan dicatat.
type fakeLetterStore struct {
	lock         *LetterLock
	participants map[int64]bool // untuk IsParticipant
	reviewers    []int64
	approverID   int64
	draft        *LetterDraft
	detail       *LetterDetail
	list         []LetterListItem

	created     *NewLetter
	updated     *LetterDraft
	applied     *Decision
	appliedNote *string
	reads       int
	filter      *LetterFilter
	published   *Publication
	versions    []DocumentVersion
}

func (f *fakeLetterStore) AddVersion(_ context.Context, v NewVersion) (int, []byte, error) {
	f.versions = append(f.versions, DocumentVersion{
		VersionNumber: len(f.versions) + 1, Type: v.Type, FileName: v.FileName, FileSize: v.FileSize,
		Checksum: v.Checksum, CreatedBy: PersonRef{ID: v.CreatedBy}, CreatedByRole: v.CreatedByRole, ObjectKey: v.ObjectKey,
	})
	return len(f.versions), testRowVersion, nil
}

func (f *fakeLetterStore) Versions(context.Context, int64) ([]DocumentVersion, error) {
	return f.versions, nil
}

func (f *fakeLetterStore) FindVersion(_ context.Context, _ int64, number int) (*DocumentVersion, error) {
	if number < 1 || number > len(f.versions) {
		return nil, db.ErrNotFound
	}
	v := f.versions[number-1]
	return &v, nil
}

func (f *fakeLetterStore) LatestVersion(_ context.Context, _ int64, versionType string) (*DocumentVersion, error) {
	for i := len(f.versions) - 1; i >= 0; i-- {
		if f.versions[i].Type == versionType {
			v := f.versions[i]
			return &v, nil
		}
	}
	return nil, db.ErrNotFound
}

func (f *fakeLetterStore) NextNumber(context.Context, int64, int) (int, string, string, error) {
	return 15, "TI", "", nil
}

func (f *fakeLetterStore) Publish(_ context.Context, _ int64, p Publication) error {
	f.published = &p
	return nil
}

func (f *fakeLetterStore) Create(_ context.Context, n NewLetter) (string, error) {
	f.created = &n
	return testLetterID, nil
}

func (f *fakeLetterStore) UpdateDraft(_ context.Context, _ int64, d *LetterDraft) error {
	f.updated = d
	return nil
}

func (f *fakeLetterStore) LoadDraft(context.Context, int64) (*LetterDraft, error) {
	return f.draft, nil
}

func (f *fakeLetterStore) LockByUUID(context.Context, string, string) (*LetterLock, error) {
	if f.lock == nil {
		return nil, db.ErrNotFound
	}
	return f.lock, nil
}

func (f *fakeLetterStore) FindByUUID(ctx context.Context, companyCode, uuid string) (*LetterLock, error) {
	return f.LockByUUID(ctx, companyCode, uuid)
}

func (f *fakeLetterStore) IsParticipant(_ context.Context, _, userID int64) (bool, error) {
	return f.participants[userID], nil
}

func (f *fakeLetterStore) Participants(context.Context, int64) ([]int64, int64, error) {
	return f.reviewers, f.approverID, nil
}

func (f *fakeLetterStore) ApplyTransition(_ context.Context, _, _ int64, d Decision, note *string) error {
	f.applied, f.appliedNote = &d, note
	return nil
}

func (f *fakeLetterStore) RecordRead(context.Context, int64, int64, string) error {
	f.reads++
	return nil
}

func (f *fakeLetterStore) FindDetail(context.Context, string, string) (*LetterDetail, error) {
	if f.detail == nil {
		return nil, db.ErrNotFound
	}
	d := *f.detail
	d.Steps = append([]LetterStep(nil), f.detail.Steps...)
	return &d, nil
}

func (f *fakeLetterStore) List(_ context.Context, filter LetterFilter) ([]LetterListItem, int, error) {
	f.filter = &filter
	return f.list, len(f.list), nil
}

func (f *fakeLetterStore) Actions(context.Context, int64) ([]LetterAction, error) {
	return []LetterAction{}, nil
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
	unitHeads     map[int64]int64   // unit -> pimpinan
	aliasMembers  map[int64][]int64 // level -> anggota
}

func (f *fakeRefs) UnitHeadIDs(context.Context, string, []int64) (map[int64]int64, error) {
	return f.unitHeads, nil
}

func (f *fakeRefs) AliasMemberIDs(context.Context, string, []int64) (map[int64][]int64, error) {
	return f.aliasMembers, nil
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
	if perm == auth.LetterApprove {
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
		return nil, db.ErrNotFound
	}
	return f.templateType, nil
}

func (f *fakeRefs) UnreachableRecipients(context.Context, string, []int64, []int64) ([]string, []string, error) {
	return f.headlessUnits, f.emptyAliases, nil
}

// fakeInbox mencatat surat yang dikirim ke inbox; holders adalah user yang memiliki surat di inbox.
type fakeInbox struct {
	to, cc  []int64
	holders map[int64]bool
}

func (f *fakeInbox) HasItem(_ context.Context, _, userID int64) (bool, error) {
	return f.holders[userID], nil
}

func (f *fakeInbox) DeliverDirect(_ context.Context, _ int64, to, cc []int64) error {
	f.to, f.cc = to, cc
	return nil
}

// fakeFiles menyimpan file naskah di memori.
type fakeFiles struct{ objects map[string][]byte }

func (f *fakeFiles) PutDocument(_ context.Context, key string, data []byte, _ string) error {
	if f.objects == nil {
		f.objects = map[string][]byte{}
	}
	f.objects[key] = data
	return nil
}

func (f *fakeFiles) OpenDocument(_ context.Context, key string) (io.ReadCloser, int64, error) {
	data, ok := f.objects[key]
	if !ok {
		return nil, 0, errors.New("objek tidak ada")
	}
	return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
}

// fakePDF "mengonversi" dengan menambah penanda, supaya urutan konversi lalu stempel bisa dicek.
type fakePDF struct {
	conversions int
	down        bool // layanan konversi mati
}

func (f *fakePDF) FromOffice(_ context.Context, _ string, data []byte) ([]byte, error) {
	if f.down {
		return nil, &apperr.UnavailableError{Message: "Layanan pembuat PDF sedang tidak tersedia"}
	}
	f.conversions++
	return append([]byte("%PDF:"), data...), nil
}

func (f *fakePDF) StampLastPage(_ context.Context, pdf, png []byte) ([]byte, error) {
	if !bytes.HasPrefix(png, []byte("\x89PNG")) {
		return nil, errors.New("bukan PNG")
	}
	return append(pdf, []byte("+QR")...), nil
}

// fakeAudit mencatat entri audit yang ditulis.
type fakeAudit struct{ entries []audit.Entry }

func (f *fakeAudit) Record(_ context.Context, e audit.Entry) error {
	f.entries = append(f.entries, e)
	return nil
}

func newTestLetterService(store *fakeLetterStore, refs *fakeRefs) *Service {
	svc, _ := newTestLetterServiceWithAudit(store, refs)
	return svc
}

func newTestLetterServiceWithAudit(store *fakeLetterStore, refs *fakeRefs) (*Service, *fakeAudit) {
	svc, audit, _ := newTestLetterServiceFull(store, refs)
	return svc, audit
}

func newTestLetterServiceFull(store *fakeLetterStore, refs *fakeRefs) (*Service, *fakeAudit, *fakeInbox) {
	audit, inbox := &fakeAudit{}, &fakeInbox{}
	return NewService(fakeTx{}, nil, Repos{
		Letters:   func(db.DBTX) Store { return store },
		Audit:     func(db.DBTX) AuditStore { return audit },
		Inbox:     func(db.DBTX) InboxStore { return inbox },
		Directory: refs,
		Lookups:   refs,
		Templates: refs,
		Files:     &fakeFiles{},
		PDF:       &fakePDF{},
	}, time.UTC), audit, inbox
}
