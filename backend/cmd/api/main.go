package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/auth"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/config"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/database"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/handler"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/httpx"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/middleware"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/permission"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/service"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if err := cfg.ValidateJWT(); err != nil {
		log.Fatal(err)
	}

	db, err := database.NewSQLServer(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("Terhubung ke SQL Server")

	minioClient, err := storage.NewMinio(cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Terhubung ke MinIO")

	// Repository -> service -> handler
	userRepo := repository.NewUserRepository(db)
	lookupRepo := repository.NewLookupRepository(db)
	unitRepo := repository.NewUnitRepository(db)

	tokens := auth.NewJWT(cfg.JWTSecret, cfg.JWTTTL)
	authService := service.NewAuthService(userRepo, tokens)
	formService := service.NewFormService(lookupRepo, unitRepo, userRepo)

	health := &handler.HealthHandler{DB: db, Minio: minioClient}
	authHandler := handler.NewAuthHandler(authService, cfg.JWTTTL)
	formHandler := handler.NewFormHandler(formService)

	r := chi.NewRouter()
	r.Use(chimw.RequestID) // ID unik per request, memudahkan melacak log
	r.Use(chimw.RealIP)    // IP asli klien bila berada di belakang proxy
	r.Use(chimw.Logger)    // mencatat setiap request di terminal
	r.Use(chimw.Recoverer) // mencegah server mati kalau ada error tak terduga
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "If-Match"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Route tidak ada / method salah tetap dijawab dalam format JSON
	r.NotFound(httpx.NotFound)
	r.MethodNotAllowed(httpx.MethodNotAllowed)

	r.Get("/health", health.Check)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(tokens, userRepo))

			r.Get("/auth/me", authHandler.Me)
			r.Get("/lookups", formHandler.Lookups)
			r.Get("/units", formHandler.Units)
			r.With(middleware.RequirePermission(permission.LetterCreate)).
				Get("/users/candidates", formHandler.Candidates)
		})
	})

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("Server berjalan di http://localhost%s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
