// Package service berisi aturan bisnis. Service tidak mengenal HTTP; handler yang
// menerjemahkan error service menjadi response.
package service

import (
	"errors"
	"strings"
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
