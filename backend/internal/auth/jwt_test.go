package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const testSecret = "rahasia-uji-yang-panjangnya-lebih-dari-32-karakter"

func TestIssueAndVerify(t *testing.T) {
	j := NewJWT(testSecret, time.Hour)
	token, err := j.Issue(42, 3)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	id, tv, err := j.Verify(token)
	if err != nil || id != 42 || tv != 3 {
		t.Fatalf("Verify() = %d, %d, %v; want 42, 3, nil", id, tv, err)
	}
}

func TestVerifyRejects(t *testing.T) {
	j := NewJWT(testSecret, time.Hour)
	future := jwt.NewNumericDate(time.Now().Add(time.Hour))

	sign := func(method jwt.SigningMethod, key any, claims jwt.Claims) string {
		t.Helper()
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatalf("membuat token uji: %v", err)
		}
		return s
	}
	valid := func() accessClaims {
		return accessClaims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "42", Issuer: Issuer, ExpiresAt: future},
			TokenVersion:     1,
		}
	}
	other, _ := NewJWT("secret-lain-yang-juga-panjangnya-lebih-dari-32", time.Hour).Issue(42, 1)

	expired := NewJWT(testSecret, time.Hour)
	expired.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	expiredToken, _ := expired.Issue(42, 1)

	wrongIssuer := valid()
	wrongIssuer.Issuer = "aplikasi-lain"
	noExpiry := valid()
	noExpiry.ExpiresAt = nil
	noTV := valid()
	noTV.TokenVersion = 0

	tests := map[string]string{
		"secret lain":                 other,
		"algoritma none":              sign(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, valid()),
		"kedaluwarsa":                 expiredToken,
		"issuer salah":                sign(jwt.SigningMethodHS256, []byte(testSecret), wrongIssuer),
		"tanpa exp":                   sign(jwt.SigningMethodHS256, []byte(testSecret), noExpiry),
		"tanpa klaim tv (token lama)": sign(jwt.SigningMethodHS256, []byte(testSecret), noTV),
		"bukan JWT":                   "abc.def.ghi",
	}
	for name, token := range tests {
		if _, _, err := j.Verify(token); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%s: err = %v, want ErrInvalidToken", name, err)
		}
	}
}

func TestPasswordChecker(t *testing.T) {
	p, err := NewPasswordChecker()
	if err != nil {
		t.Fatal(err)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("rahasia123"), bcrypt.MinCost)
	if !p.Check(string(hash), "rahasia123") || p.Check(string(hash), "salah") {
		t.Error("Check salah")
	}
	if p.CheckDummy("rahasia123") {
		t.Error("CheckDummy harus selalu false")
	}
}
