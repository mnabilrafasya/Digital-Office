package surat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
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
func (s *Service) Submit(ctx context.Context, user *auth.CurrentUser, in TransitionInput) (*LetterDetail, error) {
	in.Note = nil
	return s.transition(ctx, user, Submit, in)
}

// Approve memaraf surat (reviewer: diteruskan ke reviewer berikutnya atau ke approver), atau
// menyetujuinya (approver: surat terbit, lihat terbit.go).
func (s *Service) Approve(ctx context.Context, user *auth.CurrentUser, in TransitionInput) (*LetterDetail, error) {
	return s.transition(ctx, user, Approve, in)
}

// Return mengembalikan surat (reviewer ke drafter, approver ke reviewer) dengan catatan wajib.
func (s *Service) Return(ctx context.Context, user *auth.CurrentUser, in TransitionInput) (*LetterDetail, error) {
	return s.transition(ctx, user, Return, in)
}

// transition menjalankan satu aksi alur dalam satu transaksi (diulang otomatis saat deadlock):
// kunci baris surat → Decide → cek permission dan penerima tugas berikutnya → (terbitkan bila
// approver menyetujui) → ubah surat → catat letter_actions dan audit_logs. Klik dua kali aman:
// request kedua menunggu kunci, lalu ditolak karena status/giliran sudah berubah.
func (s *Service) transition(ctx context.Context, user *auth.CurrentUser, action Action, in TransitionInput) (*LetterDetail, error) {
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
		state := State{
			Status: l.Status, DrafterID: l.DrafterID, AssigneeID: l.AssigneeID,
			ForwardedByID: l.ForwardedByID, Reviewers: reviewers, ApproverID: approverID,
		}
		if !involves(l.DrafterID, approverID, reviewers, user.ID) {
			return ErrLetterNotFound
		}

		dec, err := Decide(state, action, user.ID)
		switch {
		case errors.Is(err, ErrInvalidState):
			return &apperr.InvalidStateError{Message: "Surat sudah diproses atau bukan giliran Anda. Muat ulang halaman untuk melihat status terbaru."}
		case errors.Is(err, ErrIncomplete):
			return apperr.Invalid("Surat belum lengkap: minimal 1 reviewer dan 1 approver wajib dipilih")
		case err != nil:
			return err
		}
		if perm := rolePermission(dec.ActorRole); perm != "" && !user.HasPermission(perm) {
			return apperr.ErrForbidden
		}
		if action == Submit {
			if err := s.checkSubmittable(ctx, letters, user.Company.Code, l.ID); err != nil {
				return err
			}
		}
		if err := s.checkNextAssignee(ctx, user.Company.Code, dec); err != nil {
			return err
		}

		var number *string
		if dec.Action == Sign {
			n, err := s.publish(ctx, letters, s.repos.Inbox(q), user.Company.Code, l.ID, user.ID)
			if err != nil {
				return err
			}
			number = &n
		}
		if err := letters.ApplyTransition(ctx, l.ID, user.ID, dec, note); err != nil {
			return err
		}
		return s.repos.Audit(q).Record(ctx, audit.Entry{
			CompanyCode: user.Company.Code,
			EntityType:  "LETTER",
			EntityID:    l.ID,
			EntityUUID:  id,
			Action:      string(dec.Action),
			ActorID:     user.ID,
			ActorName:   user.Name,
			IPAddress:   in.ClientIP,
			Snapshot:    transitionSnapshot{From: dec.From, To: dec.To, AssigneeID: dec.AssigneeID, LetterNumber: number},
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
func (s *Service) checkSubmittable(ctx context.Context, letters Store, companyCode string, letterID int64) error {
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
	return s.checkReachable(ctx, companyCode, d.Recipients)
}

// checkReachable memastikan setiap unit penerima punya pimpinan aktif dan setiap alias punya
// anggota. Diperiksa saat dikirim ke reviewer dan diulang saat surat terbit.
func (s *Service) checkReachable(ctx context.Context, companyCode string, recipients []RecipientRef) error {
	var unitIDs, aliasIDs []int64
	for _, r := range recipients {
		if r.UnitID != 0 {
			unitIDs = append(unitIDs, r.UnitID)
		} else {
			aliasIDs = append(aliasIDs, r.AliasLevelID)
		}
	}
	units, aliases, err := s.repos.Directory.UnreachableRecipients(ctx, companyCode, unitIDs, aliasIDs)
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
		return &apperr.InvalidStateError{Message: "Surat belum bisa dikirim karena penerima tidak bisa dijangkau (" + strings.Join(problems, "; ") + ")."}
	}
	return nil
}

// checkNextAssignee memastikan penerima tugas berikutnya masih aktif dan masih berwenang.
func (s *Service) checkNextAssignee(ctx context.Context, companyCode string, d Decision) error {
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
	case RoleReviewer:
		label = "Reviewer"
		found, err = s.repos.Directory.EligibleUserIDs(ctx, companyCode, auth.LetterReview, ids)
	case RoleApprover:
		label = "Approver"
		found, err = s.repos.Directory.EligibleUserIDs(ctx, companyCode, auth.LetterApprove, ids)
	default:
		label = "Pembuat surat"
		found, err = s.repos.Directory.ActiveUserIDs(ctx, companyCode, ids)
	}
	if err != nil {
		return err
	}
	if !found[d.AssigneeID] {
		return &apperr.InvalidStateError{Message: fmt.Sprintf(
			"%s berikutnya sudah tidak aktif atau tidak lagi berwenang, sehingga surat tidak bisa diteruskan. Hubungi admin.", label)}
	}
	return nil
}

// parseNote memvalidasi catatan: wajib untuk RETURN, opsional untuk aksi lain.
func parseNote(action Action, raw *string) (*string, error) {
	note, err := optionalText(raw, maxNoteLength, "Catatan")
	if err != nil {
		return nil, err
	}
	if action == Return && note == nil {
		return nil, apperr.Invalid("Catatan wajib diisi saat mengembalikan surat")
	}
	return note, nil
}
