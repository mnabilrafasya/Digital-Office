package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/workflow"
)

// Aksi yang bisa ditampilkan sebagai tombol, urut seperti di layar.
const (
	ActionEdit             = "EDIT"
	ActionDelete           = "DELETE"
	ActionUploadAttachment = "UPLOAD_ATTACHMENT"
	ActionUploadVersion    = "UPLOAD_VERSION"
)

var actionOrder = []string{
	ActionEdit, ActionDelete, string(workflow.ActionSubmit), string(workflow.ActionApprove),
	string(workflow.ActionReturn), string(workflow.ActionSign), ActionUploadAttachment, ActionUploadVersion,
}

// Status tahap persetujuan di detail surat.
const (
	stepDone    = "DONE"
	stepActive  = "ACTIVE"
	stepPending = "PENDING"
)

func parseAttachments(raw *string) ([]model.Attachment, error) {
	if raw == nil || *raw == "" {
		return []model.Attachment{}, nil
	}
	var out []model.Attachment
	if err := json.Unmarshal([]byte(*raw), &out); err != nil {
		return nil, fmt.Errorf("membaca JSON lampiran: %w", err)
	}
	return out, nil
}

// buildDetail menyusun detail surat untuk user tertentu. Label diambil dengan beberapa query
// batch (bukan satu query per item).
func (s *LetterService) buildDetail(ctx context.Context, r LetterRepos, user *model.CurrentUser, lc *letterCtx) (*model.LetterDetail, error) {
	l := lc.letter
	company := user.Company.Code

	recips, err := r.Recipients.List(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	attachments, err := parseAttachments(l.AttachmentsJSON)
	if err != nil {
		return nil, err
	}

	// Kumpulkan semua ID yang perlu label
	lookupIDs := nonNil(l.Content.LetterTypeID, l.Content.ClassificationID, l.Content.ConfidentialityID, l.Content.UrgencyID)
	unitIDs := []int64{l.SenderUnitID}
	var aliasIDs []int64
	userIDs := []int64{l.DrafterID}
	if l.AssigneeID != nil {
		userIDs = append(userIDs, *l.AssigneeID)
	}
	for _, p := range lc.parts {
		userIDs = append(userIDs, p.UserID)
	}
	for _, rc := range recips {
		if rc.UnitID != nil {
			unitIDs = append(unitIDs, *rc.UnitID)
		}
		if rc.AliasLevelID != nil {
			aliasIDs = append(aliasIDs, *rc.AliasLevelID)
		}
		if rc.ResolvedHeadID != nil {
			userIDs = append(userIDs, *rc.ResolvedHeadID)
		}
	}
	for _, a := range attachments {
		var id int64
		if _, err := fmt.Sscan(a.UploadedBy, &id); err == nil {
			userIDs = append(userIDs, id)
		}
	}

	lookups, err := r.Lookups.FindRefs(ctx, company, lookupIDs)
	if err != nil {
		return nil, err
	}
	units, err := r.Units.FindRefs(ctx, company, unitIDs)
	if err != nil {
		return nil, err
	}
	users, err := r.Users.FindLabels(ctx, company, userIDs)
	if err != nil {
		return nil, err
	}
	aliases := map[int64]model.Alias{}
	if len(aliasIDs) > 0 {
		rows, err := r.Lookups.FindAliases(ctx, company, aliasIDs)
		if err != nil {
			return nil, err
		}
		for id, row := range rows {
			aliases[id] = aliasView(row)
		}
	}

	d := &model.LetterDetail{
		ID: l.UUID, Status: l.Status, LetterNumber: l.LetterNumber,
		Title: l.Content.Title, Summary: l.Content.Summary, SpecialNote: l.Content.SpecialNote,
		LetterType:      lookupLabel(lookups, l.Content.LetterTypeID),
		Classification:  lookupLabel(lookups, l.Content.ClassificationID),
		Confidentiality: lookupLabel(lookups, l.Content.ConfidentialityID),
		Urgency:         lookupLabel(lookups, l.Content.UrgencyID),
		SenderUnit:      unitLabel(units, l.SenderUnitID),
		Drafter:         userLabel(users, l.DrafterID, false),
		Steps:           buildSteps(lc.state(), users),
		Recipients:      buildRecipients(recips, units, aliases, users),
		Attachments:     make([]model.AttachmentView, 0, len(attachments)),
		CurrentVersion:  l.CurrentVersion,
		AllowedActions:  allowedActions(user, lc),
		CreatedAt:       s.formatTime(l.CreatedAt),
		UpdatedAt:       s.formatTime(l.UpdatedAt),
	}
	if l.SenderUnitName != nil {
		d.SenderUnit.Name = *l.SenderUnitName // snapshot saat surat dikirim
	}
	if l.ApprovedAt != nil {
		date := l.ApprovedAt.In(s.cfg.Location).Format("2006-01-02")
		d.LetterDate = &date
	}
	if l.Content.TemplateID != nil {
		t, err := r.Units.FindTemplate(ctx, company, *l.Content.TemplateID)
		if err != nil && !isNotFound(err) {
			return nil, err
		}
		d.Template = t
	}
	if l.AssigneeID != nil {
		a := userLabel(users, *l.AssigneeID, false)
		d.CurrentAssignee = &a
	}
	for _, a := range attachments {
		var uploader int64
		_, _ = fmt.Sscan(a.UploadedBy, &uploader)
		label := userLabel(users, uploader, false)
		d.Attachments = append(d.Attachments, model.AttachmentView{
			ID: a.ID, FileName: a.FileName, Size: a.Size, ContentType: a.ContentType, Checksum: a.Checksum,
			UploadedAt: s.formatTime(a.UploadedAt), UploadedBy: model.PersonRef{ID: label.ID, Name: label.Name},
		})
	}

	st := workflow.Status(l.Status)
	if st == workflow.StatusReturnedToDrafter || st == workflow.StatusReturnedToReviewer {
		note, err := r.Actions.LastNote(ctx, l.ID, string(workflow.ActionReturn))
		if err != nil {
			return nil, err
		}
		d.LastReturnNote = note
	}
	return d, nil
}

func nonNil(ids ...*int64) []int64 {
	var out []int64
	for _, id := range ids {
		if id != nil {
			out = append(out, *id)
		}
	}
	return out
}

func lookupLabel(refs map[int64]model.LookupRef, id *int64) *model.LookupLabel {
	if id == nil {
		return nil
	}
	ref, ok := refs[*id]
	if !ok {
		return &model.LookupLabel{ID: *id}
	}
	return &model.LookupLabel{ID: ref.ID, Code: ref.Code, Name: ref.Name}
}

func unitLabel(units map[int64]model.UnitRef, id int64) model.UnitLabel {
	u := units[id]
	return model.UnitLabel{ID: id, Code: u.Code, Name: u.Name}
}

func userLabel(users map[int64]model.UserLabel, id int64, withUnit bool) model.UserLabel {
	u, ok := users[id]
	if !ok {
		return model.UserLabel{ID: id}
	}
	if !withUnit {
		u.UnitName = ""
	}
	return u
}

// buildSteps menghitung status setiap tahap (DONE, ACTIVE, PENDING) dari status dan assignee surat.
func buildSteps(s workflow.State, users map[int64]model.UserLabel) []model.StepView {
	activeIdx := slices.Index(s.Reviewers, s.AssigneeID)
	steps := make([]model.StepView, 0, len(s.Reviewers)+1)
	for i, id := range s.Reviewers {
		state := stepPending
		switch s.Status {
		case workflow.StatusInReview, workflow.StatusReturnedToReviewer:
			switch {
			case i < activeIdx:
				state = stepDone
			case i == activeIdx:
				state = stepActive
			}
		case workflow.StatusInApproval, workflow.StatusSent:
			state = stepDone
		}
		steps = append(steps, model.StepView{Role: model.RoleReviewer, Order: i + 1, State: state, User: userLabel(users, id, true)})
	}
	if s.ApproverID != 0 {
		state := stepPending
		switch s.Status {
		case workflow.StatusInApproval:
			state = stepActive
		case workflow.StatusSent:
			state = stepDone
		}
		steps = append(steps, model.StepView{Role: model.RoleApprover, Order: 1, State: state, User: userLabel(users, s.ApproverID, true)})
	}
	return steps
}

func buildRecipients(recips []model.Recipient, units map[int64]model.UnitRef, aliases map[int64]model.Alias, users map[int64]model.UserLabel) model.RecipientsView {
	view := model.RecipientsView{To: []model.RecipientView{}, CC: []model.RecipientView{}}
	for _, rc := range recips {
		var v model.RecipientView
		if rc.UnitID != nil {
			u := unitLabel(units, *rc.UnitID)
			v = model.RecipientView{Type: "UNIT", Unit: &u}
			if rc.ResolvedHeadID != nil {
				h := userLabel(users, *rc.ResolvedHeadID, false)
				v.ResolvedHead = &h
			}
		} else if rc.AliasLevelID != nil {
			a, ok := aliases[*rc.AliasLevelID]
			if !ok {
				a = model.Alias{LevelID: *rc.AliasLevelID}
			}
			v = model.RecipientView{Type: "ALIAS", Alias: &a}
		}
		if rc.Type == model.RecipientTo {
			view.To = append(view.To, v)
		} else {
			view.CC = append(view.CC, v)
		}
	}
	return view
}

// allowedActions menghitung tombol yang boleh tampil untuk user. Aksi alur diambil dari
// workflow.AllowedActions lalu disaring permission (reviewer: LETTER_REVIEW, approver: LETTER_APPROVE).
func allowedActions(user *model.CurrentUser, lc *letterCtx) []string {
	l := lc.letter
	st := workflow.Status(l.Status)
	allowed := map[string]bool{}

	if lc.isDrafter(user) && workflow.IsEditable(st) {
		allowed[ActionEdit] = true
		allowed[ActionUploadAttachment] = true
		allowed[ActionUploadVersion] = true
		if st == workflow.StatusDraft {
			allowed[ActionDelete] = true
		}
	}

	state := lc.state()
	for _, a := range workflow.AllowedActions(state, user.ID) {
		d, _ := workflow.Decide(state, a, user.ID)
		if hasRolePermission(user, d.ActorRole) {
			allowed[string(a)] = true
		}
	}
	if _, err := uploadRole(model.UploadVersion, lc, user); err == nil {
		allowed[ActionUploadVersion] = true
	}

	out := []string{}
	for _, a := range actionOrder {
		if allowed[a] {
			out = append(out, a)
		}
	}
	return out
}

// hasRolePermission memeriksa permission yang dibutuhkan untuk bertindak dengan peran tertentu.
func hasRolePermission(user *model.CurrentUser, role workflow.Role) bool {
	switch role {
	case workflow.RoleReviewer:
		return user.HasPermission(permission.LetterReview)
	case workflow.RoleApprover:
		return user.HasPermission(permission.LetterApprove)
	default:
		return true
	}
}
