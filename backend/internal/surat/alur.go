package surat

// Aturan perpindahan status surat (mesin status) sebagai fungsi murni:
// tanpa database dan tanpa HTTP, sehingga setiap transisi bisa diuji tuntas.
//
// Reviewer dan approver sama-sama berurutan (tahap 1..n). Setiap pengembalian, oleh reviewer maupun
// approver, selalu ke drafter; setelah diajukan ulang alur mulai lagi dari reviewer 1.
//
// Tabel transisi (status → status, penerima tugas berikutnya):
//
//	SUBMIT   drafter                DRAFT / RETURNED_TO_DRAFTER → IN_REVIEW      reviewer 1
//	APPROVE  reviewer (bukan akhir) IN_REVIEW → IN_REVIEW                        reviewer berikutnya
//	APPROVE  reviewer terakhir      IN_REVIEW → IN_APPROVAL                      approver 1
//	APPROVE  approver (bukan akhir) IN_APPROVAL → IN_APPROVAL                    approver berikutnya
//	APPROVE  approver terakhir      IN_APPROVAL → SENT (dicatat SIGN)            - (surat terbit)
//	         surat eksternal keluar IN_APPROVAL → APPROVED (dicatat SIGN)        approver terakhir (unggah scan)
//	RETURN   reviewer / approver    IN_REVIEW / IN_APPROVAL → RETURNED_TO_DRAFTER drafter
//
// APPROVED → SENT terjadi saat approver terakhir mengunggah scan bertanda tangan (eksternal.go).
// RETURNED_TO_REVIEWER hanya ada pada data lama; diperlakukan seperti IN_REVIEW.

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
	// Sign adalah APPROVE oleh approver terakhir: surat disetujui dan bernomor. Tombolnya tetap
	// APPROVE; Sign hanya muncul di Decision dan riwayat.
	Sign Action = "SIGN"
	// UploadSigned adalah unggahan scan surat eksternal bertanda tangan basah (APPROVED → SENT).
	UploadSigned Action = "UPLOAD_SIGNED"
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
	Status     string
	DrafterID  int64
	AssigneeID int64
	Reviewers  []int64 // urut sesuai step
	Approvers  []int64 // urut sesuai step
	External   bool    // surat eksternal keluar: approver terakhir menghasilkan APPROVED, bukan SENT
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
	ForwardedBy        Field  // reviewer terakhir yang meneruskan ke approver
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
		if len(s.Reviewers) == 0 || len(s.Approvers) == 0 {
			return Decision{}, ErrIncomplete
		}
		d.To = StatusInReview
		d.AssigneeID, d.AssigneeRole = s.Reviewers[0], RoleReviewer // selalu mulai dari reviewer 1
		d.ForwardedBy, d.ReturnedByReviewer, d.ReturnedByApprover = cleared(), cleared(), cleared()
		d.SnapshotSenderUnit = true

	case a == Approve && role == RoleReviewer:
		if i := slices.Index(s.Reviewers, actorID); i < len(s.Reviewers)-1 {
			d.To = StatusInReview
			d.AssigneeID, d.AssigneeRole = s.Reviewers[i+1], RoleReviewer
			break
		}
		if len(s.Approvers) == 0 {
			return Decision{}, ErrIncomplete
		}
		d.To = StatusInApproval
		d.AssigneeID, d.AssigneeRole = s.Approvers[0], RoleApprover
		d.ForwardedBy = setTo(actorID)

	case a == Approve: // approver
		if j := slices.Index(s.Approvers, actorID); j < len(s.Approvers)-1 {
			d.To = StatusInApproval
			d.AssigneeID, d.AssigneeRole = s.Approvers[j+1], RoleApprover
			break
		}
		d.Action = Sign
		if s.External {
			d.To = StatusApproved // menunggu tanda tangan basah; approver terakhir mengunggah scan
			d.AssigneeID, d.AssigneeRole = actorID, RoleApprover
		} else {
			d.To = StatusSent // tidak ada penerima tugas lagi; surat masuk inbox penerima
		}

	case a == Return:
		d.To = StatusReturnedToDrafter
		d.AssigneeID, d.AssigneeRole = s.DrafterID, RoleDrafter
		if role == RoleApprover {
			d.ReturnedByApprover = setTo(actorID)
		} else {
			d.ReturnedByReviewer = setTo(actorID)
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
	if CanUploadSigned(s, userID) {
		out = append(out, UploadSigned)
	}
	return out
}

// CanUploadSigned: userID adalah approver terakhir surat eksternal yang menunggu scan bertanda tangan.
func CanUploadSigned(s State, userID int64) bool {
	return userID != 0 && s.External && s.Status == StatusApproved && s.AssigneeID == userID &&
		len(s.Approvers) > 0 && s.Approvers[len(s.Approvers)-1] == userID
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
	isAssignedApprover := s.Status == StatusInApproval && s.AssigneeID == actorID && slices.Contains(s.Approvers, actorID)

	switch {
	case a == Submit && isDraftable && s.DrafterID == actorID:
		return RoleDrafter, true
	case (a == Approve || a == Return) && isAssignedReviewer:
		return RoleReviewer, true
	case (a == Approve || a == Return) && isAssignedApprover:
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
	case StatusInApproval, StatusApproved:
		return RoleApprover
	case StatusReturnedToDrafter:
		return RoleDrafter
	}
	return ""
}

// StepState menghitung keadaan tahap (role, order) untuk ditampilkan: DONE, ACTIVE, atau PENDING.
func StepState(s State, role string, order int) string {
	byOrder := func(list []int64) string {
		switch active := slices.Index(list, s.AssigneeID) + 1; {
		case order == active:
			return StepActive
		case order < active:
			return StepDone
		}
		return StepPending
	}
	switch s.Status {
	case StatusSent, StatusApproved:
		return StepDone
	case StatusInApproval:
		if role == RoleReviewer {
			return StepDone
		}
		return byOrder(s.Approvers)
	case StatusInReview, StatusReturnedToReviewer:
		if role == RoleApprover {
			return StepPending
		}
		return byOrder(s.Reviewers)
	}
	return StepPending // DRAFT dan RETURNED_TO_DRAFTER: proses diulang dari awal
}
