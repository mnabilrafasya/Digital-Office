package service

import (
	"context"
	"strings"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/workflow"
)

// ListQuery adalah parameter daftar surat dari query string.
type ListQuery struct {
	Scope    string // mine (default) atau tasks
	Statuses string // dipisah koma, opsional
	Search   string
	Offset   int
	Limit    int
}

// Scope daftar surat.
const (
	ScopeMine  = "mine"
	ScopeTasks = "tasks"
)

// List mengambil daftar surat milik user (mine) atau tugas user (tasks).
func (s *LetterService) List(ctx context.Context, user *model.CurrentUser, lq ListQuery) ([]model.LetterListItem, int, error) {
	f := model.LetterListFilter{CompanyCode: user.Company.Code, Offset: lq.Offset, Limit: lq.Limit}
	switch lq.Scope {
	case "", ScopeMine:
		f.DrafterID = user.ID
	case ScopeTasks:
		f.AssigneeID = user.ID
		for _, st := range workflow.TaskStatuses {
			f.TaskStatuses = append(f.TaskStatuses, string(st))
		}
	default:
		return nil, 0, invalid("Parameter scope wajib mine atau tasks")
	}

	for _, st := range strings.Split(lq.Statuses, ",") {
		if st = strings.TrimSpace(st); st == "" {
			continue
		}
		if !workflow.IsValidStatus(st) {
			return nil, 0, invalid("Status %q tidak dikenal", st)
		}
		f.Statuses = append(f.Statuses, st)
	}

	search, err := normalizeSearch(lq.Search)
	if err != nil {
		return nil, 0, err
	}
	f.Search = search

	rows, total, err := s.read.Letters.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	items := make([]model.LetterListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, s.listItem(row))
	}
	return items, total, nil
}

func (s *LetterService) listItem(row model.LetterListRow) model.LetterListItem {
	item := model.LetterListItem{
		ID: row.UUID, LetterNumber: row.LetterNumber, Title: row.Title, Status: row.Status,
		LetterType: row.LetterType, Confidentiality: row.Confidentiality, Urgency: row.Urgency,
		SenderUnit: row.SenderUnit, CurrentAssignee: row.Assignee,
		CreatedAt: s.formatTime(row.CreatedAt), UpdatedAt: s.formatTime(row.UpdatedAt),
	}
	switch workflow.Status(row.Status) {
	case workflow.StatusInReview, workflow.StatusReturnedToReviewer:
		if row.StepOrder != nil {
			item.CurrentStep = &model.StepRef{Role: model.RoleReviewer, Order: *row.StepOrder}
		}
	case workflow.StatusInApproval:
		item.CurrentStep = &model.StepRef{Role: model.RoleApprover, Order: 1}
	}
	return item
}

// History mengambil riwayat alur surat.
func (s *LetterService) History(ctx context.Context, user *model.CurrentUser, uuid string) ([]model.HistoryItem, error) {
	lc, err := loadVisible(ctx, s.read, user, uuid, false)
	if err != nil {
		return nil, err
	}
	rows, err := s.read.Actions.History(ctx, lc.letter.ID)
	if err != nil {
		return nil, err
	}
	out := make([]model.HistoryItem, 0, len(rows))
	for _, h := range rows {
		out = append(out, model.HistoryItem{
			Action: h.Action, ActorRole: h.ActorRole, Actor: h.Actor, Target: h.Target,
			FromStatus: h.FromStatus, ToStatus: h.ToStatus, Note: h.Note,
			VersionNumber: h.VersionNumber, CreatedAt: s.formatTime(h.CreatedAt),
		})
	}
	return out, nil
}
