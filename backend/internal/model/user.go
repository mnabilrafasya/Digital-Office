// Package model berisi struct data yang dipakai bersama oleh repository, service, dan handler.
// Semua ID numerik dikirim ke klien sebagai string (tag `json:",string"`).
package model

import "slices"

// CurrentUser adalah user yang sedang login beserta jabatan, unit, perusahaan, role, dan permission-nya.
type CurrentUser struct {
	ID             int64        `json:"id,string"`
	EmployeeNumber string       `json:"employeeNumber"`
	Name           string       `json:"name"`
	Email          string       `json:"email"`
	Position       UserPosition `json:"position"`
	Unit           UserUnit     `json:"unit"`
	Company        UserCompany  `json:"company"`
	Role           UserRole     `json:"role"`
	Permissions    []string     `json:"permissions"`
	TokenVersion   int          `json:"-"`
}

// UserPosition adalah jabatan user.
type UserPosition struct {
	ID         int64          `json:"id,string"`
	Name       string         `json:"name"`
	IsUnitHead bool           `json:"isUnitHead"`
	Level      *PositionLevel `json:"level"`
}

// PositionLevel adalah level jabatan (lookup POSITION_LEVEL), contoh VP.
type PositionLevel struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// UserUnit adalah unit kerja user.
type UserUnit struct {
	ID   int64  `json:"id,string"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// UserCompany adalah perusahaan user.
type UserCompany struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// UserRole adalah role user. ID tidak dikirim ke klien.
type UserRole struct {
	ID   int64  `json:"-"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// HasPermission mengecek apakah role user memiliki permission dengan kode tertentu.
func (u *CurrentUser) HasPermission(code string) bool {
	return slices.Contains(u.Permissions, code)
}

// LoginRecord adalah data minimal untuk memeriksa login dengan NIK + password.
type LoginRecord struct {
	UserID       int64
	PasswordHash *string
	IsActive     bool
	RoleIsActive bool
}

// Candidate adalah user yang bisa dipilih sebagai reviewer atau approver.
type Candidate struct {
	ID             int64  `json:"id,string"`
	Name           string `json:"name"`
	EmployeeNumber string `json:"employeeNumber"`
	PositionName   string `json:"positionName"`
	UnitName       string `json:"unitName"`
}

// ParticipantUser adalah data user untuk memvalidasi reviewer/approver.
// Eligible berarti user, jabatan, unit, dan role-nya aktif serta satu perusahaan.
type ParticipantUser struct {
	ID         int64
	Name       string
	Eligible   bool
	CanReview  bool
	CanApprove bool
}

// UserLabel adalah ringkasan user untuk ditampilkan (nama, jabatan, unit).
type UserLabel struct {
	ID           int64  `json:"id,string"`
	Name         string `json:"name"`
	PositionName string `json:"positionName"`
	UnitName     string `json:"unitName,omitempty"`
}

// PersonRef adalah referensi user paling ringkas (id dan nama).
type PersonRef struct {
	ID   int64  `json:"id,string"`
	Name string `json:"name"`
}
