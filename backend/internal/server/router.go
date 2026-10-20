// Package server berisi daftar route dan http.Server dengan graceful shutdown.
package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/handler"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/httpx"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/middleware"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
)

// RouterDeps adalah komponen yang dibutuhkan untuk menyusun router.
type RouterDeps struct {
	Health *handler.HealthHandler
	Auth   *handler.AuthHandler
	Form   *handler.FormHandler
	Letter *handler.LetterHandler

	Tokens middleware.TokenVerifier
	Users  middleware.UserLoader

	CORSOrigins       []string
	TrustProxyHeaders bool
}

// NewRouter menyusun semua route aplikasi.
func NewRouter(d RouterDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)                          // ID unik per request, memudahkan melacak log
	r.Use(middleware.ClientIP(d.TrustProxyHeaders)) // IP klien (header proxy hanya bila dipercaya)
	r.Use(chimw.Logger)                             // mencatat setiap request di terminal
	r.Use(chimw.Recoverer)                          // mencegah server mati kalau ada error tak terduga
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   d.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "If-Match"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Route tidak ada / method salah tetap dijawab dalam format JSON
	r.NotFound(httpx.NotFound)
	r.MethodNotAllowed(httpx.MethodNotAllowed)

	r.Get("/health", d.Health.Check)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", d.Auth.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(d.Tokens, d.Users))

			r.Post("/auth/logout", d.Auth.Logout)
			r.Get("/auth/me", d.Auth.Me)
			r.Get("/lookups", d.Form.Lookups)
			r.Get("/units", d.Form.Units)
			r.With(middleware.RequirePermission(permission.LetterCreate)).
				Get("/users/candidates", d.Form.Candidates)
			r.With(middleware.RequirePermission(permission.LetterCreate)).
				Get("/recipients/aliases", d.Letter.Aliases)

			r.Route("/letters", func(r chi.Router) {
				r.With(middleware.RequirePermission(permission.LetterCreate)).Post("/", d.Letter.Create)
				// Akses per surat (drafter/peserta) diperiksa service, bukan lewat permission
				r.Get("/{letterId}", d.Letter.Get)
				r.Put("/{letterId}", d.Letter.Update)
			})
		})
	})
	return r
}
