package organisasi

import (
	"errors"
	"strings"
	"testing"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
)

func TestUserGuard(t *testing.T) {
	admin := &auth.CurrentUser{ID: 1, Permissions: []string{auth.UserManage}}
	super := &auth.CurrentUser{ID: 2, Permissions: []string{auth.UserManage, auth.CompanyManage}}
	target := func(id int64, role string, active bool) *AdminUser {
		return &AdminUser{ID: id, IsActive: active, Role: RoleRef{Code: role}}
	}
	cases := []struct {
		name      string
		actor     *auth.CurrentUser
		target    *AdminUser
		newRole   string
		newActive bool
		others    int
		want      error
	}{
		{"admin menonaktifkan pegawai", admin, target(9, auth.RolePegawai, true), auth.RolePegawai, false, -1, nil},
		{"admin mengangkat admin lain", admin, target(9, auth.RolePegawai, true), auth.RoleAdmin, true, -1, nil},
		{"admin menonaktifkan diri sendiri", admin, target(1, auth.RoleAdmin, true), auth.RoleAdmin, false, 3, errSelfStatus},
		{"admin mengganti role sendiri", admin, target(1, auth.RoleAdmin, true), auth.RolePegawai, true, 3, errSelfRole},
		{"mencabut admin terakhir", admin, target(9, auth.RoleAdmin, true), auth.RolePegawai, true, 0, errLastAdmin},
		{"menonaktifkan admin terakhir", admin, target(9, auth.RoleAdmin, true), auth.RoleAdmin, false, 0, errLastAdmin},
		{"mencabut admin bila ada admin lain", admin, target(9, auth.RoleAdmin, true), auth.RolePegawai, true, 1, nil},
		{"admin memberi role SUPERADMIN", admin, target(9, auth.RolePegawai, true), auth.RoleSuperAdmin, true, -1, apperr.ErrForbidden},
		{"admin mengubah akun SUPERADMIN", admin, target(9, auth.RoleSuperAdmin, true), auth.RoleSuperAdmin, false, -1, apperr.ErrForbidden},
		{"SUPERADMIN memberi role SUPERADMIN", super, target(9, auth.RolePegawai, true), auth.RoleSuperAdmin, true, -1, nil},
	}
	for _, tc := range cases {
		if err := userGuard(tc.actor, tc.target, tc.newRole, tc.newActive, tc.others); !errors.Is(err, tc.want) {
			t.Errorf("%s: error = %v, want %v", tc.name, err, tc.want)
		}
	}
}

func TestParseUser(t *testing.T) {
	ok := UserInput{EmployeeNumber: " 2015047 ", Name: "  Andi   Kurniawan ", Email: " Andi@Pusri.test ", PositionID: "7", RoleID: "3"}
	rec, err := parseUser(ok)
	if err != nil {
		t.Fatal(err)
	}
	if rec.EmployeeNumber != "2015047" || rec.Name != "Andi Kurniawan" || rec.Email != "andi@pusri.test" ||
		rec.PositionID != 7 || rec.RoleID != 3 {
		t.Errorf("hasil parse salah: %+v", rec)
	}
	bad := []UserInput{
		{EmployeeNumber: "", Name: "A", Email: "a@b.c", PositionID: "1", RoleID: "1"},
		{EmployeeNumber: "20 15", Name: "A", Email: "a@b.c", PositionID: "1", RoleID: "1"},
		{EmployeeNumber: strings.Repeat("1", 31), Name: "A", Email: "a@b.c", PositionID: "1", RoleID: "1"},
		{EmployeeNumber: "1", Name: " ", Email: "a@b.c", PositionID: "1", RoleID: "1"},
		{EmployeeNumber: "1", Name: "A", Email: "bukan-email", PositionID: "1", RoleID: "1"},
		{EmployeeNumber: "1", Name: "A", Email: "Andi <a@b.c>", PositionID: "1", RoleID: "1"},
		{EmployeeNumber: "1", Name: "A", Email: "a@b.c", PositionID: "", RoleID: "1"},
		{EmployeeNumber: "1", Name: "A", Email: "a@b.c", PositionID: "1", RoleID: "x"},
	}
	for _, in := range bad {
		var v *apperr.ValidationError
		if _, err := parseUser(in); !errors.As(err, &v) {
			t.Errorf("%+v: error = %v, want validasi", in, err)
		}
	}
}

func TestParseUnitAndPosition(t *testing.T) {
	name, typ, parent, err := parseUnit(UnitInput{Name: " Bagian Umum ", UnitType: " bagian  umum ", ParentID: ptrStr("12")})
	if err != nil || name != "Bagian Umum" || typ != "BAGIAN UMUM" || parent == nil || *parent != 12 {
		t.Errorf("parseUnit: %q %q %v %v", name, typ, parent, err)
	}
	if _, _, parent, err := parseUnit(UnitInput{Name: "A", UnitType: "UNIT", ParentID: ptrStr(" ")}); err != nil || parent != nil {
		t.Errorf("induk kosong = unit teratas: %v %v", parent, err)
	}
	for _, in := range []UnitInput{{Name: "", UnitType: "UNIT"}, {Name: "A", UnitType: ""}, {Name: "A", UnitType: "DEP-TI"}, {Name: "A", UnitType: "UNIT", ParentID: ptrStr("x")}} {
		if _, _, _, err := parseUnit(in); err == nil {
			t.Errorf("%+v diterima", in)
		}
	}

	d, err := parsePosition(PositionInput{UnitID: "5", Name: " VP  Teknologi ", LevelID: "3", IsUnitHead: true})
	if err != nil || d.Name != "VP Teknologi" || d.UnitID != 5 || d.LevelID != 3 || !d.IsHead {
		t.Errorf("parsePosition: %+v %v", d, err)
	}
	for _, in := range []PositionInput{{UnitID: "5", Name: "", LevelID: "3"}, {UnitID: "", Name: "A", LevelID: "3"}, {UnitID: "5", Name: "A", LevelID: ""}} {
		if _, err := parsePosition(in); err == nil {
			t.Errorf("%+v diterima (posisi wajib)", in)
		}
	}
}

func ptrStr(s string) *string { return &s }
