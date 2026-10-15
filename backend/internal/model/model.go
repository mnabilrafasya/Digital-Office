// Package model berisi struct data yang dipakai bersama oleh repository, service, dan handler.
// Semua ID numerik dikirim ke klien sebagai string (tag `json:",string"`).
package model

import "slices"

// CurrentUser adalah user yang sedang login beserta jabatan, unit, perusahaan, role, dan permission-nya
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
}

type UserPosition struct {
	ID         int64  `json:"id,string"`
	Name       string `json:"name"`
	IsUnitHead bool   `json:"isUnitHead"`
}

type UserUnit struct {
	ID   int64  `json:"id,string"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type UserCompany struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type UserRole struct {
	ID   int64  `json:"-"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// HasPermission mengecek apakah role user memiliki permission dengan kode tertentu
func (u *CurrentUser) HasPermission(code string) bool {
	return slices.Contains(u.Permissions, code)
}

// LoginRecord adalah data minimal untuk memeriksa login dengan NIK + password
type LoginRecord struct {
	UserID       int64
	PasswordHash *string
	IsActive     bool
	RoleIsActive bool
}

// Lookup adalah satu pilihan dropdown (jenis surat, sifat, kecepatan, klasifikasi, status)
type Lookup struct {
	ID        int64   `json:"id,string"`
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Color     *string `json:"color"`
	Note      *string `json:"note"`
	SortOrder int     `json:"sortOrder"`
}

// Unit adalah unit kerja beserta nama jabatan pimpinannya
type Unit struct {
	ID               int64   `json:"id,string"`
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	UnitType         string  `json:"unitType"`
	ParentID         *int64  `json:"parentId,string"`
	HeadPositionName *string `json:"headPositionName"`
}

// Candidate adalah user yang bisa dipilih sebagai reviewer atau approver
type Candidate struct {
	ID             int64  `json:"id,string"`
	Name           string `json:"name"`
	EmployeeNumber string `json:"employeeNumber"`
	PositionName   string `json:"positionName"`
	UnitName       string `json:"unitName"`
}
