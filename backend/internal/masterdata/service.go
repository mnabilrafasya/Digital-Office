// Package masterdata berisi data acuan (tabel lookups): jenis surat, sifat, kecepatan,
// klasifikasi, status, dan level jabatan.
package masterdata

import (
	"context"
	"slices"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

// Kategori lookup yang boleh diminta klien (sama dengan CHECK ck_lookups_category)
var lookupCategories = []string{
	"LETTER_TYPE", "CONFIDENTIALITY", "URGENCY", "CLASSIFICATION", "LETTER_STATUS", "INBOX_STATUS",
}

// LookupRepository adalah query lookup untuk dropdown.
type LookupRepository interface {
	List(ctx context.Context, companyCode, category string) ([]Lookup, error)
}

// Service menyediakan pilihan dropdown. Data otomatis disaring ke perusahaan user yang login.
type Service struct {
	lookups LookupRepository
}

// NewService membuat Service.
func NewService(lookups LookupRepository) *Service {
	return &Service{lookups: lookups}
}

// Lookups mengambil pilihan dropdown satu kategori.
func (s *Service) Lookups(ctx context.Context, user *auth.CurrentUser, category string) ([]Lookup, error) {
	if !slices.Contains(lookupCategories, category) {
		return nil, apperr.Invalid("Parameter category wajib salah satu dari: LETTER_TYPE, CONFIDENTIALITY, URGENCY, CLASSIFICATION, LETTER_STATUS, INBOX_STATUS")
	}
	return s.lookups.List(ctx, user.Company.Code, category)
}
