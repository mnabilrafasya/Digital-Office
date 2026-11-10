// Package perusahaan berisi master data perusahaan:
//
//   - SUPERADMIN (COMPANY_MANAGE) menambah perusahaan, mengubah identitasnya (nama, kota), dan
//     mengaktifkan/menonaktifkannya. Perusahaan nonaktif: semua user-nya tidak bisa login.
//   - ADMIN (COMPANY_SETTINGS) mengatur perusahaannya sendiri: watermark naskah resmi (teks) dan
//     format nomor surat.
//
// Admin pertama perusahaan baru dibuat lewat package organisasi (kelola user).
package perusahaan

import "time"

// Company adalah satu perusahaan beserta ringkasan isinya (untuk SUPERADMIN).
type Company struct {
	Code               string    `json:"code"`
	Name               string    `json:"name"`
	City               *string   `json:"city"`
	LetterNumberFormat string    `json:"letterNumberFormat"`
	WatermarkText      *string   `json:"watermarkText"`
	IsActive           bool      `json:"isActive"`
	ActiveUnits        int       `json:"activeUnits"`
	ActiveUsers        int       `json:"activeUsers"`
	Letters            int       `json:"letters"`
	CreatedAt          time.Time `json:"createdAt"`
}

// Settings adalah pengaturan perusahaan yang diatur admin perusahaan.
type Settings struct {
	Code               string  `json:"code"`
	Name               string  `json:"name"`
	LetterNumberFormat string  `json:"letterNumberFormat"`
	NumberPreview      string  `json:"numberPreview"` // contoh nomor surat ke-1 bulan ini
	WatermarkText      *string `json:"watermarkText"` // nil = tanpa watermark
}

// CompanyInput adalah isi form tambah/ubah perusahaan (Code hanya dipakai saat tambah).
type CompanyInput struct {
	Code string  `json:"code"`
	Name string  `json:"name"`
	City *string `json:"city"`
}

// SettingsInput adalah isi form pengaturan perusahaan. WatermarkText nil atau kosong = tanpa
// watermark.
type SettingsInput struct {
	LetterNumberFormat string  `json:"letterNumberFormat"`
	WatermarkText      *string `json:"watermarkText"`
}

// StatusInput adalah isi permintaan aktif/nonaktif.
type StatusInput struct {
	Active *bool `json:"active"`
}

// CompanyQuery adalah filter daftar perusahaan, apa adanya dari query string.
type CompanyQuery struct {
	Search string
	Status string // "" | ACTIVE | INACTIVE
	Offset int
	Limit  int
}

// companySnapshot adalah isi audit_logs.snapshot untuk perubahan perusahaan.
type companySnapshot struct {
	Code   string         `json:"code"`
	Before map[string]any `json:"before,omitempty"`
	After  map[string]any `json:"after,omitempty"`
}
