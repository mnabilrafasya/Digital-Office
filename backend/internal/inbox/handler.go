package inbox

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
	"office.backend/internal/platform/middleware"
)

// Handler menangani endpoint inbox: daftar, buka item, disposisi, dan pengalihan.
type Handler struct {
	svc *Service
}

// NewHandler membuat Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// dispositionBody adalah body POST /inbox/{itemId}/disposition.
type dispositionBody struct {
	UserIDs []string `json:"userIds"`
	UnitIDs []string `json:"unitIds"`
	Note    *string  `json:"note"`
}

// redirectBody adalah body POST /inbox/{itemId}/redirect.
type redirectBody struct {
	UserID string  `json:"userId"`
	Note   *string `json:"note"`
}

// Get menangani GET /api/v1/inbox/{itemId} (item UNREAD otomatis menjadi READ).
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Get(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "itemId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

// Dispose menangani POST /api/v1/inbox/{itemId}/disposition
func (h *Handler) Dispose(w http.ResponseWriter, r *http.Request) {
	var body dispositionBody
	if !httpx.DecodeJSON(w, r, &body) {
		return
	}
	d, err := h.svc.Dispose(r.Context(), auth.CurrentUserFrom(r.Context()), DispositionInput{
		ItemID:   chi.URLParam(r, "itemId"),
		UserIDs:  body.UserIDs,
		UnitIDs:  body.UnitIDs,
		Note:     body.Note,
		ClientIP: middleware.ClientIPFrom(r.Context()),
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

// Redirect menangani POST /api/v1/inbox/{itemId}/redirect
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	var body redirectBody
	if !httpx.DecodeJSON(w, r, &body) {
		return
	}
	d, err := h.svc.Redirect(r.Context(), auth.CurrentUserFrom(r.Context()), RedirectInput{
		ItemID:   chi.URLParam(r, "itemId"),
		UserID:   body.UserID,
		Note:     body.Note,
		ClientIP: middleware.ClientIPFrom(r.Context()),
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

// List menangani GET /api/v1/inbox?status=&q=&page=&pageSize=
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, err := httpx.ParsePage(query)
	if err != nil {
		httpx.Validation(w, err.Error())
		return
	}
	res, err := h.svc.List(r.Context(), auth.CurrentUserFrom(r.Context()), ListQuery{
		Status:   query.Get("status"),
		Search:   query.Get("q"),
		Page:     page.Page,
		PageSize: page.PageSize,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSONPageWithCounts(w, res.Items, httpx.PageMeta{Page: page.Page, PageSize: page.PageSize, Total: res.Total}, res.Counts)
}
