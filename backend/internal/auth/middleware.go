package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"office.backend/internal/platform/db"
	"office.backend/internal/platform/httpx"
)

// TokenVerifier memeriksa access token dan mengembalikan user id serta token_version di dalamnya.
type TokenVerifier interface {
	Verify(token string) (userID int64, tokenVersion int, err error)
}

// UserLoader memuat user aktif dari database (db.ErrNotFound bila tidak aktif/tidak ada)
type UserLoader interface {
	FindActiveByID(ctx context.Context, userID int64) (*CurrentUser, error)
}

type ctxKey struct{}

// CurrentUserFrom mengambil user yang login dari context (nil bila belum melewati Authenticate)
func CurrentUserFrom(ctx context.Context) *CurrentUser {
	u, _ := ctx.Value(ctxKey{}).(*CurrentUser)
	return u
}

// WithCurrentUser menyimpan user ke context (dipakai Authenticate dan tes)
func WithCurrentUser(ctx context.Context, u *CurrentUser) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

// Authenticate mewajibkan header "Authorization: Bearer <token>" yang valid. User dimuat
// ulang dari database setiap request, sehingga penonaktifan user, perubahan role, dan logout
// (token_version naik) langsung berlaku tanpa menunggu token kedaluwarsa.
func Authenticate(tokens TokenVerifier, users UserLoader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				httpx.Unauthorized(w)
				return
			}

			userID, tokenVersion, err := tokens.Verify(token)
			if err != nil {
				httpx.Unauthorized(w)
				return
			}

			user, err := users.FindActiveByID(r.Context(), userID)
			if errors.Is(err, db.ErrNotFound) {
				httpx.Unauthorized(w)
				return
			}
			if err != nil {
				httpx.Internal(w, r, err)
				return
			}
			if user.TokenVersion != tokenVersion {
				httpx.Unauthorized(w) // token dibuat sebelum logout
				return
			}

			next.ServeHTTP(w, r.WithContext(WithCurrentUser(r.Context(), user)))
		})
	}
}

// RequirePermission menolak (403) user yang role-nya tidak punya permission tertentu.
// Harus dipasang setelah Authenticate.
func RequirePermission(code string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := CurrentUserFrom(r.Context())
			if user == nil {
				httpx.Unauthorized(w)
				return
			}
			if !user.HasPermission(code) {
				httpx.Forbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// bearerToken mengambil token dari nilai header "Bearer <token>" (skema tidak peka huruf besar/kecil)
func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	if token == "" || strings.ContainsAny(token, " \t") {
		return "", false
	}
	return token, true
}
