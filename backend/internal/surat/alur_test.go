package surat

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// Pemeran uji: drafter Andi (15), reviewer Fajar (9) lalu Dewi (4), approver Agus (3) lalu
// Ratna (2), orang luar (77).
const (
	andi     int64 = 15
	fajar    int64 = 9
	dewi     int64 = 4
	agus     int64 = 3
	ratna    int64 = 2
	outsider int64 = 77
)

func state(status string, assignee int64) State {
	return State{Status: status, DrafterID: andi, AssigneeID: assignee,
		Reviewers: []int64{fajar, dewi}, Approvers: []int64{agus, ratna}}
}

func external(s State) State {
	s.External = true
	return s
}

// TestDecideValid menguji setiap baris tabel transisi.
func TestDecideValid(t *testing.T) {
	submit := Decision{Action: Submit, ActorRole: RoleDrafter, To: StatusInReview, AssigneeID: fajar, AssigneeRole: RoleReviewer,
		ForwardedBy: cleared(), ReturnedByReviewer: cleared(), ReturnedByApprover: cleared(), SnapshotSenderUnit: true}
	withFrom := func(d Decision, from string) Decision { d.From = from; return d }
	tests := []struct {
		name   string
		state  State
		action Action
		actor  int64
		want   Decision
	}{
		{"submit draf", state(StatusDraft, 0), Submit, andi, withFrom(submit, StatusDraft)},
		{"submit ulang mulai dari reviewer 1", state(StatusReturnedToDrafter, andi), Submit, andi,
			withFrom(submit, StatusReturnedToDrafter)},
		{"reviewer 1 setuju ke reviewer 2", state(StatusInReview, fajar), Approve, fajar, Decision{
			Action: Approve, ActorRole: RoleReviewer, From: StatusInReview, To: StatusInReview,
			AssigneeID: dewi, AssigneeRole: RoleReviewer}},
		{"reviewer terakhir setuju ke approver 1", state(StatusInReview, dewi), Approve, dewi, Decision{
			Action: Approve, ActorRole: RoleReviewer, From: StatusInReview, To: StatusInApproval,
			AssigneeID: agus, AssigneeRole: RoleApprover, ForwardedBy: setTo(dewi)}},
		{"approver 1 setuju ke approver 2", state(StatusInApproval, agus), Approve, agus, Decision{
			Action: Approve, ActorRole: RoleApprover, From: StatusInApproval, To: StatusInApproval,
			AssigneeID: ratna, AssigneeRole: RoleApprover}},
		{"approver terakhir setuju, surat terbit", state(StatusInApproval, ratna), Approve, ratna, Decision{
			Action: Sign, ActorRole: RoleApprover, From: StatusInApproval, To: StatusSent}},
		{"surat eksternal: approver terakhir setuju, menunggu tanda tangan", external(state(StatusInApproval, ratna)), Approve, ratna, Decision{
			Action: Sign, ActorRole: RoleApprover, From: StatusInApproval, To: StatusApproved,
			AssigneeID: ratna, AssigneeRole: RoleApprover}},
		{"reviewer kembalikan ke drafter", state(StatusInReview, dewi), Return, dewi, Decision{
			Action: Return, ActorRole: RoleReviewer, From: StatusInReview, To: StatusReturnedToDrafter,
			AssigneeID: andi, AssigneeRole: RoleDrafter, ReturnedByReviewer: setTo(dewi)}},
		{"approver 2 kembalikan ke drafter (bukan ke reviewer)", state(StatusInApproval, ratna), Return, ratna, Decision{
			Action: Return, ActorRole: RoleApprover, From: StatusInApproval, To: StatusReturnedToDrafter,
			AssigneeID: andi, AssigneeRole: RoleDrafter, ReturnedByApprover: setTo(ratna)}},
		{"data lama dikembalikan ke reviewer: reviewer terakhir lanjut ke approver 1", state(StatusReturnedToReviewer, dewi), Approve, dewi, Decision{
			Action: Approve, ActorRole: RoleReviewer, From: StatusReturnedToReviewer, To: StatusInApproval,
			AssigneeID: agus, AssigneeRole: RoleApprover, ForwardedBy: setTo(dewi)}},
	}
	for _, tt := range tests {
		got, err := Decide(tt.state, tt.action, tt.actor)
		if err != nil || got != tt.want {
			t.Errorf("%s:\n got  %+v, %v\n want %+v", tt.name, got, err, tt.want)
		}
	}
}

// TestDecideMatrix mencoba semua kombinasi status × aksi × pemeran. Hanya kombinasi di allowed
// yang boleh; sisanya wajib ErrInvalidState, dan AllowedActions harus selalu sejalan dengan Decide.
func TestDecideMatrix(t *testing.T) {
	states := []State{
		state(StatusDraft, 0),
		state(StatusInReview, fajar),
		state(StatusReturnedToDrafter, andi),
		state(StatusInApproval, agus),
		state(StatusInApproval, ratna),
		state(StatusReturnedToReviewer, dewi),
		state(StatusApproved, ratna),
		state(StatusSent, 0),
		state(StatusReceived, 0),
		state(StatusCancelled, 0),
	}
	allowed := map[string]bool{
		fmt.Sprint(StatusDraft, Submit, andi, 0):                  true,
		fmt.Sprint(StatusReturnedToDrafter, Submit, andi, andi):   true,
		fmt.Sprint(StatusInReview, Approve, fajar, fajar):         true,
		fmt.Sprint(StatusInReview, Return, fajar, fajar):          true,
		fmt.Sprint(StatusInApproval, Approve, agus, agus):         true,
		fmt.Sprint(StatusInApproval, Return, agus, agus):          true,
		fmt.Sprint(StatusInApproval, Approve, ratna, ratna):       true,
		fmt.Sprint(StatusInApproval, Return, ratna, ratna):        true,
		fmt.Sprint(StatusReturnedToReviewer, Approve, dewi, dewi): true,
		fmt.Sprint(StatusReturnedToReviewer, Return, dewi, dewi):  true,
	}
	count := 0
	for _, s := range states {
		for _, actor := range []int64{andi, fajar, dewi, agus, ratna, outsider} {
			var wantActions []Action
			for _, a := range []Action{Submit, Approve, Return} {
				key := fmt.Sprint(s.Status, a, actor, s.AssigneeID)
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
	s := State{Status: StatusDraft, DrafterID: andi, Reviewers: []int64{andi, dewi}, Approvers: []int64{agus}}
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
		{Status: StatusDraft, DrafterID: andi, Approvers: []int64{agus}},
		{Status: StatusDraft, DrafterID: andi, Reviewers: []int64{fajar}},
	} {
		if _, err := Decide(s, Submit, andi); !errors.Is(err, ErrIncomplete) {
			t.Errorf("submit tidak lengkap %+v: error = %v, want ErrIncomplete", s, err)
		}
	}

	// Approver 2 tidak bisa menyetujui sebelum gilirannya
	if _, err := Decide(state(StatusInApproval, agus), Approve, ratna); !errors.Is(err, ErrInvalidState) {
		t.Errorf("approver 2 sebelum giliran: error = %v", err)
	}

	// Actor 0 (tidak dikenal) tidak pernah boleh beraksi, walau assignee kosong
	if _, err := Decide(State{Status: StatusDraft}, Submit, 0); !errors.Is(err, ErrInvalidState) {
		t.Errorf("actor 0: error = %v", err)
	}
}

// Unggah scan bertanda tangan: hanya approver terakhir surat eksternal yang berstatus APPROVED.
func TestCanUploadSigned(t *testing.T) {
	ok := external(state(StatusApproved, ratna))
	if !CanUploadSigned(ok, ratna) || !slices.Contains(AllowedActions(ok, ratna), UploadSigned) {
		t.Error("approver terakhir surat eksternal APPROVED harus bisa mengunggah scan")
	}
	for name, tc := range map[string]struct {
		s     State
		actor int64
	}{
		"bukan approver terakhir": {ok, agus},
		"drafter":                 {ok, andi},
		"surat internal":          {state(StatusApproved, ratna), ratna},
		"belum disetujui":         {external(state(StatusInApproval, ratna)), ratna},
		"sudah terkirim":          {external(state(StatusSent, 0)), ratna},
	} {
		if CanUploadSigned(tc.s, tc.actor) || slices.Contains(AllowedActions(tc.s, tc.actor), UploadSigned) {
			t.Errorf("%s: tidak boleh mengunggah scan", name)
		}
	}
}

func TestStepState(t *testing.T) {
	steps := func(status string, assignee int64) State {
		return State{Status: status, DrafterID: andi, AssigneeID: assignee,
			Reviewers: []int64{fajar, dewi, 5}, Approvers: []int64{agus, ratna}}
	}
	tests := []struct {
		name  string
		state State
		want  string // R1 R2 R3 A1 A2
	}{
		{"draf", steps(StatusDraft, 0), "PENDING PENDING PENDING PENDING PENDING"},
		{"di reviewer 2", steps(StatusInReview, dewi), "DONE ACTIVE PENDING PENDING PENDING"},
		{"dikembalikan ke drafter", steps(StatusReturnedToDrafter, andi), "PENDING PENDING PENDING PENDING PENDING"},
		{"di approver 1", steps(StatusInApproval, agus), "DONE DONE DONE ACTIVE PENDING"},
		{"di approver 2", steps(StatusInApproval, ratna), "DONE DONE DONE DONE ACTIVE"},
		{"menunggu tanda tangan", steps(StatusApproved, ratna), "DONE DONE DONE DONE DONE"},
		{"terkirim", steps(StatusSent, 0), "DONE DONE DONE DONE DONE"},
	}
	for _, tt := range tests {
		got := strings.Join([]string{StepState(tt.state, RoleReviewer, 1), StepState(tt.state, RoleReviewer, 2),
			StepState(tt.state, RoleReviewer, 3), StepState(tt.state, RoleApprover, 1), StepState(tt.state, RoleApprover, 2)}, " ")
		if got != tt.want {
			t.Errorf("%s: %s, want %s", tt.name, got, tt.want)
		}
	}
}

// TestSamePersonAllRoles: pejabat yang sekaligus drafter, satu-satunya reviewer, dan approver bisa
// menjalankan seluruh alur sendiri; pengembalian olehnya sebagai approver kembali ke dirinya sebagai drafter.
func TestSamePersonAllRoles(t *testing.T) {
	s := State{Status: StatusDraft, DrafterID: agus, Reviewers: []int64{agus}, Approvers: []int64{agus}}
	steps := []struct {
		action   Action
		allowed  string // tombol sebelum aksi
		role     string
		to       string
		recorded Action
	}{
		{Submit, "SUBMIT", RoleDrafter, StatusInReview, Submit},
		{Approve, "APPROVE,RETURN", RoleReviewer, StatusInApproval, Approve},
		{Return, "APPROVE,RETURN", RoleApprover, StatusReturnedToDrafter, Return},
		{Submit, "SUBMIT", RoleDrafter, StatusInReview, Submit},
		{Approve, "APPROVE,RETURN", RoleReviewer, StatusInApproval, Approve},
		{Approve, "APPROVE,RETURN", RoleApprover, StatusSent, Sign},
	}
	for i, st := range steps {
		var allowed []string
		for _, a := range AllowedActions(s, agus) {
			allowed = append(allowed, string(a))
		}
		d, err := Decide(s, st.action, agus)
		if err != nil || strings.Join(allowed, ",") != st.allowed || d.ActorRole != st.role || d.To != st.to || d.Action != st.recorded {
			t.Fatalf("langkah %d (%s di %s): allowed=%v err=%v decision=%+v", i+1, st.action, s.Status, allowed, err, d)
		}
		s.Status, s.AssigneeID = d.To, d.AssigneeID
		if d.To != StatusSent && d.AssigneeID != agus {
			t.Fatalf("langkah %d: surat harus tetap dipegang orang yang sama, assignee=%d", i+1, d.AssigneeID)
		}
	}
}

func TestAssigneeRole(t *testing.T) {
	want := map[string]string{
		StatusDraft: "", StatusInReview: RoleReviewer, StatusReturnedToReviewer: RoleReviewer,
		StatusInApproval: RoleApprover, StatusApproved: RoleApprover, StatusReturnedToDrafter: RoleDrafter,
		StatusSent: "", StatusReceived: "",
	}
	for status, role := range want {
		if got := AssigneeRole(status); got != role {
			t.Errorf("AssigneeRole(%s) = %q, want %q", status, got, role)
		}
	}
}
