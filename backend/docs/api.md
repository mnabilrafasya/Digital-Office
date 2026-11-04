# Perintah Back-end Digital Office

Jalankan dari folder `backend`, berurutan.

| No | Perintah | Fungsi |
|---|---|---|
| 1 | `docker compose up -d` | Menjalankan SQL Server, MinIO, dan Gotenberg |
| 2 | Simpan snapshot HC ke `data/hc/organization.json` | Data unit kerja PSP (didapat lewat jalur internal, jangan di-commit) |
| 3 | `go run ./cmd/migrate reset --yes` | Mengosongkan database lalu membuat ulang strukturnya |
| 4 | `go run ./cmd/seed all` | Mengisi data acuan, unit PSP dari snapshot HC, dan akun dummy |
| 5 | `go run ./cmd/api` | Menjalankan server di `http://localhost:8080` |

Perintah tambahan:

| Perintah | Fungsi |
|---|---|
| `go run ./cmd/sync-hc` | Memperbarui unit PSP dari snapshot HC |
| `go run ./cmd/sync-hc -api` | Memperbarui unit PSP langsung dari API HC (butuh VPN dan token di `.env`) |
| `go test ./...` | Menjalankan semua test |
