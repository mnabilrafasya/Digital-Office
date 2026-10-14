package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlserver"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/microsoft/go-mssqldb"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/config"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/database"
)

const migrationsSource = "file://migrations"

// logger menampilkan setiap migration yang dijalankan
type logger struct{}

func (logger) Printf(format string, v ...any) { fmt.Printf(format, v...) }
func (logger) Verbose() bool                  { return false }

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		usage()
	}
	cmd := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Gagal membaca .env: %v", err)
	}

	if cmd == "up" {
		if err := ensureDatabase(cfg); err != nil {
			log.Fatalf("Gagal menyiapkan database: %v", err)
		}
	}

	db, err := sql.Open("sqlserver", database.BuildDSN(cfg, cfg.DBName))
	if err != nil {
		log.Fatalf("Gagal membuka koneksi: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Gagal terhubung ke database %q: %v\nPastikan Docker berjalan dan database sudah dibuat (go run ./cmd/migrate up).", cfg.DBName, err)
	}

	driver, err := sqlserver.WithInstance(db, &sqlserver.Config{})
	if err != nil {
		log.Fatalf("Gagal menyiapkan driver migration: %v", err)
	}
	m, err := migrate.NewWithDatabaseInstance(migrationsSource, "sqlserver", driver)
	if err != nil {
		log.Fatalf("Gagal membaca folder migrations (jalankan dari folder backend): %v", err)
	}
	m.Log = logger{}

	switch cmd {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-argInt("down", "jumlah migration yang dibongkar, contoh: down 1"))
	case "force":
		err = m.Force(argInt("force", "nomor versi, contoh: force 7"))
	case "version":
		// cukup tampilkan versi di bawah
	default:
		usage()
	}

	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("Tidak ada perubahan, database sudah versi terbaru.")
		err = nil
	}
	if err != nil {
		log.Fatalf("Migration gagal: %v", err)
	}
	printVersion(m)
}

// ensureDatabase membuat database (misalnya digioffice) bila belum ada.
func ensureDatabase(cfg *config.Config) error {
	db, err := sql.Open("sqlserver", database.BuildDSN(cfg, "master"))
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var exists int
	if err := db.QueryRowContext(ctx,
		"SELECT CASE WHEN DB_ID(@p1) IS NULL THEN 0 ELSE 1 END", cfg.DBName).Scan(&exists); err != nil {
		return err
	}
	if exists == 1 {
		return nil
	}

	// QUOTENAME mencegah nama database disalahgunakan (SQL injection)
	if _, err := db.ExecContext(ctx,
		"DECLARE @sql NVARCHAR(400) = N'CREATE DATABASE ' + QUOTENAME(@p1); EXEC (@sql);", cfg.DBName); err != nil {
		return err
	}
	fmt.Printf("Database %q dibuat.\n", cfg.DBName)
	return nil
}

func argInt(cmd, hint string) int {
	if len(os.Args) < 3 {
		log.Fatalf("Perintah %q butuh angka: %s", cmd, hint)
	}
	n, err := strconv.Atoi(os.Args[2])
	if err != nil || n < 1 {
		log.Fatalf("Angka tidak valid untuk %q: %s", cmd, hint)
	}
	return n
}

func printVersion(m *migrate.Migrate) {
	v, dirty, err := m.Version()
	switch {
	case errors.Is(err, migrate.ErrNilVersion):
		fmt.Println("Versi database: belum ada migration yang dijalankan.")
	case err != nil:
		log.Fatalf("Gagal membaca versi: %v", err)
	case dirty:
		fmt.Printf("Versi database: %d (DIRTY: migration ini gagal di tengah jalan. Cek database, lalu jalankan 'force' ke versi terakhir yang benar-benar berhasil.)\n", v)
	default:
		fmt.Printf("Versi database: %d\n", v)
	}
}

func usage() {
	fmt.Println(`Pemakaian (dari folder backend):
  go run ./cmd/migrate up          buat database bila belum ada + jalankan migration baru
  go run ./cmd/migrate down N      bongkar N migration terakhir
  go run ./cmd/migrate version     tampilkan versi saat ini
  go run ./cmd/migrate force V     tandai versi V sebagai bersih (perbaikan status dirty)`)
	os.Exit(1)
}
