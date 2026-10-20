package httpx

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
)

// ErrInvalidIfMatch dikembalikan bila header If-Match tidak berformat ETag yang dikenal.
var ErrInvalidIfMatch = errors.New("header If-Match tidak valid")

// FormatETag mengubah ROWVERSION (8 byte) menjadi ETag kuat, contoh "00000000000007D3".
func FormatETag(rowVersion []byte) string {
	return `"` + strings.ToUpper(hex.EncodeToString(rowVersion)) + `"`
}

// SetETag menulis header ETag dari ROWVERSION.
func SetETag(w http.ResponseWriter, rowVersion []byte) {
	if len(rowVersion) > 0 {
		w.Header().Set("ETag", FormatETag(rowVersion))
	}
}

// ParseIfMatch membaca header If-Match menjadi ROWVERSION. present=false berarti header tidak
// dikirim. Hanya satu ETag kuat yang diterima (bukan "*", bukan W/"...", bukan daftar), karena
// klien wajib menyebut versi yang ia edit.
func ParseIfMatch(header string) (rowVersion []byte, present bool, err error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return nil, false, nil
	}
	if len(header) < 3 || header[0] != '"' || header[len(header)-1] != '"' {
		return nil, true, ErrInvalidIfMatch
	}
	v, err := hex.DecodeString(header[1 : len(header)-1])
	if err != nil || len(v) != 8 {
		return nil, true, ErrInvalidIfMatch
	}
	return v, true, nil
}
