package auth

import (
	"errors"
	"testing"
	"time"
)

const letterA = "6f1c2a9e-0000-4000-8000-000000000001"

func TestUploadTokenRoundTrip(t *testing.T) {
	u := NewUploadTokens(testSecret, 15*time.Minute)
	in := UploadClaims{UserID: 8, LetterUUID: letterA, Kind: "ATTACHMENT", ObjectID: "obj-1", FileName: "a.pdf"}
	token, exp, err := u.Issue(in)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Until(exp); d < 29*time.Minute || d > 31*time.Minute {
		t.Errorf("masa berlaku = %v, want sekitar URL TTL + 15 menit", d)
	}
	got, err := u.Verify(token, 8, letterA, "ATTACHMENT")
	if err != nil || got != in {
		t.Fatalf("Verify() = %+v, %v; want %+v", got, err, in)
	}
}

func TestUploadTokenRejected(t *testing.T) {
	u := NewUploadTokens(testSecret, 15*time.Minute)
	claims := UploadClaims{UserID: 8, LetterUUID: letterA, Kind: "ATTACHMENT", ObjectID: "obj-1", FileName: "a.pdf"}
	token, _, _ := u.Issue(claims)

	expiredIssuer := NewUploadTokens(testSecret, 15*time.Minute)
	expiredIssuer.now = func() time.Time { return time.Now().Add(-time.Hour) }
	expired, _, _ := expiredIssuer.Issue(claims)

	forged, _, _ := NewUploadTokens("secret-lain-yang-juga-panjangnya-lebih-dari-32", 15*time.Minute).Issue(claims)
	accessToken, _ := NewJWT(testSecret, time.Hour).Issue(8, 1)

	tests := []struct {
		name, token, letter, kind string
		user                      int64
	}{
		{"kedaluwarsa", expired, letterA, "ATTACHMENT", 8},
		{"dipalsukan (kunci lain)", forged, letterA, "ATTACHMENT", 8},
		{"surat berbeda", token, "6f1c2a9e-0000-4000-8000-000000000002", "ATTACHMENT", 8},
		{"jenis berbeda", token, letterA, "VERSION", 8},
		{"user berbeda", token, letterA, "ATTACHMENT", 9},
		{"access token dipakai sebagai upload token", accessToken, letterA, "ATTACHMENT", 8},
	}
	for _, tt := range tests {
		if _, err := u.Verify(tt.token, tt.user, tt.letter, tt.kind); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%s: err = %v, want ErrInvalidToken", tt.name, err)
		}
	}

	// Kebalikannya: upload token tidak boleh diterima sebagai access token
	if _, _, err := NewJWT(testSecret, time.Hour).Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("upload token diterima sebagai access token: err = %v", err)
	}
}
