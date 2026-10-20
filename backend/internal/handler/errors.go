// Package handler membaca request, memanggil service, dan menulis response. Handler tidak berisi SQL
// maupun aturan bisnis.
package handler

import (
	"errors"
	"net/http"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/httpx"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/service"
)

// writeServiceError menerjemahkan error dari service menjadi response HTTP.
func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	if msg, ok := service.IsNotFound(err); ok {
		httpx.NotFoundMessage(w, msg)
		return
	}

	var (
		ve *service.ValidationError
		se *service.StateError
		vc *service.VersionConflictError
		tm *service.TooManyAttemptsError
		ft *service.FileTooLargeError
	)
	switch {
	case errors.As(err, &ve):
		httpx.Validation(w, ve.Message)
	case errors.As(err, &se):
		httpx.Error(w, http.StatusConflict, httpx.CodeInvalidState, se.Message)
	case errors.As(err, &vc):
		httpx.SetETag(w, vc.Current)
		httpx.Error(w, http.StatusPreconditionFailed, httpx.CodeVersionConflict, vc.Error())
	case errors.As(err, &tm):
		httpx.TooManyRequests(w, tm.RetryAfter, tm.Error())
	case errors.As(err, &ft):
		httpx.Error(w, http.StatusRequestEntityTooLarge, httpx.CodeFileTooLarge, ft.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		httpx.Error(w, http.StatusUnauthorized, httpx.CodeInvalidCredentials, err.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Forbidden(w)
	case errors.Is(err, service.ErrStorageFull):
		httpx.Error(w, http.StatusInsufficientStorage, httpx.CodeStorageFull, err.Error())
	default:
		httpx.Internal(w, r, err)
	}
}
