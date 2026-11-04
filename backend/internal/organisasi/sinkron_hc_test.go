package organisasi

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNormalizeUnitName(t *testing.T) {
	tests := map[string]string{
		"Dep. Teknologi Informasi PSP":             "Departemen Teknologi Informasi PSP",
		"Dept. Pemeliharaan Listrik":               "Departemen Pemeliharaan Listrik",
		"Dep Rendal Pemeliharaan":                  "Departemen Perencanaan & Pengendalian Pemeliharaan",
		"DEPARTEMEN HUKUM":                         "Departemen Hukum",
		"DEP. PERENCANAAN & PENGENDALIAN PRODUKSI": "Departemen Perencanaan & Pengendalian Produksi",
		"Komp. Sekretaris Perusahaan":              "Kompartemen Sekretaris Perusahaan",
		"Bag. Pemantauan & Pelaporan LH":           "Bagian Pemantauan & Pelaporan Lingkungan Hidup",
		"Fungs. TK & Manrisk":                      "Fungsional Tata Kelola & Manajemen Risiko",
		"KEL. ELECTRONIC INSTRUMENT ENGINEERING":   "Kelompok Electronic Instrument Engineering",
		"Dep. Perenc,Penerimaan&Pergudangan":       "Departemen Perencanaan, Penerimaan & Pergudangan",
		"Bagian Rendal & Manajemen Proyek":         "Bagian Perencanaan, Pengendalian & Manajemen Proyek",
		"Dep. Komunikasi & Adm. Korporat":          "Departemen Komunikasi & Administrasi Korporat",
		"Dep. Admin Pemasaran & Penjualan":         "Departemen Administrasi Pemasaran & Penjualan",
		"Fungs. Tata Kelola & Manj Risiko":         "Fungsional Tata Kelola & Manajemen Risiko",
		"Dep. Sisman Terpadu & Inovasi":            "Departemen Sistem Manajemen Terpadu & Inovasi",
		// Istilah resmi dan awalan yang sudah benar dibiarkan
		"Fungs. Teknologi & K3LH":       "Fungsional Teknologi & K3LH",
		"Dep. HSSE P-IIIB":              "Departemen HSSE P-IIIB",
		"Direktorat Utama":              "Direktorat Utama",
		"Proyek Pusri-IIIB":             "Proyek Pusri-IIIB",
		"  Kantor  Perwakilan Jakarta ": "Kantor Perwakilan Jakarta",
	}
	for in, want := range tests {
		if got := NormalizeUnitName(in); got != want {
			t.Errorf("NormalizeUnitName(%q) = %q, want %q", in, got, want)
		}
	}
}

const (
	uidRoot  = "d734ccab-d1fb-46a4-9a2c-cfde0a787d39"
	uidChild = "f7ab142d-9a74-49e0-8e31-d10b10bbac24"
)

func TestParseAndDecodeHC(t *testing.T) {
	array := `[{"uid":"` + uidRoot + `","uid_parent":"#","text":"Direktorat Utama","level":1,"desc_level":"DIREKTORAT","status":"active"},` +
		`{"uid":"` + uidChild + `","uid_parent":"` + uidRoot + `","text":"Komp. Transformasi Bisnis","level":2,"desc_level":"KOMPARTEMEN","status":"in-active"}]`

	units, err := ParseHCUnits(append([]byte("\xEF\xBB\xBF"), array...)) // snapshot dari PowerShell diawali BOM
	if err != nil || len(units) != 2 || units[1].ParentUID != uidRoot || units[1].Status != "in-active" {
		t.Fatalf("ParseHCUnits: %+v %v", units, err)
	}

	page := `{"data":"` + base64.StdEncoding.EncodeToString([]byte(array)) + `","meta":{"limit":2,"page":1,"total":250}}`
	units, total, err := DecodeHCPage([]byte(page))
	if err != nil || len(units) != 2 || total != 250 || units[0].DescLevel != "DIREKTORAT" {
		t.Fatalf("DecodeHCPage: %+v total=%d %v", units, total, err)
	}

	for name, bad := range map[string]string{
		"uid dobel":       strings.Replace(array, uidChild, uidRoot, 1),
		"induk tidak ada": strings.Replace(array, `"uid_parent":"`+uidRoot, `"uid_parent":"00000000-0000-0000-0000-000000000000`, 1),
		"uid tidak valid": strings.Replace(array, uidRoot+`","uid_parent":"#"`, `abc","uid_parent":"#"`, 1),
		"bukan JSON":      "bukan json",
	} {
		if _, err := ParseHCUnits([]byte(bad)); err == nil {
			t.Errorf("%s: harus error", name)
		}
	}
}
