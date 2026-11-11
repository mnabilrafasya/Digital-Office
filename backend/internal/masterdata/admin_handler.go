package masterdata

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
)

// AdminHandler menangani master data lookup perusahaan (LETTER_TYPE dan CLASSIFICATION).
// Permission dipasang di router: ORG_MANAGE.
type AdminHandler struct {
	svc *AdminService
}

// NewAdminHandler membuat AdminHandler.
func NewAdminHandler(svc *AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// List menangani GET /api/v1/admin/lookups/{category}?q=&status=
func (h *AdminHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := h.svc.List(
		r.Context(),
		auth.CurrentUserFrom(r.Context()),
		chi.URLParam(r, "category"),
		q.Get("status"),
		q.Get("q"),
	)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// Save menangani POST /api/v1/admin/lookups/{category} dan PUT /api/v1/admin/lookups/{category}/{lookupId}
func (h *AdminHandler) Save(w http.ResponseWriter, r *http.Request) {
	var in LookupInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	id := chi.URLParam(r, "lookupId")
	item, err := h.svc.Save(
		r.Context(),
		auth.CurrentUserFrom(r.Context()),
		chi.URLParam(r, "category"),
		id,
		in,
	)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	status := http.StatusOK
	if id == "" {
		status = http.StatusCreated
	}
	httpx.JSON(w, status, item)
}

// Status menangani PATCH /api/v1/admin/lookups/{category}/{lookupId}/status {active}
func (h *AdminHandler) Status(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Active *bool `json:"active"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	item, err := h.svc.SetStatus(
		r.Context(),
		auth.CurrentUserFrom(r.Context()),
		chi.URLParam(r, "category"),
		chi.URLParam(r, "lookupId"),
		in.Active,
	)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

// Delete menangani DELETE /api/v1/admin/lookups/{category}/{lookupId}
func (h *AdminHandler) Delete(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Delete(
		r.Context(),
		auth.CurrentUserFrom(r.Context()),
		chi.URLParam(r, "category"),
		chi.URLParam(r, "lookupId"),
	)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}
