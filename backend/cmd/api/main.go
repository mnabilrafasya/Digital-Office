package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/config"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/database"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/handler"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
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

	health := &handler.HealthHandler{DB: db, Minio: minioClient}

	r := chi.NewRouter()
	r.Use(middleware.RequestID) // ID unik per request, memudahkan melacak log
	r.Use(middleware.Logger)    // mencatat setiap request di terminal
	r.Use(middleware.Recoverer) // mencegah server mati kalau ada error tak terduga

	r.Get("/health", health.Check)

	addr := ":" + cfg.AppPort
	log.Printf("Server berjalan di http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}