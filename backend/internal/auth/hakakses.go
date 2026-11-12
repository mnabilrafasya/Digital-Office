package auth

import (
	"slices"
	"strings"
)

// Kode permission (hak akses) yang dipakai di seluruh aplikasi.
// Nilainya harus sama dengan kolom permissions.code di database.
//
// Hak akses user = permission role-nya + permission dari posisinya (level jabatan). Role hanya
// menentukan akses sistem (SUPERADMIN, ADMIN, PEGAWAI); kewenangan surat pejabat (approve dan
// disposisi) datang dari posisi untuk semua role, sehingga mutasi jabatan langsung mengubah haknya.
const (
	LetterCreate   = "LETTER_CREATE"
	LetterReview   = "LETTER_REVIEW"
	LetterApprove  = "LETTER_APPROVE"
	InboxDispose   = "INBOX_DISPOSE"
	TemplateCreate = "TEMPLATE_CREATE"
	AuditView      = "AUDIT_VIEW"

	CompanyManage   = "COMPANY_MANAGE"   // SUPERADMIN: kelola perusahaan
	CompanySettings = "COMPANY_SETTINGS" // ADMIN: watermark dan format nomor perusahaannya
	OrgManage       = "ORG_MANAGE"       // ADMIN: unit dan jabatan
	UserManage      = "USER_MANAGE"      // ADMIN: user
	TemplateManage  = "TEMPLATE_MANAGE"  // ADMIN: aktif/nonaktif template
)

// Kode role (kolom roles.code).
const (
	RoleSuperAdmin = "SUPERADMIN"
	RoleAdmin      = "ADMIN"
	RolePegawai    = "PEGAWAI"
)

// officialLevels adalah posisi (kode lookup POSITION_LEVEL) pejabat, sama untuk semua perusahaan.
var officialLevels = []string{"DIREKTUR", "SVP", "VP", "AVP"}

// positionPermissions adalah permission yang otomatis dimiliki pemegang posisi pejabat.
var positionPermissions = []string{LetterApprove, InboxDispose}

// OfficialLevels mengembalikan kode posisi pejabat.
func OfficialLevels() []string {
	return slices.Clone(officialLevels)
}

// IsOfficialLevel mengecek apakah kode posisi termasuk pejabat.
func IsOfficialLevel(levelCode string) bool {
	return slices.Contains(officialLevels, strings.ToUpper(levelCode))
}

// GrantedByPosition mengecek apakah permission diberikan oleh posisi pejabat.
func GrantedByPosition(code string) bool {
	return slices.Contains(positionPermissions, code)
}

// mergePermissions menggabungkan permission role dengan permission dari posisi, urut dan unik.
func mergePermissions(rolePerms []string, levelCode string) []string {
	out := slices.Clone(rolePerms)
	if IsOfficialLevel(levelCode) {
		out = append(out, positionPermissions...)
	}
	slices.Sort(out)
	out = slices.Compact(out)
	if out == nil {
		out = []string{}
	}
	return out
}
