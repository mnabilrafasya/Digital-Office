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

// Error yang pesannya aman ditampilkan ke pengguna.
var (
	// ErrInvalidCredentials: NIK/password salah, user nonaktif, belum punya password, atau role nonaktif.
	ErrInvalidCredentials = errors.New("NIK atau password salah")
	// ErrNotFound: data tidak ada atau user tidak berhak melihatnya (keberadaannya tidak dibocorkan).
	ErrNotFound = errors.New("Surat tidak ditemukan")
	// ErrForbidden: user bisa melihat data tetapi perannya tidak pernah boleh melakukan tindakan ini.
	ErrForbidden = errors.New("Anda tidak memiliki akses untuk tindakan ini")
	// ErrStorageFull: sisa ruang storage di bawah batas aman.
	ErrStorageFull = errors.New("Ruang penyimpanan hampir penuh. Hubungi administrator.")
)

// ValidationError menandakan input dari klien tidak valid (400).
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalid(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}

// StateError menandakan aksi tidak sesuai status surat saat ini (409).
type StateError struct{ Message string }

func (e *StateError) Error() string { return e.Message }

func conflict(format string, args ...any) error {
	return &StateError{Message: fmt.Sprintf(format, args...)}
}

// Pesan 409 yang dipakai berulang.
const (
	msgAlreadyProcessed = "Surat sudah diproses. Muat ulang halaman untuk melihat status terbaru."
	msgNotYourTurn      = "Surat sedang tidak menunggu tindakan Anda. Muat ulang halaman untuk melihat status terbaru."
)

// VersionConflictError menandakan If-Match tidak sama dengan versi terbaru (412).
type VersionConflictError struct{ Current []byte }

func (e *VersionConflictError) Error() string {
	return "Surat sudah diubah di tab atau perangkat lain. Muat ulang untuk melihat versi terbaru."
}

// TooManyAttemptsError menandakan percobaan login dibatasi sementara (429).
type TooManyAttemptsError struct{ RetryAfter time.Duration }

func (e *TooManyAttemptsError) Error() string {
	minutes := int((e.RetryAfter + time.Minute - 1) / time.Minute)
	return fmt.Sprintf("Terlalu banyak percobaan login. Coba lagi dalam %d menit.", max(minutes, 1))
}

// FileTooLargeError menandakan file melebihi batas ukuran (413).
type FileTooLargeError struct{ MaxBytes int64 }

func (e *FileTooLargeError) Error() string {
	return fmt.Sprintf("Ukuran file melebihi batas %d MB.", e.MaxBytes>>20)
}

// problems mengumpulkan beberapa masalah validasi menjadi satu pesan.
type problems []string

func (p *problems) add(format string, args ...any) { *p = append(*p, fmt.Sprintf(format, args...)) }

func (p problems) err() error {
	if len(p) == 0 {
		return nil
	}
	return &ValidationError{Message: strings.Join(p, "; ")}
}

// maxSearchLength adalah panjang maksimal teks pencarian q (karakter).
const maxSearchLength = 100

// normalizeSearch merapikan teks pencarian: trim spasi, "" berarti tidak mencari.
func normalizeSearch(q string) (string, error) {
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) > maxSearchLength {
		return "", invalid("Teks pencarian maksimal %d karakter", maxSearchLength)
	}
	return q, nil
}
