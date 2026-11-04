// Sinkronisasi unit kerja PSP dari HC PUSRI ke database. Jalankan dari folder backend:
//
//	go run ./cmd/sync-hc        -> dari snapshot data/hc/organization.json
//	go run ./cmd/sync-hc -api   -> langsung dari API HC (PUSRI_HC_ORGANIZATION_URL + PUSRI_API_TOKEN di .env)
//
// Data hanya dibaca dari HC lalu ditulis ke database lokal. Snapshot (folder data/) dan token
// (.env) diabaikan git dan tidak boleh di-commit.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"office.backend/internal/organisasi"
	"office.backend/internal/platform/config"
	"office.backend/internal/platform/db"
)

const pageSize = 100

func main() {
	log.SetFlags(0)
	file := flag.String("file", "data/hc/organization.json", "snapshot unit HC (array JSON)")
	fromAPI := flag.Bool("api", false, "ambil langsung dari API HC")
	company := flag.String("company", "PSP", "kode perusahaan yang disinkronkan")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Gagal membaca .env: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var units []organisasi.HCUnit
	if *fromAPI {
		units, err = fetchAPI(ctx, cfg)
	} else {
		units, err = readSnapshot(*file)
	}
	if err != nil {
		log.Fatal(err)
	}

	conn, err := db.Open(ctx, cfg)
	if err != nil {
		log.Fatalf("Gagal terhubung ke database: %v", err)
	}
	defer conn.Close()

	var res organisasi.SyncResult
	err = db.NewRunner(conn, 1).InTx(ctx, func(q db.DBTX) error {
		res, err = organisasi.NewRepository(q).SyncHCUnits(ctx, *company, units)
		return err
	})
	if err != nil {
		log.Fatalf("Sinkronisasi dibatalkan, tidak ada perubahan: %v", err)
	}
	fmt.Printf("Unit %s dari HC: %d unit | ditambah %d, diperbarui %d, tetap %d, dinonaktifkan %d\n",
		*company, len(units), res.Added, res.Updated, res.Unchanged, res.Deactivated)
}

func readSnapshot(path string) ([]organisasi.HCUnit, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("snapshot HC tidak ditemukan (%s). Minta file lewat jalur internal atau pakai -api: %w", path, err)
	}
	return organisasi.ParseHCUnits(raw)
}

// fetchAPI mengambil semua halaman unit dari API HC. Token tidak pernah dicetak.
func fetchAPI(ctx context.Context, cfg *config.Config) ([]organisasi.HCUnit, error) {
	if cfg.HCOrganizationURL == "" || cfg.HCToken == "" {
		return nil, fmt.Errorf("PUSRI_HC_ORGANIZATION_URL dan PUSRI_API_TOKEN wajib diisi di .env untuk -api")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	var all []organisasi.HCUnit
	for page := 1; ; page++ {
		u, err := url.Parse(cfg.HCOrganizationURL)
		if err != nil {
			return nil, fmt.Errorf("PUSRI_HC_ORGANIZATION_URL tidak valid: %w", err)
		}
		q := u.Query()
		q.Set("page", fmt.Sprint(page))
		q.Set("limit", fmt.Sprint(pageSize))
		u.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+cfg.HCToken)
		req.Header.Set("Accept", "*/*")
		res, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("API HC tidak bisa dihubungi (perlu jaringan kantor/VPN?): %w", err)
		}
		body, err := io.ReadAll(io.LimitReader(res.Body, 20<<20))
		res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("membaca response API HC: %w", err)
		}
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API HC menjawab HTTP %d", res.StatusCode)
		}
		units, total, err := organisasi.DecodeHCPage(body)
		if err != nil {
			return nil, err
		}
		all = append(all, units...)
		if len(units) == 0 || len(all) >= total {
			return all, nil
		}
	}
}
