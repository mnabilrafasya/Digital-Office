package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/inbox"
	"office.backend/internal/masterdata"
	"office.backend/internal/organisasi"
	"office.backend/internal/platform/httpx"
	"office.backend/internal/platform/middleware"
	"office.backend/internal/platform/server"
	"office.backend/internal/surat"
)

// routerDeps adalah komponen yang dibutuhkan untuk menyusun router.
type routerDeps struct {
	Health     *server.HealthHandler
	Auth       *auth.Handler
	MasterData *masterdata.Handler
	Organisasi *organisasi.Handler
	Surat      *surat.Handler
	Inbox      *inbox.Handler
	Audit      *audit.Handler

	Tokens auth.TokenVerifier
	Users  auth.UserLoader

	CORSOrigins       []string
	TrustProxyHeaders bool
}

// newRouter menyusun semua route aplikasi.
func newRouter(d routerDeps) http.Handler {
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
			r.Use(auth.Authenticate(d.Tokens, d.Users))

			r.Post("/auth/logout", d.Auth.Logout)
			r.Get("/auth/me", d.Auth.Me)
			r.Get("/lookups", d.MasterData.Lookups)
			r.Get("/units", d.Organisasi.Units)
			r.With(auth.RequirePermission(auth.LetterCreate)).
				Get("/users/candidates", d.Organisasi.Candidates)
			r.With(auth.RequirePermission(auth.LetterCreate)).
				Get("/recipients/aliases", d.Organisasi.Aliases)

			r.Route("/letters", func(r chi.Router) {
				r.Get("/", d.Surat.List)
				r.With(auth.RequirePermission(auth.LetterCreate)).Post("/", d.Surat.Create)

				// Akses per surat (drafter/peserta) dan permission peran diperiksa service
				r.Route("/{letterId}", func(r chi.Router) {
					r.Get("/", d.Surat.Get)
					r.Put("/", d.Surat.Update)
					r.Get("/history", d.Surat.History)
					r.Get("/qr", d.Surat.ApprovalQR)
					r.Get("/versions", d.Surat.Versions)
					r.Get("/document", d.Surat.Document)
					r.Put("/document", d.Surat.SaveDocument)
					r.Post("/attachments", d.Surat.AddAttachment)
					r.Get("/attachments/{attachmentId}", d.Surat.Attachment)
					r.Delete("/attachments/{attachmentId}", d.Surat.RemoveAttachment)
					r.Post("/submit", d.Surat.Submit)
					r.Post("/approve", d.Surat.Approve)
					r.Post("/return", d.Surat.Return)
					r.Get("/tracking", d.Audit.Tracking)
					r.With(auth.RequirePermission(auth.AuditView)).Get("/audit", d.Audit.LetterLogs)
				})
			})

			// Template naskah: semua user perusahaan boleh memakai; membuat dan mengganti butuh
			// TEMPLATE_CREATE (dimiliki semua role)
			r.Route("/templates", func(r chi.Router) {
				r.Get("/", d.Surat.Templates)
				r.With(auth.RequirePermission(auth.TemplateCreate)).Post("/", d.Surat.CreateTemplate)
				r.Get("/{templateId}/file", d.Surat.TemplateFile)
				r.With(auth.RequirePermission(auth.TemplateCreate)).Put("/{templateId}", d.Surat.UpdateTemplate)
			})

			// Audit log dan pantauan surat hanya untuk pemegang AUDIT_VIEW
			r.Group(func(r chi.Router) {
				r.Use(auth.RequirePermission(auth.AuditView))
				r.Get("/audit-logs", d.Audit.Logs)
				r.Get("/audit-logs/export", d.Audit.Export)
				r.Get("/audit-logs/{logId}", d.Audit.Log)
				r.Get("/monitoring/letters", d.Audit.Monitor)
			})

			// Inbox selalu milik user yang login. Disposisi butuh INBOX_DISPOSE (diperiksa service,
			// supaya item milik orang lain tetap dijawab 404, bukan 403).
			r.Route("/inbox", func(r chi.Router) {
				r.Get("/", d.Inbox.List)
				r.Route("/{itemId}", func(r chi.Router) {
					r.Get("/", d.Inbox.Get)
					r.Post("/disposition", d.Inbox.Dispose)
					r.Post("/redirect", d.Inbox.Redirect)
				})
			})
		})
	})
	return r
}
