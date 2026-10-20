package model

// Lookup adalah satu pilihan dropdown (jenis surat, sifat, kecepatan, klasifikasi, status).
type Lookup struct {
	ID        int64   `json:"id,string"`
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Color     *string `json:"color"`
	Note      *string `json:"note"`
	SortOrder int     `json:"sortOrder"`
}
