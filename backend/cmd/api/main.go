// Server API digioffice. File ini hanya merangkai komponen: config -> dependency -> server.Run.
package main

import (
	"context"
	"log"

	_ "time/tzdata" // data zona waktu ikut di binary, tidak bergantung pada OS

	"github.com/mnabilrafasya/Digital-Office/backend/internal/auth"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/config"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/handler"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/repository"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/server"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/service"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/storage"
)

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
	if err := cfg.ValidateJWT(); err != nil {
		return err
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
	log.Println("Terhubung ke MinIO")

	// Repository -> service -> handler
	users := repository.NewUserRepository(conn)
	tokens := auth.NewJWT(cfg.JWTSecret, cfg.JWTTTL)
	authService := service.NewAuthService(users, tokens)
	formService := service.NewFormService(repository.NewLookupRepository(conn), repository.NewUnitRepository(conn), users)

	router := server.NewRouter(server.RouterDeps{
		Health:      handler.NewHealthHandler(conn, files),
		Auth:        handler.NewAuthHandler(authService, cfg.JWTTTL),
		Form:        handler.NewFormHandler(formService),
		Tokens:      tokens,
		Users:       users,
		CORSOrigins: cfg.CORSAllowedOrigins,
	})
	return server.Run(ctx, ":"+cfg.AppPort, router)
}
