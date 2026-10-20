package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// UploadAudience adalah klaim "aud" upload token; access token tidak memilikinya.
const UploadAudience = "digioffice-upload"

// uploadKeyLabel dipakai menurunkan kunci upload token dari JWT_SECRET, sehingga access token
// dan upload token tidak bisa saling dipakai.
const uploadKeyLabel = "digioffice/upload-token/v1"

// uploadTokenGrace adalah tambahan masa berlaku upload token di atas masa berlaku presigned URL,
// supaya unggahan besar yang dimulai di menit terakhir masih bisa dikonfirmasi.
const uploadTokenGrace = 15 * time.Minute

// UploadClaims adalah isi upload token: izin satu user untuk mengonfirmasi satu object
// pada satu surat dengan jenis unggahan tertentu.
type UploadClaims struct {
	UserID     int64
	LetterUUID string
	Kind       string // ATTACHMENT atau VERSION
	ObjectID   string // uuid object
	FileName   string // nama file yang sudah disanitasi
}

type uploadJWTClaims struct {
	jwt.RegisteredClaims
	LetterUUID string `json:"lid"`
	Kind       string `json:"knd"`
	ObjectID   string `json:"oid"`
	FileName   string `json:"fn"`
}

// UploadTokens membuat dan memeriksa upload token (stateless, HS256).
type UploadTokens struct {
	key []byte
	ttl time.Duration
	now func() time.Time
}

// NewUploadTokens membuat penerbit upload token. urlTTL adalah masa berlaku presigned URL.
func NewUploadTokens(jwtSecret string, urlTTL time.Duration) *UploadTokens {
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write([]byte(uploadKeyLabel))
	return &UploadTokens{key: mac.Sum(nil), ttl: urlTTL + uploadTokenGrace, now: time.Now}
}

// Issue membuat upload token dan mengembalikan waktu kedaluwarsanya.
func (u *UploadTokens) Issue(c UploadClaims) (string, time.Time, error) {
	now := u.now()
	exp := now.Add(u.ttl)
	claims := uploadJWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(c.UserID, 10),
			Issuer:    Issuer,
			Audience:  jwt.ClaimStrings{UploadAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
		LetterUUID: c.LetterUUID, Kind: c.Kind, ObjectID: c.ObjectID, FileName: c.FileName,
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(u.key)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("membuat upload token: %w", err)
	}
	return token, exp, nil
}

// Verify memeriksa upload token dan memastikan token itu milik userID, untuk surat letterUUID,
// dan jenis unggahan kind. Selain itu ErrInvalidToken.
func (u *UploadTokens) Verify(token string, userID int64, letterUUID, kind string) (UploadClaims, error) {
	var claims uploadJWTClaims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return u.key, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(UploadAudience),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(u.now),
	)
	if err != nil {
		return UploadClaims{}, ErrInvalidToken
	}
	sub, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || sub != userID || claims.LetterUUID != letterUUID || claims.Kind != kind ||
		claims.ObjectID == "" || claims.FileName == "" {
		return UploadClaims{}, ErrInvalidToken
	}
	return UploadClaims{
		UserID: sub, LetterUUID: claims.LetterUUID, Kind: claims.Kind,
		ObjectID: claims.ObjectID, FileName: claims.FileName,
	}, nil
}
