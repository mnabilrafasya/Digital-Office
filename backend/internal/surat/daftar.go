package surat

import (
	"context"
	"slices"
	"strings"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Cakupan daftar surat.
const (
	ScopeMine  = "mine"  // surat yang saya buat
	ScopeTasks = "tasks" // surat yang sedang menunggu tindakan saya
)

// allStatuses adalah semua status surat (sama dengan CHECK ck_letters_status).
func allStatuses() []string {
	return []string{
		StatusDraft, StatusInReview, StatusReturnedToDrafter, StatusInApproval,
		StatusReturnedToReviewer, StatusSent, StatusCancelled,
	}
}

// taskStatuses adalah status yang berarti surat menunggu tindakan penerima tugasnya.
func taskStatuses() []string {
	return []string{
		StatusInReview, StatusInApproval, StatusReturnedToReviewer, StatusReturnedToDrafter,
	}
}

// LetterListQuery adalah parameter daftar surat. Scope, Status, dan Search apa adanya dari query
// string; Page dan PageSize sudah divalidasi handler.
type LetterListQuery struct {
	Scope    string // "" berarti mine
	Status   string // opsional, beberapa status dipisah koma
	Search   string
	Page     int
	PageSize int
}

// List mengambil satu halaman daftar surat beserta jumlah totalnya.
func (s *Service) List(ctx context.Context, user *auth.CurrentUser, lq LetterListQuery) ([]LetterListItem, int, error) {
	statuses, err := parseStatuses(lq.Status)
	if err != nil {
		return nil, 0, err
	}
	search, err := db.NormalizeSearch(lq.Search)
	if err != nil {
		return nil, 0, err
	}
	f := LetterFilter{
		CompanyCode: user.Company.Code,
		Statuses:    statuses,
		Search:      search,
		Offset:      (lq.Page - 1) * lq.PageSize,
		Limit:       lq.PageSize,
	}

	switch lq.Scope {
	case "", ScopeMine:
		f.DrafterID = user.ID
	case ScopeTasks:
		f.AssigneeID = user.ID
		f.Statuses = taskStatuses()
		if len(statuses) > 0 {
			f.Statuses = slices.DeleteFunc(f.Statuses, func(st string) bool { return !slices.Contains(statuses, st) })
			if len(f.Statuses) == 0 {
				return []LetterListItem{}, 0, nil // status yang diminta tidak pernah menjadi tugas
			}
		}
	default:
		return nil, 0, apperr.Invalid("Parameter scope wajib mine atau tasks")
	}

	items, total, err := s.reader.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		it := &items[i]
		it.CurrentStep = currentStep(it.Status, it.AssigneeStep)
		it.CreatedAt = it.CreatedAt.In(s.loc)
		it.UpdatedAt = it.UpdatedAt.In(s.loc)
	}
	return items, total, nil
}

// currentStep adalah tahap yang sedang berjalan, atau nil bila surat tidak sedang di reviewer/approver.
func currentStep(status string, reviewerStep int) *StepRef {
	switch status {
	case StatusInReview, StatusReturnedToReviewer:
		if reviewerStep > 0 {
			return &StepRef{Role: RoleReviewer, Order: reviewerStep}
		}
	case StatusInApproval:
		return &StepRef{Role: RoleApprover, Order: 1}
	}
	return nil
}

// parseStatuses membaca "IN_REVIEW,SENT" menjadi daftar status unik yang valid ("" = semua).
func parseStatuses(raw string) ([]string, error) {
	var out []string
	for part := range strings.SplitSeq(raw, ",") {
		st := strings.ToUpper(strings.TrimSpace(part))
		if st == "" || slices.Contains(out, st) {
			continue
		}
		if !slices.Contains(allStatuses(), st) {
			return nil, apperr.Invalid("Parameter status wajib salah satu dari: " + strings.Join(allStatuses(), ", "))
		}
		out = append(out, st)
	}
	return out, nil
}
