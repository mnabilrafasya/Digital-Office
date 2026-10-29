// Package organisasi berisi data organisasi untuk form surat: unit kerja, kandidat
// reviewer/approver, dan alias penerima (level jabatan).
package organisasi

import (
	"context"
	"strconv"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// Jenis kandidat yang bisa dicari di form Buat Surat
const (
	CandidateReviewer = "REVIEWER"
	CandidateApprover = "APPROVER"
)

const (
	defaultCandidateLimit = 20
	maxCandidateLimit     = 50
)

// UnitRepository adalah query daftar unit.
type UnitRepository interface {
	ListUnits(ctx context.Context, companyCode, search string) ([]Unit, error)
}

// CandidateRepository adalah query kandidat reviewer/approver.
type CandidateRepository interface {
	ListCandidates(ctx context.Context, f CandidateFilter) ([]Candidate, error)
}

// AliasRepository adalah query alias penerima.
type AliasRepository interface {
	ListAliases(ctx context.Context, companyCode string) ([]Alias, error)
}

// Service menyediakan data organisasi untuk form Buat Surat. Semua data otomatis disaring
// ke perusahaan user yang login; perusahaan tidak pernah dibaca dari request.
type Service struct {
	units      UnitRepository
	candidates CandidateRepository
	aliases    AliasRepository
}

// NewService membuat Service.
func NewService(units UnitRepository, candidates CandidateRepository, aliases AliasRepository) *Service {
	return &Service{units: units, candidates: candidates, aliases: aliases}
}

// Units mengambil unit aktif di perusahaan user, opsional dicari berdasarkan nama.
func (s *Service) Units(ctx context.Context, user *auth.CurrentUser, q string) ([]Unit, error) {
	search, err := db.NormalizeSearch(q)
	if err != nil {
		return nil, err
	}
	return s.units.ListUnits(ctx, user.Company.Code, search)
}

// CandidateQuery adalah parameter pencarian kandidat, apa adanya dari query string.
// Limit "" berarti memakai default.
type CandidateQuery struct {
	Type   string
	Search string
	Limit  string
}

// Candidates mencari kandidat reviewer/approver di perusahaan user yang login.
// Drafter boleh memilih dirinya sendiri sebagai reviewer, tetapi tidak sebagai approver.
func (s *Service) Candidates(ctx context.Context, user *auth.CurrentUser, cq CandidateQuery) ([]Candidate, error) {
	f := CandidateFilter{CompanyCode: user.Company.Code}

	switch cq.Type {
	case CandidateReviewer:
		f.PermissionCode = auth.LetterReview
	case CandidateApprover:
		f.PermissionCode = auth.LetterApprove
		f.ExcludeUserID = user.ID
	default:
		return nil, apperr.Invalid("Parameter type wajib REVIEWER atau APPROVER")
	}

	f.Limit = defaultCandidateLimit
	if cq.Limit != "" {
		n, err := strconv.Atoi(cq.Limit)
		if err != nil || n < 1 || n > maxCandidateLimit {
			return nil, apperr.Invalid("Parameter limit harus angka antara 1 dan 50")
		}
		f.Limit = n
	}

	search, err := db.NormalizeSearch(cq.Search)
	if err != nil {
		return nil, err
	}
	f.Search = search

	return s.candidates.ListCandidates(ctx, f)
}

// Aliases mengambil alias penerima (level jabatan) yang punya minimal 1 anggota.
func (s *Service) Aliases(ctx context.Context, user *auth.CurrentUser) ([]Alias, error) {
	return s.aliases.ListAliases(ctx, user.Company.Code)
}
