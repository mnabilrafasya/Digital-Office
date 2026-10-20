package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
)

// Panjang maksimal isian surat (mengikuti form front-end).
const (
	maxTitleLength       = 250
	maxSpecialNoteLength = 500
	maxSummaryLength     = 1000
)

// LetterInput adalah isi draf dari klien. Semua field opsional selama draf.
type LetterInput struct {
	LetterTypeID      *int64
	TemplateID        *int64
	ClassificationID  *int64
	ConfidentialityID *int64
	UrgencyID         *int64
	Title             *string
	Summary           *string
	SpecialNote       *string
	ReviewerIDs       []int64 // urutan = urutan tahap
	ApproverID        *int64
	To                []RecipientInput
	CC                []RecipientInput
}

// RecipientInput adalah satu penerima: tepat satu dari UnitID atau AliasLevelID.
type RecipientInput struct {
	UnitID       *int64
	AliasLevelID *int64
}

// draft adalah isi surat yang sudah dirapikan dan lolos pemeriksaan bentuk.
type draft struct {
	content    model.LetterContent
	reviewers  []int64
	approverID int64 // 0 = belum dipilih
	to, cc     []RecipientInput
}

// normalizeInput merapikan teks (trim, kosong = null) dan memeriksa bentuk input tanpa database.
func normalizeInput(in LetterInput) (draft, error) {
	var p problems
	d := draft{
		content: model.LetterContent{
			TemplateID: in.TemplateID, LetterTypeID: in.LetterTypeID, ClassificationID: in.ClassificationID,
			ConfidentialityID: in.ConfidentialityID, UrgencyID: in.UrgencyID,
			Title:       cleanText(in.Title),
			Summary:     cleanText(in.Summary),
			SpecialNote: cleanText(in.SpecialNote),
		},
		reviewers: in.ReviewerIDs,
		to:        in.To,
		cc:        in.CC,
	}
	checkLength(&p, d.content.Title, "Perihal", maxTitleLength)
	checkLength(&p, d.content.SpecialNote, "Keterangan khusus", maxSpecialNoteLength)
	checkLength(&p, d.content.Summary, "Isi ringkas", maxSummaryLength)

	seen := map[int64]bool{}
	for _, id := range d.reviewers {
		if seen[id] {
			p.add("Reviewer yang sama tidak boleh dipilih lebih dari sekali")
			break
		}
		seen[id] = true
	}
	if in.ApproverID != nil {
		d.approverID = *in.ApproverID
	}

	units, aliases := map[int64]bool{}, map[int64]bool{}
	for _, list := range [][]RecipientInput{d.to, d.cc} {
		for _, r := range list {
			switch {
			case (r.UnitID == nil) == (r.AliasLevelID == nil):
				p.add("Setiap penerima wajib berisi tepat satu dari unitId atau aliasLevelId")
			case r.UnitID != nil && units[*r.UnitID]:
				p.add("Unit penerima yang sama tidak boleh muncul lebih dari sekali di Kepada dan Tembusan")
			case r.AliasLevelID != nil && aliases[*r.AliasLevelID]:
				p.add("Alias penerima yang sama tidak boleh muncul lebih dari sekali")
			case r.UnitID != nil:
				units[*r.UnitID] = true
			default:
				aliases[*r.AliasLevelID] = true
			}
		}
	}
	return d, p.err()
}

func cleanText(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func checkLength(p *problems, s *string, label string, limit int) {
	if s != nil && utf8.RuneCountInString(*s) > limit {
		p.add("%s maksimal %d karakter", label, limit)
	}
}

// draftFromStored menyusun draft dari data surat yang tersimpan (untuk validasi saat submit).
func draftFromStored(l *model.Letter, parts []model.Participant, recips []model.Recipient) draft {
	d := draft{content: l.Content}
	d.reviewers, d.approverID = splitParticipants(parts)
	for _, r := range recips {
		ri := RecipientInput{UnitID: r.UnitID, AliasLevelID: r.AliasLevelID}
		if r.Type == model.RecipientTo {
			d.to = append(d.to, ri)
		} else {
			d.cc = append(d.cc, ri)
		}
	}
	return d
}

// splitParticipants memisahkan reviewer (urut tahap) dan approver.
func splitParticipants(parts []model.Participant) (reviewers []int64, approverID int64) {
	sorted := slices.Clone(parts)
	slices.SortFunc(sorted, func(a, b model.Participant) int { return a.StepOrder - b.StepOrder })
	for _, p := range sorted {
		if p.Role == model.RoleReviewer {
			reviewers = append(reviewers, p.UserID)
		} else if p.Role == model.RoleApprover {
			approverID = p.UserID
		}
	}
	return reviewers, approverID
}

// participants adalah daftar peserta yang diinginkan: reviewer tahap 1..n dan approver tahap 1.
func (d draft) participants() []model.NewParticipant {
	out := make([]model.NewParticipant, 0, len(d.reviewers)+1)
	for i, id := range d.reviewers {
		out = append(out, model.NewParticipant{UserID: id, Role: model.RoleReviewer, StepOrder: i + 1})
	}
	if d.approverID != 0 {
		out = append(out, model.NewParticipant{UserID: d.approverID, Role: model.RoleApprover, StepOrder: 1})
	}
	return out
}

// recipients adalah daftar penerima yang diinginkan; sort_order = urutan di daftarnya.
func (d draft) recipients() []model.NewRecipient {
	var out []model.NewRecipient
	add := func(typ string, list []RecipientInput) {
		for i, r := range list {
			out = append(out, model.NewRecipient{Type: typ, UnitID: r.UnitID, AliasLevelID: r.AliasLevelID, SortOrder: i})
		}
	}
	add(model.RecipientTo, d.to)
	add(model.RecipientCC, d.cc)
	return out
}

// diffParticipants membandingkan peserta tersimpan dengan yang diinginkan. Baris yang sama
// dibiarkan; sisanya dihapus (soft delete) lalu disisipkan ulang. Karena penghapusan dilakukan
// lebih dulu, perubahan urutan reviewer aman terhadap unique index (letter_id, role, step_order).
func diffParticipants(existing []model.Participant, desired []model.NewParticipant) (deleteIDs []int64, insert []model.NewParticipant) {
	key := func(userID int64, role string, step int) string { return fmt.Sprintf("%s|%d|%d", role, userID, step) }
	want := map[string]bool{}
	for _, d := range desired {
		want[key(d.UserID, d.Role, d.StepOrder)] = true
	}
	kept := map[string]bool{}
	for _, e := range existing {
		k := key(e.UserID, e.Role, e.StepOrder)
		if want[k] && !kept[k] {
			kept[k] = true
			continue
		}
		deleteIDs = append(deleteIDs, e.ID)
	}
	for _, d := range desired {
		if !kept[key(d.UserID, d.Role, d.StepOrder)] {
			insert = append(insert, d)
		}
	}
	return deleteIDs, insert
}

// diffRecipients sama dengan diffParticipants untuk penerima (kunci: tipe, unit/alias, urutan).
func diffRecipients(existing []model.Recipient, desired []model.NewRecipient) (deleteIDs []int64, insert []model.NewRecipient) {
	key := func(typ string, unit, alias *int64, sort int) string {
		return fmt.Sprintf("%s|%d|%d|%d", typ, deref(unit), deref(alias), sort)
	}
	want := map[string]bool{}
	for _, d := range desired {
		want[key(d.Type, d.UnitID, d.AliasLevelID, d.SortOrder)] = true
	}
	kept := map[string]bool{}
	for _, e := range existing {
		k := key(e.Type, e.UnitID, e.AliasLevelID, e.SortOrder)
		if want[k] && !kept[k] {
			kept[k] = true
			continue
		}
		deleteIDs = append(deleteIDs, e.ID)
	}
	for _, d := range desired {
		if !kept[key(d.Type, d.UnitID, d.AliasLevelID, d.SortOrder)] {
			insert = append(insert, d)
		}
	}
	return deleteIDs, insert
}

func deref(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// validateDraft memeriksa referensi (selalu) dan kelengkapan (bila strict): lookup, template,
// unit, alias, reviewer, dan approver harus aktif dan milik perusahaan user.
func validateDraft(ctx context.Context, r LetterRepos, user *model.CurrentUser, d draft, drafterID int64, strict bool) error {
	var p problems
	company := user.Company.Code
	c := d.content

	if strict {
		for _, f := range []struct {
			v     any
			label string
		}{
			{c.LetterTypeID, "Jenis surat"}, {c.ClassificationID, "Klasifikasi arsip"},
			{c.ConfidentialityID, "Sifat surat"}, {c.UrgencyID, "Kecepatan"}, {c.Title, "Perihal"},
		} {
			if isNilPointer(f.v) {
				p.add("%s wajib diisi", f.label)
			}
		}
		if len(d.reviewers) == 0 {
			p.add("Minimal 1 reviewer wajib dipilih")
		}
		if d.approverID == 0 {
			p.add("Approver wajib dipilih")
		}
		if len(d.to) == 0 {
			p.add("Minimal 1 penerima Kepada wajib dipilih")
		}
	}

	if err := checkLookups(ctx, r, company, c, &p); err != nil {
		return err
	}
	if c.TemplateID != nil {
		t, err := r.Units.FindTemplate(ctx, company, *c.TemplateID)
		if err != nil && !isNotFound(err) {
			return err
		}
		if t == nil || !t.IsActive {
			p.add("Template surat tidak ditemukan atau tidak aktif")
		}
	}
	if err := checkRecipientRefs(ctx, r, company, d, &p); err != nil {
		return err
	}
	if err := checkParticipants(ctx, r, company, d, drafterID, &p); err != nil {
		return err
	}
	return p.err()
}

func isNilPointer(v any) bool {
	switch x := v.(type) {
	case *int64:
		return x == nil
	case *string:
		return x == nil
	}
	return v == nil
}

func checkLookups(ctx context.Context, r LetterRepos, company string, c model.LetterContent, p *problems) error {
	fields := []struct {
		id       *int64
		category string
		label    string
	}{
		{c.LetterTypeID, model.CategoryLetterType, "Jenis surat"},
		{c.ClassificationID, model.CategoryClassification, "Klasifikasi arsip"},
		{c.ConfidentialityID, model.CategoryConfidentiality, "Sifat surat"},
		{c.UrgencyID, model.CategoryUrgency, "Kecepatan"},
	}
	var ids []int64
	for _, f := range fields {
		if f.id != nil {
			ids = append(ids, *f.id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	refs, err := r.Lookups.FindRefs(ctx, company, ids)
	if err != nil {
		return err
	}
	for _, f := range fields {
		if f.id == nil {
			continue
		}
		ref, ok := refs[*f.id]
		if !ok || ref.Category != f.category || !ref.IsActive {
			p.add("%s tidak ditemukan atau tidak aktif", f.label)
		}
	}
	return nil
}

func checkRecipientRefs(ctx context.Context, r LetterRepos, company string, d draft, p *problems) error {
	var unitIDs, aliasIDs []int64
	for _, list := range [][]RecipientInput{d.to, d.cc} {
		for _, rc := range list {
			if rc.UnitID != nil {
				unitIDs = append(unitIDs, *rc.UnitID)
			}
			if rc.AliasLevelID != nil {
				aliasIDs = append(aliasIDs, *rc.AliasLevelID)
			}
		}
	}
	if len(unitIDs) > 0 {
		units, err := r.Units.FindRefs(ctx, company, unitIDs)
		if err != nil {
			return err
		}
		for _, id := range unitIDs {
			if u, ok := units[id]; !ok || !u.IsActive {
				p.add("Unit penerima (id %d) tidak ditemukan atau tidak aktif", id)
			}
		}
	}
	if len(aliasIDs) > 0 {
		aliases, err := r.Lookups.FindAliases(ctx, company, aliasIDs)
		if err != nil {
			return err
		}
		for _, id := range aliasIDs {
			if a, ok := aliases[id]; !ok || !a.IsActive {
				p.add("Alias penerima (id %d) tidak ditemukan atau tidak aktif", id)
			}
		}
	}
	return nil
}

func checkParticipants(ctx context.Context, r LetterRepos, company string, d draft, drafterID int64, p *problems) error {
	if d.approverID != 0 {
		if d.approverID == drafterID {
			p.add("Approver tidak boleh pembuat surat")
		}
		if slices.Contains(d.reviewers, d.approverID) {
			p.add("Approver tidak boleh sekaligus menjadi reviewer")
		}
	}
	ids := slices.Clone(d.reviewers)
	if d.approverID != 0 {
		ids = append(ids, d.approverID)
	}
	if len(ids) == 0 {
		return nil
	}
	users, err := r.Users.FindParticipantUsers(ctx, company, ids, permission.LetterReview, permission.LetterApprove)
	if err != nil {
		return err
	}
	for i, id := range d.reviewers {
		if u, ok := users[id]; !ok || !u.Eligible || !u.CanReview {
			p.add("Reviewer tahap %d tidak dapat dipilih (tidak aktif atau tidak berhak mereview)", i+1)
		}
	}
	if d.approverID != 0 {
		if u, ok := users[d.approverID]; !ok || !u.Eligible || !u.CanApprove {
			p.add("Approver tidak dapat dipilih (tidak aktif atau tidak berhak menyetujui)")
		}
	}
	return nil
}
