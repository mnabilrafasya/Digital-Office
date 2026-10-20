package service

import (
	"context"
	"slices"
	"strings"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
)

// Cakupan daftar surat.
const (
	ScopeMine  = "mine"  // surat yang saya buat
	ScopeTasks = "tasks" // surat yang sedang menunggu tindakan saya
)

// allStatuses adalah semua status surat (sama dengan CHECK ck_letters_status).
func allStatuses() []string {
	return []string{
		model.StatusDraft, model.StatusInReview, model.StatusReturnedToDrafter, model.StatusInApproval,
		model.StatusReturnedToReviewer, model.StatusSent, model.StatusCancelled,
	}
}

// taskStatuses adalah status yang berarti surat menunggu tindakan penerima tugasnya.
func taskStatuses() []string {
	return []string{
		model.StatusInReview, model.StatusInApproval, model.StatusReturnedToReviewer, model.StatusReturnedToDrafter,
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
func (s *LetterService) List(ctx context.Context, user *model.CurrentUser, lq LetterListQuery) ([]model.LetterListItem, int, error) {
	statuses, err := parseStatuses(lq.Status)
	if err != nil {
		return nil, 0, err
	}
	search, err := normalizeSearch(lq.Search)
	if err != nil {
		return nil, 0, err
	}
	f := repository.LetterFilter{
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
				return []model.LetterListItem{}, 0, nil // status yang diminta tidak pernah menjadi tugas
			}
		}
	default:
		return nil, 0, invalid("Parameter scope wajib mine atau tasks")
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

// History mengambil riwayat aksi surat (urut dari yang paling lama) untuk drafter dan peserta.
func (s *LetterService) History(ctx context.Context, user *model.CurrentUser, letterID string) ([]model.LetterAction, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	l, err := s.reader.FindByUUID(ctx, user.Company.Code, id)
	if err != nil {
		return nil, notFoundAs(err)
	}
	if l.DrafterID != user.ID {
		isParticipant, err := s.reader.IsParticipant(ctx, l.ID, user.ID)
		if err != nil {
			return nil, err
		}
		if !isParticipant {
			return nil, ErrLetterNotFound
		}
	}

	actions, err := s.reader.Actions(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	for i := range actions {
		actions[i].CreatedAt = actions[i].CreatedAt.In(s.loc)
	}
	return actions, nil
}

// currentStep adalah tahap yang sedang berjalan, atau nil bila surat tidak sedang di reviewer/approver.
func currentStep(status string, reviewerStep int) *model.StepRef {
	switch status {
	case model.StatusInReview, model.StatusReturnedToReviewer:
		if reviewerStep > 0 {
			return &model.StepRef{Role: model.RoleReviewer, Order: reviewerStep}
		}
	case model.StatusInApproval:
		return &model.StepRef{Role: model.RoleApprover, Order: 1}
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
			return nil, invalid("Parameter status wajib salah satu dari: " + strings.Join(allStatuses(), ", "))
		}
		out = append(out, st)
	}
	return out, nil
}
