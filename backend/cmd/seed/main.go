// Pengisi data digioffice. File seed di-embed ke binary (paket database).
// Jalankan dari folder backend:
//
//	go run ./cmd/seed reference   -> data acuan, idempoten, aman di semua environment
//	go run ./cmd/seed dev         -> data dummy, hanya APP_ENV=development (butuh reference)
//	go run ./cmd/seed all         -> reference lalu dev (development)
//
// Setiap set dijalankan dalam satu transaksi; file diurutkan menurut nama.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sort"
	"time"

	"github.com/mnabilrafasya/Digital-Office/backend/database"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/auth"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/config"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/db"
)

const minSeedPasswordLength = 8

// summaryTables adalah tabel yang jumlah barisnya dicetak di akhir.
var summaryTables = []string{"companies", "roles", "permissions", "lookups", "units", "positions", "users"}

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		usage()
	}
	cmd := os.Args[1]
	if cmd != "reference" && cmd != "dev" && cmd != "all" {
		usage()
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Gagal membaca .env: %v", err)
	}
	if (cmd == "dev" || cmd == "all") && !cfg.IsDevelopment() {
		log.Fatalf("Ditolak: data dummy hanya boleh diisi saat APP_ENV=development (sekarang %q).", cfg.AppEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	conn, err := db.Open(ctx, cfg)
	if err != nil {
		log.Fatalf("Gagal terhubung ke database: %v", err)
	}
	defer conn.Close()

	if cmd == "reference" || cmd == "all" {
		if err := runSet(ctx, conn, "reference", database.ReferenceSeeds); err != nil {
			log.Fatalf("Seed reference gagal, semua perubahan dibatalkan: %v", err)
		}
		fmt.Println("Seed reference selesai.")
	}
	if cmd == "dev" || cmd == "all" {
		if err := runDev(ctx, conn); err != nil {
			log.Fatalf("Seed dev gagal, semua perubahan dibatalkan: %v", err)
		}
	}
	printSummary(ctx, conn)
}

func runDev(ctx context.Context, conn *sql.DB) error {
	var users, levels int
	err := conn.QueryRowContext(ctx, `
SELECT (SELECT COUNT(*) FROM users),
       (SELECT COUNT(*) FROM lookups WHERE category = 'POSITION_LEVEL' AND deleted_at IS NULL)`).Scan(&users, &levels)
	if err != nil {
		return fmt.Errorf("memeriksa data (sudah menjalankan go run ./cmd/migrate up?): %w", err)
	}
	if users > 0 {
		fmt.Println("Seed dev dilewati: tabel users sudah berisi data.")
		fmt.Println("Untuk mengulang dari nol (development saja): go run ./cmd/migrate reset --yes, lalu go run ./cmd/seed all")
		return nil
	}
	if levels == 0 {
		return fmt.Errorf("data acuan belum ada, jalankan dulu: go run ./cmd/seed reference")
	}

	password := os.Getenv("SEED_PASSWORD")
	if len(password) < minSeedPasswordLength {
		return fmt.Errorf("SEED_PASSWORD di .env wajib diisi, minimal %d karakter", minSeedPasswordLength)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	if err := runSet(ctx, conn, "dev", database.DevSeeds, sql.Named("password_hash", hash)); err != nil {
		return err
	}
	fmt.Println("Seed dev selesai. Semua user dummy memakai password dari SEED_PASSWORD di .env.")
	return nil
}

// runSet menjalankan semua file .sql di fsys (urut nama) dalam satu transaksi.
func runSet(ctx context.Context, conn *sql.DB, name string, fsys fs.FS, args ...any) error {
	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)

	return db.WithTx(ctx, conn, func(tx db.DBTX) error {
		for _, f := range files {
			body, err := fs.ReadFile(fsys, f)
			if err != nil {
				return fmt.Errorf("membaca %s/%s: %w", name, f, err)
			}
			if _, err := tx.ExecContext(ctx, string(body), args...); err != nil {
				return fmt.Errorf("%s/%s: %w", name, f, err)
			}
			fmt.Printf("  %s/%s\n", name, f)
		}
		return nil
	})
}

func printSummary(ctx context.Context, conn *sql.DB) {
	fmt.Println("Jumlah baris:")
	for _, t := range summaryTables {
		var n int
		// Nama tabel berasal dari daftar konstanta di atas, bukan dari input
		if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+t).Scan(&n); err != nil {
			fmt.Printf("  %-12s (gagal dihitung: %v)\n", t, err)
			continue
		}
		fmt.Printf("  %-12s %d\n", t, n)
	}
}

func usage() {
	fmt.Println(`Pemakaian (dari folder backend):
  go run ./cmd/seed reference   data acuan (idempoten, aman di semua environment)
  go run ./cmd/seed dev         data dummy (hanya APP_ENV=development, butuh reference)
  go run ./cmd/seed all         reference lalu dev (development)`)
	os.Exit(1)
}
