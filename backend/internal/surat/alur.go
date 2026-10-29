package surat

// Aturan perpindahan status surat (mesin status) sebagai fungsi murni:
// tanpa database dan tanpa HTTP, sehingga setiap transisi bisa diuji tuntas.
//
// Tabel transisi (status → status, penerima tugas berikutnya):
//
//	SUBMIT   drafter              DRAFT / RETURNED_TO_DRAFTER → IN_REVIEW        reviewer 1
//	APPROVE  reviewer aktif       IN_REVIEW → IN_REVIEW                          reviewer berikutnya
//	APPROVE  reviewer terakhir    IN_REVIEW → IN_APPROVAL                        approver
//	RETURN   reviewer aktif       IN_REVIEW → RETURNED_TO_DRAFTER                drafter
//	RETURN   approver             IN_APPROVAL → RETURNED_TO_REVIEWER             reviewer yang meneruskan
//	APPROVE  approver             IN_APPROVAL → SENT (dicatat sebagai SIGN)      - (surat terbit)
//	APPROVE  reviewer dikembalikan RETURNED_TO_REVIEWER → IN_APPROVAL            approver (langsung)
//	RETURN   reviewer dikembalikan RETURNED_TO_REVIEWER → RETURNED_TO_DRAFTER    drafter

import (
	"errors"
	"slices"
)

// Action adalah aksi alur persetujuan.
type Action string

// Aksi yang dikenal.
const (
	Submit  Action = "SUBMIT"
	Approve Action = "APPROVE"
	Return  Action = "RETURN"
	// Sign adalah APPROVE oleh approver: surat disetujui dan terbit. Tombolnya tetap APPROVE;
	// Sign hanya muncul di Decision dan riwayat.
	Sign Action = "SIGN"
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
		d.To = StatusInReview
		d.AssigneeID, d.AssigneeRole = s.Reviewers[0], RoleReviewer // selalu mulai dari reviewer 1
		d.ForwardedBy, d.ReturnedByReviewer = cleared(), cleared()
		d.SnapshotSenderUnit = true

	case a == Approve && s.Status == StatusInReview:
		d.ForwardedBy = setTo(actorID)
		if i := slices.Index(s.Reviewers, actorID); i < len(s.Reviewers)-1 {
			d.To = StatusInReview
			d.AssigneeID, d.AssigneeRole = s.Reviewers[i+1], RoleReviewer
		} else {
			if s.ApproverID == 0 {
				return Decision{}, ErrIncomplete
			}
			d.To = StatusInApproval
			d.AssigneeID, d.AssigneeRole = s.ApproverID, RoleApprover
		}

	case a == Approve && role == RoleApprover:
		d.Action = Sign
		d.To = StatusSent // tidak ada penerima tugas lagi; surat masuk inbox penerima

	case a == Approve && s.Status == StatusReturnedToReviewer:
		// Reviewer yang dikembalikan approver meneruskan langsung ke approver, tanpa mengulang reviewer
		if s.ApproverID == 0 {
			return Decision{}, ErrIncomplete
		}
		d.To = StatusInApproval
		d.AssigneeID, d.AssigneeRole = s.ApproverID, RoleApprover
		d.ForwardedBy, d.ReturnedByApprover = setTo(actorID), cleared()

	case a == Return && role == RoleApprover:
		d.To = StatusReturnedToReviewer
		d.AssigneeID, d.AssigneeRole = s.ForwardedByID, RoleReviewer
		if d.AssigneeID == 0 && len(s.Reviewers) > 0 {
			d.AssigneeID = s.Reviewers[len(s.Reviewers)-1] // data lama tanpa forwarded_by_id
		}
		d.ReturnedByApprover = setTo(actorID)

	case a == Return: // reviewer, dari IN_REVIEW atau RETURNED_TO_REVIEWER
		d.To = StatusReturnedToDrafter
		d.AssigneeID, d.AssigneeRole = s.DrafterID, RoleDrafter
		d.ReturnedByReviewer = setTo(actorID)
		if s.Status == StatusReturnedToReviewer {
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
	isDraftable := s.Status == StatusDraft || s.Status == StatusReturnedToDrafter
	isReviewing := s.Status == StatusInReview || s.Status == StatusReturnedToReviewer
	isAssignedReviewer := isReviewing && s.AssigneeID == actorID && slices.Contains(s.Reviewers, actorID)
	isAssignedApprover := s.Status == StatusInApproval && s.AssigneeID == actorID && s.ApproverID == actorID

	switch {
	case a == Submit && isDraftable && s.DrafterID == actorID:
		return RoleDrafter, true
	case a == Approve && isAssignedReviewer:
		return RoleReviewer, true
	case a == Return && isAssignedReviewer:
		return RoleReviewer, true
	case a == Approve && isAssignedApprover:
		return RoleApprover, true
	case a == Return && isAssignedApprover:
		return RoleApprover, true
	}
	return "", false
}

// AssigneeRole adalah peran penerima tugas pada status tertentu ("" bila status tidak punya
// penerima tugas, misalnya DRAFT).
func AssigneeRole(status string) string {
	switch status {
	case StatusInReview, StatusReturnedToReviewer:
		return RoleReviewer
	case StatusInApproval:
		return RoleApprover
	case StatusReturnedToDrafter:
		return RoleDrafter
	}
	return ""
}

// StepState menghitung keadaan tahap (role, order) untuk ditampilkan: DONE, ACTIVE, atau PENDING.
func StepState(s State, role string, order int) string {
	switch s.Status {
	case StatusSent:
		return StepDone
	case StatusInApproval:
		if role == RoleApprover {
			return StepActive
		}
		return StepDone
	case StatusInReview, StatusReturnedToReviewer:
		if role == RoleApprover {
			return StepPending
		}
		active := slices.Index(s.Reviewers, s.AssigneeID) + 1
		switch {
		case order == active:
			return StepActive
		case order < active || s.Status == StatusReturnedToReviewer:
			return StepDone // saat dikembalikan approver, semua reviewer lain sudah memaraf
		}
	}
	return StepPending // DRAFT dan RETURNED_TO_DRAFTER: proses diulang dari awal
}
