package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/auth"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
)

const testSecret = "rahasia-uji-yang-panjangnya-lebih-dari-32-karakter"

// fakeUsers adalah pengganti repository user untuk tes
type fakeUsers map[int64]*model.CurrentUser

func (f fakeUsers) FindActiveByID(_ context.Context, id int64) (*model.CurrentUser, error) {
	if u, ok := f[id]; ok {
		return u, nil
	}
	return nil, repository.ErrNotFound
}

var andi = &model.CurrentUser{
	ID:           8,
	Name:         "Andi Kurniawan",
	Permissions:  []string{permission.LetterCreate, permission.LetterReview, permission.TemplateCreate},
	TokenVersion: 2,
}

// okHandler menulis nama user yang login, untuk memastikan context terisi
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(CurrentUser(r.Context()).Name))
})

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) errorResponse {
	t.Helper()
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body bukan JSON error yang valid: %v (%s)", err, rec.Body.String())
	}
	return body
}

func signToken(t *testing.T, secret string, sub string, exp time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub": sub,
		"iss": auth.Issuer,
		"iat": exp.Add(-time.Hour).Unix(),
		"exp": exp.Unix(),
		"tv":  andi.TokenVersion,
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("membuat token: %v", err)
	}
	return token
}

func TestAuthenticate(t *testing.T) {
	tokens := auth.NewJWT(testSecret, time.Hour)
	users := fakeUsers{8: andi}
	h := Authenticate(tokens, users)(okHandler)

	validToken, _ := tokens.Issue(8, andi.TokenVersion)
	loggedOutToken, _ := tokens.Issue(8, andi.TokenVersion-1)
	future := time.Now().Add(time.Hour)

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{"tanpa header", "", http.StatusUnauthorized},
		{"tanpa skema Bearer", validToken, http.StatusUnauthorized},
		{"skema Basic", "Basic " + validToken, http.StatusUnauthorized},
		{"Bearer tanpa token", "Bearer ", http.StatusUnauthorized},
		{"tanda tangan salah", "Bearer " + signToken(t, "secret-lain-yang-juga-panjangnya-lebih-dari-32", "8", future), http.StatusUnauthorized},
		{"kedaluwarsa", "Bearer " + signToken(t, testSecret, "8", time.Now().Add(-time.Minute)), http.StatusUnauthorized},
		{"user tidak aktif/terhapus", "Bearer " + signToken(t, testSecret, "99", future), http.StatusUnauthorized},
		{"token lama setelah logout", "Bearer " + loggedOutToken, http.StatusUnauthorized},
		{"token valid", "Bearer " + validToken, http.StatusOK},
		{"skema huruf kecil", "bearer " + validToken, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				if rec.Body.String() != andi.Name {
					t.Fatalf("CurrentUser tidak tersimpan di context, body = %q", rec.Body.String())
				}
				return
			}
			if body := decodeError(t, rec); body.Error.Code != "UNAUTHORIZED" {
				t.Fatalf("code = %q, want UNAUTHORIZED", body.Error.Code)
			}
		})
	}
}

func TestRequirePermission(t *testing.T) {
	serve := func(user *model.CurrentUser, code string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/candidates", nil)
		if user != nil {
			req = req.WithContext(WithCurrentUser(req.Context(), user))
		}
		rec := httptest.NewRecorder()
		RequirePermission(code)(okHandler).ServeHTTP(rec, req)
		return rec
	}

	t.Run("punya permission", func(t *testing.T) {
		if rec := serve(andi, permission.LetterCreate); rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("tidak punya permission", func(t *testing.T) {
		rec := serve(andi, permission.LetterApprove)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Fatalf("Content-Type = %q", ct)
		}
		body := decodeError(t, rec)
		if body.Error.Code != "FORBIDDEN" || body.Error.Message != "Anda tidak memiliki akses untuk tindakan ini" {
			t.Fatalf("error = %+v", body.Error)
		}
	})

	t.Run("belum login", func(t *testing.T) {
		if rec := serve(nil, permission.LetterCreate); rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}
