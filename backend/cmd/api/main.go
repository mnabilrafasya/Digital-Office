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
	"office.backend/internal/perusahaan"
	"office.backend/internal/platform/config"
	"office.backend/internal/platform/db"
	"office.backend/internal/platform/pdf"
	"office.backend/internal/platform/server"
	"office.backend/internal/platform/storage"
	"office.backend/internal/platform/upload"
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
		Letters:        func(q db.DBTX) surat.Store { return surat.NewRepository(q) },
		Audit:          func(q db.DBTX) surat.AuditStore { return audit.NewRepository(q) },
		Inbox:          func(q db.DBTX) surat.InboxStore { return inbox.NewRepository(q) },
		TemplateWriter: func(q db.DBTX) surat.TemplateWriter { return surat.NewRepository(q) },
		Directory:      orgRepo,
		Lookups:        masterRepo,
		Templates:      surat.NewRepository(conn),
		Files:          files,
		PDF:            pdf.NewMaker(cfg.GotenbergURL),
	}, cfg.Location)

	inboxService := inbox.NewService(db.NewRunner(conn, deadlockAttempts), conn, inbox.Repos{
		Items:     func(q db.DBTX) inbox.Store { return inbox.NewRepository(q) },
		Audit:     func(q db.DBTX) inbox.AuditStore { return audit.NewRepository(q) },
		Directory: orgRepo,
	}, cfg.Location)

	// Master data: perusahaan (SUPERADMIN, pengaturan oleh ADMIN) dan unit/jabatan/user (ADMIN)
	companyService := perusahaan.NewService(db.NewRunner(conn, deadlockAttempts), conn, perusahaan.Repos{
		Companies: func(q db.DBTX) perusahaan.Store { return perusahaan.NewRepository(q) },
		Audit:     func(q db.DBTX) perusahaan.AuditStore { return audit.NewRepository(q) },
	}, cfg.Location)
	orgAdmin := organisasi.NewAdminService(db.NewRunner(conn, deadlockAttempts), conn,
		organisasi.NewRepository,
		func(q db.DBTX) organisasi.AuditStore { return audit.NewRepository(q) })

	router := newRouter(routerDeps{
		Health:      server.NewHealthHandler(conn, files),
		Auth:        auth.NewHandler(authService, cfg.JWTTTL),
		MasterData:  masterdata.NewHandler(masterdata.NewService(masterRepo)),
		LookupAdmin: masterdata.NewAdminHandler(masterdata.NewAdminService(db.NewRunner(conn, deadlockAttempts), conn)),
		Organisasi:  organisasi.NewHandler(organisasi.NewService(orgRepo, orgRepo, orgRepo)),
		OrgAdmin:    organisasi.NewAdminHandler(orgAdmin),
		Perusahaan:  perusahaan.NewHandler(companyService),
		Surat: surat.NewHandler(suratService, surat.UploadOptions{
			Limiter:     upload.NewLimiter(int64(cfg.UploadMemoryMB)<<20, cfg.UploadWait),
			ReadTimeout: cfg.UploadReadTimeout,
			Timeout:     cfg.UploadTimeout,
		}),
		Inbox:             inbox.NewHandler(inboxService),
		Audit:             audit.NewHandler(audit.NewService(conn, cfg.Location)),
		Tokens:            tokens,
		Users:             users,
		CORSOrigins:       cfg.CORSAllowedOrigins,
		TrustProxyHeaders: cfg.TrustProxyHeaders,
	})
	return server.Run(ctx, ":"+cfg.AppPort, router)
}
