package perusahaan

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
)

// Handler menangani master data perusahaan. Permission dipasang di router: COMPANY_MANAGE untuk
// /admin/companies, COMPANY_SETTINGS untuk /company.
type Handler struct {
	svc *Service
}

// NewHandler membuat Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Companies menangani GET /api/v1/admin/companies?q=&status=&page=&pageSize=
func (h *Handler) Companies(w http.ResponseWriter, r *http.Request) {
	page, err := httpx.ParsePage(r.URL.Query())
	if err != nil {
		httpx.Validation(w, err.Error())
		return
	}
	items, total, err := h.svc.Companies(r.Context(), CompanyQuery{
		Search: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"),
		Offset: page.Offset(), Limit: page.PageSize,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSONPage(w, items, httpx.PageMeta{Page: page.Page, PageSize: page.PageSize, Total: total})
}

// Company menangani GET /api/v1/admin/companies/{code}
func (h *Handler) Company(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Company(r.Context(), chi.URLParam(r, "code"))
	respond(w, r, http.StatusOK, c, err)
}

// CreateCompany menangani POST /api/v1/admin/companies {code, name, city}
func (h *Handler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	var in CompanyInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	c, err := h.svc.CreateCompany(r.Context(), auth.CurrentUserFrom(r.Context()), in)
	respond(w, r, http.StatusCreated, c, err)
}

// UpdateCompany menangani PUT /api/v1/admin/companies/{code} {name, city}
func (h *Handler) UpdateCompany(w http.ResponseWriter, r *http.Request) {
	var in CompanyInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	c, err := h.svc.UpdateCompany(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "code"), in)
	respond(w, r, http.StatusOK, c, err)
}

// CompanyStatus menangani PATCH /api/v1/admin/companies/{code}/status {active}
func (h *Handler) CompanyStatus(w http.ResponseWriter, r *http.Request) {
	var in StatusInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	c, err := h.svc.SetCompanyStatus(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "code"), in)
	respond(w, r, http.StatusOK, c, err)
}

// Settings menangani GET /api/v1/company
func (h *Handler) Settings(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.Settings(r.Context(), auth.CurrentUserFrom(r.Context()))
	respond(w, r, http.StatusOK, st, err)
}

// UpdateSettings menangani PUT /api/v1/company {letterNumberFormat, watermarkText}
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var in SettingsInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	st, err := h.svc.UpdateSettings(r.Context(), auth.CurrentUserFrom(r.Context()), in)
	respond(w, r, http.StatusOK, st, err)
}

// NumberPreview menangani GET /api/v1/company/number-preview?format=
func (h *Handler) NumberPreview(w http.ResponseWriter, r *http.Request) {
	n, err := h.svc.NumberPreview(auth.CurrentUserFrom(r.Context()), r.URL.Query().Get("format"))
	respond(w, r, http.StatusOK, map[string]string{"numberPreview": n}, err)
}

// WatermarkPreview menangani GET /api/v1/company/watermark-preview?text= (PNG). Tanpa parameter
// text: watermark yang tersimpan.
func (h *Handler) WatermarkPreview(w http.ResponseWriter, r *http.Request) {
	var text *string
	if r.URL.Query().Has("text") {
		t := r.URL.Query().Get("text")
		text = &t
	}
	img, err := h.svc.WatermarkPreview(r.Context(), auth.CurrentUserFrom(r.Context()), text)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(img)
}

func respond(w http.ResponseWriter, r *http.Request, status int, data any, err error) {
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, status, data)
}
