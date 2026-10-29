// Server API digioffice. Package ini hanya merangkai komponen: config -> dependency -> router
// (router.go) -> server.Run.
package main

import (
	"context"
	"log"
	"runtime/debug"

	_ "time/tzdata" // data zona waktu ikut di binary, tidak bergantung pada OS

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/inbox"
	"office.backend/internal/masterdata"
	"office.backend/internal/organisasi"
	"office.backend/internal/platform/config"
	"office.backend/internal/platform/db"
	"office.backend/internal/platform/pdf"
	"office.backend/internal/platform/server"
	"office.backend/internal/platform/storage"
	"office.backend/internal/surat"
)

// deadlockAttempts adalah jumlah percobaan maksimal transaksi yang terkena deadlock SQL Server.
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
	log.Println("Terhubung ke MinIO")

	passwords, err := auth.NewPasswordChecker()
	if err != nil {
		return err
	}
	limiter := auth.NewLoginLimiter(nil)
	go limiter.RunJanitor(ctx, auth.JanitorInterval) // berhenti saat server shutdown

	// Repository -> service -> handler, per fitur
	users := auth.NewRepository(conn)
	tokens := auth.NewJWT(cfg.JWTSecret, cfg.JWTTTL)
	authService := auth.NewService(users, tokens, passwords, limiter)

	masterRepo := masterdata.NewRepository(conn)
	orgRepo := organisasi.NewRepository(conn)

	// Surat memakai fitur lain lewat interface miliknya sendiri (Directory, Lookups, Templates, Inbox)
	suratService := surat.NewService(db.NewRunner(conn, deadlockAttempts), conn, surat.Repos{
		Letters:   func(q db.DBTX) surat.Store { return surat.NewRepository(q) },
		Audit:     func(q db.DBTX) surat.AuditStore { return audit.NewRepository(q) },
		Inbox:     func(q db.DBTX) surat.InboxStore { return inbox.NewRepository(q) },
		Directory: orgRepo,
		Lookups:   masterRepo,
		Templates: surat.NewRepository(conn),
		Files:     files,
		PDF:       pdf.NewMaker(cfg.GotenbergURL),
	}, cfg.Location)

	inboxService := inbox.NewService(db.NewRunner(conn, deadlockAttempts), conn, inbox.Repos{
		Items:     func(q db.DBTX) inbox.Store { return inbox.NewRepository(q) },
		Audit:     func(q db.DBTX) inbox.AuditStore { return audit.NewRepository(q) },
		Directory: orgRepo,
	}, cfg.Location)

	router := newRouter(routerDeps{
		Health:            server.NewHealthHandler(conn, files),
		Auth:              auth.NewHandler(authService, cfg.JWTTTL),
		MasterData:        masterdata.NewHandler(masterdata.NewService(masterRepo)),
		Organisasi:        organisasi.NewHandler(organisasi.NewService(orgRepo, orgRepo, orgRepo)),
		Surat:             surat.NewHandler(suratService),
		Inbox:             inbox.NewHandler(inboxService),
		Tokens:            tokens,
		Users:             users,
		CORSOrigins:       cfg.CORSAllowedOrigins,
		TrustProxyHeaders: cfg.TrustProxyHeaders,
	})
	return server.Run(ctx, ":"+cfg.AppPort, router)
}
