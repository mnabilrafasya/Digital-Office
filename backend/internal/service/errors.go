// Package service berisi aturan bisnis. Service tidak mengenal HTTP; handler yang
// menerjemahkan error service menjadi response.
package service

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// ErrInvalidCredentials: NIK/password salah, user nonaktif, belum punya password, atau role nonaktif.
var ErrInvalidCredentials = errors.New("NIK atau password salah")

// ValidationError menandakan input dari klien tidak valid; Message aman ditampilkan ke pengguna.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(message string) error { return &ValidationError{Message: message} }

// ErrLetterNotFound: surat tidak ada, sudah dihapus, atau user tidak berhak melihatnya (404,
// supaya keberadaan surat tidak bocor).
var ErrLetterNotFound = errors.New("Surat tidak ditemukan")

// ErrForbidden: user boleh melihat surat, tetapi tidak boleh melakukan tindakan ini (403).
var ErrForbidden = errors.New("Anda tidak memiliki akses untuk tindakan ini")

// InvalidStateError menandakan tindakan tidak bisa dilakukan pada status surat saat ini (409).
type InvalidStateError struct {
	Message string
}

func (e *InvalidStateError) Error() string { return e.Message }

// VersionConflictError menandakan If-Match tidak sama dengan versi terbaru (412).
// RowVersion adalah versi terbaru, dikirim ke klien sebagai ETag.
type VersionConflictError struct {
	RowVersion []byte
}

func (e *VersionConflictError) Error() string {
	return "Surat sudah diubah di tempat lain. Muat ulang untuk melihat versi terbaru."
}

// TooManyAttemptsError menandakan percobaan login dibatasi sementara (429).
type TooManyAttemptsError struct{ RetryAfter time.Duration }

func (e *TooManyAttemptsError) Error() string {
	minutes := int((e.RetryAfter + time.Minute - 1) / time.Minute)
	return fmt.Sprintf("Terlalu banyak percobaan login. Coba lagi dalam %d menit.", max(minutes, 1))
}

// maxSearchLength adalah panjang maksimal teks pencarian q (karakter).
const maxSearchLength = 100

// normalizeSearch merapikan teks pencarian: trim spasi, "" berarti tidak mencari.
func normalizeSearch(q string) (string, error) {
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) > maxSearchLength {
		return "", invalid("Teks pencarian maksimal 100 karakter")
	}
	return q, nil
}
