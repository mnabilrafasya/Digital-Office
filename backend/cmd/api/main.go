// Server API digioffice. File ini hanya merangkai komponen: config -> dependency -> server.Run.
package main

import (
	"context"
	"log"
	"runtime/debug"

	_ "time/tzdata" // data zona waktu ikut di binary, tidak bergantung pada OS

	"github.com/mnabilrafasya/Digital-Office/backend/internal/auth"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/config"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/handler"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/ratelimit"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/server"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/service"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/storage"
)

// deadlockAttempts adalah jumlah percobaan transaksi saat SQL Server memilihnya sebagai korban deadlock.
const deadlockAttempts = 3

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.ValidateAPI(); err != nil {
		return err
	}
	if cfg.MemoryLimitMB > 0 {
		debug.SetMemoryLimit(int64(cfg.MemoryLimitMB) << 20)
		log.Printf("Batas memori Go: %d MB", cfg.MemoryLimitMB)
	}

	ctx, stop := server.SignalContext(context.Background())
	defer stop()

	conn, err := db.Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Println("Terhubung ke SQL Server")

	files, err := storage.New(ctx, cfg)
	if err != nil {
		return err
	}
	capacity, err := storage.NewCapacity(cfg)
	if err != nil {
		return err
	}
	log.Println("Terhubung ke MinIO")

	passwords, err := auth.NewPasswordChecker()
	if err != nil {
		return err
	}
	limiter := ratelimit.NewLoginLimiter(nil)
	go limiter.RunJanitor(ctx, ratelimit.JanitorInterval)

	users := repository.NewUserRepository(conn)
	tokens := auth.NewJWT(cfg.JWTSecret, cfg.JWTTTL)
	repos := func(q db.DBTX) service.LetterRepos {
		return service.LetterRepos{
			Letters:      repository.NewLetterRepository(q),
			Participants: repository.NewParticipantRepository(q),
			Recipients:   repository.NewRecipientRepository(q),
			Actions:      repository.NewActionRepository(q),
			Versions:     repository.NewVersionRepository(q),
			Inbox:        repository.NewInboxRepository(q),
			Audit:        repository.NewAuditRepository(q),
			Numbering:    repository.NewNumberingRepository(q),
			Lookups:      repository.NewLookupRepository(q),
			Units:        repository.NewUnitRepository(q),
			Users:        repository.NewUserRepository(q),
		}
	}

	authService := service.NewAuthService(users, tokens, passwords, limiter)
	formService := service.NewFormService(repository.NewLookupRepository(conn), repository.NewUnitRepository(conn), users)
	letterService := service.NewLetterService(service.LetterDeps{
		Tx:       db.NewRunner(conn, deadlockAttempts),
		Repos:    repos,
		Read:     repos(conn),
		Files:    files,
		Capacity: capacity,
		Uploads:  auth.NewUploadTokens(cfg.JWTSecret, cfg.UploadURLTTL),
		Config: service.LetterConfig{
			Location:       cfg.Location,
			MaxUploadBytes: cfg.UploadMaxBytes(),
			UploadURLTTL:   cfg.UploadURLTTL,
			DownloadURLTTL: cfg.DownloadURLTTL,
		},
	})

	router := server.NewRouter(server.RouterDeps{
		Health:            handler.NewHealthHandler(conn, files),
		Auth:              handler.NewAuthHandler(authService, cfg.JWTTTL),
		Form:              handler.NewFormHandler(formService),
		Letters:           handler.NewLetterHandler(letterService),
		Tokens:            tokens,
		Users:             users,
		CORSOrigins:       cfg.CORSAllowedOrigins,
		TrustProxyHeaders: cfg.TrustProxyHeaders,
	})
	return server.Run(ctx, ":"+cfg.AppPort, router)
}
