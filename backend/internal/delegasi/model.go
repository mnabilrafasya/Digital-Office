// Package delegasi mengelola jadwal delegasi tugas: selama periode tertentu tugas review/approval
// seorang pegawai dikerjakan penggantinya atas nama pegawai itu, dan surat yang masuk ke inbox
// pegawai itu ditembuskan ke penggantinya. Efeknya pada surat dan inbox ada di paket surat,
// inbox, dan audit (lewat tabel delegations); paket ini hanya mengisi dan membaca jadwalnya.
package delegasi

import "time"

// Status jadwal delegasi, dihitung dari tanggal hari ini (zona waktu aplikasi).
const (
	StatusActive   = "ACTIVE"   // sedang berlaku
	StatusUpcoming = "UPCOMING" // belum mulai
	StatusEnded    = "ENDED"    // sudah lewat
)

// PersonRef adalah ringkasan user di jadwal delegasi.
type PersonRef struct {
	ID           int64  `json:"id,string"`
	Name         string `json:"name"`
	PositionName string `json:"positionName"`
}

// Delegation adalah satu jadwal delegasi. From dan Until berformat YYYY-MM-DD (tanggal penuh).
type Delegation struct {
	ID        int64     `json:"id,string"`
	User      PersonRef `json:"user"`     // pegawai yang mendelegasikan
	Delegate  PersonRef `json:"delegate"` // pengganti
	From      string    `json:"from"`
	Until     string    `json:"until"`
	Note      *string   `json:"note"`
	Status    string    `json:"status"` // ACTIVE, UPCOMING, atau ENDED
	CreatedBy PersonRef `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	companyCode string
}

// Input adalah isi form delegasi apa adanya dari request. UserID hanya dipakai admin (pegawai
// mengatur delegasinya sendiri).
type Input struct {
	UserID       string  `json:"userId"`
	SubstituteID string  `json:"substituteId"`
	From         string  `json:"from"`
	Until        string  `json:"until"`
	Note         *string `json:"note"`
}

// Mine adalah delegasi milik user yang login (berlaku dan akan datang) beserta jadwal orang lain
// yang ia wakili (berlaku dan akan datang).
type Mine struct {
	Delegations  []Delegation `json:"delegations"`
	Representing []Delegation `json:"representing"`
}

// ListQuery adalah filter daftar delegasi perusahaan (admin), apa adanya dari query string.
type ListQuery struct {
	Status string // ACTIVE, UPCOMING, ENDED, atau "" (semua)
	UserID string // pegawai yang mendelegasikan atau penggantinya
	Offset int
	Limit  int
}

// period adalah isian delegasi yang sudah divalidasi.
type period struct {
	UserID     int64
	DelegateID int64
	From       string
	Until      string
	Note       *string
}
