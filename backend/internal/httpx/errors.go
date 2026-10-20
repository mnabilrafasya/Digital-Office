package httpx

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Kode error yang dikirim ke klien.
const (
	CodeValidation           = "VALIDATION_ERROR"
	CodeInvalidCredentials   = "INVALID_CREDENTIALS"
	CodeUnauthorized         = "UNAUTHORIZED"
	CodeForbidden            = "FORBIDDEN"
	CodeNotFound             = "NOT_FOUND"
	CodeMethodNotAllowed     = "METHOD_NOT_ALLOWED"
	CodeInvalidState         = "INVALID_STATE"
	CodeVersionConflict      = "VERSION_CONFLICT"
	CodeFileTooLarge         = "FILE_TOO_LARGE"
	CodePreconditionRequired = "PRECONDITION_REQUIRED"
	CodeTooManyRequests      = "TOO_MANY_REQUESTS"
	CodeStorageFull          = "STORAGE_FULL"
	CodeInternal             = "INTERNAL_ERROR"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error menulis response error {"error": {"code", "message"}}.
func Error(w http.ResponseWriter, status int, code, message string) {
	write(w, status, map[string]any{"error": errorBody{Code: code, Message: message}})
}

// Validation menulis 400 VALIDATION_ERROR.
func Validation(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, CodeValidation, message)
}

// Unauthorized menulis 401 UNAUTHORIZED.
func Unauthorized(w http.ResponseWriter) {
	Error(w, http.StatusUnauthorized, CodeUnauthorized, "Silakan login terlebih dahulu")
}

// Forbidden menulis 403 FORBIDDEN.
func Forbidden(w http.ResponseWriter) {
	Error(w, http.StatusForbidden, CodeForbidden, "Anda tidak memiliki akses untuk tindakan ini")
}

// NotFoundMessage menulis 404 NOT_FOUND dengan pesan tertentu.
func NotFoundMessage(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, CodeNotFound, message)
}

// PreconditionRequired menulis 428 bila header If-Match wajib tetapi tidak dikirim.
func PreconditionRequired(w http.ResponseWriter) {
	Error(w, http.StatusPreconditionRequired, CodePreconditionRequired,
		"Header If-Match wajib dikirim. Ambil detail surat terbaru lalu kirim nilai ETag-nya.")
}

// TooManyRequests menulis 429 beserta header Retry-After (detik).
func TooManyRequests(w http.ResponseWriter, retryAfter time.Duration, message string) {
	secs := int((retryAfter + time.Second - 1) / time.Second)
	if secs < 1 {
		secs = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	Error(w, http.StatusTooManyRequests, CodeTooManyRequests, message)
}

// Internal mencatat detail error ke log server lalu menulis 500 tanpa detail ke klien.
func Internal(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("[%s] %s %s: %v", middleware.GetReqID(r.Context()), r.Method, r.URL.Path, err)
	Error(w, http.StatusInternalServerError, CodeInternal, "Terjadi kesalahan pada server, silakan coba lagi")
}

// NotFound dipakai sebagai handler route yang tidak ada.
func NotFound(w http.ResponseWriter, _ *http.Request) {
	NotFoundMessage(w, "Alamat tidak ditemukan")
}

// MethodNotAllowed dipakai sebagai handler method HTTP yang salah.
func MethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	Error(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "Method HTTP tidak diizinkan untuk alamat ini")
}
