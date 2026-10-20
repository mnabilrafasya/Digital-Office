package service

import (
	"context"
	"slices"
	"strconv"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
)

// lookupCategories adalah kategori lookup yang boleh diminta klien lewat GET /lookups.
var lookupCategories = []string{
	model.CategoryLetterType, model.CategoryConfidentiality, model.CategoryUrgency,
	model.CategoryClassification, model.CategoryLetterStatus, model.CategoryInboxStatus,
}

// Jenis kandidat yang bisa dicari di form Buat Surat.
const (
	CandidateReviewer = "REVIEWER"
	CandidateApprover = "APPROVER"
)

const (
	defaultCandidateLimit = 20
	maxCandidateLimit     = 50
)

// LookupRepository adalah query lookup untuk dropdown dan alias penerima.
type LookupRepository interface {
	List(ctx context.Context, companyCode, category string) ([]model.Lookup, error)
	ListAliases(ctx context.Context, companyCode string) ([]repository.AliasRow, error)
}

// UnitRepository adalah query daftar unit.
type UnitRepository interface {
	List(ctx context.Context, companyCode, search string) ([]model.Unit, error)
}

// CandidateRepository adalah query kandidat reviewer/approver.
type CandidateRepository interface {
	ListCandidates(ctx context.Context, f repository.CandidateFilter) ([]model.Candidate, error)
}

// FormService menyediakan data untuk form Buat Surat. Semua data otomatis disaring
// ke perusahaan user yang login; perusahaan tidak pernah dibaca dari request.
type FormService struct {
	lookups    LookupRepository
	units      UnitRepository
	candidates CandidateRepository
}

// NewFormService membuat FormService.
func NewFormService(lookups LookupRepository, units UnitRepository, candidates CandidateRepository) *FormService {
	return &FormService{lookups: lookups, units: units, candidates: candidates}
}

// Lookups mengambil pilihan dropdown satu kategori.
func (s *FormService) Lookups(ctx context.Context, user *model.CurrentUser, category string) ([]model.Lookup, error) {
	if !slices.Contains(lookupCategories, category) {
		return nil, invalid("Parameter category wajib salah satu dari: LETTER_TYPE, CONFIDENTIALITY, URGENCY, CLASSIFICATION, LETTER_STATUS, INBOX_STATUS")
	}
	return s.lookups.List(ctx, user.Company.Code, category)
}

// Units mengambil unit aktif di perusahaan user, opsional dicari berdasarkan nama.
func (s *FormService) Units(ctx context.Context, user *model.CurrentUser, q string) ([]model.Unit, error) {
	search, err := normalizeSearch(q)
	if err != nil {
		return nil, err
	}
	return s.units.List(ctx, user.Company.Code, search)
}

// Aliases mengambil alias penerima (level jabatan) yang punya minimal satu anggota.
func (s *FormService) Aliases(ctx context.Context, user *model.CurrentUser) ([]model.Alias, error) {
	rows, err := s.lookups.ListAliases(ctx, user.Company.Code)
	if err != nil {
		return nil, err
	}
	out := []model.Alias{}
	for _, r := range rows {
		if r.MemberCount > 0 {
			out = append(out, aliasView(r))
		}
	}
	return out, nil
}

func aliasView(r repository.AliasRow) model.Alias {
	a := r.Alias
	a.Label = "Seluruh " + r.Name
	return a
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
func (s *FormService) Candidates(ctx context.Context, user *model.CurrentUser, cq CandidateQuery) ([]model.Candidate, error) {
	f := repository.CandidateFilter{CompanyCode: user.Company.Code}

	switch cq.Type {
	case CandidateReviewer:
		f.PermissionCode = permission.LetterReview
	case CandidateApprover:
		f.PermissionCode = permission.LetterApprove
		f.ExcludeUserID = user.ID
	default:
		return nil, invalid("Parameter type wajib REVIEWER atau APPROVER")
	}

	f.Limit = defaultCandidateLimit
	if cq.Limit != "" {
		n, err := strconv.Atoi(cq.Limit)
		if err != nil || n < 1 || n > maxCandidateLimit {
			return nil, invalid("Parameter limit harus angka antara 1 dan 50")
		}
		f.Limit = n
	}

	search, err := normalizeSearch(cq.Search)
	if err != nil {
		return nil, err
	}
	f.Search = search

	return s.candidates.ListCandidates(ctx, f)
}
