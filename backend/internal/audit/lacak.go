package audit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"office.backend/internal/auth"
	"office.backend/internal/platform/db"
)

// Lacak Surat: posisi surat saat ini, timeline kejadian, dan distribusi setelah terbit.
//
//	drafter dan peserta (reviewer/approver)   semua (Scope FULL)
//	penerima inbox (surat SENT)               posisi, kejadian terbit, dan cabang distribusi yang
//	                                          berawal dari dirinya (Scope RECIPIENT)
//
// Sumber data: letters (posisi), letter_actions (timeline), inbox_items + letter_forwards
// (distribusi). IP tidak pernah ditampilkan di sini; itu hanya untuk audit log.

// Cakupan pelacakan.
const (
	ScopeFull      = "FULL"
	ScopeRecipient = "RECIPIENT"
)

// Tracking mengambil posisi dan perjalanan satu surat.
func (s *Service) Tracking(ctx context.Context, user *auth.CurrentUser, letterID string) (*Tracking, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	c, err := s.repo.findLetter(ctx, user.Company.Code, id)
	if errors.Is(err, db.ErrNotFound) {
		return nil, ErrLetterNotFound
	}
	if err != nil {
		return nil, err
	}
	participant, recipient, err := s.repo.involvement(ctx, c.ID, user.ID)
	if err != nil {
		return nil, err
	}
	scope := ScopeFull
	if user.ID != c.Drafter.ID && !participant {
		if !recipient || c.Status != statusSent {
			return nil, ErrLetterNotFound // keberadaan surat tidak dibocorkan
		}
		scope = ScopeRecipient
	}

	acts, err := s.repo.actions(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	t := &Tracking{
		LetterID: c.UUID, LetterNumber: c.Number, Title: c.Title, Status: c.Status, Scope: scope,
		Position:     position(c, s.now()),
		Timeline:     timeline(c, acts, scope),
		Distribution: []DistributionNode{},
	}
	if c.Status == statusSent {
		rows, err := s.repo.distribution(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		branchOf := int64(0)
		if scope == ScopeRecipient {
			branchOf = user.ID
		}
		t.Distribution = buildDistribution(rows, branchOf)
	}
	s.localizeTracking(t)
	return t, nil
}

// position menghitung tempat surat saat ini dan lama surat di sana.
func position(c *letterCore, now time.Time) Position {
	p := Position{Since: c.Since, Holder: c.Holder}
	role := func(r string) *string { return &r }
	switch c.Status {
	case statusDraft:
		p.Label = "Masih berupa draf di pembuat surat"
		p.Holder, p.Role = &c.Drafter, role("DRAFTER")
	case statusReturnedToDrafter:
		p.Label = "Dikembalikan ke pembuat surat untuk diperbaiki"
		p.Role = role("DRAFTER")
	case statusInReview, statusReturnedToReviewer:
		p.Role, p.Step = role("REVIEWER"), c.HolderStep
		p.Label = "Sedang diperiksa reviewer"
		if c.HolderStep != nil {
			p.Label = fmt.Sprintf("Sedang diperiksa Reviewer %d", *c.HolderStep)
		}
		if c.Status == statusReturnedToReviewer {
			p.Label += " (dikembalikan approver)"
		}
	case statusInApproval:
		p.Label = "Menunggu persetujuan approver"
		p.Role = role("APPROVER")
	case statusSent:
		p.Label = "Sudah terbit dan dikirim ke penerima"
		p.Holder = nil
		if c.ApprovedAt != nil {
			p.Since = *c.ApprovedAt
		}
	case statusCancelled:
		p.Label = "Dibatalkan"
		p.Holder = nil
	default:
		p.Label = c.Status
	}
	if p.Holder != nil {
		h := waitingHours(p.Since, now)
		p.WaitingHours = &h
		p.Stuck = c.Status != statusDraft && h >= StuckAfterHours
	}
	return p
}

// timeline menyusun kejadian surat menjadi kalimat siap tampil. Penerima inbox hanya melihat
// kejadian surat terbit; catatan pengembalian dan proses review bersifat internal.
func timeline(c *letterCore, acts []letterAction, scope string) []TimelineEvent {
	out := []TimelineEvent{}
	if scope == ScopeFull {
		drafter := "DRAFTER"
		out = append(out, TimelineEvent{At: c.CreatedAt, Action: ActionCreate, Label: "Draf dibuat",
			Actor: c.Drafter, Role: &drafter})
	}
	for _, a := range acts {
		if scope == ScopeRecipient && a.Action != "SIGN" {
			continue
		}
		role := a.ActorRole
		out = append(out, TimelineEvent{At: a.At, Action: a.Action, Label: actionLabel(a),
			Actor: a.Actor, Role: &role, Target: a.Target, Note: a.Note})
	}
	return out
}

// actionLabel adalah kalimat untuk satu aksi alur surat.
func actionLabel(a letterAction) string {
	switch a.Action {
	case "SUBMIT":
		return "Dikirim untuk diperiksa"
	case "APPROVE":
		if a.ToStatus == statusInApproval {
			return "Disetujui reviewer, diteruskan ke approver"
		}
		return "Disetujui reviewer, diteruskan ke reviewer berikutnya"
	case "RETURN":
		if a.ActorRole == "APPROVER" {
			return "Dikembalikan approver ke reviewer"
		}
		return "Dikembalikan ke pembuat surat"
	case "SIGN":
		return "Disetujui approver, surat terbit"
	case "READ":
		return "Membuka surat"
	case "SAVE":
		if a.VersionNumber != nil {
			return fmt.Sprintf("Menyimpan naskah versi %d", *a.VersionNumber)
		}
		return "Menyimpan naskah"
	}
	return a.Action
}

// buildDistribution menyusun item inbox menjadi pohon: penerima langsung (TO/CC) di akar, lalu
// disposisi/pengalihan darinya bertingkat. branchOf 0 = seluruh pohon; selain itu hanya cabang
// yang berawal dari item milik user tersebut.
func buildDistribution(rows []inboxRow, branchOf int64) []DistributionNode {
	byID := make(map[int64]inboxRow, len(rows))
	children := map[int64][]int64{}
	for _, r := range rows {
		byID[r.Node.InboxItemID] = r
		children[r.ParentID] = append(children[r.ParentID], r.Node.InboxItemID)
	}

	var build func(id int64) DistributionNode
	build = func(id int64) DistributionNode {
		n := byID[id].Node
		n.Forwards = []DistributionNode{}
		for _, child := range children[id] {
			n.Forwards = append(n.Forwards, build(child))
		}
		return n
	}

	out := []DistributionNode{}
	if branchOf == 0 {
		for _, id := range children[0] {
			out = append(out, build(id))
		}
		return out
	}
	// Item milik user yang tidak berada di bawah item miliknya yang lain menjadi akar cabang
	for _, r := range rows {
		if r.OwnerID != branchOf || hasAncestorOwnedBy(byID, r.ParentID, branchOf) {
			continue
		}
		out = append(out, build(r.Node.InboxItemID))
	}
	return out
}

func hasAncestorOwnedBy(byID map[int64]inboxRow, parentID, ownerID int64) bool {
	for parentID != 0 {
		p, ok := byID[parentID]
		if !ok {
			return false
		}
		if p.OwnerID == ownerID {
			return true
		}
		parentID = p.ParentID
	}
	return false
}

// localizeTracking mengubah semua waktu ke zona waktu tampilan.
func (s *Service) localizeTracking(t *Tracking) {
	t.Position.Since = t.Position.Since.In(s.loc)
	for i := range t.Timeline {
		t.Timeline[i].At = t.Timeline[i].At.In(s.loc)
	}
	var walk func(nodes []DistributionNode)
	walk = func(nodes []DistributionNode) {
		for i := range nodes {
			nodes[i].ReceivedAt = nodes[i].ReceivedAt.In(s.loc)
			if nodes[i].ReadAt != nil {
				v := nodes[i].ReadAt.In(s.loc)
				nodes[i].ReadAt = &v
			}
			walk(nodes[i].Forwards)
		}
	}
	walk(t.Distribution)
}
