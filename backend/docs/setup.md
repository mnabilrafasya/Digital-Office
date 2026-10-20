# Setup back-end digioffice

Panduan menyiapkan, menjalankan, dan men-deploy back-end. Kontrak API ada di [api.md](api.md).

## 1. Prasyarat

- Go sesuai versi di `go.mod`
- Docker Desktop (SQL Server 2022 dan MinIO AIStor untuk development)
- File lisensi MinIO `minio.license` di folder `backend` (minta ke pembimbing; **jangan** di-commit atau dibagikan)

## 2. Setup pertama kali (PowerShell, dari folder `backend`)

```powershell
copy .env.example .env      # lalu ganti semua nilai berawalan ganti_dengan_
docker compose up -d        # SQL Server dan MinIO
go run ./cmd/migrate up     # development: database dibuat otomatis bila belum ada
go run ./cmd/seed all       # data acuan + data dummy
go run ./cmd/api            # http://localhost:8080
```

`JWT_SECRET` dibuat acak, contoh: `$b = New-Object byte[] 48; [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($b); [Convert]::ToBase64String($b)`.

Bila kredensial MinIO atau `CORS_ALLOWED_ORIGINS` di `.env` diubah, buat ulang container MinIO dengan `docker compose up -d minio` (data volume tetap).

## 3. Migration dan seed

| Perintah | Fungsi |
|---|---|
| `go run ./cmd/migrate up` | Jalankan migration baru. Development: buat database bila belum ada |
| `go run ./cmd/migrate down N` | Bongkar N migration terakhir |
| `go run ./cmd/migrate version` | Lihat versi migration |
| `go run ./cmd/migrate force V` | Perbaiki status dirty (hanya bila migration gagal di tengah jalan) |
| `go run ./cmd/migrate reset --yes` | **Hanya development**: hapus database, buat ulang, lalu `up` |
| `go run ./cmd/seed reference` | Data acuan (perusahaan, role, permission, lookup global, level jabatan). Idempoten, aman di semua environment |
| `go run ./cmd/seed dev` | Data dummy (unit, jabatan, user, klasifikasi). Hanya development; dilewati bila tabel users sudah berisi |
| `go run ./cmd/seed all` | `reference` lalu `dev` |

File migration dan seed di-embed ke binary, jadi alat ini tidak bergantung pada folder kerja.

**Pindah dari skema Sprint 1:** database lama (versi 18) tidak kompatibel dengan baseline 000001-000016. Di komputer development jalankan `go run ./cmd/migrate reset --yes` lalu `go run ./cmd/seed all` (seluruh data lokal terhapus).

## 4. Struktur folder

| Path | Fungsi |
|---|---|
| `cmd/api` | Server API (hanya merangkai komponen) |
| `cmd/migrate`, `cmd/seed` | Alat migration dan seed |
| `database/migrations` | Migration baseline (satu tabel satu file, dalam transaksi) |
| `database/seeds/reference`, `database/seeds/dev` | Seed acuan (MERGE, idempoten) dan seed dummy |
| `internal/config` | Membaca `.env` |
| `internal/db` | Koneksi, transaksi (`WithTx`), pengulangan otomatis saat deadlock |
| `internal/server` | Daftar route, http.Server, graceful shutdown |
| `internal/handler` | Baca request, panggil service, tulis response (tanpa SQL) |
| `internal/service` | Aturan bisnis (tanpa HTTP) |
| `internal/workflow` | Mesin status surat dan format nomor surat (fungsi murni) |
| `internal/repository` | Query SQL (tanpa aturan bisnis) |
| `internal/storage` | MinIO: presigned URL, operasi object, cek kapasitas |
| `internal/auth` | Access token, upload token, bcrypt |
| `internal/middleware` | Autentikasi, permission, IP klien |
| `internal/ratelimit` | Pembatasan percobaan login |
| `internal/httpx`, `internal/model`, `internal/permission` | Helper HTTP, struct data, kode permission |
| `scripts/smoke-test.ps1` | Uji E2E alur surat |

## 5. Variabel `.env`

| Variabel | Default | Keterangan |
|---|---|---|
| `APP_ENV` | `production` | `development` hanya di komputer developer (mengizinkan seed dev, reset, dan sertifikat DB tanpa verifikasi) |
| `APP_PORT` | `8080` | Port API |
| `APP_TIMEZONE` | `Asia/Jakarta` | Zona waktu bisnis (nomor surat, tampilan waktu) |
| `APP_MEMORY_LIMIT_MB` | `0` | Batas memori Go (`debug.SetMemoryLimit`), 0 = nonaktif |
| `TRUST_PROXY_HEADERS` | `false` | `true` hanya di belakang reverse proxy yang menimpa `X-Forwarded-For`/`X-Real-IP` |
| `SEED_PASSWORD` | - | Password semua user dummy (rahasia, hanya development) |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_NAME` | `localhost`, `1433`, `sa`, `digioffice` | Koneksi SQL Server |
| `DB_PASSWORD` | - | Rahasia, wajib |
| `DB_TRUST_SERVER_CERTIFICATE` | `true` di development, `false` di lainnya | `false` = sertifikat TLS SQL Server diverifikasi |
| `MINIO_ENDPOINT`, `MINIO_USE_SSL` | `localhost:9000`, `false` | Alamat MinIO dari sisi server |
| `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD` | - | Rahasia, wajib |
| `MINIO_PUBLIC_ENDPOINT`, `MINIO_PUBLIC_USE_SSL` | sama dengan `MINIO_ENDPOINT` | Alamat MinIO yang dijangkau browser (dipakai di presigned URL) |
| `MINIO_REGION` | `us-east-1` | Region MinIO (presign tanpa panggilan jaringan) |
| `JWT_SECRET` | - | Rahasia, wajib untuk API, minimal 32 karakter, bukan nilai contoh |
| `JWT_TTL` | `8h` | Masa berlaku access token |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | Origin front-end (dipisah koma); juga dipakai CORS MinIO |
| `UPLOAD_MAX_FILE_SIZE_MB` | `25` | Batas ukuran satu file (1 sampai 1024) |
| `UPLOAD_URL_TTL` | `15m` | Masa berlaku izin unggah |
| `DOWNLOAD_URL_TTL` | `5m` | Masa berlaku URL unduh |
| `STORAGE_MIN_FREE_MB` | `1024` | Sisa ruang storage minimal yang dijaga |

## 6. Catatan keamanan

- Jangan pernah commit `.env`, `minio.license`, `*.pem`, `*.key`. `.env.example` hanya berisi placeholder.
- Port SQL Server dan MinIO di `docker-compose.yml` hanya terbuka di `127.0.0.1`.
- Password disimpan sebagai bcrypt (cost 12); login gagal memakan waktu yang sama apa pun penyebabnya.
- Presigned URL, token, dan password tidak pernah ditulis ke log.
- Surat yang tidak boleh dilihat dijawab 404 (keberadaannya tidak dibocorkan).
- Semua query memakai parameter SQL; daftar ID dikirim sebagai JSON dan dibaca dengan `OPENJSON`.

## 7. Test

```powershell
go test ./...                                                  # unit test, tanpa Docker
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1   # E2E, API harus berjalan
```

Sebelum commit: `gofmt -l .` harus kosong, lalu `go vet ./...`, `go build ./...`, `go test ./...`.

## 8. Pipeline server perusahaan

Urutan deploy:

1. DBA membuat database (alat migrate **tidak** membuat database di luar development).
2. `migrate up`
3. `seed reference`
4. Jalankan API dengan `APP_ENV=production`, `DB_TRUST_SERVER_CERTIFICATE=false`, dan `TRUST_PROXY_HEADERS=true` bila di belakang reverse proxy.

**Jangan pernah** menjalankan `seed dev` atau `migrate reset` di server perusahaan (keduanya juga ditolak bila `APP_ENV` bukan `development`).
