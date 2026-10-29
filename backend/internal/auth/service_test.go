package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
)

// ---- Login ----

type fakeAuthUsers struct {
	logins map[string]*LoginRecord
	active map[int64]*CurrentUser
	bumped []int64
}

func (f *fakeAuthUsers) FindLoginByEmployeeNumber(_ context.Context, nik string) (*LoginRecord, error) {
	if r, ok := f.logins[nik]; ok {
		return r, nil
	}
	return nil, db.ErrNotFound
}

func (f *fakeAuthUsers) FindActiveByID(_ context.Context, id int64) (*CurrentUser, error) {
	if u, ok := f.active[id]; ok {
		return u, nil
	}
	return nil, db.ErrNotFound
}

func (f *fakeAuthUsers) IncrementTokenVersion(_ context.Context, id int64) error {
	f.bumped = append(f.bumped, id)
	return nil
}

type fakeTokens struct{ lastTV int }

func (f *fakeTokens) Issue(_ int64, tv int) (string, error) { f.lastTV = tv; return "token-uji", nil }

// fakePasswords memeriksa password teks biasa (tanpa bcrypt supaya tes cepat).
type fakePasswords struct{ dummyCalls int }

func (f *fakePasswords) Check(hash, password string) bool { return hash == "hash:"+password }
func (f *fakePasswords) CheckDummy(string) bool           { f.dummyCalls++; return false }

type fakeLimiter struct {
	blocked           time.Duration
	failures, success int
}

func (f *fakeLimiter) Check(string, string) time.Duration { return f.blocked }
func (f *fakeLimiter) Failure(string, string)             { f.failures++ }
func (f *fakeLimiter) Success(string, string)             { f.success++ }

func TestLogin(t *testing.T) {
	hash := "hash:benar123"
	users := &fakeAuthUsers{
		logins: map[string]*LoginRecord{
			"1": {UserID: 1, PasswordHash: &hash, IsActive: true, RoleIsActive: true},
			"2": {UserID: 2, PasswordHash: &hash, IsActive: false, RoleIsActive: true},
			"3": {UserID: 3, PasswordHash: nil, IsActive: true, RoleIsActive: true},
			"4": {UserID: 4, PasswordHash: &hash, IsActive: true, RoleIsActive: false},
		},
		active: map[int64]*CurrentUser{1: {ID: 1, Name: "Aktif", TokenVersion: 3}},
	}
	tokens, passwords, limiter := &fakeTokens{}, &fakePasswords{}, &fakeLimiter{}
	svc := NewService(users, tokens, passwords, limiter)
	ctx := context.Background()

	res, err := svc.Login(ctx, LoginInput{EmployeeNumber: " 1 ", Password: "benar123", ClientIP: "10.0.0.1"})
	if err != nil || res.AccessToken != "token-uji" || res.User.ID != 1 || tokens.lastTV != 3 || limiter.success != 1 {
		t.Fatalf("login valid gagal: res=%+v err=%v tv=%d", res, err, tokens.lastTV)
	}

	gagal := []struct{ name, nik, pw string }{
		{"password salah", "1", "salah"},
		{"NIK tidak ada", "9999999", "benar123"},
		{"user nonaktif", "2", "benar123"},
		{"belum punya password", "3", "benar123"},
		{"role nonaktif", "4", "benar123"},
	}
	for _, tt := range gagal {
		if _, err := svc.Login(ctx, LoginInput{EmployeeNumber: tt.nik, Password: tt.pw}); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("%s: err = %v, want ErrInvalidCredentials", tt.name, err)
		}
	}
	if limiter.failures != len(gagal) {
		t.Errorf("kegagalan tercatat %d, want %d", limiter.failures, len(gagal))
	}
	if passwords.dummyCalls != 2 {
		t.Errorf("hash palsu dipakai %d kali, want 2 (NIK tidak ada dan belum punya password)", passwords.dummyCalls)
	}

	// Input kosong ditolak tanpa dihitung sebagai kegagalan login
	var ve *apperr.ValidationError
	for _, in := range []LoginInput{{EmployeeNumber: "  ", Password: "x"}, {EmployeeNumber: "1"}} {
		if _, err := svc.Login(ctx, in); !errors.As(err, &ve) {
			t.Errorf("%+v: err = %v, want apperr.ValidationError", in, err)
		}
	}
	if limiter.failures != len(gagal) {
		t.Errorf("input kosong ikut dihitung sebagai kegagalan")
	}

	// Diblokir: ditolak sebelum memeriksa password
	limiter.blocked = 7 * time.Minute
	var tm *TooManyAttemptsError
	_, err = svc.Login(ctx, LoginInput{EmployeeNumber: "1", Password: "benar123"})
	if !errors.As(err, &tm) || tm.RetryAfter != 7*time.Minute || !strings.Contains(tm.Error(), "7 menit") {
		t.Errorf("diblokir: err = %v, want TooManyAttemptsError 7 menit", err)
	}

	if err := svc.Logout(ctx, &CurrentUser{ID: 1}); err != nil || len(users.bumped) != 1 {
		t.Errorf("Logout: err=%v bumped=%v", err, users.bumped)
	}
}
