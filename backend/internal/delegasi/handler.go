package delegasi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
)

// Handler menangani jadwal delegasi: /me/delegations untuk semua user, /admin/delegations untuk
// pemegang USER_MANAGE (dipasang di router dan diperiksa ulang di service).
type Handler struct {
	svc *Service
}

// NewHandler membuat Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Mine menangani GET /api/v1/me/delegations
func (h *Handler) Mine(w http.ResponseWriter, r *http.Request) {
	m, err := h.svc.Mine(r.Context(), auth.CurrentUserFrom(r.Context()))
	respond(w, r, http.StatusOK, m, err)
}

// CreateMine menangani POST /api/v1/me/delegations {substituteId, from, until, note}
func (h *Handler) CreateMine(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	d, err := h.svc.CreateMine(r.Context(), auth.CurrentUserFrom(r.Context()), in)
	respond(w, r, http.StatusCreated, d, err)
}

// UpdateMine menangani PUT /api/v1/me/delegations/{delegationId}
func (h *Handler) UpdateMine(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	d, err := h.svc.UpdateMine(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "delegationId"), in)
	respond(w, r, http.StatusOK, d, err)
}

// DeleteMine menangani DELETE /api/v1/me/delegations/{delegationId}
func (h *Handler) DeleteMine(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteMine(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "delegationId")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// List menangani GET /api/v1/admin/delegations?status=&userId=&page=&pageSize=
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, err := httpx.ParsePage(r.URL.Query())
	if err != nil {
		httpx.Validation(w, err.Error())
		return
	}
	items, total, err := h.svc.List(r.Context(), auth.CurrentUserFrom(r.Context()), ListQuery{
		Status: r.URL.Query().Get("status"), UserID: r.URL.Query().Get("userId"),
		Offset: page.Offset(), Limit: page.PageSize,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSONPage(w, items, httpx.PageMeta{Page: page.Page, PageSize: page.PageSize, Total: total})
}

// Create menangani POST /api/v1/admin/delegations {userId, substituteId, from, until, note}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	d, err := h.svc.Create(r.Context(), auth.CurrentUserFrom(r.Context()), in)
	respond(w, r, http.StatusCreated, d, err)
}

// Update menangani PUT /api/v1/admin/delegations/{delegationId} {substituteId, from, until, note}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	d, err := h.svc.Update(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "delegationId"), in)
	respond(w, r, http.StatusOK, d, err)
}

// Delete menangani DELETE /api/v1/admin/delegations/{delegationId}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "delegationId")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func respond(w http.ResponseWriter, r *http.Request, status int, data any, err error) {
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, status, data)
}
