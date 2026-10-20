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
		is *service.InvalidStateError
		vc *service.VersionConflictError
	)
	switch {
	case errors.As(err, &ve):
		httpx.Validation(w, ve.Message)
	case errors.Is(err, service.ErrLetterNotFound):
		httpx.NotFoundMessage(w, service.ErrLetterNotFound.Error())
	case errors.Is(err, service.ErrForbidden):
		httpx.Forbidden(w)
	case errors.As(err, &is):
		httpx.Error(w, http.StatusConflict, httpx.CodeInvalidState, is.Message)
	case errors.As(err, &vc):
		httpx.SetETag(w, vc.RowVersion)
		httpx.Error(w, http.StatusPreconditionFailed, httpx.CodeVersionConflict, vc.Error())
	case errors.As(err, &tm):
		httpx.TooManyRequests(w, tm.RetryAfter, tm.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		httpx.Error(w, http.StatusUnauthorized, httpx.CodeInvalidCredentials, service.ErrInvalidCredentials.Error())
	default:
		httpx.Internal(w, r, err)
	}
}
