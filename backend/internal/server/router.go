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
	Health  *handler.HealthHandler
	Auth    *handler.AuthHandler
	Form    *handler.FormHandler
	Letters *handler.LetterHandler

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
		ExposedHeaders:   []string{"ETag", "Location", "Retry-After"},
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
			canCreate := middleware.RequirePermission(permission.LetterCreate)

			r.Post("/auth/logout", d.Auth.Logout)
			r.Get("/auth/me", d.Auth.Me)
			r.Get("/lookups", d.Form.Lookups)
			r.Get("/units", d.Form.Units)
			r.Get("/recipients/aliases", d.Form.Aliases)
			r.With(canCreate).Get("/users/candidates", d.Form.Candidates)

			r.With(canCreate).Post("/letters", d.Letters.Create)
			r.Get("/letters", d.Letters.List)
			r.Route("/letters/{letterId}", func(r chi.Router) {
				r.Get("/", d.Letters.Get)
				r.Put("/", d.Letters.Update)
				r.Delete("/", d.Letters.Delete)
				r.Get("/history", d.Letters.History)

				r.Post("/submit", d.Letters.Submit)
				r.Post("/approve", d.Letters.Approve)
				r.Post("/return", d.Letters.Return)
				r.Post("/sign", d.Letters.Sign)

				r.Post("/attachments/upload-url", d.Letters.AttachmentUploadURL)
				r.Post("/attachments", d.Letters.CompleteAttachment)
				r.Delete("/attachments/{attachmentId}", d.Letters.DeleteAttachment)
				r.Get("/attachments/{attachmentId}/download-url", d.Letters.AttachmentDownload)

				r.Post("/versions/upload-url", d.Letters.VersionUploadURL)
				r.Post("/versions", d.Letters.CompleteVersion)
				r.Get("/versions", d.Letters.Versions)
				r.Get("/versions/{versionNumber}/download-url", d.Letters.VersionDownload)
			})
		})
	})
	return r
}
