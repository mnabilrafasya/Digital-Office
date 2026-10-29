package organisasi

import (
	"context"
	"errors"
	"strings"
	"testing"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

// ---- Kandidat ----

type fakeCandidates struct{ got *CandidateFilter }

func (f *fakeCandidates) ListCandidates(_ context.Context, filter CandidateFilter) ([]Candidate, error) {
	*f.got = filter
	return nil, nil
}

func TestCandidates(t *testing.T) {
	var got CandidateFilter
	svc := NewService(nil, &fakeCandidates{got: &got}, nil)
	user := &auth.CurrentUser{ID: 8, Company: auth.UserCompany{Code: "PSP"}}
	ctx := context.Background()

	if _, err := svc.Candidates(ctx, user, CandidateQuery{Type: "REVIEWER"}); err != nil {
		t.Fatal(err)
	}
	if got.PermissionCode != auth.LetterReview || got.ExcludeUserID != 0 || got.CompanyCode != "PSP" || got.Limit != 20 {
		t.Errorf("REVIEWER: filter = %+v", got)
	}

	if _, err := svc.Candidates(ctx, user, CandidateQuery{Type: "APPROVER", Search: "  agus ", Limit: "50"}); err != nil {
		t.Fatal(err)
	}
	if got.PermissionCode != auth.LetterApprove || got.ExcludeUserID != 8 || got.Search != "agus" || got.Limit != 50 {
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
		var ve *apperr.ValidationError
		if _, err := svc.Candidates(ctx, user, q); !errors.As(err, &ve) {
			t.Errorf("%+v: err = %v, want apperr.ValidationError", q, err)
		}
	}
}
