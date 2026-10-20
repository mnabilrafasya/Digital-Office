package service

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
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
func parseDraft(in LetterInput, drafterID int64) (*model.LetterDraft, error) {
	var (
		d   model.LetterDraft
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
		return nil, invalid(fmt.Sprintf("Reviewer maksimal %d orang", maxReviewers))
	}
	for _, raw := range in.ReviewerIDs {
		id, err := parseID(raw, "Reviewer")
		if err != nil {
			return nil, err
		}
		if slices.Contains(d.ReviewerIDs, id) {
			return nil, invalid("Reviewer yang sama tidak boleh dipilih lebih dari sekali")
		}
		d.ReviewerIDs = append(d.ReviewerIDs, id)
	}
	if d.ApproverID != nil {
		switch {
		case *d.ApproverID == drafterID:
			return nil, invalid("Approver tidak boleh pembuat surat")
		case slices.Contains(d.ReviewerIDs, *d.ApproverID):
			return nil, invalid("Approver tidak boleh sekaligus menjadi reviewer")
		}
	}

	if d.Recipients, err = parseRecipients(in.Recipients.To, in.Recipients.CC); err != nil {
		return nil, err
	}
	return &d, nil
}

// parseRecipients menggabungkan penerima Kepada lalu Tembusan (urutan dipertahankan) dan menolak
// unit atau alias yang muncul lebih dari sekali, termasuk di Kepada dan Tembusan sekaligus.
func parseRecipients(to, cc []RecipientInput) ([]model.RecipientRef, error) {
	if len(to)+len(cc) > maxRecipients {
		return nil, invalid(fmt.Sprintf("Penerima maksimal %d", maxRecipients))
	}
	out := make([]model.RecipientRef, 0, len(to)+len(cc))
	seen := map[model.RecipientRef]bool{}
	add := func(typ string, items []RecipientInput) error {
		for _, it := range items {
			if (it.UnitID == nil) == (it.AliasLevelID == nil) {
				return invalid("Setiap penerima harus berisi tepat satu dari unitId atau aliasLevelId")
			}
			ref := model.RecipientRef{Type: typ}
			key := model.RecipientRef{}
			if it.UnitID != nil {
				id, err := parseID(*it.UnitID, "Unit penerima")
				if err != nil {
					return err
				}
				ref.UnitID, key.UnitID = id, id
			} else {
				id, err := parseID(*it.AliasLevelID, "Alias penerima")
				if err != nil {
					return err
				}
				ref.AliasLevelID, key.AliasLevelID = id, id
			}
			if seen[key] {
				return invalid("Unit atau alias penerima yang sama tidak boleh dipilih lebih dari sekali (termasuk di Kepada dan Tembusan)")
			}
			seen[key] = true
			out = append(out, ref)
		}
		return nil
	}
	if err := add(model.RecipientTo, to); err != nil {
		return nil, err
	}
	if err := add(model.RecipientCC, cc); err != nil {
		return nil, err
	}
	return out, nil
}

// checkComplete memastikan surat siap diproses (dipakai saat submit dan saat menyimpan surat
// yang dikembalikan ke drafter). Semua kekurangan disebut sekaligus.
func checkComplete(d *model.LetterDraft) error {
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
	add(!slices.ContainsFunc(d.Recipients, func(r model.RecipientRef) bool { return r.Type == model.RecipientTo }),
		"minimal 1 penerima Kepada")
	if len(missing) > 0 {
		return invalid("Surat belum lengkap: " + strings.Join(missing, ", "))
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
		return 0, invalid(label + " tidak valid")
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
		return nil, invalid(fmt.Sprintf("%s maksimal %d karakter", label, maxLen))
	}
	return &s, nil
}
