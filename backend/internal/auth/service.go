package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// UserStore adalah query user yang dibutuhkan untuk login dan logout.
type UserStore interface {
	FindLoginByEmployeeNumber(ctx context.Context, employeeNumber string) (*LoginRecord, error)
	FindActiveByID(ctx context.Context, userID int64) (*CurrentUser, error)
	IncrementTokenVersion(ctx context.Context, userID int64) error
}

// TokenIssuer membuat access token untuk user yang berhasil login.
type TokenIssuer interface {
	Issue(userID int64, tokenVersion int) (string, error)
}

// PasswordVerifier memeriksa password terhadap hash bcrypt, dengan penyetaraan waktu.
type PasswordVerifier interface {
	Check(hash, password string) bool
	CheckDummy(password string) bool
}

// AttemptLimiter membatasi percobaan login gagal (dipenuhi *LoginLimiter).
type AttemptLimiter interface {
	Check(ip, employeeNumber string) time.Duration
	Failure(ip, employeeNumber string)
	Success(ip, employeeNumber string)
}

// Service menangani login NIK + password dan logout.
type Service struct {
	users     UserStore
	tokens    TokenIssuer
	passwords PasswordVerifier
	limiter   AttemptLimiter
}

// NewService membuat Service.
func NewService(users UserStore, tokens TokenIssuer, passwords PasswordVerifier, limiter AttemptLimiter) *Service {
	return &Service{users: users, tokens: tokens, passwords: passwords, limiter: limiter}
}

// LoginInput adalah data login dari klien.
type LoginInput struct {
	EmployeeNumber string
	Password       string
	ClientIP       string
}

// LoginResult adalah hasil login yang berhasil.
type LoginResult struct {
	AccessToken string
	User        *CurrentUser
}

// Login memeriksa NIK + password. Semua kegagalan login mengembalikan ErrInvalidCredentials yang
// sama, supaya klien tidak bisa menebak NIK mana yang terdaftar. Pembatasan percobaan dicek
// sebelum bcrypt, sehingga serangan tebak password tidak membebani CPU server.
func (s *Service) Login(ctx context.Context, in LoginInput) (*LoginResult, error) {
	nik := strings.TrimSpace(in.EmployeeNumber)
	if nik == "" || in.Password == "" {
		return nil, apperr.Invalid("NIK dan password wajib diisi")
	}
	if wait := s.limiter.Check(in.ClientIP, nik); wait > 0 {
		return nil, &TooManyAttemptsError{RetryAfter: wait}
	}

	user, err := s.authenticate(ctx, nik, in.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		s.limiter.Failure(in.ClientIP, nik)
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	s.limiter.Success(in.ClientIP, nik)

	token, err := s.tokens.Issue(user.ID, user.TokenVersion)
	if err != nil {
		return nil, err
	}
	return &LoginResult{AccessToken: token, User: user}, nil
}

func (s *Service) authenticate(ctx context.Context, nik, password string) (*CurrentUser, error) {
	rec, err := s.users.FindLoginByEmployeeNumber(ctx, nik)
	if errors.Is(err, db.ErrNotFound) {
		s.passwords.CheckDummy(password)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	// Selalu jalankan bcrypt sekali, apa pun status user, supaya waktu respons setara
	var ok bool
	if rec.PasswordHash != nil {
		ok = s.passwords.Check(*rec.PasswordHash, password)
	} else {
		ok = s.passwords.CheckDummy(password)
	}
	if !ok || !rec.IsActive || !rec.RoleIsActive {
		return nil, ErrInvalidCredentials
	}

	user, err := s.users.FindActiveByID(ctx, rec.UserID)
	if errors.Is(err, db.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	return user, err
}

// Logout mengakhiri semua sesi user (semua perangkat) dengan menaikkan token_version.
func (s *Service) Logout(ctx context.Context, user *CurrentUser) error {
	return s.users.IncrementTokenVersion(ctx, user.ID)
}
