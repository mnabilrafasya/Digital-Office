package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"office.backend/internal/platform/httpx"
)

// ErrInvalidCredentials: NIK/password salah, user nonaktif, belum punya password, atau role nonaktif.
var ErrInvalidCredentials = errors.New("NIK atau password salah")

// TooManyAttemptsError menandakan percobaan login dibatasi sementara (429).
type TooManyAttemptsError struct{ RetryAfter time.Duration }

func (e *TooManyAttemptsError) Error() string {
	minutes := int((e.RetryAfter + time.Minute - 1) / time.Minute)
	return fmt.Sprintf("Terlalu banyak percobaan login. Coba lagi dalam %d menit.", max(minutes, 1))
}

// writeError menerjemahkan error login (429, 401 INVALID_CREDENTIALS) dan error umum menjadi response.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var tm *TooManyAttemptsError
	switch {
	case errors.As(err, &tm):
		httpx.TooManyRequests(w, tm.RetryAfter, tm.Error())
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(w, http.StatusUnauthorized, httpx.CodeInvalidCredentials, ErrInvalidCredentials.Error())
	default:
		httpx.WriteError(w, r, err)
	}
}
