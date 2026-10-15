package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "rahasia-uji-yang-panjangnya-lebih-dari-32-karakter"

func TestIssueAndVerify(t *testing.T) {
	j := NewJWT(testSecret, time.Hour)
	token, err := j.Issue(42)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	id, err := j.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if id != 42 {
		t.Fatalf("user id = %d, want 42", id)
	}
}

func TestVerifyRejectsOtherSecret(t *testing.T) {
	token, _ := NewJWT("secret-lain-yang-juga-panjangnya-lebih-dari-32", time.Hour).Issue(42)
	if _, err := NewJWT(testSecret, time.Hour).Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyRejectsAlgNone(t *testing.T) {
	claims := jwt.RegisteredClaims{
		Subject:   "42",
		Issuer:    Issuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("membuat token none: %v", err)
	}
	if _, err := NewJWT(testSecret, time.Hour).Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	j := NewJWT(testSecret, time.Hour)
	j.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	token, _ := j.Issue(42)

	j.now = time.Now
	if _, err := j.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyRejectsWrongIssuer(t *testing.T) {
	claims := jwt.RegisteredClaims{
		Subject:   "42",
		Issuer:    "aplikasi-lain",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if _, err := NewJWT(testSecret, time.Hour).Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyRejectsMissingExpiry(t *testing.T) {
	claims := jwt.RegisteredClaims{Subject: "42", Issuer: Issuer}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if _, err := NewJWT(testSecret, time.Hour).Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}
