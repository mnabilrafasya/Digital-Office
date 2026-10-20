package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost adalah cost hash password user (sama dengan cmd/seed).
const BcryptCost = 12

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

// HashPassword membuat hash bcrypt dari password (dipakai cmd/seed).
func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("membuat hash password: %w", err)
	}
	return string(h), nil
}
