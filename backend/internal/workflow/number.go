package workflow

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DefaultNumberFormat dipakai bila perusahaan belum mengatur letter_number_format.
const DefaultNumberFormat = "{SEQ}/{UNIT}/{COMP}/{MM_ROMAN}/{YYYY}"

// maxLetterNumberLength sama dengan ukuran kolom letters.letter_number.
const maxLetterNumberLength = 100

// ErrInvalidNumberFormat dikembalikan bila format nomor surat salah (token tidak dikenal,
// kurung kurawal tidak berpasangan) atau hasilnya terlalu panjang.
var ErrInvalidNumberFormat = errors.New("format nomor surat tidak valid")

var romanMonths = [...]string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}

// NumberParts adalah bahan pembentuk nomor surat.
type NumberParts struct {
	Seq         int
	UnitCode    string
	CompanyCode string
	Month       time.Month
	Year        int
}

// FormatLetterNumber membentuk nomor surat dari format, contoh "{SEQ}/{UNIT}/{COMP}/{MM_ROMAN}/{YYYY}"
// menjadi "001/PA/PSP/X/2026". Token: {SEQ} (minimal 3 digit), {UNIT}, {COMP}, {MM_ROMAN}, {YYYY}.
func FormatLetterNumber(format string, p NumberParts) (string, error) {
	if format == "" {
		format = DefaultNumberFormat
	}
	if p.Seq < 1 || p.Month < time.January || p.Month > time.December || p.Year < 1 {
		return "", fmt.Errorf("%w: nomor urut, bulan, atau tahun tidak valid", ErrInvalidNumberFormat)
	}

	var b strings.Builder
	for i := 0; i < len(format); {
		switch format[i] {
		case '{':
			end := strings.IndexByte(format[i:], '}')
			if end < 0 {
				return "", fmt.Errorf("%w: kurung kurawal tidak ditutup", ErrInvalidNumberFormat)
			}
			token := format[i+1 : i+end]
			switch token {
			case "SEQ":
				b.WriteString(fmt.Sprintf("%03d", p.Seq))
			case "UNIT":
				b.WriteString(p.UnitCode)
			case "COMP":
				b.WriteString(p.CompanyCode)
			case "MM_ROMAN":
				b.WriteString(romanMonths[p.Month-1])
			case "YYYY":
				b.WriteString(strconv.Itoa(p.Year))
			default:
				return "", fmt.Errorf("%w: token {%s} tidak dikenal", ErrInvalidNumberFormat, token)
			}
			i += end + 1
		case '}':
			return "", fmt.Errorf("%w: kurung kurawal tutup tanpa pasangan", ErrInvalidNumberFormat)
		default:
			b.WriteByte(format[i])
			i++
		}
	}

	if b.Len() > maxLetterNumberLength {
		return "", fmt.Errorf("%w: hasil lebih dari %d karakter", ErrInvalidNumberFormat, maxLetterNumberLength)
	}
	return b.String(), nil
}
