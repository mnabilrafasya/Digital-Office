package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
)

// ---- Login ----

type fakeAuthUsers struct {
	logins map[string]*model.LoginRecord
	active map[int64]*model.CurrentUser
}

func (f fakeAuthUsers) FindLoginByEmployeeNumber(_ context.Context, nik string) (*model.LoginRecord, error) {
	if r, ok := f.logins[nik]; ok {
		return r, nil
	}
	return nil, repository.ErrNotFound
}

func (f fakeAuthUsers) FindActiveByID(_ context.Context, id int64) (*model.CurrentUser, error) {
	if u, ok := f.active[id]; ok {
		return u, nil
	}
	return nil, repository.ErrNotFound
}

type fakeTokens struct{}

func (fakeTokens) Issue(int64) (string, error) { return "token-uji", nil }

func TestLogin(t *testing.T) {
	hashBytes, _ := bcrypt.GenerateFromPassword([]byte("benar123"), bcrypt.MinCost)
	hash := string(hashBytes)

	users := fakeAuthUsers{
		logins: map[string]*model.LoginRecord{
			"1": {UserID: 1, PasswordHash: &hash, IsActive: true, RoleIsActive: true},
			"2": {UserID: 2, PasswordHash: &hash, IsActive: false, RoleIsActive: true},
			"3": {UserID: 3, PasswordHash: nil, IsActive: true, RoleIsActive: true},
			"4": {UserID: 4, PasswordHash: &hash, IsActive: true, RoleIsActive: false},
		},
		active: map[int64]*model.CurrentUser{1: {ID: 1, Name: "Aktif"}},
	}
	svc := NewAuthService(users, fakeTokens{})

	res, err := svc.Login(context.Background(), " 1 ", "benar123")
	if err != nil || res.AccessToken != "token-uji" || res.User.ID != 1 {
		t.Fatalf("login valid gagal: res=%+v err=%v", res, err)
	}

	gagal := []struct{ name, nik, pw string }{
		{"password salah", "1", "salah"},
		{"NIK tidak ada", "9999999", "benar123"},
		{"user nonaktif", "2", "benar123"},
		{"belum punya password", "3", "benar123"},
		{"role nonaktif", "4", "benar123"},
	}
	for _, tt := range gagal {
		if _, err := svc.Login(context.Background(), tt.nik, tt.pw); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("%s: err = %v, want ErrInvalidCredentials", tt.name, err)
		}
	}

	var ve *ValidationError
	if _, err := svc.Login(context.Background(), "  ", "x"); !errors.As(err, &ve) {
		t.Errorf("NIK kosong: err = %v, want ValidationError", err)
	}
	if _, err := svc.Login(context.Background(), "1", ""); !errors.As(err, &ve) {
		t.Errorf("password kosong: err = %v, want ValidationError", err)
	}
}

// ---- Kandidat ----

type fakeCandidates struct{ got *repository.CandidateFilter }

func (f *fakeCandidates) ListCandidates(_ context.Context, filter repository.CandidateFilter) ([]model.Candidate, error) {
	*f.got = filter
	return nil, nil
}

func TestCandidates(t *testing.T) {
	var got repository.CandidateFilter
	svc := NewFormService(nil, nil, &fakeCandidates{got: &got})
	user := &model.CurrentUser{ID: 8, Company: model.UserCompany{Code: "PSP"}}
	ctx := context.Background()

	if _, err := svc.Candidates(ctx, user, CandidateQuery{Type: "REVIEWER"}); err != nil {
		t.Fatal(err)
	}
	if got.PermissionCode != permission.LetterReview || got.ExcludeUserID != 0 || got.CompanyCode != "PSP" || got.Limit != 20 {
		t.Errorf("REVIEWER: filter = %+v", got)
	}

	if _, err := svc.Candidates(ctx, user, CandidateQuery{Type: "APPROVER", Search: "  agus ", Limit: "50"}); err != nil {
		t.Fatal(err)
	}
	if got.PermissionCode != permission.LetterApprove || got.ExcludeUserID != 8 || got.Search != "agus" || got.Limit != 50 {
		t.Errorf("APPROVER: filter = %+v", got)
	}

	invalidQueries := []CandidateQuery{
		{Type: ""},
		{Type: "ABC"},
		{Type: "reviewer"},
		{Type: "REVIEWER", Limit: "0"},
		{Type: "REVIEWER", Limit: "51"},
		{Type: "REVIEWER", Limit: "abc"},
		{Type: "REVIEWER", Search: strings.Repeat("a", 101)},
	}
	for _, q := range invalidQueries {
		var ve *ValidationError
		if _, err := svc.Candidates(ctx, user, q); !errors.As(err, &ve) {
			t.Errorf("%+v: err = %v, want ValidationError", q, err)
		}
	}
}
