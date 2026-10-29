package surat

import (
	"context"

	"office.backend/internal/auth"
)

// History mengambil riwayat aksi surat (urut dari yang paling lama) untuk drafter dan peserta.
func (s *Service) History(ctx context.Context, user *auth.CurrentUser, letterID string) ([]LetterAction, error) {
	l, err := s.readable(ctx, user, letterID)
	if err != nil {
		return nil, err
	}
	actions, err := s.reader.Actions(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	for i := range actions {
		actions[i].CreatedAt = actions[i].CreatedAt.In(s.loc)
	}
	return actions, nil
}
