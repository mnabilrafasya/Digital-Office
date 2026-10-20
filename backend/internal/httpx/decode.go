package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// MaxBodyBytes adalah batas ukuran body JSON (1 MB).
const MaxBodyBytes = 1 << 20

// DecodeJSON membaca body JSON (maksimal MaxBodyBytes) ke dst.
// Kalau gagal, response 400 sudah ditulis dan fungsi mengembalikan false.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	err := json.NewDecoder(r.Body).Decode(dst)
	if err == nil {
		return true
	}

	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		Validation(w, "Body request terlalu besar (maksimal 1 MB)")
	case errors.Is(err, io.EOF):
		Validation(w, "Body request kosong")
	default:
		Validation(w, "Format JSON tidak valid")
	}
	return false
}

// DecodeOptionalJSON sama dengan DecodeJSON, tetapi body kosong diterima (dst tidak diubah).
func DecodeOptionalJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	err := json.NewDecoder(r.Body).Decode(dst)
	if err == nil || errors.Is(err, io.EOF) {
		return true
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		Validation(w, "Body request terlalu besar (maksimal 1 MB)")
	} else {
		Validation(w, "Format JSON tidak valid")
	}
	return false
}
