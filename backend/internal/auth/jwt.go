// Package auth berisi pembuatan/pemeriksaan token dan pengecekan password.
// Login NIK + password bersifat sementara sampai SSO tersedia; middleware hanya
// bergantung pada token, sehingga cara login bisa diganti tanpa mengubah middleware.
package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Issuer adalah nilai klaim "iss" pada token aplikasi ini.
const Issuer = "digioffice"

// ErrInvalidToken dikembalikan bila token rusak, salah tanda tangan, kedaluwarsa, atau klaimnya salah.
var ErrInvalidToken = errors.New("token tidak valid")

// accessClaims adalah isi access token. tv = token_version user saat token dibuat;
// token ditolak bila token_version di database sudah naik (logout).
type accessClaims struct {
	jwt.RegisteredClaims
	TokenVersion int `json:"tv"`
}

// JWT membuat dan memeriksa access token (HS256).
type JWT struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewJWT membuat penerbit access token dengan secret dan masa berlaku tertentu.
func NewJWT(secret string, ttl time.Duration) *JWT {
	return &JWT{secret: []byte(secret), ttl: ttl, now: time.Now}
}

// TTL mengembalikan masa berlaku token.
func (j *JWT) TTL() time.Duration { return j.ttl }

// Issue membuat access token untuk user dengan token_version tertentu.
func (j *JWT) Issue(userID int64, tokenVersion int) (string, error) {
	now := j.now()
	claims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			Issuer:    Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
		TokenVersion: tokenVersion,
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
	if err != nil {
		return "", fmt.Errorf("membuat token: %w", err)
	}
	return token, nil
}

// Verify memeriksa tanda tangan, exp, dan iss, lalu mengembalikan user id ("sub") dan
// token_version ("tv"). Token tanpa klaim tv (dibuat sebelum fitur logout) dianggap tidak valid.
func (j *JWT) Verify(token string) (userID int64, tokenVersion int, err error) {
	var claims accessClaims
	_, err = jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return j.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(j.now),
	)
	if err != nil {
		return 0, 0, ErrInvalidToken
	}

	userID, err = strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 || claims.TokenVersion < 1 {
		return 0, 0, ErrInvalidToken
	}
	return userID, claims.TokenVersion, nil
}
