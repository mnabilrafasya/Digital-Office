package organisasi

import "time"

// Master data organisasi yang dikelola admin perusahaan (ORG_MANAGE untuk unit dan jabatan,
// USER_MANAGE untuk user), selalu di perusahaan pengguna itu sendiri. SUPERADMIN memiliki seluruh
// akses sistem dan dapat menambah admin perusahaan lain (kelola_user.go). Tidak ada hapus permanen: surat dan audit
// lama merujuk unit, jabatan, dan user, jadi semuanya cukup dinonaktifkan.

// AdminUnit adalah unit kerja untuk halaman master data.
type AdminUnit struct {
	ID              int64   `json:"id,string"`
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	UnitType        string  `json:"unitType"`
	ParentID        *int64  `json:"parentId,string"`
	ParentName      *string `json:"parentName"`
	IsActive        bool    `json:"isActive"`
	HeadPosition    *string `json:"headPosition"` // jabatan pimpinan unit (aktif)
	HeadName        *string `json:"headName"`     // pemegang jabatan pimpinan unit
	ActivePositions int     `json:"activePositions"`
	ActiveUsers     int     `json:"activeUsers"`
	ActiveChildren  int     `json:"activeChildren"`
}

// UnitInput adalah isi form unit.
type UnitInput struct {
	Name     string  `json:"name"`
	UnitType string  `json:"unitType"`
	ParentID *string `json:"parentId"` // nil atau "" = unit teratas
}

// LevelRef adalah posisi (lookup POSITION_LEVEL), contoh VP.
type LevelRef struct {
	ID   int64  `json:"id,string"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// AdminPosition adalah jabatan untuk halaman master data.
type AdminPosition struct {
	ID          int64     `json:"id,string"`
	Name        string    `json:"name"`
	UnitID      int64     `json:"unitId,string"`
	UnitName    string    `json:"unitName"`
	Level       *LevelRef `json:"level"` // posisi; DIREKTUR/SVP/VP/AVP = pejabat (approve & disposisi)
	IsOfficial  bool      `json:"isOfficial"`
	IsUnitHead  bool      `json:"isUnitHead"`
	IsActive    bool      `json:"isActive"`
	ActiveUsers int       `json:"activeUsers"`
	Holders     []string  `json:"holders"` // nama pemegang aktif
}

// PositionInput adalah isi form jabatan.
type PositionInput struct {
	UnitID     string `json:"unitId"`
	Name       string `json:"name"`
	LevelID    string `json:"levelId"`
	IsUnitHead bool   `json:"isUnitHead"`
}

// RoleRef adalah role user.
type RoleRef struct {
	ID   int64  `json:"id,string"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// AdminUser adalah user untuk halaman master data. Password tidak pernah dikirim.
type AdminUser struct {
	ID             int64        `json:"id,string"`
	EmployeeNumber string       `json:"employeeNumber"`
	Name           string       `json:"name"`
	Email          string       `json:"email"`
	IsActive       bool         `json:"isActive"`
	HasPassword    bool         `json:"hasPassword"`
	Role           RoleRef      `json:"role"`
	Position       UserPosition `json:"position"`
	Unit           UserUnit     `json:"unit"`
	CreatedAt      time.Time    `json:"createdAt"`
	UpdatedAt      time.Time    `json:"updatedAt"`
}

// UserPosition adalah jabatan user beserta posisinya.
type UserPosition struct {
	ID         int64     `json:"id,string"`
	Name       string    `json:"name"`
	Level      *LevelRef `json:"level"`
	IsOfficial bool      `json:"isOfficial"`
	IsUnitHead bool      `json:"isUnitHead"`
}

// UserUnit adalah unit user.
type UserUnit struct {
	ID   int64  `json:"id,string"`
	Name string `json:"name"`
}

// UserInput adalah isi form user. Password hanya dipakai saat tambah user.
type UserInput struct {
	EmployeeNumber string `json:"employeeNumber"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	PositionID     string `json:"positionId"`
	RoleID         string `json:"roleId"`
	Password       string `json:"password"`
}

// PasswordInput adalah isi reset password.
type PasswordInput struct {
	Password string `json:"password"`
}

// StatusInput adalah isi permintaan aktif/nonaktif.
type StatusInput struct {
	Active *bool `json:"active"`
}

// CompanyAdminInput adalah isi form SUPERADMIN untuk menambah admin sebuah perusahaan. Unit:
// UnitID (unit yang sudah ada) atau UnitName (unit baru, untuk perusahaan yang belum punya unit).
type CompanyAdminInput struct {
	UnitID         *string `json:"unitId"`
	UnitName       *string `json:"unitName"`
	PositionName   string  `json:"positionName"`
	EmployeeNumber string  `json:"employeeNumber"`
	Name           string  `json:"name"`
	Email          string  `json:"email"`
	Password       string  `json:"password"`
}

// ListQuery adalah filter daftar master data, apa adanya dari query string.
type ListQuery struct {
	Search string
	Status string // "" | ACTIVE | INACTIVE
	UnitID string
	RoleID string
	Offset int
	Limit  int
}

// adminSnapshot adalah isi audit_logs.snapshot untuk perubahan master data.
type adminSnapshot struct {
	Before any `json:"before,omitempty"`
	After  any `json:"after,omitempty"`
}
