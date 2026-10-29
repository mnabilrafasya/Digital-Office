package httpx

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"office.backend/internal/platform/apperr"
)

// WriteError menerjemahkan error aplikasi (package apperr) menjadi response HTTP. Error lain
// dianggap tak terduga: dicatat di log lalu dijawab 500 tanpa detail.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		ve *apperr.ValidationError
		nf *apperr.NotFoundError
		is *apperr.InvalidStateError
		vc *apperr.VersionConflictError
		ue *apperr.UnavailableError
	)
	switch {
	case errors.As(err, &ve):
		Validation(w, ve.Message)
	case errors.As(err, &nf):
		NotFoundMessage(w, nf.Message)
	case errors.Is(err, apperr.ErrForbidden):
		Forbidden(w)
	case errors.As(err, &is):
		Error(w, http.StatusConflict, CodeInvalidState, is.Message)
	case errors.As(err, &vc):
		SetETag(w, vc.RowVersion)
		Error(w, http.StatusPreconditionFailed, CodeVersionConflict, vc.Error())
	case errors.As(err, &ue):
		log.Printf("[%s] %s %s: %v", middleware.GetReqID(r.Context()), r.Method, r.URL.Path, err)
		Error(w, http.StatusServiceUnavailable, CodeServiceUnavailable, ue.Message)
	default:
		Internal(w, r, err)
	}
}
