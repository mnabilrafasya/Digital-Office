package workflow

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestFormatLetterNumber(t *testing.T) {
	base := NumberParts{Seq: 1, UnitCode: "PA", CompanyCode: "PSP", Month: time.October, Year: 2026}
	with := func(f func(p *NumberParts)) NumberParts { p := base; f(&p); return p }

	tests := []struct {
		name   string
		format string
		parts  NumberParts
		want   string
	}{
		{"format bawaan", DefaultNumberFormat, base, "001/PA/PSP/X/2026"},
		{"format kosong memakai bawaan", "", base, "001/PA/PSP/X/2026"},
		{"padding dua digit", DefaultNumberFormat, with(func(p *NumberParts) { p.Seq = 42 }), "042/PA/PSP/X/2026"},
		{"lewat 999 tidak dipotong", DefaultNumberFormat, with(func(p *NumberParts) { p.Seq = 1234 }), "1234/PA/PSP/X/2026"},
		{"teks tetap dan urutan lain", "ND-{SEQ}/{COMP}.{UNIT}/{YYYY}", base, "ND-001/PSP.PA/2026"},
	}
	for _, tt := range tests {
		got, err := FormatLetterNumber(tt.format, tt.parts)
		if err != nil || got != tt.want {
			t.Errorf("%s: got %q, %v; want %q", tt.name, got, err, tt.want)
		}
	}
}

func TestFormatLetterNumberRomanMonths(t *testing.T) {
	want := []string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}
	for i, roman := range want {
		got, err := FormatLetterNumber("{MM_ROMAN}", NumberParts{Seq: 1, Month: time.Month(i + 1), Year: 2026})
		if err != nil || got != roman {
			t.Errorf("bulan %d: got %q, %v; want %q", i+1, got, err, roman)
		}
	}
}

func TestFormatLetterNumberInvalid(t *testing.T) {
	ok := NumberParts{Seq: 1, UnitCode: "PA", CompanyCode: "PSP", Month: time.March, Year: 2026}
	tests := []struct {
		name   string
		format string
		parts  NumberParts
	}{
		{"token tidak dikenal", "{SEQ}/{DEPT}/{YYYY}", ok},
		{"token huruf kecil", "{seq}/{YYYY}", ok},
		{"kurung tidak ditutup", "{SEQ/{YYYY}", ok},
		{"kurung tutup tanpa pasangan", "SEQ}/{YYYY}", ok},
		{"bulan 13", DefaultNumberFormat, NumberParts{Seq: 1, Month: 13, Year: 2026}},
		{"nomor urut 0", DefaultNumberFormat, NumberParts{Seq: 0, Month: time.March, Year: 2026}},
		{"hasil terlalu panjang", strings.Repeat("X", 99) + "{SEQ}", ok},
	}
	for _, tt := range tests {
		if _, err := FormatLetterNumber(tt.format, tt.parts); !errors.Is(err, ErrInvalidNumberFormat) {
			t.Errorf("%s: err = %v, want ErrInvalidNumberFormat", tt.name, err)
		}
	}
}
