package surat

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// Pemeran uji: drafter Andi (15), reviewer Fajar (9) lalu Dewi (4), approver Agus (3), orang luar (77).
const (
	andi     int64 = 15
	fajar    int64 = 9
	dewi     int64 = 4
	agus     int64 = 3
	outsider int64 = 77
)

func state(status string, assignee, forwardedBy int64) State {
	return State{Status: status, DrafterID: andi, AssigneeID: assignee, ForwardedByID: forwardedBy,
		Reviewers: []int64{fajar, dewi}, ApproverID: agus}
}

// TestDecideValid menguji 9 baris tabel transisi (4.2 baris 1-8, ditambah approver menyetujui).
func TestDecideValid(t *testing.T) {
	tests := []struct {
		name   string
		state  State
		action Action
		actor  int64
		want   Decision
	}{
		{"1 submit draf", state(StatusDraft, 0, 0), Submit, andi, Decision{
			Action: Submit, ActorRole: RoleDrafter, From: StatusDraft, To: StatusInReview,
			AssigneeID: fajar, AssigneeRole: RoleReviewer,
			ForwardedBy: cleared(), ReturnedByReviewer: cleared(), SnapshotSenderUnit: true}},
		{"2 submit ulang mulai dari reviewer 1", state(StatusReturnedToDrafter, andi, dewi), Submit, andi, Decision{
			Action: Submit, ActorRole: RoleDrafter, From: StatusReturnedToDrafter, To: StatusInReview,
			AssigneeID: fajar, AssigneeRole: RoleReviewer,
			ForwardedBy: cleared(), ReturnedByReviewer: cleared(), SnapshotSenderUnit: true}},
		{"3 reviewer 1 approve ke reviewer 2", state(StatusInReview, fajar, 0), Approve, fajar, Decision{
			Action: Approve, ActorRole: RoleReviewer, From: StatusInReview, To: StatusInReview,
			AssigneeID: dewi, AssigneeRole: RoleReviewer, ForwardedBy: setTo(fajar)}},
		{"4 reviewer terakhir approve ke approver", state(StatusInReview, dewi, fajar), Approve, dewi, Decision{
			Action: Approve, ActorRole: RoleReviewer, From: StatusInReview, To: StatusInApproval,
			AssigneeID: agus, AssigneeRole: RoleApprover, ForwardedBy: setTo(dewi)}},
		{"5 reviewer return ke drafter", state(StatusInReview, dewi, fajar), Return, dewi, Decision{
			Action: Return, ActorRole: RoleReviewer, From: StatusInReview, To: StatusReturnedToDrafter,
			AssigneeID: andi, AssigneeRole: RoleDrafter, ReturnedByReviewer: setTo(dewi)}},
		{"6 approver return ke reviewer yang meneruskan", state(StatusInApproval, agus, dewi), Return, agus, Decision{
			Action: Return, ActorRole: RoleApprover, From: StatusInApproval, To: StatusReturnedToReviewer,
			AssigneeID: dewi, AssigneeRole: RoleReviewer, ReturnedByApprover: setTo(agus)}},
		{"7 reviewer dikembalikan approve langsung ke approver", state(StatusReturnedToReviewer, dewi, dewi), Approve, dewi, Decision{
			Action: Approve, ActorRole: RoleReviewer, From: StatusReturnedToReviewer, To: StatusInApproval,
			AssigneeID: agus, AssigneeRole: RoleApprover, ForwardedBy: setTo(dewi), ReturnedByApprover: cleared()}},
		{"8 reviewer dikembalikan return ke drafter", state(StatusReturnedToReviewer, dewi, dewi), Return, dewi, Decision{
			Action: Return, ActorRole: RoleReviewer, From: StatusReturnedToReviewer, To: StatusReturnedToDrafter,
			AssigneeID: andi, AssigneeRole: RoleDrafter, ReturnedByReviewer: setTo(dewi), ReturnedByApprover: cleared()}},
		{"9 approver approve, surat terbit", state(StatusInApproval, agus, dewi), Approve, agus, Decision{
			Action: Sign, ActorRole: RoleApprover, From: StatusInApproval, To: StatusSent}},
	}
	for _, tt := range tests {
		got, err := Decide(tt.state, tt.action, tt.actor)
		if err != nil || got != tt.want {
			t.Errorf("%s:\n got  %+v, %v\n want %+v", tt.name, got, err, tt.want)
		}
	}
}

// TestDecideMatrix mencoba semua kombinasi status × aksi × pemeran. Hanya 8 kombinasi yang
// boleh; sisanya (bukan giliran, status salah, peran salah, orang luar) wajib ErrInvalidState,
// dan AllowedActions harus selalu sejalan dengan Decide.
func TestDecideMatrix(t *testing.T) {
	states := []State{
		state(StatusDraft, 0, 0),
		state(StatusInReview, fajar, 0),
		state(StatusReturnedToDrafter, andi, 0),
		state(StatusInApproval, agus, dewi),
		state(StatusReturnedToReviewer, dewi, dewi),
		state(StatusSent, 0, dewi),
		state(StatusCancelled, 0, 0),
	}
	allowed := map[string]bool{
		fmt.Sprint(StatusDraft, Submit, andi):               true,
		fmt.Sprint(StatusReturnedToDrafter, Submit, andi):   true,
		fmt.Sprint(StatusInReview, Approve, fajar):          true,
		fmt.Sprint(StatusInReview, Return, fajar):           true,
		fmt.Sprint(StatusInApproval, Approve, agus):         true,
		fmt.Sprint(StatusInApproval, Return, agus):          true,
		fmt.Sprint(StatusReturnedToReviewer, Approve, dewi): true,
		fmt.Sprint(StatusReturnedToReviewer, Return, dewi):  true,
	}
	count := 0
	for _, s := range states {
		for _, actor := range []int64{andi, fajar, dewi, agus, outsider} {
			var wantActions []Action
			for _, a := range []Action{Submit, Approve, Return} {
				key := fmt.Sprint(s.Status, a, actor)
				_, err := Decide(s, a, actor)
				if allowed[key] {
					count++
					wantActions = append(wantActions, a)
					if err != nil {
						t.Errorf("%s: error = %v, want nil", key, err)
					}
				} else if !errors.Is(err, ErrInvalidState) {
					t.Errorf("%s: error = %v, want ErrInvalidState", key, err)
				}
			}
			if got := AllowedActions(s, actor); !slices.Equal(got, wantActions) && (len(got) > 0 || len(wantActions) > 0) {
				t.Errorf("AllowedActions(%s, %d) = %v, want %v", s.Status, actor, got, wantActions)
			}
		}
	}
	if count != len(allowed) {
		t.Errorf("kombinasi yang diizinkan teruji %d, want %d", count, len(allowed))
	}
}

func TestDecideEdgeCases(t *testing.T) {
	// Drafter boleh menjadi reviewer 1: setelah submit, surat menjadi tugasnya sebagai reviewer
	s := State{Status: StatusDraft, DrafterID: andi, Reviewers: []int64{andi, dewi}, ApproverID: agus}
	d, err := Decide(s, Submit, andi)
	if err != nil || d.AssigneeID != andi {
		t.Fatalf("submit drafter-reviewer: %+v, %v", d, err)
	}
	s.Status, s.AssigneeID = d.To, d.AssigneeID
	if d, err := Decide(s, Approve, andi); err != nil || d.ActorRole != RoleReviewer || d.AssigneeID != dewi {
		t.Errorf("drafter sebagai reviewer approve: %+v, %v", d, err)
	}

	// Submit tanpa reviewer/approver
	for _, s := range []State{
		{Status: StatusDraft, DrafterID: andi, ApproverID: agus},
		{Status: StatusDraft, DrafterID: andi, Reviewers: []int64{fajar}},
	} {
		if _, err := Decide(s, Submit, andi); !errors.Is(err, ErrIncomplete) {
			t.Errorf("submit tidak lengkap %+v: error = %v, want ErrIncomplete", s, err)
		}
	}

	// Approver mengembalikan surat lama tanpa forwarded_by_id: ke reviewer terakhir
	if d, err := Decide(state(StatusInApproval, agus, 0), Return, agus); err != nil || d.AssigneeID != dewi {
		t.Errorf("return tanpa forwarded_by: %+v, %v", d, err)
	}

	// Actor 0 (tidak dikenal) tidak pernah boleh beraksi, walau assignee kosong
	if _, err := Decide(State{Status: StatusDraft}, Submit, 0); !errors.Is(err, ErrInvalidState) {
		t.Errorf("actor 0: error = %v", err)
	}
}

func TestStepState(t *testing.T) {
	three := func(status string, assignee int64) State {
		return State{Status: status, DrafterID: andi, AssigneeID: assignee, Reviewers: []int64{fajar, dewi, 5}, ApproverID: agus}
	}
	tests := []struct {
		name  string
		state State
		want  string // R1 R2 R3 A
	}{
		{"draf", three(StatusDraft, 0), "PENDING PENDING PENDING PENDING"},
		{"di reviewer 2", three(StatusInReview, dewi), "DONE ACTIVE PENDING PENDING"},
		{"dikembalikan ke drafter", three(StatusReturnedToDrafter, andi), "PENDING PENDING PENDING PENDING"},
		{"di approver", three(StatusInApproval, agus), "DONE DONE DONE ACTIVE"},
		{"dikembalikan approver ke reviewer 2", three(StatusReturnedToReviewer, dewi), "DONE ACTIVE DONE PENDING"},
		{"terkirim", three(StatusSent, 0), "DONE DONE DONE DONE"},
	}
	for _, tt := range tests {
		got := strings.Join([]string{StepState(tt.state, RoleReviewer, 1), StepState(tt.state, RoleReviewer, 2),
			StepState(tt.state, RoleReviewer, 3), StepState(tt.state, RoleApprover, 1)}, " ")
		if got != tt.want {
			t.Errorf("%s: %s, want %s", tt.name, got, tt.want)
		}
	}
}

func TestAssigneeRole(t *testing.T) {
	want := map[string]string{
		StatusDraft: "", StatusInReview: RoleReviewer, StatusReturnedToReviewer: RoleReviewer,
		StatusInApproval: RoleApprover, StatusReturnedToDrafter: RoleDrafter, StatusSent: "",
	}
	for status, role := range want {
		if got := AssigneeRole(status); got != role {
			t.Errorf("AssigneeRole(%s) = %q, want %q", status, got, role)
		}
	}
}
