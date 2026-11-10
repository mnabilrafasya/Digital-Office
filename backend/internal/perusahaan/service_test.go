package perusahaan

import (
	"strings"
	"testing"
)

func TestParseCompany(t *testing.T) {
	city := "  Gresik "
	got, err := parseCompany(CompanyInput{Code: " pkg ", Name: "  PT Petrokimia Gresik ", City: &city}, true)
	if err != nil || got.Code != "PKG" || got.Name != "PT Petrokimia Gresik" || got.City == nil || *got.City != "Gresik" {
		t.Errorf("parseCompany: %+v %v", got, err)
	}
	empty := " "
	if got, err := parseCompany(CompanyInput{Name: "PT A", City: &empty}, false); err != nil || got.City != nil || got.Code != "" {
		t.Errorf("kota kosong = NULL, kode diabaikan saat ubah: %+v %v", got, err)
	}
	long := strings.Repeat("a", 51)
	for _, in := range []CompanyInput{
		{Code: "P", Name: "PT A"}, {Code: "PS-P", Name: "PT A"}, {Code: strings.Repeat("A", 21), Name: "PT A"},
		{Code: "PSP", Name: ""}, {Code: "PSP", Name: strings.Repeat("a", 151)}, {Code: "PSP", Name: "PT A", City: &long},
	} {
		if _, err := parseCompany(in, true); err == nil {
			t.Errorf("%+v diterima", in)
		}
	}
}

func TestParseWatermark(t *testing.T) {
	if got, err := parseWatermark(nil); got != nil || err != nil {
		t.Errorf("nil: %v %v", got, err)
	}
	blank := "   "
	if got, err := parseWatermark(&blank); got != nil || err != nil {
		t.Errorf("kosong = tanpa watermark: %v %v", got, err)
	}
	text := "  PUPUK   SRIWIDJAJA "
	if got, err := parseWatermark(&text); err != nil || got == nil || *got != "PUPUK SRIWIDJAJA" {
		t.Errorf("dirapikan: %v %v", got, err)
	}
	long := strings.Repeat("A", 101)
	if _, err := parseWatermark(&long); err == nil {
		t.Error("watermark lebih dari 100 karakter diterima")
	}
}
