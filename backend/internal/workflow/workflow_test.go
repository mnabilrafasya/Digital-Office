package workflow

import (
	"errors"
	"slices"
	"testing"
)

// Aktor uji: drafter, dua reviewer berurutan, satu approver, dan orang luar.
const (
	drafter  int64 = 1
	rev1     int64 = 2
	rev2     int64 = 3
	approver int64 = 4
	outsider int64 = 9
)

func state(st Status, assignee, forwardedBy int64) State {
	return State{
		Status: st, DrafterID: drafter, AssigneeID: assignee, ForwardedByID: forwardedBy,
		Reviewers: []int64{rev1, rev2}, ApproverID: approver,
	}
}

// Sembilan baris tabel mesin status (dokumen Sprint 2 bagian 4.2).
func TestDecideValidTransitions(t *testing.T) {
	tests := []struct {
		name   string
		s      State
		action Action
		actor  int64
		want   Decision
	}{
		{"1 submit draf", state(StatusDraft, 0, 0), ActionSubmit, drafter, Decision{
			ActorRole: RoleDrafter, To: StatusInReview, NextAssigneeID: rev1, SnapshotSenderUnit: true}},
		{"2 submit ulang mulai reviewer 1", state(StatusReturnedToDrafter, drafter, rev1), ActionSubmit, drafter, Decision{
			ActorRole: RoleDrafter, To: StatusInReview, NextAssigneeID: rev1,
			ReturnedByReviewer: clearID, ForwardedBy: clearID}},
		{"3 reviewer 1 setuju ke reviewer 2", state(StatusInReview, rev1, 0), ActionApprove, rev1, Decision{
			ActorRole: RoleReviewer, To: StatusInReview, NextAssigneeID: rev2, ForwardedBy: set(rev1)}},
		{"4 reviewer terakhir setuju ke approver", state(StatusInReview, rev2, rev1), ActionApprove, rev2, Decision{
			ActorRole: RoleReviewer, To: StatusInApproval, NextAssigneeID: approver, ForwardedBy: set(rev2)}},
		{"5 reviewer kembalikan ke drafter", state(StatusInReview, rev2, rev1), ActionReturn, rev2, Decision{
			ActorRole: RoleReviewer, To: StatusReturnedToDrafter, NextAssigneeID: drafter, ReturnedByReviewer: set(rev2)}},
		{"6 approver kembalikan ke reviewer penerus", state(StatusInApproval, approver, rev2), ActionReturn, approver, Decision{
			ActorRole: RoleApprover, To: StatusReturnedToReviewer, NextAssigneeID: rev2, ReturnedByApprover: set(approver)}},
		{"7 reviewer yang dikembalikan setuju langsung ke approver", state(StatusReturnedToReviewer, rev2, rev2), ActionApprove, rev2, Decision{
			ActorRole: RoleReviewer, To: StatusInApproval, NextAssigneeID: approver,
			ForwardedBy: set(rev2), ReturnedByApprover: clearID}},
		{"8 reviewer yang dikembalikan kembalikan ke drafter", state(StatusReturnedToReviewer, rev2, rev2), ActionReturn, rev2, Decision{
			ActorRole: RoleReviewer, To: StatusReturnedToDrafter, NextAssigneeID: drafter,
			ReturnedByReviewer: set(rev2), ReturnedByApprover: clearID}},
		{"9 approver tanda tangan", state(StatusInApproval, approver, rev2), ActionSign, approver, Decision{
			ActorRole: RoleApprover, To: StatusSent, NextAssigneeID: 0, Sign: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decide(tt.s, tt.action, tt.actor)
			if err != nil {
				t.Fatalf("Decide() error = %v", err)
			}
			tt.want.Action, tt.want.From = tt.action, tt.s.Status
			if got != tt.want {
				t.Fatalf("Decide()\n got  %+v\n want %+v", got, tt.want)
			}
		})
	}
}

func TestDecideRejected(t *testing.T) {
	noReviewers := state(StatusDraft, 0, 0)
	noReviewers.Reviewers = nil
	noApprover := state(StatusDraft, 0, 0)
	noApprover.ApproverID = 0

	tests := []struct {
		name    string
		s       State
		action  Action
		actor   int64
		wantErr error
	}{
		// SUBMIT
		{"submit dua kali", state(StatusInReview, rev1, 0), ActionSubmit, drafter, ErrInvalidState},
		{"submit oleh bukan drafter", state(StatusDraft, 0, 0), ActionSubmit, rev1, ErrNotAllowed},
		{"submit tanpa reviewer", noReviewers, ActionSubmit, drafter, ErrIncomplete},
		{"submit tanpa approver", noApprover, ActionSubmit, drafter, ErrIncomplete},
		{"submit saat menunggu approver", state(StatusInApproval, approver, rev2), ActionSubmit, drafter, ErrInvalidState},
		{"submit saat dikembalikan ke reviewer", state(StatusReturnedToReviewer, rev2, rev2), ActionSubmit, drafter, ErrInvalidState},
		{"submit surat terkirim", state(StatusSent, 0, rev2), ActionSubmit, drafter, ErrInvalidState},
		{"submit surat batal", state(StatusCancelled, 0, 0), ActionSubmit, drafter, ErrInvalidState},
		// APPROVE
		{"setuju sebelum gilirannya", state(StatusInReview, rev1, 0), ActionApprove, rev2, ErrNotAllowed},
		{"setuju dua kali (sudah bukan assignee)", state(StatusInReview, rev2, rev1), ActionApprove, rev1, ErrNotAllowed},
		{"drafter setuju padahal bukan reviewer", state(StatusInReview, rev1, 0), ActionApprove, drafter, ErrNotAllowed},
		{"orang luar setuju", state(StatusInReview, rev1, 0), ActionApprove, outsider, ErrNotAllowed},
		{"approver memakai setuju", state(StatusInApproval, approver, rev2), ActionApprove, approver, ErrInvalidState},
		{"setuju draf", state(StatusDraft, 0, 0), ActionApprove, rev1, ErrInvalidState},
		{"setuju saat dikembalikan ke drafter", state(StatusReturnedToDrafter, drafter, 0), ActionApprove, rev1, ErrInvalidState},
		{"setuju surat terkirim", state(StatusSent, 0, rev2), ActionApprove, rev2, ErrInvalidState},
		{"reviewer lain setuju saat dikembalikan ke reviewer", state(StatusReturnedToReviewer, rev2, rev2), ActionApprove, rev1, ErrNotAllowed},
		// RETURN
		{"kembalikan oleh reviewer bukan assignee", state(StatusInReview, rev1, 0), ActionReturn, rev2, ErrNotAllowed},
		{"reviewer kembalikan saat menunggu approver", state(StatusInApproval, approver, rev2), ActionReturn, rev2, ErrNotAllowed},
		{"approver kembalikan tanpa reviewer penerus", state(StatusInApproval, approver, 0), ActionReturn, approver, ErrInvalidState},
		{"kembalikan draf", state(StatusDraft, 0, 0), ActionReturn, drafter, ErrInvalidState},
		{"kembalikan saat di drafter", state(StatusReturnedToDrafter, drafter, 0), ActionReturn, drafter, ErrInvalidState},
		{"kembalikan surat terkirim", state(StatusSent, 0, rev2), ActionReturn, approver, ErrInvalidState},
		{"approver kembalikan saat di reviewer", state(StatusReturnedToReviewer, rev2, rev2), ActionReturn, approver, ErrNotAllowed},
		// SIGN
		{"tanda tangan saat review", state(StatusInReview, rev1, 0), ActionSign, approver, ErrInvalidState},
		{"reviewer tanda tangan", state(StatusInApproval, approver, rev2), ActionSign, rev2, ErrNotAllowed},
		{"drafter tanda tangan", state(StatusInApproval, approver, rev2), ActionSign, drafter, ErrNotAllowed},
		{"tanda tangan dua kali", state(StatusSent, 0, rev2), ActionSign, approver, ErrInvalidState},
		{"tanda tangan saat dikembalikan ke reviewer", state(StatusReturnedToReviewer, rev2, rev2), ActionSign, approver, ErrInvalidState},
		// Lain-lain
		{"aksi tidak dikenal", state(StatusDraft, 0, 0), Action("CANCEL"), drafter, ErrUnknownAction},
		{"aktor kosong", state(StatusInReview, 0, 0), ActionApprove, 0, ErrNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Decide(tt.s, tt.action, tt.actor); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Decide() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// Drafter boleh menjadi reviewer: saat gilirannya tiba ia bertindak sebagai reviewer.
func TestDrafterAsReviewer(t *testing.T) {
	s := State{Status: StatusInReview, DrafterID: drafter, AssigneeID: drafter,
		Reviewers: []int64{drafter, rev2}, ApproverID: approver}
	d, err := Decide(s, ActionApprove, drafter)
	if err != nil || d.ActorRole != RoleReviewer || d.NextAssigneeID != rev2 {
		t.Fatalf("Decide() = %+v, %v", d, err)
	}
}

func TestAllowedActions(t *testing.T) {
	tests := []struct {
		name string
		s    State
		user int64
		want []Action
	}{
		{"drafter di draf", state(StatusDraft, 0, 0), drafter, []Action{ActionSubmit}},
		{"drafter saat dikembalikan", state(StatusReturnedToDrafter, drafter, 0), drafter, []Action{ActionSubmit}},
		{"reviewer aktif", state(StatusInReview, rev1, 0), rev1, []Action{ActionApprove, ActionReturn}},
		{"reviewer belum giliran", state(StatusInReview, rev1, 0), rev2, nil},
		{"approver aktif", state(StatusInApproval, approver, rev2), approver, []Action{ActionReturn, ActionSign}},
		{"reviewer yang dikembalikan", state(StatusReturnedToReviewer, rev2, rev2), rev2, []Action{ActionApprove, ActionReturn}},
		{"orang luar", state(StatusInReview, rev1, 0), outsider, nil},
		{"surat terkirim", state(StatusSent, 0, rev2), approver, nil},
	}
	for _, tt := range tests {
		if got := AllowedActions(tt.s, tt.user); !slices.Equal(got, tt.want) {
			t.Errorf("%s: AllowedActions() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestHelpers(t *testing.T) {
	s := state(StatusInReview, rev1, 0)
	if ReviewerStep(s, rev2) != 2 || ReviewerStep(s, outsider) != 0 {
		t.Error("ReviewerStep salah")
	}
	if !IsEditable(StatusDraft) || !IsEditable(StatusReturnedToDrafter) || IsEditable(StatusInReview) {
		t.Error("IsEditable salah")
	}
	if !IsValidStatus("SENT") || IsValidStatus("sent") || IsValidStatus("ABC") {
		t.Error("IsValidStatus salah")
	}
}
