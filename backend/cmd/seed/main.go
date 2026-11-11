// Pengisi data digioffice. File seed di-embed ke binary (paket database).
// Jalankan dari folder backend:
//
//	go run ./cmd/seed reference   -> data acuan, idempoten, aman di semua environment
//	go run ./cmd/seed dev         -> data dummy, hanya APP_ENV=development (butuh reference;
//	                                 unit PSP diambil dari snapshot HC bila belum ada). Bila
//	                                 database sudah berisi user, hanya seed lanjutan yang idempoten
//	                                 (topUpDevSeeds) yang dijalankan.
//	go run ./cmd/seed all         -> reference, dev, lalu templates (development)
//	go run ./cmd/seed templates   -> template naskah bawaan ke MinIO, idempoten (butuh user admin)
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

	"golang.org/x/crypto/bcrypt"

	"office.backend/database"
	"office.backend/internal/organisasi"
	"office.backend/internal/platform/config"
	"office.backend/internal/platform/db"
)

const (
	minSeedPasswordLength = 8
	bcryptCost            = 12 // sama dengan cost hash palsu di internal/auth
)

// topUpDevSeeds adalah seed dev yang idempoten (hanya menambah yang belum ada), sehingga juga
// dijalankan pada database yang sudah berisi data dev.
var topUpDevSeeds = []string{"005_pkg_alur_lengkap.sql"}

// summaryTables adalah tabel yang jumlah barisnya dicetak di akhir (daftar tetap, bukan input).
var summaryTables = []string{"companies", "roles", "permissions", "lookups", "units", "positions", "users", "letter_templates"}

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		usage()
	}
	cmd := os.Args[1]
	if cmd != "reference" && cmd != "dev" && cmd != "all" && cmd != "templates" {
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
	if cmd == "all" || cmd == "templates" {
		if err := seedTemplates(ctx, conn, cfg); err != nil {
			log.Fatalf("Seed template gagal: %v", err)
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
	if levels == 0 {
		return fmt.Errorf("data acuan belum ada, jalankan dulu: go run ./cmd/seed reference")
	}
	password := os.Getenv("SEED_PASSWORD")
	if len(password) < minSeedPasswordLength {
		return fmt.Errorf("SEED_PASSWORD di .env wajib diisi, minimal %d karakter", minSeedPasswordLength)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return fmt.Errorf("membuat hash password: %w", err)
	}
	hashArg := sql.Named("password_hash", string(hash))

	if users > 0 {
		fmt.Println("Data dev dasar sudah ada: hanya menambah data dev lanjutan yang belum ada.")
		fmt.Println("Untuk mengulang dari nol (development saja): go run ./cmd/migrate reset --yes, lalu go run ./cmd/seed all")
		if err := runFiles(ctx, conn, "dev", database.DevSeeds, topUpDevSeeds, hashArg); err != nil {
			return err
		}
		fmt.Println("Seed dev lanjutan selesai.")
		return nil
	}
	if err := syncUnitsIfEmpty(ctx, conn); err != nil {
		return err
	}
	if err := runSet(ctx, conn, "dev", database.DevSeeds, hashArg); err != nil {
		return err
	}
	fmt.Println("Seed dev selesai. Semua user dummy memakai password dari SEED_PASSWORD di .env.")
	return nil
}

// hcSnapshot adalah snapshot unit HC (folder data/ diabaikan git; dibagikan lewat jalur internal).
const hcSnapshot = "data/hc/organization.json"

// syncUnitsIfEmpty mengisi unit PSP dari snapshot HC bila belum ada, supaya seed dev (jabatan dan
// user dummy di unit asli) bisa langsung jalan. Sama dengan "go run ./cmd/sync-hc".
func syncUnitsIfEmpty(ctx context.Context, conn *sql.DB) error {
	var n int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM units WHERE company_code = 'PSP'").Scan(&n); err != nil {
		return fmt.Errorf("memeriksa unit PSP: %w", err)
	}
	if n > 0 {
		return nil
	}
	raw, err := os.ReadFile(hcSnapshot)
	if err != nil {
		return fmt.Errorf("unit PSP belum ada dan snapshot HC tidak ditemukan (%s); minta file lewat jalur internal atau jalankan go run ./cmd/sync-hc -api: %w", hcSnapshot, err)
	}
	units, err := organisasi.ParseHCUnits(raw)
	if err != nil {
		return err
	}
	var res organisasi.SyncResult
	err = db.WithTx(ctx, conn, func(tx db.DBTX) error {
		res, err = organisasi.NewRepository(tx).SyncHCUnits(ctx, "PSP", units)
		return err
	})
	if err != nil {
		return fmt.Errorf("sinkronisasi unit HC: %w", err)
	}
	fmt.Printf("  unit PSP dari snapshot HC: %d unit ditambah\n", res.Added)
	return nil
}

// runSet menjalankan semua file .sql di fsys (urut nama) dalam satu transaksi.
func runSet(ctx context.Context, conn *sql.DB, name string, fsys fs.FS, args ...any) error {
	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	return runFiles(ctx, conn, name, fsys, files, args...)
}

// runFiles menjalankan file .sql tertentu di fsys (urutan sesuai daftar) dalam satu transaksi.
func runFiles(ctx context.Context, conn *sql.DB, name string, fsys fs.FS, files []string, args ...any) error {
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
			fmt.Printf("  %-17s (gagal dihitung: %v)\n", t, err)
			continue
		}
		fmt.Printf("  %-17s %d\n", t, n)
	}
}

func usage() {
	fmt.Println(`Pemakaian (dari folder backend):
  go run ./cmd/seed reference   data acuan (idempoten, aman di semua environment)
  go run ./cmd/seed dev         data dummy (hanya APP_ENV=development, butuh reference)
  go run ./cmd/seed all         reference, dev, lalu templates (development)
  go run ./cmd/seed templates   template naskah bawaan ke MinIO (idempoten, butuh user admin)`)
	os.Exit(1)
}
