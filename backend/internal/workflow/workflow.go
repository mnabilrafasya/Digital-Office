// Package workflow berisi aturan perpindahan status surat (mesin status) sebagai fungsi murni:
// tanpa database dan tanpa HTTP, sehingga setiap transisi bisa diuji tuntas.
//
// Tabel transisi (status → status, penerima tugas berikutnya):
//
//	SUBMIT   drafter              DRAFT / RETURNED_TO_DRAFTER → IN_REVIEW        reviewer 1
//	APPROVE  reviewer aktif       IN_REVIEW → IN_REVIEW                          reviewer berikutnya
//	APPROVE  reviewer terakhir    IN_REVIEW → IN_APPROVAL                        approver
//	RETURN   reviewer aktif       IN_REVIEW → RETURNED_TO_DRAFTER                drafter
//	RETURN   approver             IN_APPROVAL → RETURNED_TO_REVIEWER             reviewer yang meneruskan
//	APPROVE  reviewer dikembalikan RETURNED_TO_REVIEWER → IN_APPROVAL            approver (langsung)
//	RETURN   reviewer dikembalikan RETURNED_TO_REVIEWER → RETURNED_TO_DRAFTER    drafter
package workflow

import (
	"errors"
	"slices"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// Action adalah aksi alur persetujuan.
type Action string

// Aksi yang dikenal.
const (
	Submit  Action = "SUBMIT"
	Approve Action = "APPROVE"
	Return  Action = "RETURN"
)

var (
	// ErrInvalidState: aksi tidak tersedia untuk user ini pada status surat sekarang (bukan
	// gilirannya, status salah, atau perannya tidak cocok). Termasuk klik dua kali.
	ErrInvalidState = errors.New("aksi tidak sesuai status surat")
	// ErrIncomplete: surat belum punya reviewer/approver yang dibutuhkan transisi.
	ErrIncomplete = errors.New("reviewer atau approver surat belum lengkap")
)

// State adalah data surat yang menentukan transisi. ID 0 berarti kosong.
type State struct {
	Status        string
	DrafterID     int64
	AssigneeID    int64
	ForwardedByID int64   // reviewer terakhir yang meneruskan ke approver
	Reviewers     []int64 // urut sesuai step
	ApproverID    int64
}

// Field adalah perubahan satu kolom user id di surat. Changed=false: kolom dibiarkan;
// UserID 0: kolom dikosongkan (NULL).
type Field struct {
	Changed bool
	UserID  int64
}

func setTo(userID int64) Field { return Field{Changed: true, UserID: userID} }
func cleared() Field           { return Field{Changed: true} }

// Decision adalah hasil transisi yang harus diterapkan ke surat.
type Decision struct {
	Action             Action
	ActorRole          string // peran aktor: DRAFTER, REVIEWER, atau APPROVER
	From, To           string
	AssigneeID         int64  // penerima tugas berikutnya; 0 = tidak ada
	AssigneeRole       string // peran penerima tugas berikutnya
	ForwardedBy        Field
	ReturnedByReviewer Field
	ReturnedByApprover Field
	SnapshotSenderUnit bool // simpan nama unit pengirim saat surat mulai diproses
}

// Decide menghitung transisi aksi oleh actorID. ErrInvalidState bila aksi tidak tersedia untuk
// actorID, ErrIncomplete bila reviewer/approver yang dibutuhkan belum ada.
func Decide(s State, a Action, actorID int64) (Decision, error) {
	role, ok := actorRole(s, a, actorID)
	if !ok {
		return Decision{}, ErrInvalidState
	}
	d := Decision{Action: a, ActorRole: role, From: s.Status}

	switch {
	case a == Submit:
		if len(s.Reviewers) == 0 || s.ApproverID == 0 {
			return Decision{}, ErrIncomplete
		}
		d.To = model.StatusInReview
		d.AssigneeID, d.AssigneeRole = s.Reviewers[0], model.RoleReviewer // selalu mulai dari reviewer 1
		d.ForwardedBy, d.ReturnedByReviewer = cleared(), cleared()
		d.SnapshotSenderUnit = true

	case a == Approve && s.Status == model.StatusInReview:
		d.ForwardedBy = setTo(actorID)
		if i := slices.Index(s.Reviewers, actorID); i < len(s.Reviewers)-1 {
			d.To = model.StatusInReview
			d.AssigneeID, d.AssigneeRole = s.Reviewers[i+1], model.RoleReviewer
		} else {
			if s.ApproverID == 0 {
				return Decision{}, ErrIncomplete
			}
			d.To = model.StatusInApproval
			d.AssigneeID, d.AssigneeRole = s.ApproverID, model.RoleApprover
		}

	case a == Approve && s.Status == model.StatusReturnedToReviewer:
		// Reviewer yang dikembalikan approver meneruskan langsung ke approver, tanpa mengulang reviewer
		if s.ApproverID == 0 {
			return Decision{}, ErrIncomplete
		}
		d.To = model.StatusInApproval
		d.AssigneeID, d.AssigneeRole = s.ApproverID, model.RoleApprover
		d.ForwardedBy, d.ReturnedByApprover = setTo(actorID), cleared()

	case a == Return && role == model.RoleApprover:
		d.To = model.StatusReturnedToReviewer
		d.AssigneeID, d.AssigneeRole = s.ForwardedByID, model.RoleReviewer
		if d.AssigneeID == 0 && len(s.Reviewers) > 0 {
			d.AssigneeID = s.Reviewers[len(s.Reviewers)-1] // data lama tanpa forwarded_by_id
		}
		d.ReturnedByApprover = setTo(actorID)

	case a == Return: // reviewer, dari IN_REVIEW atau RETURNED_TO_REVIEWER
		d.To = model.StatusReturnedToDrafter
		d.AssigneeID, d.AssigneeRole = s.DrafterID, model.RoleDrafter
		d.ReturnedByReviewer = setTo(actorID)
		if s.Status == model.StatusReturnedToReviewer {
			d.ReturnedByApprover = cleared()
		}
	}
	return d, nil
}

// AllowedActions mengembalikan aksi alur yang boleh dilakukan userID saat ini (untuk tombol di
// front-end). Kelengkapan surat tidak diperiksa di sini; itu divalidasi saat aksi dijalankan.
func AllowedActions(s State, userID int64) []Action {
	out := []Action{}
	for _, a := range []Action{Submit, Approve, Return} {
		if _, ok := actorRole(s, a, userID); ok {
			out = append(out, a)
		}
	}
	return out
}

// actorRole menentukan peran actorID untuk aksi a, atau false bila aksi tidak tersedia baginya.
// Selain SUBMIT, hanya penerima tugas saat ini (assignee) yang boleh beraksi.
func actorRole(s State, a Action, actorID int64) (string, bool) {
	if actorID == 0 {
		return "", false
	}
	isDraftable := s.Status == model.StatusDraft || s.Status == model.StatusReturnedToDrafter
	isReviewing := s.Status == model.StatusInReview || s.Status == model.StatusReturnedToReviewer
	isAssignedReviewer := isReviewing && s.AssigneeID == actorID && slices.Contains(s.Reviewers, actorID)
	isAssignedApprover := s.Status == model.StatusInApproval && s.AssigneeID == actorID && s.ApproverID == actorID

	switch {
	case a == Submit && isDraftable && s.DrafterID == actorID:
		return model.RoleDrafter, true
	case a == Approve && isAssignedReviewer:
		return model.RoleReviewer, true
	case a == Return && isAssignedReviewer:
		return model.RoleReviewer, true
	case a == Return && isAssignedApprover:
		return model.RoleApprover, true
	}
	return "", false
}

// AssigneeRole adalah peran penerima tugas pada status tertentu ("" bila status tidak punya
// penerima tugas, misalnya DRAFT).
func AssigneeRole(status string) string {
	switch status {
	case model.StatusInReview, model.StatusReturnedToReviewer:
		return model.RoleReviewer
	case model.StatusInApproval:
		return model.RoleApprover
	case model.StatusReturnedToDrafter:
		return model.RoleDrafter
	}
	return ""
}

// StepState menghitung keadaan tahap (role, order) untuk ditampilkan: DONE, ACTIVE, atau PENDING.
func StepState(s State, role string, order int) string {
	switch s.Status {
	case model.StatusSent:
		return model.StepDone
	case model.StatusInApproval:
		if role == model.RoleApprover {
			return model.StepActive
		}
		return model.StepDone
	case model.StatusInReview, model.StatusReturnedToReviewer:
		if role == model.RoleApprover {
			return model.StepPending
		}
		active := slices.Index(s.Reviewers, s.AssigneeID) + 1
		switch {
		case order == active:
			return model.StepActive
		case order < active || s.Status == model.StatusReturnedToReviewer:
			return model.StepDone // saat dikembalikan approver, semua reviewer lain sudah memaraf
		}
	}
	return model.StepPending // DRAFT dan RETURNED_TO_DRAFTER: proses diulang dari awal
}
