package surat

import (
	"context"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

// checkParticipants memastikan reviewer punya LETTER_REVIEW dan approver punya LETTER_APPROVE, aktif,
// di perusahaan yang sama (syarat yang sama dengan pencarian kandidat).
func (s *Service) checkParticipants(ctx context.Context, companyCode string, d *LetterDraft) error {
	reviewers, err := s.repos.Directory.EligibleUserIDs(ctx, companyCode, auth.LetterReview, d.ReviewerIDs)
	if err != nil {
		return err
	}
	if !allIn(d.ReviewerIDs, reviewers) {
		return apperr.Invalid("Reviewer tidak ditemukan, tidak aktif, atau tidak berwenang memaraf surat")
	}
	if d.ApproverID != nil {
		approver, err := s.repos.Directory.EligibleUserIDs(ctx, companyCode, auth.LetterApprove, []int64{*d.ApproverID})
		if err != nil {
			return err
		}
		if !approver[*d.ApproverID] {
			return apperr.Invalid("Approver tidak ditemukan, tidak aktif, atau tidak berwenang menandatangani surat")
		}
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
