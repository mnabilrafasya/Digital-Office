package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/workflow"
)

// maxNoteLength adalah panjang maksimal catatan approve/return (karakter).
const maxNoteLength = 1000

// TransitionInput adalah permintaan aksi alur persetujuan.
type TransitionInput struct {
	LetterID string
	Note     *string // wajib untuk RETURN, opsional untuk APPROVE, diabaikan untuk SUBMIT
	ClientIP string  // dicatat di audit
}

// Submit mengirim draf (atau surat yang dikembalikan) ke reviewer pertama.
func (s *LetterService) Submit(ctx context.Context, user *model.CurrentUser, in TransitionInput) (*model.LetterDetail, error) {
	in.Note = nil
	return s.transition(ctx, user, workflow.Submit, in)
}

// Approve memaraf surat: diteruskan ke reviewer berikutnya atau ke approver.
func (s *LetterService) Approve(ctx context.Context, user *model.CurrentUser, in TransitionInput) (*model.LetterDetail, error) {
	return s.transition(ctx, user, workflow.Approve, in)
}

// Return mengembalikan surat (reviewer ke drafter, approver ke reviewer) dengan catatan wajib.
func (s *LetterService) Return(ctx context.Context, user *model.CurrentUser, in TransitionInput) (*model.LetterDetail, error) {
	return s.transition(ctx, user, workflow.Return, in)
}

// transition menjalankan satu aksi alur dalam satu transaksi (diulang otomatis saat deadlock):
// kunci baris surat → workflow.Decide → cek permission dan penerima tugas berikutnya → ubah surat
// → catat letter_actions dan audit_logs. Klik dua kali aman: request kedua menunggu kunci, lalu
// ditolak karena status/giliran sudah berubah.
func (s *LetterService) transition(ctx context.Context, user *model.CurrentUser, action workflow.Action, in TransitionInput) (*model.LetterDetail, error) {
	id, ok := parseLetterID(in.LetterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	note, err := parseNote(action, in.Note)
	if err != nil {
		return nil, err
	}

	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		letters := s.repos.Letters(q)
		l, err := letters.LockByUUID(ctx, user.Company.Code, id)
		if err != nil {
			return notFoundAs(err)
		}
		reviewers, approverID, err := letters.Participants(ctx, l.ID)
		if err != nil {
			return err
		}
		state := workflow.State{
			Status: l.Status, DrafterID: l.DrafterID, AssigneeID: l.AssigneeID,
			ForwardedByID: l.ForwardedByID, Reviewers: reviewers, ApproverID: approverID,
		}
		if user.ID != l.DrafterID && user.ID != approverID && !slices.Contains(reviewers, user.ID) {
			return ErrLetterNotFound
		}

		dec, err := workflow.Decide(state, action, user.ID)
		switch {
		case errors.Is(err, workflow.ErrInvalidState):
			return &InvalidStateError{Message: "Surat sudah diproses atau bukan giliran Anda. Muat ulang halaman untuk melihat status terbaru."}
		case errors.Is(err, workflow.ErrIncomplete):
			return invalid("Surat belum lengkap: minimal 1 reviewer dan 1 approver wajib dipilih")
		case err != nil:
			return err
		}
		if perm := rolePermission(dec.ActorRole); perm != "" && !user.HasPermission(perm) {
			return ErrForbidden
		}
		if action == workflow.Submit {
			if err := s.checkSubmittable(ctx, letters, user.Company.Code, l.ID); err != nil {
				return err
			}
		}
		if err := s.checkNextAssignee(ctx, user.Company.Code, dec); err != nil {
			return err
		}

		if err := letters.ApplyTransition(ctx, l.ID, user.ID, dec, note); err != nil {
			return err
		}
		return s.repos.Audit(q).Record(ctx, repository.AuditEntry{
			CompanyCode: user.Company.Code,
			EntityType:  "LETTER",
			EntityID:    l.ID,
			EntityUUID:  id,
			Action:      string(action),
			ActorID:     user.ID,
			ActorName:   user.Name,
			IPAddress:   in.ClientIP,
			Snapshot:    transitionSnapshot{From: dec.From, To: dec.To, AssigneeID: dec.AssigneeID},
		})
	})
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, user, id)
}

// transitionSnapshot adalah isi kolom audit_logs.snapshot untuk transisi surat.
type transitionSnapshot struct {
	From         string  `json:"from"`
	To           string  `json:"to"`
	AssigneeID   int64   `json:"assigneeId,omitempty"`
	LetterNumber *string `json:"letterNumber"`
}

// checkSubmittable memastikan surat siap dikirim: isi lengkap, semua rujukan masih valid
// (reviewer/approver bisa saja dinonaktifkan sejak draf disimpan), setiap unit penerima punya
// pimpinan aktif, dan setiap alias punya anggota.
func (s *LetterService) checkSubmittable(ctx context.Context, letters LetterStore, companyCode string, letterID int64) error {
	d, err := letters.LoadDraft(ctx, letterID)
	if err != nil {
		return err
	}
	if err := checkComplete(d); err != nil {
		return err
	}
	if err := s.checkRefs(ctx, companyCode, d); err != nil {
		return err
	}

	var unitIDs, aliasIDs []int64
	for _, r := range d.Recipients {
		if r.UnitID != 0 {
			unitIDs = append(unitIDs, r.UnitID)
		} else {
			aliasIDs = append(aliasIDs, r.AliasLevelID)
		}
	}
	units, aliases, err := s.repos.Refs.UnreachableRecipients(ctx, companyCode, unitIDs, aliasIDs)
	if err != nil {
		return err
	}
	var problems []string
	if len(units) > 0 {
		problems = append(problems, "unit tanpa pimpinan aktif: "+strings.Join(units, ", "))
	}
	if len(aliases) > 0 {
		problems = append(problems, "alias tanpa anggota: "+strings.Join(aliases, ", "))
	}
	if len(problems) > 0 {
		return &InvalidStateError{Message: "Surat belum bisa dikirim karena penerima tidak bisa dijangkau (" + strings.Join(problems, "; ") + ")."}
	}
	return nil
}

// checkNextAssignee memastikan penerima tugas berikutnya masih aktif dan masih berwenang.
func (s *LetterService) checkNextAssignee(ctx context.Context, companyCode string, d workflow.Decision) error {
	if d.AssigneeID == 0 {
		return nil
	}
	var (
		found map[int64]bool
		err   error
		label string
	)
	ids := []int64{d.AssigneeID}
	switch d.AssigneeRole {
	case model.RoleReviewer:
		label = "Reviewer"
		found, err = s.repos.Refs.EligibleUserIDs(ctx, companyCode, permission.LetterReview, ids)
	case model.RoleApprover:
		label = "Approver"
		found, err = s.repos.Refs.EligibleUserIDs(ctx, companyCode, permission.LetterApprove, ids)
	default:
		label = "Pembuat surat"
		found, err = s.repos.Refs.ActiveUserIDs(ctx, companyCode, ids)
	}
	if err != nil {
		return err
	}
	if !found[d.AssigneeID] {
		return &InvalidStateError{Message: fmt.Sprintf(
			"%s berikutnya sudah tidak aktif atau tidak lagi berwenang, sehingga surat tidak bisa diteruskan. Hubungi admin.", label)}
	}
	return nil
}

// parseNote memvalidasi catatan: wajib untuk RETURN, opsional untuk aksi lain.
func parseNote(action workflow.Action, raw *string) (*string, error) {
	note, err := optionalText(raw, maxNoteLength, "Catatan")
	if err != nil {
		return nil, err
	}
	if action == workflow.Return && note == nil {
		return nil, invalid("Catatan wajib diisi saat mengembalikan surat")
	}
	return note, nil
}
