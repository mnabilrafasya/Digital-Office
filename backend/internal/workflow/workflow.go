// Package workflow berisi mesin status surat sebagai fungsi murni (tanpa database dan HTTP),
// supaya setiap perpindahan status bisa diuji tuntas.
package workflow

import (
	"errors"
	"slices"
)

// Status adalah status surat (sama dengan CHECK ck_letters_status).
type Status string

// Daftar status surat.
const (
	StatusDraft              Status = "DRAFT"
	StatusInReview           Status = "IN_REVIEW"
	StatusReturnedToDrafter  Status = "RETURNED_TO_DRAFTER"
	StatusInApproval         Status = "IN_APPROVAL"
	StatusReturnedToReviewer Status = "RETURNED_TO_REVIEWER"
	StatusSent               Status = "SENT"
	StatusCancelled          Status = "CANCELLED"
)

// AllStatuses adalah semua status yang dikenal.
var AllStatuses = []Status{
	StatusDraft, StatusInReview, StatusReturnedToDrafter, StatusInApproval,
	StatusReturnedToReviewer, StatusSent, StatusCancelled,
}

// TaskStatuses adalah status yang berarti surat sedang menunggu tindakan assignee-nya.
var TaskStatuses = []Status{StatusInReview, StatusInApproval, StatusReturnedToReviewer, StatusReturnedToDrafter}

// Action adalah aksi alur persetujuan.
type Action string

// Daftar aksi alur persetujuan.
const (
	ActionSubmit  Action = "SUBMIT"
	ActionApprove Action = "APPROVE"
	ActionReturn  Action = "RETURN"
	ActionSign    Action = "SIGN"
)

// Actions adalah semua aksi alur persetujuan, urut seperti tombol di layar.
var Actions = []Action{ActionSubmit, ActionApprove, ActionReturn, ActionSign}

// Role adalah peran aktor saat melakukan aksi.
type Role string

// Daftar peran aktor.
const (
	RoleDrafter  Role = "DRAFTER"
	RoleReviewer Role = "REVIEWER"
	RoleApprover Role = "APPROVER"
)

// Error keputusan alur.
var (
	// ErrInvalidState: aksi tidak berlaku untuk status surat saat ini (contoh: kirim dua kali).
	ErrInvalidState = errors.New("aksi tidak sesuai status surat")
	// ErrNotAllowed: status cocok, tetapi aktor bukan orang yang sedang memproses surat.
	ErrNotAllowed = errors.New("user bukan pemroses surat pada tahap ini")
	// ErrIncomplete: surat belum punya reviewer atau approver.
	ErrIncomplete = errors.New("reviewer dan approver surat belum lengkap")
	// ErrUnknownAction: aksi tidak dikenal.
	ErrUnknownAction = errors.New("aksi tidak dikenal")
)

// State adalah data surat yang menentukan aksi apa yang boleh dilakukan. ID 0 berarti kosong.
type State struct {
	Status        Status
	DrafterID     int64
	AssigneeID    int64
	ForwardedByID int64
	Reviewers     []int64 // urut step_order (step 1 di indeks 0)
	ApproverID    int64
}

// IDUpdate adalah perubahan satu kolom ID: Set=false berarti tidak diubah,
// Set=true dengan ID 0 berarti dikosongkan (NULL).
type IDUpdate struct {
	Set bool
	ID  int64
}

// Decision adalah hasil perhitungan satu aksi: status dan assignee baru serta kolom yang berubah.
type Decision struct {
	Action             Action
	ActorRole          Role
	From               Status
	To                 Status
	NextAssigneeID     int64 // 0 = tidak ada assignee
	ForwardedBy        IDUpdate
	ReturnedByReviewer IDUpdate
	ReturnedByApprover IDUpdate
	SnapshotSenderUnit bool // isi sender_unit_name (submit pertama)
	Sign               bool // nomori surat dan kirim ke penerima
}

func set(id int64) IDUpdate { return IDUpdate{Set: true, ID: id} }

var clearID = IDUpdate{Set: true}

// Decide menghitung hasil aksi a oleh actorID terhadap surat s. Hanya assignee saat ini yang boleh
// melakukan APPROVE, RETURN, dan SIGN, dengan peran yang cocok.
func Decide(s State, a Action, actorID int64) (Decision, error) {
	d := Decision{Action: a, From: s.Status}
	switch a {
	case ActionSubmit:
		return decideSubmit(s, d, actorID)
	case ActionApprove:
		return decideApprove(s, d, actorID)
	case ActionReturn:
		return decideReturn(s, d, actorID)
	case ActionSign:
		return decideSign(s, d, actorID)
	default:
		return d, ErrUnknownAction
	}
}

func decideSubmit(s State, d Decision, actor int64) (Decision, error) {
	if s.Status != StatusDraft && s.Status != StatusReturnedToDrafter {
		return d, ErrInvalidState
	}
	if actor != s.DrafterID {
		return d, ErrNotAllowed
	}
	if len(s.Reviewers) == 0 || s.ApproverID == 0 {
		return d, ErrIncomplete
	}
	d.ActorRole, d.To, d.NextAssigneeID = RoleDrafter, StatusInReview, s.Reviewers[0]
	if s.Status == StatusDraft {
		d.SnapshotSenderUnit = true
	} else {
		// Dikirim ulang: mulai lagi dari reviewer tahap 1
		d.ReturnedByReviewer, d.ForwardedBy = clearID, clearID
	}
	return d, nil
}

func decideApprove(s State, d Decision, actor int64) (Decision, error) {
	switch s.Status {
	case StatusInReview:
		step, ok := activeReviewerStep(s, actor)
		if !ok {
			return d, ErrNotAllowed
		}
		d.ActorRole, d.ForwardedBy = RoleReviewer, set(actor)
		if step < len(s.Reviewers)-1 {
			d.To, d.NextAssigneeID = StatusInReview, s.Reviewers[step+1]
		} else {
			d.To, d.NextAssigneeID = StatusInApproval, s.ApproverID
		}
		return d, nil

	case StatusReturnedToReviewer:
		if _, ok := activeReviewerStep(s, actor); !ok {
			return d, ErrNotAllowed
		}
		// Reviewer yang dikembalikan approver langsung meneruskan ke approver lagi
		d.ActorRole, d.To, d.NextAssigneeID = RoleReviewer, StatusInApproval, s.ApproverID
		d.ForwardedBy, d.ReturnedByApprover = set(actor), clearID
		return d, nil

	default:
		return d, ErrInvalidState
	}
}

func decideReturn(s State, d Decision, actor int64) (Decision, error) {
	switch s.Status {
	case StatusInReview:
		if _, ok := activeReviewerStep(s, actor); !ok {
			return d, ErrNotAllowed
		}
		d.ActorRole, d.To, d.NextAssigneeID = RoleReviewer, StatusReturnedToDrafter, s.DrafterID
		d.ReturnedByReviewer = set(actor)
		return d, nil

	case StatusInApproval:
		if !isActiveApprover(s, actor) {
			return d, ErrNotAllowed
		}
		if s.ForwardedByID == 0 {
			return d, ErrInvalidState
		}
		d.ActorRole, d.To, d.NextAssigneeID = RoleApprover, StatusReturnedToReviewer, s.ForwardedByID
		d.ReturnedByApprover = set(actor)
		return d, nil

	case StatusReturnedToReviewer:
		if _, ok := activeReviewerStep(s, actor); !ok {
			return d, ErrNotAllowed
		}
		d.ActorRole, d.To, d.NextAssigneeID = RoleReviewer, StatusReturnedToDrafter, s.DrafterID
		d.ReturnedByReviewer, d.ReturnedByApprover = set(actor), clearID
		return d, nil

	default:
		return d, ErrInvalidState
	}
}

func decideSign(s State, d Decision, actor int64) (Decision, error) {
	if s.Status != StatusInApproval {
		return d, ErrInvalidState
	}
	if !isActiveApprover(s, actor) {
		return d, ErrNotAllowed
	}
	d.ActorRole, d.To, d.NextAssigneeID, d.Sign = RoleApprover, StatusSent, 0, true
	return d, nil
}

// activeReviewerStep mengembalikan indeks tahap reviewer bila actor adalah assignee dan reviewer.
func activeReviewerStep(s State, actor int64) (int, bool) {
	if actor == 0 || actor != s.AssigneeID {
		return 0, false
	}
	i := slices.Index(s.Reviewers, actor)
	return i, i >= 0
}

func isActiveApprover(s State, actor int64) bool {
	return actor != 0 && actor == s.AssigneeID && actor == s.ApproverID
}

// AllowedActions mengembalikan aksi alur yang boleh dilakukan userID terhadap surat s saat ini.
func AllowedActions(s State, userID int64) []Action {
	var out []Action
	for _, a := range Actions {
		if _, err := Decide(s, a, userID); err == nil {
			out = append(out, a)
		}
	}
	return out
}

// ReviewerStep mengembalikan nomor tahap (mulai 1) userID sebagai reviewer, atau 0 bila bukan reviewer.
func ReviewerStep(s State, userID int64) int {
	return slices.Index(s.Reviewers, userID) + 1
}

// IsEditable mengembalikan true bila isi surat masih boleh diubah drafter.
func IsEditable(st Status) bool {
	return st == StatusDraft || st == StatusReturnedToDrafter
}

// IsValidStatus mengembalikan true bila st adalah status yang dikenal.
func IsValidStatus(st string) bool {
	return slices.Contains(AllStatuses, Status(st))
}
