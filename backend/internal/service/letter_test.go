package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/workflow"
)

func ptr[T any](v T) *T { return &v }

// ---- Fake repository untuk uji otorisasi ----

type fakeTx struct{}

func (fakeTx) InTx(_ context.Context, fn func(q db.DBTX) error) error { return fn(nil) }

// fakeLetters hanya mengimplementasikan Get/GetForUpdate; method lain panic bila terpanggil
// (artinya alur lolos pemeriksaan yang seharusnya menolak).
type fakeLetters struct {
	LetterStore
	letter *model.Letter
}

func (f *fakeLetters) Get(_ context.Context, company, uuid string) (*model.Letter, error) {
	if f.letter == nil || f.letter.CompanyCode != company || f.letter.UUID != uuid {
		return nil, repository.ErrNotFound
	}
	l := *f.letter
	return &l, nil
}

func (f *fakeLetters) GetForUpdate(ctx context.Context, company, uuid string) (*model.Letter, error) {
	return f.Get(ctx, company, uuid)
}

type fakeParticipants struct {
	ParticipantStore
	parts []model.Participant
}

func (f *fakeParticipants) List(context.Context, int64) ([]model.Participant, error) {
	return f.parts, nil
}

const testUUID = "6f1c2a9e-0000-4000-8000-000000000001"

// Peserta surat uji: drafter 1, reviewer 2 lalu 3, approver 4; orang luar 9 (PSP) dan 20 (PKG).
func newTestService(status workflow.Status, assignee int64) *LetterService {
	letter := &model.Letter{
		ID: 100, UUID: testUUID, CompanyCode: "PSP", Status: string(status),
		DrafterID: 1, SenderUnitID: 7, RowVersion: []byte{0, 0, 0, 0, 0, 0, 0, 1},
	}
	if assignee != 0 {
		letter.AssigneeID = ptr(assignee)
	}
	repos := LetterRepos{
		Letters: &fakeLetters{letter: letter},
		Participants: &fakeParticipants{parts: []model.Participant{
			{ID: 1, UserID: 2, Role: model.RoleReviewer, StepOrder: 1},
			{ID: 2, UserID: 3, Role: model.RoleReviewer, StepOrder: 2},
			{ID: 3, UserID: 4, Role: model.RoleApprover, StepOrder: 1},
		}},
	}
	return NewLetterService(LetterDeps{Tx: fakeTx{}, Repos: func(db.DBTX) LetterRepos { return repos }, Read: repos})
}

func user(id int64, company string, perms ...string) *model.CurrentUser {
	return &model.CurrentUser{ID: id, Name: "Uji", Company: model.UserCompany{Code: company}, Permissions: perms}
}

func TestTransitionAuthorization(t *testing.T) {
	ctx := context.Background()
	reviewer := []string{permission.LetterReview}
	approver := []string{permission.LetterApprove}

	tests := []struct {
		name     string
		status   workflow.Status
		assignee int64
		actor    *model.CurrentUser
		action   workflow.Action
		note     *string
		check    func(error) bool
	}{
		{"bukan peserta -> tidak ditemukan", workflow.StatusInReview, 2, user(9, "PSP", reviewer...),
			workflow.ActionApprove, nil, func(err error) bool { return errors.Is(err, ErrNotFound) }},
		{"perusahaan lain -> tidak ditemukan", workflow.StatusInReview, 2, user(20, "PKG", reviewer...),
			workflow.ActionApprove, nil, func(err error) bool { return errors.Is(err, ErrNotFound) }},
		{"submit dua kali -> INVALID_STATE", workflow.StatusInReview, 2, user(1, "PSP", permission.LetterCreate),
			workflow.ActionSubmit, nil, isStateError},
		{"reviewer belum giliran -> INVALID_STATE", workflow.StatusInReview, 2, user(3, "PSP", reviewer...),
			workflow.ActionApprove, nil, isStateError},
		{"reviewer tanda tangan -> INVALID_STATE", workflow.StatusInApproval, 4, user(2, "PSP", reviewer...),
			workflow.ActionSign, nil, isStateError},
		{"reviewer tanpa LETTER_REVIEW -> FORBIDDEN", workflow.StatusInReview, 2, user(2, "PSP"),
			workflow.ActionApprove, nil, func(err error) bool { return errors.Is(err, ErrForbidden) }},
		{"approver tanpa LETTER_APPROVE -> FORBIDDEN", workflow.StatusInApproval, 4, user(4, "PSP", reviewer...),
			workflow.ActionSign, nil, func(err error) bool { return errors.Is(err, ErrForbidden) }},
		{"kembalikan tanpa catatan -> VALIDATION", workflow.StatusInApproval, 4, user(4, "PSP", approver...),
			workflow.ActionReturn, ptr("   "), isValidationError},
		{"catatan terlalu panjang -> VALIDATION", workflow.StatusInReview, 2, user(2, "PSP", reviewer...),
			workflow.ActionApprove, ptr(strings.Repeat("a", 1001)), isValidationError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(tt.status, tt.assignee)
			_, err := svc.Transition(ctx, tt.actor, testUUID, TransitionInput{Action: tt.action, Note: tt.note})
			if !tt.check(err) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestUpdateAuthorization(t *testing.T) {
	ctx := context.Background()
	staleETag := []byte{0, 0, 0, 0, 0, 0, 0, 9}

	// Bukan drafter (reviewer) -> FORBIDDEN
	svc := newTestService(workflow.StatusDraft, 0)
	if _, err := svc.Update(ctx, user(2, "PSP"), testUUID, nil, LetterInput{}); !errors.Is(err, ErrForbidden) {
		t.Errorf("reviewer mengedit: err = %v, want ErrForbidden", err)
	}
	// Surat sudah dikirim -> INVALID_STATE
	svc = newTestService(workflow.StatusInReview, 2)
	if _, err := svc.Update(ctx, user(1, "PSP"), testUUID, nil, LetterInput{}); !isStateError(err) {
		t.Errorf("edit saat review: err = %v, want StateError", err)
	}
	// ETag lama -> VERSION_CONFLICT dengan versi terbaru
	svc = newTestService(workflow.StatusDraft, 0)
	var vc *VersionConflictError
	_, err := svc.Update(ctx, user(1, "PSP"), testUUID, staleETag, LetterInput{})
	if !errors.As(err, &vc) || vc.Current[7] != 1 {
		t.Errorf("ETag lama: err = %v, want VersionConflictError berisi versi terbaru", err)
	}
	// Orang luar -> tidak ditemukan
	if _, err := svc.Update(ctx, user(9, "PSP"), testUUID, nil, LetterInput{}); !errors.Is(err, ErrNotFound) {
		t.Errorf("orang luar: err = %v, want ErrNotFound", err)
	}
}

func isStateError(err error) bool {
	var se *StateError
	return errors.As(err, &se)
}

func isValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

// ---- Fungsi murni ----

func TestNormalizeInput(t *testing.T) {
	d, err := normalizeInput(LetterInput{
		Title: ptr("  Undangan Rapat  "), Summary: ptr("   "),
		ReviewerIDs: []int64{9, 4}, ApproverID: ptr(int64(3)),
		To: []RecipientInput{{UnitID: ptr(int64(7))}, {AliasLevelID: ptr(int64(60))}},
		CC: []RecipientInput{{UnitID: ptr(int64(9))}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if *d.content.Title != "Undangan Rapat" || d.content.Summary != nil || d.approverID != 3 {
		t.Errorf("hasil normalisasi salah: %+v", d)
	}

	invalidInputs := map[string]LetterInput{
		"perihal 251 karakter":    {Title: ptr(strings.Repeat("a", 251))},
		"keterangan 501 karakter": {SpecialNote: ptr(strings.Repeat("a", 501))},
		"ringkasan 1001 karakter": {Summary: ptr(strings.Repeat("a", 1001))},
		"reviewer ganda":          {ReviewerIDs: []int64{9, 9}},
		"penerima tanpa target":   {To: []RecipientInput{{}}},
		"penerima dua target":     {To: []RecipientInput{{UnitID: ptr(int64(7)), AliasLevelID: ptr(int64(60))}}},
		"unit di TO dan CC":       {To: []RecipientInput{{UnitID: ptr(int64(7))}}, CC: []RecipientInput{{UnitID: ptr(int64(7))}}},
		"alias muncul dua kali":   {To: []RecipientInput{{AliasLevelID: ptr(int64(60))}}, CC: []RecipientInput{{AliasLevelID: ptr(int64(60))}}},
	}
	for name, in := range invalidInputs {
		if _, err := normalizeInput(in); !isValidationError(err) {
			t.Errorf("%s: err = %v, want ValidationError", name, err)
		}
	}
	// Batas tepat masih diterima
	if _, err := normalizeInput(LetterInput{Title: ptr(strings.Repeat("é", 250))}); err != nil {
		t.Errorf("perihal 250 karakter (multibyte) ditolak: %v", err)
	}
}

func TestDiffParticipants(t *testing.T) {
	existing := []model.Participant{
		{ID: 1, UserID: 9, Role: model.RoleReviewer, StepOrder: 1},
		{ID: 2, UserID: 4, Role: model.RoleReviewer, StepOrder: 2},
		{ID: 3, UserID: 3, Role: model.RoleApprover, StepOrder: 1},
	}
	// Urutan reviewer ditukar, approver tetap
	desired := draft{reviewers: []int64{4, 9}, approverID: 3}.participants()
	del, add := diffParticipants(existing, desired)
	if !slices.Equal(del, []int64{1, 2}) || len(add) != 2 || add[0].UserID != 4 || add[0].StepOrder != 1 {
		t.Errorf("tukar urutan: del=%v add=%+v", del, add)
	}

	// Tidak ada perubahan -> tidak ada baris yang disentuh
	del, add = diffParticipants(existing, draft{reviewers: []int64{9, 4}, approverID: 3}.participants())
	if len(del) != 0 || len(add) != 0 {
		t.Errorf("tanpa perubahan: del=%v add=%v", del, add)
	}
}

func TestDiffRecipients(t *testing.T) {
	existing := []model.Recipient{
		{ID: 10, Type: model.RecipientTo, UnitID: ptr(int64(7)), SortOrder: 0},
		{ID: 11, Type: model.RecipientCC, UnitID: ptr(int64(9)), SortOrder: 0},
	}
	d := draft{
		to: []RecipientInput{{UnitID: ptr(int64(7))}, {AliasLevelID: ptr(int64(60))}},
		cc: nil,
	}
	del, add := diffRecipients(existing, d.recipients())
	if !slices.Equal(del, []int64{11}) || len(add) != 1 || *add[0].AliasLevelID != 60 || add[0].SortOrder != 1 {
		t.Errorf("del=%v add=%+v", del, add)
	}
}

// Skenario Lampiran B no. 18: TO = unit SDM + alias VP, CC = unit AK.
// VP SDM (user 15) terkena TO dua kali -> 1 baris TO. VP AK (user 17) terkena TO (alias) dan CC (unit) -> TO.
func TestInboxEntriesDedupe(t *testing.T) {
	const unitSDM, unitAK, aliasVP = 7, 8, 60
	recips := []model.Recipient{
		{Type: model.RecipientTo, UnitID: ptr(int64(unitSDM))},
		{Type: model.RecipientTo, AliasLevelID: ptr(int64(aliasVP))},
		{Type: model.RecipientCC, UnitID: ptr(int64(unitAK))},
	}
	heads := map[int64]int64{unitSDM: 15, unitAK: 17}
	members := map[int64][]int64{aliasVP: {15, 17}}

	got := inboxEntries(recips, heads, members)
	want := []model.InboxEntry{{UserID: 15, RecipientType: "TO"}, {UserID: 17, RecipientType: "TO"}}
	if !slices.Equal(got, want) {
		t.Errorf("inboxEntries() = %+v, want %+v", got, want)
	}
}

func TestBuildSteps(t *testing.T) {
	s := workflow.State{Status: workflow.StatusInReview, AssigneeID: 3, Reviewers: []int64{2, 3}, ApproverID: 4}
	states := func(st workflow.State) []string {
		var out []string
		for _, step := range buildSteps(st, nil) {
			out = append(out, step.State)
		}
		return out
	}
	if got := states(s); !slices.Equal(got, []string{"DONE", "ACTIVE", "PENDING"}) {
		t.Errorf("IN_REVIEW tahap 2: %v", got)
	}
	s.Status, s.AssigneeID = workflow.StatusInApproval, 4
	if got := states(s); !slices.Equal(got, []string{"DONE", "DONE", "ACTIVE"}) {
		t.Errorf("IN_APPROVAL: %v", got)
	}
	s.Status, s.AssigneeID = workflow.StatusSent, 0
	if got := states(s); !slices.Equal(got, []string{"DONE", "DONE", "DONE"}) {
		t.Errorf("SENT: %v", got)
	}
	s.Status, s.AssigneeID = workflow.StatusReturnedToDrafter, 1
	if got := states(s); !slices.Equal(got, []string{"PENDING", "PENDING", "PENDING"}) {
		t.Errorf("RETURNED_TO_DRAFTER: %v", got)
	}
}

func TestAllowedActionsWithPermissions(t *testing.T) {
	lc := func(status workflow.Status, assignee int64) *letterCtx {
		l := &model.Letter{Status: string(status), DrafterID: 1}
		if assignee != 0 {
			l.AssigneeID = ptr(assignee)
		}
		if status == workflow.StatusInApproval {
			l.ForwardedByID = ptr(int64(2)) // reviewer terakhir yang meneruskan ke approver
		}
		return &letterCtx{letter: l, parts: []model.Participant{
			{UserID: 2, Role: model.RoleReviewer, StepOrder: 1},
			{UserID: 4, Role: model.RoleApprover, StepOrder: 1},
		}}
	}
	tests := []struct {
		name string
		u    *model.CurrentUser
		lc   *letterCtx
		want []string
	}{
		{"drafter di draf", user(1, "PSP", permission.LetterCreate), lc(workflow.StatusDraft, 0),
			[]string{"EDIT", "DELETE", "SUBMIT", "UPLOAD_ATTACHMENT", "UPLOAD_VERSION"}},
		{"drafter saat dikembalikan", user(1, "PSP"), lc(workflow.StatusReturnedToDrafter, 1),
			[]string{"EDIT", "SUBMIT", "UPLOAD_ATTACHMENT", "UPLOAD_VERSION"}},
		{"reviewer aktif", user(2, "PSP", permission.LetterReview), lc(workflow.StatusInReview, 2),
			[]string{"APPROVE", "RETURN", "UPLOAD_VERSION"}},
		{"reviewer aktif tanpa permission", user(2, "PSP"), lc(workflow.StatusInReview, 2), []string{}},
		{"approver aktif", user(4, "PSP", permission.LetterApprove), lc(workflow.StatusInApproval, 4),
			[]string{"RETURN", "SIGN"}},
		{"drafter saat review", user(1, "PSP"), lc(workflow.StatusInReview, 2), []string{}},
	}
	for _, tt := range tests {
		if got := allowedActions(tt.u, tt.lc); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := map[string]string{
		"Lampiran Anggaran.xlsx":          "Lampiran Anggaran.xlsx",
		`C:\Users\x\Desktop\nota.pdf`:     "nota.pdf",
		"../../etc/passwd":                "passwd",
		`a<b>c:d"e|f?g*h.txt`:             "abcdefgh.txt",
		"  spasi.doc  ":                   "spasi.doc",
		"baris\nbaru\t.pdf":               "barisbaru.pdf",
		"":                                "berkas",
		"///":                             "berkas",
		"..":                              "berkas",
		strings.Repeat("a", 300) + ".pdf": strings.Repeat("a", 255),
	}
	for in, want := range tests {
		if got := SanitizeFileName(in); got != want {
			t.Errorf("SanitizeFileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFileExtension(t *testing.T) {
	tests := map[string]string{
		"Surat.PDF": "pdf", "arsip.tar.gz": "gz", "tanpa-ekstensi": "bin",
		"x.abcdefghijklmno": "abcdefghij",
	}
	for in, want := range tests {
		if got := FileExtension(in); got != want {
			t.Errorf("FileExtension(%q) = %q, want %q", in, got, want)
		}
	}
}
