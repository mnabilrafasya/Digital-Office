package surat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

const testLetterID = "6f1c2a9e-0000-4000-8000-000000000001"

var testRowVersion = []byte{0, 0, 0, 0, 0, 0, 0x07, 0xD3}

func ptr[T any](v T) *T { return &v }

// drafter adalah Andi (id 15) di perusahaan PSP, unit 8.
var drafter = &auth.CurrentUser{ID: 15, Name: "Andi", Unit: auth.UserUnit{ID: 8}, Company: auth.UserCompany{Code: "PSP"}}

// completeInput adalah draf lengkap: reviewer [9, 4], approver 3, Kepada unit 7 + alias 60, Tembusan unit 9.
func completeInput() LetterInput {
	in := LetterInput{
		LetterTypeID: ptr("12"), ClassificationID: ptr("40"), ConfidentialityID: ptr("1"), UrgencyID: ptr("5"),
		Title: ptr("  Undangan Rapat  "), ReviewerIDs: []string{"9", "4"}, ApproverID: ptr("3"),
	}
	in.Recipients.To = []RecipientInput{{UnitID: ptr("7")}, {AliasLevelID: ptr("60")}}
	in.Recipients.CC = []RecipientInput{{UnitID: ptr("9")}}
	return in
}

func TestParseDraft(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(in *LetterInput)
		wantErr string
	}{
		{"lengkap", func(*LetterInput) {}, ""},
		{"kosong semua", func(in *LetterInput) { *in = LetterInput{} }, ""},
		{"ID bukan angka", func(in *LetterInput) { in.LetterTypeID = ptr("abc") }, "Jenis surat tidak valid"},
		{"ID nol", func(in *LetterInput) { in.UrgencyID = ptr("0") }, "Kecepatan tidak valid"},
		{"perihal 251 karakter", func(in *LetterInput) { in.Title = ptr(strings.Repeat("a", 251)) }, "Perihal maksimal 250"},
		{"emoji dihitung 2", func(in *LetterInput) { in.Title = ptr(strings.Repeat("😀", 126)) }, "Perihal maksimal 250"},
		{"reviewer duplikat", func(in *LetterInput) { in.ReviewerIDs = []string{"9", "9"} }, "Reviewer yang sama"},
		{"reviewer kosong", func(in *LetterInput) { in.ReviewerIDs = []string{""} }, "Reviewer tidak valid"},
		{"approver = drafter", func(in *LetterInput) { in.ApproverID = ptr("15") }, "pembuat surat"},
		{"approver juga reviewer", func(in *LetterInput) { in.ApproverID = ptr("9") }, "sekaligus menjadi reviewer"},
		{"penerima tanpa unit/alias", func(in *LetterInput) { in.Recipients.To = []RecipientInput{{}} }, "tepat satu"},
		{"penerima unit dan alias sekaligus", func(in *LetterInput) {
			in.Recipients.To = []RecipientInput{{UnitID: ptr("7"), AliasLevelID: ptr("60")}}
		}, "tepat satu"},
		{"unit di Kepada dan Tembusan", func(in *LetterInput) { in.Recipients.CC = []RecipientInput{{UnitID: ptr("7")}} }, "tidak boleh dipilih lebih dari sekali"},
		{"alias dua kali", func(in *LetterInput) { in.Recipients.CC = []RecipientInput{{AliasLevelID: ptr("60")}} }, "tidak boleh dipilih lebih dari sekali"},
	}
	for _, tt := range tests {
		in := completeInput()
		tt.mutate(&in)
		d, err := parseDraft(in, drafter.ID)
		if tt.wantErr == "" {
			if err != nil {
				t.Errorf("%s: error = %v, want nil", tt.name, err)
			}
			continue
		}
		var ve *apperr.ValidationError
		if !errors.As(err, &ve) || !strings.Contains(ve.Message, tt.wantErr) {
			t.Errorf("%s: error = %v, want apperr.ValidationError berisi %q (draft %+v)", tt.name, err, tt.wantErr, d)
		}
	}

	d, _ := parseDraft(completeInput(), drafter.ID)
	if *d.Title != "Undangan Rapat" || len(d.Recipients) != 3 || d.Recipients[2].Type != RecipientCC {
		t.Errorf("hasil parse salah: title=%q recipients=%+v", *d.Title, d.Recipients)
	}
	d, _ = parseDraft(LetterInput{Title: ptr("   "), Summary: ptr("")}, drafter.ID)
	if d.Title != nil || d.Summary != nil {
		t.Error("teks kosong harus menjadi nil")
	}
}

func TestCheckComplete(t *testing.T) {
	d, _ := parseDraft(completeInput(), drafter.ID)
	if err := checkComplete(d); err != nil {
		t.Fatalf("draf lengkap: %v", err)
	}
	in := completeInput()
	in.Title, in.ApproverID = nil, nil
	in.Recipients.To = nil
	d, _ = parseDraft(in, drafter.ID)
	err := checkComplete(d)
	for _, want := range []string{"perihal", "approver", "minimal 1 penerima Kepada"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("error %v tidak menyebut %q", err, want)
		}
	}
}

func TestLetterCheckRefs(t *testing.T) {
	tests := []struct {
		name    string
		refs    func(r *fakeRefs)
		wantErr string
	}{
		{"semua valid", func(*fakeRefs) {}, ""},
		{"jenis surat salah kategori", func(r *fakeRefs) { r.lookups[12] = "URGENCY" }, "Jenis surat"},
		{"alias bukan level jabatan", func(r *fakeRefs) { delete(r.lookups, 60) }, "Alias penerima"},
		{"unit penerima nonaktif", func(r *fakeRefs) { r.badUnits = map[int64]bool{9: true} }, "Unit penerima"},
		{"reviewer tanpa LETTER_REVIEW", func(r *fakeRefs) { delete(r.reviewers, 4) }, "Reviewer tidak ditemukan"},
		{"approver tanpa LETTER_APPROVE", func(r *fakeRefs) { delete(r.approvers, 3) }, "Approver tidak ditemukan"},
		{"template tidak ada", func(r *fakeRefs) { r.noTemplate = true }, "Template tidak ditemukan"},
		{"template beda jenis surat", func(r *fakeRefs) { r.templateType = ptr(int64(99)) }, "Template tidak sesuai"},
		{"template jenis sama", func(r *fakeRefs) { r.templateType = ptr(int64(12)) }, ""},
	}
	for _, tt := range tests {
		refs := newFakeRefs()
		tt.refs(refs)
		in := completeInput()
		in.TemplateID = ptr("2")
		d, err := parseDraft(in, drafter.ID)
		if err == nil {
			err = newTestLetterService(&fakeLetterStore{}, refs).checkRefs(context.Background(), drafter.Company.Code, d)
		}
		if tt.wantErr == "" && err != nil || tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
			t.Errorf("%s: error = %v, want %q", tt.name, err, tt.wantErr)
		}
	}
}

func TestLetterUpdate(t *testing.T) {
	lock := func(drafterID int64, status string) *LetterLock {
		return &LetterLock{ID: 1, DrafterID: drafterID, Status: status, RowVersion: testRowVersion}
	}
	tests := []struct {
		name    string
		store   *fakeLetterStore
		ifMatch []byte
		in      func(in *LetterInput)
		check   func(err error) bool
	}{
		{"surat tidak ada / perusahaan lain", &fakeLetterStore{}, testRowVersion, nil,
			func(err error) bool { return errors.Is(err, ErrLetterNotFound) }},
		{"bukan peserta (referensi tidak valid pun tetap 404)", &fakeLetterStore{lock: lock(99, StatusDraft)}, testRowVersion,
			func(in *LetterInput) { in.LetterTypeID = ptr("999") },
			func(err error) bool { return errors.Is(err, ErrLetterNotFound) }},
		{"reviewer mencoba mengedit", &fakeLetterStore{lock: lock(99, StatusDraft), participants: map[int64]bool{15: true}}, testRowVersion, nil,
			func(err error) bool { return errors.Is(err, apperr.ErrForbidden) }},
		{"status sedang direview", &fakeLetterStore{lock: lock(15, StatusInReview)}, testRowVersion, nil,
			func(err error) bool { var e *apperr.InvalidStateError; return errors.As(err, &e) }},
		{"ETag lama", &fakeLetterStore{lock: lock(15, StatusDraft)}, []byte{0, 0, 0, 0, 0, 0, 0, 1}, nil,
			func(err error) bool {
				var e *apperr.VersionConflictError
				return errors.As(err, &e) && string(e.RowVersion) == string(testRowVersion)
			}},
		{"dikembalikan, belum lengkap", &fakeLetterStore{lock: lock(15, StatusReturnedToDrafter)}, testRowVersion,
			func(in *LetterInput) { in.Title = nil },
			func(err error) bool { var e *apperr.ValidationError; return errors.As(err, &e) }},
		{"draf boleh belum lengkap", &fakeLetterStore{lock: lock(15, StatusDraft),
			detail: &LetterDetail{ID: testLetterID, Status: StatusDraft, Drafter: PersonRef{ID: 15}}},
			testRowVersion, func(in *LetterInput) { in.Title = nil },
			func(err error) bool { return err == nil }},
	}
	for _, tt := range tests {
		in := completeInput()
		if tt.in != nil {
			tt.in(&in)
		}
		_, err := newTestLetterService(tt.store, newFakeRefs()).Update(context.Background(), drafter, testLetterID, tt.ifMatch, in)
		if !tt.check(err) {
			t.Errorf("%s: error = %v", tt.name, err)
		}
		if wantSaved := tt.name == "draf boleh belum lengkap"; (tt.store.updated != nil) != wantSaved {
			t.Errorf("%s: UpdateDraft dipanggil = %v, want %v", tt.name, tt.store.updated != nil, wantSaved)
		}
	}
}

func TestLetterGetAccess(t *testing.T) {
	detail := &LetterDetail{
		ID: testLetterID, Status: StatusDraft,
		Drafter: PersonRef{ID: 15},
		Steps:   []LetterStep{{Role: RoleReviewer, Order: 1, User: StepUser{ID: 9}}},
	}
	svc := newTestLetterService(&fakeLetterStore{detail: detail}, newFakeRefs())
	ctx := context.Background()

	d, err := svc.Get(ctx, drafter, strings.ToUpper(testLetterID))
	if err != nil || strings.Join(d.AllowedActions, ",") != "EDIT,SUBMIT" {
		t.Errorf("drafter: err=%v allowedActions=%v, want [EDIT SUBMIT]", err, d.AllowedActions)
	}
	d, err = svc.Get(ctx, &auth.CurrentUser{ID: 9}, testLetterID)
	if err != nil || len(d.AllowedActions) != 0 {
		t.Errorf("reviewer: err=%v allowedActions=%v, want []", err, d.AllowedActions)
	}
	for _, tc := range []struct {
		user *auth.CurrentUser
		id   string
	}{{&auth.CurrentUser{ID: 77}, testLetterID}, {drafter, "bukan-uuid"}} {
		if _, err := svc.Get(ctx, tc.user, tc.id); !errors.Is(err, ErrLetterNotFound) {
			t.Errorf("user %d id %q: error = %v, want ErrLetterNotFound", tc.user.ID, tc.id, err)
		}
	}
}

func TestLetterCreate(t *testing.T) {
	store := &fakeLetterStore{}
	_, _ = newTestLetterService(store, newFakeRefs()).Create(context.Background(), drafter, LetterInput{})
	if store.created == nil || store.created.CompanyCode != "PSP" || store.created.SenderUnitID != 8 || store.created.DrafterID != 15 {
		t.Errorf("draf baru harus memakai perusahaan, unit, dan id user yang login: %+v", store.created)
	}
}
