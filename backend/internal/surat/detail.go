package surat

import (
	"context"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/masterdata"
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
	if d.Confidentiality != nil && d.Confidentiality.Code == ConfidentialitySecret {
		// Setiap pembukaan surat Rahasia dicatat; surat lain tidak, supaya audit log tidak terlalu ramai
		if err := s.audit.Record(ctx, letterEntry(user, d.InternalID, d.ID, audit.ActionView, nil)); err != nil {
			return nil, err
		}
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
// boleh: drafter, peserta, dan pemilik inbox surat yang sudah terbit atau surat masuk eksternal
// (langsung atau penerusan).
func (s *Service) detail(ctx context.Context, user *auth.CurrentUser, id string) (*LetterDetail, error) {
	d, err := s.reader.FindDetail(ctx, user.Company.Code, id)
	if err != nil {
		return nil, notFoundAs(err)
	}
	var assigneeID int64
	if d.CurrentAssignee != nil {
		assigneeID = d.CurrentAssignee.ID
	}
	del, err := s.assigneeDelegation(ctx, s.reader, user.Company.Code, d.Status, assigneeID)
	if err != nil {
		return nil, err
	}
	if !d.IsParticipant(user.ID) && (del == nil || del.Delegate.ID != user.ID) {
		visible, err := s.reader.HasActed(ctx, d.InternalID, user.ID) // pernah bertindak sebagai pengganti
		if err != nil {
			return nil, err
		}
		if !visible && (d.Status == StatusSent || d.Status == StatusReceived) {
			if visible, err = s.inbox.HasItem(ctx, d.InternalID, user.ID); err != nil {
				return nil, err
			}
		}
		if !visible {
			return nil, ErrLetterNotFound
		}
	}
	s.present(d, user, del)
	return d, nil
}

// present melengkapi detail untuk user yang melihat: keadaan tiap tahap, catatan pengembalian,
// delegasi penerima tugas (del), tombol yang boleh ditampilkan, dan zona waktu. Pengganti mendapat
// tombol penerima tugas yang diwakilinya; penerima tugas yang sedang mendelegasikan tidak mendapat
// tombol alur.
func (s *Service) present(d *LetterDetail, user *auth.CurrentUser, del *Delegation) {
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
	actor := user.ID
	if del != nil && d.CurrentAssignee != nil {
		d.DelegatedTo = &DelegationRef{Delegate: del.Delegate, Until: del.Until}
		switch user.ID {
		case del.Delegate.ID:
			actor = d.CurrentAssignee.ID
			owner := *d.CurrentAssignee
			d.OnBehalfOf = &owner
		case d.CurrentAssignee.ID:
			actor = 0 // sedang mendelegasikan: dikerjakan pengganti
		}
	}
	for _, a := range AllowedActions(state, actor) {
		// Tombol disembunyikan bila permission peran yang dibutuhkan sudah dicabut (pengganti memakai
		// kewenangan orang yang diwakilinya; diperiksa saat aksi dijalankan)
		perm := rolePermission(actorRoleFor(a, d.Status))
		if perm == "" || d.OnBehalfOf != nil || user.HasPermission(perm) {
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
	for i := range d.Attachments {
		d.Attachments[i].UploadedAt = d.Attachments[i].UploadedAt.In(s.loc)
	}
}

// detailState menyusun state workflow dari detail surat.
func detailState(d *LetterDetail) State {
	s := State{Status: d.Status, DrafterID: d.Drafter.ID, External: d.Direction == masterdata.DirectionOutgoing}
	if d.CurrentAssignee != nil {
		s.AssigneeID = d.CurrentAssignee.ID
	}
	for _, step := range d.Steps { // repository mengurutkan reviewer lalu approver, sesuai step
		if step.Role == RoleApprover {
			s.Approvers = append(s.Approvers, step.User.ID)
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
