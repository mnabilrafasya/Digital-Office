// Alat migration digioffice. File migration di-embed ke binary (paket database).
// Jalankan dari folder backend:
//
//	go run ./cmd/migrate up            -> jalankan migration baru (development: buat database bila belum ada)
//	go run ./cmd/migrate down N        -> bongkar N migration terakhir
//	go run ./cmd/migrate version       -> tampilkan versi migration saat ini
//	go run ./cmd/migrate force V       -> tandai versi V sebagai bersih (hanya untuk memperbaiki status dirty)
//	go run ./cmd/migrate reset --yes   -> HANYA development: hapus database, buat ulang, lalu up
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlserver"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"office.backend/database"
	"office.backend/internal/platform/config"
	"office.backend/internal/platform/db"
)

// systemDatabases tidak boleh dihapus oleh reset.
var systemDatabases = []string{"master", "model", "msdb", "tempdb"}

// logger menampilkan setiap migration yang dijalankan, contoh: "9/u create_letters (25ms)"
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
	ctx := context.Background()

	switch cmd {
	case "reset":
		if err := resetDatabase(ctx, cfg); err != nil {
			log.Fatalf("Reset gagal: %v", err)
		}
		cmd = "up"
	case "up":
		if cfg.IsDevelopment() {
			if err := ensureDatabase(ctx, cfg); err != nil {
				log.Fatalf("Gagal menyiapkan database: %v", err)
			}
		}
	}

	m, closeMigrator := newMigrator(ctx, cfg)
	defer closeMigrator()

	switch cmd {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-argInt("down", "jumlah migration yang dibongkar, contoh: down 1"))
	case "force":
		err = m.Force(argInt("force", "nomor versi, contoh: force 16"))
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
		if strings.Contains(err.Error(), "no migration found for version") {
			log.Fatalf("Migration gagal: versi database tidak dikenal oleh file migration saat ini (kemungkinan masih skema lama).\n" +
				"Development: go run ./cmd/migrate reset --yes lalu go run ./cmd/seed all (seluruh data lokal terhapus).")
		}
		log.Fatalf("Migration gagal: %v", err)
	}
	printVersion(m)
}

func newMigrator(ctx context.Context, cfg *config.Config) (*migrate.Migrate, func()) {
	conn, err := sql.Open("sqlserver", db.BuildDSN(cfg, cfg.DBName))
	if err != nil {
		log.Fatalf("Gagal membuka koneksi: %v", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := conn.PingContext(pingCtx); err != nil {
		hint := "Pastikan Docker berjalan dan database sudah dibuat (development: go run ./cmd/migrate up; server perusahaan: dibuat DBA)."
		log.Fatalf("Gagal terhubung ke database %q: %v\n%s", cfg.DBName, err, hint)
	}

	driver, err := sqlserver.WithInstance(conn, &sqlserver.Config{})
	if err != nil {
		log.Fatalf("Gagal menyiapkan driver migration: %v", err)
	}
	source, err := iofs.New(database.Migrations, ".")
	if err != nil {
		log.Fatalf("Gagal membaca migration yang di-embed: %v", err)
	}
	m, err := migrate.NewWithInstance("iofs", source, "sqlserver", driver)
	if err != nil {
		log.Fatalf("Gagal menyiapkan migration: %v", err)
	}
	m.Log = logger{}
	return m, func() { _, _ = m.Close() }
}

// openMaster membuka koneksi ke database master (untuk membuat atau menghapus database aplikasi).
func openMaster(cfg *config.Config) (*sql.DB, error) {
	return sql.Open("sqlserver", db.BuildDSN(cfg, "master"))
}

// ensureDatabase membuat database aplikasi bila belum ada (hanya development).
func ensureDatabase(ctx context.Context, cfg *config.Config) error {
	conn, err := openMaster(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var exists bool
	if err := conn.QueryRowContext(ctx,
		"SELECT CASE WHEN DB_ID(@name) IS NULL THEN 0 ELSE 1 END", sql.Named("name", cfg.DBName)).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	// QUOTENAME mencegah nama database disalahgunakan (SQL injection)
	if _, err := conn.ExecContext(ctx,
		"DECLARE @sql NVARCHAR(400) = N'CREATE DATABASE ' + QUOTENAME(@name); EXEC (@sql);",
		sql.Named("name", cfg.DBName)); err != nil {
		return err
	}
	fmt.Printf("Database %q dibuat.\n", cfg.DBName)
	return nil
}

// resetDatabase menghapus database aplikasi lalu membuatnya ulang. Hanya development, wajib --yes.
func resetDatabase(ctx context.Context, cfg *config.Config) error {
	if !cfg.IsDevelopment() {
		return fmt.Errorf("ditolak: reset hanya boleh saat APP_ENV=development (sekarang %q)", cfg.AppEnv)
	}
	if len(os.Args) < 3 || os.Args[2] != "--yes" {
		return errors.New("reset menghapus SELURUH data database. Ulangi dengan: go run ./cmd/migrate reset --yes")
	}
	if slices.Contains(systemDatabases, strings.ToLower(cfg.DBName)) {
		return fmt.Errorf("ditolak: database sistem %q tidak boleh dihapus", cfg.DBName)
	}

	conn, err := openMaster(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	const q = `
IF DB_ID(@name) IS NOT NULL
BEGIN
    DECLARE @sql NVARCHAR(600) =
        N'ALTER DATABASE ' + QUOTENAME(@name) + N' SET SINGLE_USER WITH ROLLBACK IMMEDIATE; ' +
        N'DROP DATABASE ' + QUOTENAME(@name) + N';';
    EXEC (@sql);
END`
	if _, err := conn.ExecContext(ctx, q, sql.Named("name", cfg.DBName)); err != nil {
		return err
	}
	fmt.Printf("Database %q dihapus.\n", cfg.DBName)
	return ensureDatabase(ctx, cfg)
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
  go run ./cmd/migrate up            jalankan migration baru (development: buat database bila belum ada)
  go run ./cmd/migrate down N        bongkar N migration terakhir
  go run ./cmd/migrate version       tampilkan versi saat ini
  go run ./cmd/migrate force V       tandai versi V sebagai bersih (perbaikan status dirty)
  go run ./cmd/migrate reset --yes   HANYA development: hapus database, buat ulang, lalu up`)
	os.Exit(1)
}
