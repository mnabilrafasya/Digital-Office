// Package database berisi aset SQL (migration dan seeder) yang di-embed ke binary, sehingga
// alat migrate dan seed tidak bergantung pada folder kerja saat dijalankan di pipeline.
package database

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

//go:embed seeds/reference/*.sql
var referenceFS embed.FS

//go:embed seeds/dev/*.sql
var devFS embed.FS

var (
	// Migrations berisi file NNNNNN_nama.up.sql / .down.sql untuk golang-migrate.
	Migrations = mustSub(migrationsFS, "migrations")
	// ReferenceSeeds berisi data acuan yang idempoten (MERGE) dan aman untuk semua environment.
	ReferenceSeeds = mustSub(referenceFS, "seeds/reference")
	// DevSeeds berisi data dummy, hanya untuk APP_ENV=development.
	DevSeeds = mustSub(devFS, "seeds/dev")
)

// mustSub hanya gagal bila nama folder salah ketik (kesalahan programmer, terdeteksi saat start).
func mustSub(f embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic("database: folder embed tidak valid: " + dir)
	}
	return sub
}
