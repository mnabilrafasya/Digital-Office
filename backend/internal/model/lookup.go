package model

// Kategori lookup (sama dengan CHECK ck_lookups_category).
const (
	CategoryLetterType      = "LETTER_TYPE"
	CategoryConfidentiality = "CONFIDENTIALITY"
	CategoryUrgency         = "URGENCY"
	CategoryClassification  = "CLASSIFICATION"
	CategoryLetterStatus    = "LETTER_STATUS"
	CategoryInboxStatus     = "INBOX_STATUS"
	CategoryPositionLevel   = "POSITION_LEVEL"
)

// Lookup adalah satu pilihan dropdown (jenis surat, sifat, kecepatan, klasifikasi, status).
type Lookup struct {
	ID        int64   `json:"id,string"`
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Color     *string `json:"color"`
	Note      *string `json:"note"`
	SortOrder int     `json:"sortOrder"`
}

// LookupRef adalah lookup untuk validasi referensi dan label tampilan.
// Hanya lookup global atau milik perusahaan user yang dikembalikan repository.
type LookupRef struct {
	ID       int64
	Category string
	Code     string
	Name     string
	IsActive bool
}

// LookupLabel adalah lookup yang ditampilkan di detail/daftar surat.
type LookupLabel struct {
	ID   int64  `json:"id,string"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// Alias adalah alias penerima berdasarkan level jabatan, contoh "Seluruh VP".
type Alias struct {
	LevelID     int64  `json:"levelId,string"`
	Code        string `json:"code"`
	Label       string `json:"label"`
	MemberCount int    `json:"memberCount"`
}
