package surat

import (
	"context"

	"office.backend/internal/auth"
	"office.backend/internal/platform/db"
)

// Get mengambil detail surat untuk drafter, peserta, dan penerima inbox (selain itu
// ErrLetterNotFound; status baca inbox diubah lewat GET /inbox/{itemId}). Kalau yang
// membuka adalah penerima tugas saat ini, aksi READ dicatat (sekali sejak transisi terakhir).
func (s *Service) Get(ctx context.Context, user *auth.CurrentUser, letterID string) (*LetterDetail, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	d, err := s.detail(ctx, user, id)
	if err != nil {
		return nil, err
	}
	if role := AssigneeRole(d.Status); role != "" && d.CurrentAssignee != nil && d.CurrentAssignee.ID == user.ID {
		err := s.tx.InTx(ctx, func(q db.DBTX) error {
			return s.repos.Letters(q).RecordRead(ctx, d.InternalID, user.ID, role)
		})
		if err != nil {
			return nil, err
		}
	}
	return d, nil
}

// detail memuat detail surat (uuid sudah baku) dan memeriksa akses, tanpa mencatat READ. Yang
// boleh: drafter, peserta, dan penerima inbox surat yang sudah terbit (langsung atau penerusan).
func (s *Service) detail(ctx context.Context, user *auth.CurrentUser, id string) (*LetterDetail, error) {
	d, err := s.reader.FindDetail(ctx, user.Company.Code, id)
	if err != nil {
		return nil, notFoundAs(err)
	}
	if !d.IsParticipant(user.ID) {
		if d.Status != StatusSent {
			return nil, ErrLetterNotFound
		}
		received, err := s.inbox.HasItem(ctx, d.InternalID, user.ID)
		if err != nil {
			return nil, err
		}
		if !received {
			return nil, ErrLetterNotFound
		}
	}
	s.present(d, user)
	return d, nil
}

// present melengkapi detail untuk user yang melihat: keadaan tiap tahap, catatan pengembalian,
// tombol yang boleh ditampilkan, dan zona waktu.
func (s *Service) present(d *LetterDetail, user *auth.CurrentUser) {
	state := detailState(d)
	for i := range d.Steps {
		d.Steps[i].State = StepState(state, d.Steps[i].Role, d.Steps[i].Order)
	}
	if d.Status != StatusReturnedToDrafter && d.Status != StatusReturnedToReviewer {
		d.LastReturnNote = nil
	}

	d.AllowedActions = []string{}
	if d.Drafter.ID == user.ID && isEditable(d.Status) {
		d.AllowedActions = append(d.AllowedActions, ActionEdit)
	}
	for _, a := range AllowedActions(state, user.ID) {
		// Tombol disembunyikan bila permission peran yang dibutuhkan sudah dicabut
		if perm := rolePermission(actorRoleFor(a, d.Status)); perm == "" || user.HasPermission(perm) {
			d.AllowedActions = append(d.AllowedActions, string(a))
		}
	}

	d.CreatedAt = d.CreatedAt.In(s.loc)
	d.UpdatedAt = d.UpdatedAt.In(s.loc)
	if d.LetterDate != nil {
		t := d.LetterDate.In(s.loc)
		d.LetterDate = &t
	}
	if d.Approval != nil {
		d.Approval.ApprovedAt = d.Approval.ApprovedAt.In(s.loc)
	}
}

// detailState menyusun state workflow dari detail surat.
func detailState(d *LetterDetail) State {
	s := State{Status: d.Status, DrafterID: d.Drafter.ID, ForwardedByID: d.ForwardedByID}
	if d.CurrentAssignee != nil {
		s.AssigneeID = d.CurrentAssignee.ID
	}
	for _, step := range d.Steps { // repository mengurutkan reviewer sesuai step
		if step.Role == RoleApprover {
			s.ApproverID = step.User.ID
		} else {
			s.Reviewers = append(s.Reviewers, step.User.ID)
		}
	}
	return s
}

// actorRoleFor adalah peran pelaku aksi a pada status tertentu.
func actorRoleFor(a Action, status string) string {
	if a == Submit {
		return RoleDrafter
	}
	return AssigneeRole(status)
}
