package surat

import (
	"office.backend/internal/platform/apperr"
)

// ErrLetterNotFound: surat tidak ada, sudah dihapus, atau user tidak berhak melihatnya (404,
// supaya keberadaan surat tidak bocor).
var ErrLetterNotFound error = &apperr.NotFoundError{Message: "Surat tidak ditemukan"}
