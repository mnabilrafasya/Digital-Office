package surat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Batas isi surat (mengikuti form front-end).
const (
	maxTitleLength       = 250
	maxSpecialNoteLength = 500
	maxSummaryLength     = 1000
	maxReviewers         = 10
	maxRecipients        = 50
)

// LetterInput adalah isi form surat apa adanya dari request POST/PUT. Semua field opsional;
// field yang tidak dikirim berarti kosong (PUT mengganti seluruh isi draf).
type LetterInput struct {
	LetterTypeID      *string  `json:"letterTypeId"`
	TemplateID        *string  `json:"templateId"`
	ClassificationID  *string  `json:"classificationId"`
	ConfidentialityID *string  `json:"confidentialityId"`
	UrgencyID         *string  `json:"urgencyId"`
	Title             *string  `json:"title"`
	Summary           *string  `json:"summary"`
	SpecialNote       *string  `json:"specialNote"`
	ReviewerIDs       []string `json:"reviewerIds"`
	ApproverID        *string  `json:"approverId"`
	Recipients        struct {
		To []RecipientInput `json:"to"`
		CC []RecipientInput `json:"cc"`
	} `json:"recipients"`
}

// RecipientInput adalah satu penerima: isi tepat satu dari UnitID atau AliasLevelID.
type RecipientInput struct {
	UnitID       *string `json:"unitId"`
	AliasLevelID *string `json:"aliasLevelId"`
}

// parseDraft memeriksa format input tanpa database: ID, panjang teks, duplikat, dan aturan
// peran (approver bukan drafter dan bukan reviewer). Teks di-trim; kosong menjadi nil.
func parseDraft(in LetterInput, drafterID int64) (*LetterDraft, error) {
	var (
		d   LetterDraft
		err error
	)
	ids := []struct {
		dst   **int64
		src   *string
		label string
	}{
		{&d.LetterTypeID, in.LetterTypeID, "Jenis surat"},
		{&d.TemplateID, in.TemplateID, "Template"},
		{&d.ClassificationID, in.ClassificationID, "Klasifikasi"},
		{&d.ConfidentialityID, in.ConfidentialityID, "Sifat surat"},
		{&d.UrgencyID, in.UrgencyID, "Kecepatan"},
		{&d.ApproverID, in.ApproverID, "Approver"},
	}
	for _, f := range ids {
		if *f.dst, err = optionalID(f.src, f.label); err != nil {
			return nil, err
		}
	}
	texts := []struct {
		dst   **string
		src   *string
		max   int
		label string
	}{
		{&d.Title, in.Title, maxTitleLength, "Perihal"},
		{&d.Summary, in.Summary, maxSummaryLength, "Ringkasan"},
		{&d.SpecialNote, in.SpecialNote, maxSpecialNoteLength, "Catatan khusus"},
	}
	for _, f := range texts {
		if *f.dst, err = optionalText(f.src, f.max, f.label); err != nil {
			return nil, err
		}
	}

	if len(in.ReviewerIDs) > maxReviewers {
		return nil, apperr.Invalid(fmt.Sprintf("Reviewer maksimal %d orang", maxReviewers))
	}
	for _, raw := range in.ReviewerIDs {
		id, err := parseID(raw, "Reviewer")
		if err != nil {
			return nil, err
		}
		if slices.Contains(d.ReviewerIDs, id) {
			return nil, apperr.Invalid("Reviewer yang sama tidak boleh dipilih lebih dari sekali")
		}
		d.ReviewerIDs = append(d.ReviewerIDs, id)
	}
	if d.ApproverID != nil {
		switch {
		case *d.ApproverID == drafterID:
			return nil, apperr.Invalid("Approver tidak boleh pembuat surat")
		case slices.Contains(d.ReviewerIDs, *d.ApproverID):
			return nil, apperr.Invalid("Approver tidak boleh sekaligus menjadi reviewer")
		}
	}

	if d.Recipients, err = parseRecipients(in.Recipients.To, in.Recipients.CC); err != nil {
		return nil, err
	}
	return &d, nil
}

// checkComplete memastikan surat siap diproses (dipakai saat submit dan saat menyimpan surat
// yang dikembalikan ke drafter). Semua kekurangan disebut sekaligus.
func checkComplete(d *LetterDraft) error {
	var missing []string
	add := func(empty bool, label string) {
		if empty {
			missing = append(missing, label)
		}
	}
	add(d.LetterTypeID == nil, "jenis surat")
	add(d.ClassificationID == nil, "klasifikasi")
	add(d.ConfidentialityID == nil, "sifat surat")
	add(d.UrgencyID == nil, "kecepatan")
	add(d.Title == nil, "perihal")
	add(len(d.ReviewerIDs) == 0, "minimal 1 reviewer")
	add(d.ApproverID == nil, "approver")
	add(!slices.ContainsFunc(d.Recipients, func(r RecipientRef) bool { return r.Type == RecipientTo }),
		"minimal 1 penerima Kepada")
	if len(missing) > 0 {
		return apperr.Invalid("Surat belum lengkap: " + strings.Join(missing, ", "))
	}
	return nil
}

// optionalID membaca ID opsional: nil atau "" berarti kosong.
func optionalID(raw *string, label string) (*int64, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	id, err := parseID(*raw, label)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// parseID membaca ID wajib berupa angka positif dalam bentuk string.
func parseID(raw, label string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id < 1 {
		return 0, apperr.Invalid(label + " tidak valid")
	}
	return id, nil
}

// optionalText merapikan teks opsional: di-trim, kosong menjadi nil. Panjang dihitung dalam satuan
// UTF-16 seperti NVARCHAR SQL Server, jadi emoji dihitung 2 dan tidak bisa melewati batas kolom.
func optionalText(raw *string, maxLen int, label string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	s := strings.TrimSpace(*raw)
	if s == "" {
		return nil, nil
	}
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	if n > maxLen {
		return nil, apperr.Invalid(fmt.Sprintf("%s maksimal %d karakter", label, maxLen))
	}
	return &s, nil
}

// Create membuat draf surat baru. Unit pengirim dan perusahaan diambil dari user yang login.
func (s *Service) Create(ctx context.Context, user *auth.CurrentUser, in LetterInput) (*LetterDetail, error) {
	d, err := parseDraft(in, user.ID)
	if err != nil {
		return nil, err
	}
	if err := s.checkRefs(ctx, user.Company.Code, d); err != nil {
		return nil, err
	}

	var id string
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		var err error
		id, err = s.repos.Letters(q).Create(ctx, NewLetter{
			CompanyCode:  user.Company.Code,
			SenderUnitID: user.Unit.ID,
			DrafterID:    user.ID,
			Draft:        d,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, user, id)
}

// Update mengganti seluruh isi draf. ifMatch wajib sama dengan versi terbaru (ETag), supaya dua
// tab yang mengedit surat yang sama tidak saling menimpa.
func (s *Service) Update(ctx context.Context, user *auth.CurrentUser, letterID string, ifMatch []byte, in LetterInput) (*LetterDetail, error) {
	id, ok := parseLetterID(letterID)
	if !ok {
		return nil, ErrLetterNotFound
	}
	d, err := parseDraft(in, user.ID)
	if err != nil {
		return nil, err
	}

	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		letters := s.repos.Letters(q)
		l, err := letters.LockByUUID(ctx, user.Company.Code, id)
		if err != nil {
			return notFoundAs(err)
		}
		if l.DrafterID != user.ID {
			isParticipant, err := letters.IsParticipant(ctx, l.ID, user.ID)
			if err != nil {
				return err
			}
			if isParticipant {
				return apperr.ErrForbidden
			}
			return ErrLetterNotFound
		}
		if !isEditable(l.Status) {
			return &apperr.InvalidStateError{Message: "Surat sedang diproses dan tidak bisa diubah. Muat ulang halaman untuk melihat status terbaru."}
		}
		if !bytes.Equal(ifMatch, l.RowVersion) {
			return &apperr.VersionConflictError{RowVersion: l.RowVersion}
		}
		// Referensi diperiksa setelah akses, supaya user tanpa akses selalu mendapat 404
		if err := s.checkRefs(ctx, user.Company.Code, d); err != nil {
			return err
		}
		// Surat yang dikembalikan ke drafter pernah lengkap; CHECK ck_letters_complete_after_draft
		// juga mewajibkannya, jadi aturannya sama dengan submit.
		if l.Status == StatusReturnedToDrafter {
			if err := checkComplete(d); err != nil {
				return err
			}
		}
		return letters.UpdateDraft(ctx, l.ID, d)
	})
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, user, id)
}

// checkRefs memastikan semua data yang dirujuk draf ada, aktif, dan milik perusahaan user:
// lookup sesuai kategori, template (jenisnya cocok), unit penerima, alias, reviewer, dan approver.
// Berlaku sejak draf, walaupun surat belum lengkap.
func (s *Service) checkRefs(ctx context.Context, companyCode string, d *LetterDraft) error {
	if err := s.checkLookups(ctx, companyCode, d); err != nil {
		return err
	}
	if err := s.checkTemplate(ctx, companyCode, d); err != nil {
		return err
	}
	if err := s.checkRecipientUnits(ctx, companyCode, d); err != nil {
		return err
	}
	return s.checkParticipants(ctx, companyCode, d)
}

// checkLookups memeriksa semua lookup draf (termasuk alias penerima) dengan satu query.
func (s *Service) checkLookups(ctx context.Context, companyCode string, d *LetterDraft) error {
	type want struct {
		id       int64
		category string
		message  string
	}
	var wants []want
	add := func(id *int64, category, message string) {
		if id != nil {
			wants = append(wants, want{*id, category, message})
		}
	}
	add(d.LetterTypeID, "LETTER_TYPE", "Jenis surat tidak ditemukan atau tidak aktif")
	add(d.ClassificationID, "CLASSIFICATION", "Klasifikasi tidak ditemukan atau tidak aktif")
	add(d.ConfidentialityID, "CONFIDENTIALITY", "Sifat surat tidak ditemukan atau tidak aktif")
	add(d.UrgencyID, "URGENCY", "Kecepatan tidak ditemukan atau tidak aktif")
	for _, r := range d.Recipients {
		if r.AliasLevelID != 0 {
			add(&r.AliasLevelID, "POSITION_LEVEL", "Alias penerima tidak ditemukan atau tidak aktif")
		}
	}
	if len(wants) == 0 {
		return nil
	}

	ids := make([]int64, len(wants))
	for i, w := range wants {
		ids[i] = w.id
	}
	categories, err := s.repos.Lookups.LookupCategories(ctx, companyCode, ids)
	if err != nil {
		return err
	}
	for _, w := range wants {
		if categories[w.id] != w.category {
			return apperr.Invalid(w.message)
		}
	}
	return nil
}

// checkTemplate memastikan template aktif milik perusahaan, dan kalau template dibuat untuk jenis
// surat tertentu, jenisnya sama dengan jenis surat draf.
func (s *Service) checkTemplate(ctx context.Context, companyCode string, d *LetterDraft) error {
	if d.TemplateID == nil {
		return nil
	}
	typeID, err := s.repos.Templates.TemplateLetterType(ctx, companyCode, *d.TemplateID)
	if errors.Is(err, db.ErrNotFound) {
		return apperr.Invalid("Template tidak ditemukan atau tidak aktif")
	}
	if err != nil {
		return err
	}
	if typeID != nil && d.LetterTypeID != nil && *typeID != *d.LetterTypeID {
		return apperr.Invalid("Template tidak sesuai dengan jenis surat yang dipilih")
	}
	return nil
}
