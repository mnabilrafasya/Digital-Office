package service

import (
	"context"
	"errors"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
)

// checkRefs memastikan semua data yang dirujuk draf ada, aktif, dan milik perusahaan user:
// lookup sesuai kategori, template (jenisnya cocok), unit penerima, alias, reviewer, dan approver.
// Berlaku sejak draf, walaupun surat belum lengkap.
func (s *LetterService) checkRefs(ctx context.Context, companyCode string, d *model.LetterDraft) error {
	if err := s.checkLookups(ctx, companyCode, d); err != nil {
		return err
	}
	if err := s.checkTemplate(ctx, companyCode, d); err != nil {
		return err
	}

	var unitIDs []int64
	for _, r := range d.Recipients {
		if r.UnitID != 0 {
			unitIDs = append(unitIDs, r.UnitID)
		}
	}
	units, err := s.repos.Refs.ActiveUnitIDs(ctx, companyCode, unitIDs)
	if err != nil {
		return err
	}
	if !allIn(unitIDs, units) {
		return invalid("Unit penerima tidak ditemukan atau tidak aktif")
	}

	reviewers, err := s.repos.Refs.EligibleUserIDs(ctx, companyCode, permission.LetterReview, d.ReviewerIDs)
	if err != nil {
		return err
	}
	if !allIn(d.ReviewerIDs, reviewers) {
		return invalid("Reviewer tidak ditemukan, tidak aktif, atau tidak berwenang memaraf surat")
	}
	if d.ApproverID != nil {
		approver, err := s.repos.Refs.EligibleUserIDs(ctx, companyCode, permission.LetterApprove, []int64{*d.ApproverID})
		if err != nil {
			return err
		}
		if !approver[*d.ApproverID] {
			return invalid("Approver tidak ditemukan, tidak aktif, atau tidak berwenang menandatangani surat")
		}
	}
	return nil
}

// checkLookups memeriksa semua lookup draf (termasuk alias penerima) dengan satu query.
func (s *LetterService) checkLookups(ctx context.Context, companyCode string, d *model.LetterDraft) error {
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
	categories, err := s.repos.Refs.LookupCategories(ctx, companyCode, ids)
	if err != nil {
		return err
	}
	for _, w := range wants {
		if categories[w.id] != w.category {
			return invalid(w.message)
		}
	}
	return nil
}

// checkTemplate memastikan template aktif milik perusahaan, dan kalau template dibuat untuk jenis
// surat tertentu, jenisnya sama dengan jenis surat draf.
func (s *LetterService) checkTemplate(ctx context.Context, companyCode string, d *model.LetterDraft) error {
	if d.TemplateID == nil {
		return nil
	}
	typeID, err := s.repos.Refs.TemplateLetterType(ctx, companyCode, *d.TemplateID)
	if errors.Is(err, repository.ErrNotFound) {
		return invalid("Template tidak ditemukan atau tidak aktif")
	}
	if err != nil {
		return err
	}
	if typeID != nil && d.LetterTypeID != nil && *typeID != *d.LetterTypeID {
		return invalid("Template tidak sesuai dengan jenis surat yang dipilih")
	}
	return nil
}

func allIn(ids []int64, found map[int64]bool) bool {
	for _, id := range ids {
		if !found[id] {
			return false
		}
	}
	return true
}
