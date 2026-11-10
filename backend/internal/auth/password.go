package auth

import (
	"fmt"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"office.backend/internal/platform/apperr"
)

// BcryptCost adalah cost hash password user (sama dengan cmd/seed).
const BcryptCost = 12

// Batas panjang password yang diatur admin. bcrypt hanya membaca 72 byte pertama.
const (
	MinPasswordLength = 8
	maxPasswordBytes  = 72
)

// HashPassword memvalidasi panjang password lalu membuat hash bcrypt-nya.
func HashPassword(password string) (string, error) {
	if utf8.RuneCountInString(password) < MinPasswordLength {
		return "", apperr.Invalid(fmt.Sprintf("Password minimal %d karakter", MinPasswordLength))
	}
	if len(password) > maxPasswordBytes {
		return "", apperr.Invalid(fmt.Sprintf("Password maksimal %d byte", maxPasswordBytes))
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("membuat hash password: %w", err)
	}
	return string(h), nil
}

// PasswordChecker memeriksa password bcrypt. Ia menyimpan hash palsu supaya kegagalan login
// (NIK tidak ada, user nonaktif, belum punya password) memakan waktu yang sama dengan
// pengecekan password sungguhan, sehingga waktu respons tidak membocorkan NIK yang terdaftar.
type PasswordChecker struct {
	dummyHash []byte
}

// NewPasswordChecker membuat PasswordChecker dan menyiapkan hash palsu (butuh beberapa ratus ms).
func NewPasswordChecker() (*PasswordChecker, error) {
	// Isi hash palsu tidak penting karena hasil perbandingannya selalu diabaikan
	h, err := bcrypt.GenerateFromPassword([]byte("digioffice-dummy-password"), BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("menyiapkan hash palsu: %w", err)
	}
	return &PasswordChecker{dummyHash: h}, nil
}

// Check mengembalikan true bila password cocok dengan hash bcrypt.
func (p *PasswordChecker) Check(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// CheckDummy menjalankan perbandingan bcrypt terhadap hash palsu dan selalu mengembalikan false.
func (p *PasswordChecker) CheckDummy(password string) bool {
	_ = bcrypt.CompareHashAndPassword(p.dummyHash, []byte(password))
	return false
}
