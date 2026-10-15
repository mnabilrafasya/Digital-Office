// Package httpx berisi helper untuk menulis response JSON dengan format seragam:
//
//	sukses: {"data": ...}
//	error:  {"error": {"code": "KODE", "message": "Pesan untuk pengguna"}}
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// MaxBodyBytes adalah batas ukuran body JSON (1 MB)
const MaxBodyBytes = 1 << 20

// Kode error yang dikirim ke klien
const (
	CodeValidation         = "VALIDATION_ERROR"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeForbidden          = "FORBIDDEN"
	CodeNotFound           = "NOT_FOUND"
	CodeMethodNotAllowed   = "METHOD_NOT_ALLOWED"
	CodeInternal           = "INTERNAL_ERROR"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON menulis response sukses {"data": data}
func JSON(w http.ResponseWriter, status int, data any) {
	write(w, status, map[string]any{"data": data})
}

// Error menulis response error {"error": {"code", "message"}}
func Error(w http.ResponseWriter, status int, code, message string) {
	write(w, status, map[string]any{"error": errorBody{Code: code, Message: message}})
}

// Validation menulis 400 VALIDATION_ERROR
func Validation(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, CodeValidation, message)
}

// Unauthorized menulis 401 UNAUTHORIZED
func Unauthorized(w http.ResponseWriter) {
	Error(w, http.StatusUnauthorized, CodeUnauthorized, "Silakan login terlebih dahulu")
}

// Forbidden menulis 403 FORBIDDEN
func Forbidden(w http.ResponseWriter) {
	Error(w, http.StatusForbidden, CodeForbidden, "Anda tidak memiliki akses untuk tindakan ini")
}

// Internal mencatat detail error ke log server lalu menulis 500 tanpa detail ke klien
func Internal(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("[%s] %s %s: %v", middleware.GetReqID(r.Context()), r.Method, r.URL.Path, err)
	Error(w, http.StatusInternalServerError, CodeInternal, "Terjadi kesalahan pada server, silakan coba lagi")
}

// NotFound dipakai sebagai handler route yang tidak ada
func NotFound(w http.ResponseWriter, _ *http.Request) {
	Error(w, http.StatusNotFound, CodeNotFound, "Alamat tidak ditemukan")
}

// MethodNotAllowed dipakai sebagai handler method HTTP yang salah
func MethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	Error(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "Method HTTP tidak diizinkan untuk alamat ini")
}

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

func write(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
