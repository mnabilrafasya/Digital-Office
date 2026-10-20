// Package handler membaca request, memanggil service, dan menulis response. Handler tidak berisi
// SQL maupun aturan bisnis.
package handler

import (
	"errors"
	"net/http"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/httpx"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/service"
)

// writeServiceError menerjemahkan error dari service menjadi response HTTP.
func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		ve *service.ValidationError
		tm *service.TooManyAttemptsError
	)
	switch {
	case errors.As(err, &ve):
		httpx.Validation(w, ve.Message)
	case errors.As(err, &tm):
		httpx.TooManyRequests(w, tm.RetryAfter, tm.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		httpx.Error(w, http.StatusUnauthorized, httpx.CodeInvalidCredentials, service.ErrInvalidCredentials.Error())
	default:
		httpx.Internal(w, r, err)
	}
}
