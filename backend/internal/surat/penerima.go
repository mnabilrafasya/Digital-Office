package surat

import (
	"context"
	"fmt"

	"office.backend/internal/platform/apperr"
)

// parseRecipients menggabungkan penerima Kepada lalu Tembusan (urutan dipertahankan) dan menolak
// unit atau alias yang muncul lebih dari sekali, termasuk di Kepada dan Tembusan sekaligus.
func parseRecipients(to, cc []RecipientInput) ([]RecipientRef, error) {
	if len(to)+len(cc) > maxRecipients {
		return nil, apperr.Invalid(fmt.Sprintf("Penerima maksimal %d", maxRecipients))
	}
	out := make([]RecipientRef, 0, len(to)+len(cc))
	seen := map[RecipientRef]bool{}
	add := func(typ string, items []RecipientInput) error {
		for _, it := range items {
			if (it.UnitID == nil) == (it.AliasLevelID == nil) {
				return apperr.Invalid("Setiap penerima harus berisi tepat satu dari unitId atau aliasLevelId")
			}
			ref := RecipientRef{Type: typ}
			key := RecipientRef{}
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
				return apperr.Invalid("Unit atau alias penerima yang sama tidak boleh dipilih lebih dari sekali (termasuk di Kepada dan Tembusan)")
			}
			seen[key] = true
			out = append(out, ref)
		}
		return nil
	}
	if err := add(RecipientTo, to); err != nil {
		return nil, err
	}
	if err := add(RecipientCC, cc); err != nil {
		return nil, err
	}
	return out, nil
}

// checkRecipientUnits memastikan setiap unit penerima aktif dan milik perusahaan user.
func (s *Service) checkRecipientUnits(ctx context.Context, companyCode string, d *LetterDraft) error {
	var unitIDs []int64
	for _, r := range d.Recipients {
		if r.UnitID != 0 {
			unitIDs = append(unitIDs, r.UnitID)
		}
	}
	units, err := s.repos.Directory.ActiveUnitIDs(ctx, companyCode, unitIDs)
	if err != nil {
		return err
	}
	if !allIn(unitIDs, units) {
		return apperr.Invalid("Unit penerima tidak ditemukan atau tidak aktif")
	}
	return nil
}
