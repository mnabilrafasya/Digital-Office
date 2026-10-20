package service

import (
	"context"
	"errors"
	"strings"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/auth"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
)

// AuthUserRepository adalah query user yang dibutuhkan untuk login
type AuthUserRepository interface {
	FindLoginByEmployeeNumber(ctx context.Context, employeeNumber string) (*model.LoginRecord, error)
	FindActiveByID(ctx context.Context, userID int64) (*model.CurrentUser, error)
}

// TokenIssuer membuat access token untuk user yang berhasil login
type TokenIssuer interface {
	Issue(userID int64) (string, error)
}

// AuthService menangani login NIK + password.
type AuthService struct {
	users  AuthUserRepository
	tokens TokenIssuer
}

// NewAuthService membuat AuthService.
func NewAuthService(users AuthUserRepository, tokens TokenIssuer) *AuthService {
	// Siapkan hash palsu sekarang, supaya login pertama dengan NIK tak terdaftar tidak lebih lambat
	auth.CheckDummyPassword("")
	return &AuthService{users: users, tokens: tokens}
}

// LoginResult adalah hasil login yang berhasil
type LoginResult struct {
	AccessToken string
	User        *model.CurrentUser
}

// Login memeriksa NIK + password. Semua kegagalan login mengembalikan ErrInvalidCredentials
// yang sama, supaya klien tidak bisa menebak NIK mana yang terdaftar.
func (s *AuthService) Login(ctx context.Context, employeeNumber, password string) (*LoginResult, error) {
	employeeNumber = strings.TrimSpace(employeeNumber)
	if employeeNumber == "" || password == "" {
		return nil, invalid("NIK dan password wajib diisi")
	}

	rec, err := s.users.FindLoginByEmployeeNumber(ctx, employeeNumber)
	if errors.Is(err, repository.ErrNotFound) {
		auth.CheckDummyPassword(password)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	// Selalu jalankan bcrypt sekali, apa pun status user, supaya waktu respons setara
	var ok bool
	if rec.PasswordHash != nil {
		ok = auth.CheckPassword(*rec.PasswordHash, password)
	} else {
		ok = auth.CheckDummyPassword(password)
	}
	if !ok || !rec.IsActive || !rec.RoleIsActive {
		return nil, ErrInvalidCredentials
	}

	user, err := s.users.FindActiveByID(ctx, rec.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	token, err := s.tokens.Issue(user.ID)
	if err != nil {
		return nil, err
	}
	return &LoginResult{AccessToken: token, User: user}, nil
}
