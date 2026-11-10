package organisasi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
)

// AdminHandler menangani master data organisasi. Permission dipasang di router: ORG_MANAGE untuk
// unit dan jabatan, USER_MANAGE untuk user, COMPANY_MANAGE untuk admin perusahaan lain.
type AdminHandler struct {
	svc *AdminService
}

// NewAdminHandler membuat AdminHandler.
func NewAdminHandler(svc *AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func listQuery(r *http.Request) ListQuery {
	q := r.URL.Query()
	return ListQuery{Search: q.Get("q"), Status: q.Get("status"), UnitID: q.Get("unitId"), RoleID: q.Get("roleId")}
}

// pagedQuery membaca filter daftar beserta halamannya; false bila sudah dijawab error.
func pagedQuery(w http.ResponseWriter, r *http.Request) (ListQuery, httpx.Page, bool) {
	page, err := httpx.ParsePage(r.URL.Query())
	if err != nil {
		httpx.Validation(w, err.Error())
		return ListQuery{}, page, false
	}
	lq := listQuery(r)
	lq.Offset, lq.Limit = page.Offset(), page.PageSize
	return lq, page, true
}

func respond(w http.ResponseWriter, r *http.Request, status int, data any, err error) {
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, status, data)
}

func currentUser(r *http.Request) *auth.CurrentUser { return auth.CurrentUserFrom(r.Context()) }

// --- Unit ---

// Units menangani GET /api/v1/admin/units?q=&status=
func (h *AdminHandler) Units(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Units(r.Context(), currentUser(r), listQuery(r))
	respond(w, r, http.StatusOK, items, err)
}

// CreateUnit menangani POST /api/v1/admin/units {name, unitType, parentId}
func (h *AdminHandler) CreateUnit(w http.ResponseWriter, r *http.Request) {
	var in UnitInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	u, err := h.svc.CreateUnit(r.Context(), currentUser(r), in)
	respond(w, r, http.StatusCreated, u, err)
}

// UpdateUnit menangani PUT /api/v1/admin/units/{unitId} {name, unitType, parentId}
func (h *AdminHandler) UpdateUnit(w http.ResponseWriter, r *http.Request) {
	var in UnitInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	u, err := h.svc.UpdateUnit(r.Context(), currentUser(r), chi.URLParam(r, "unitId"), in)
	respond(w, r, http.StatusOK, u, err)
}

// UnitStatus menangani PATCH /api/v1/admin/units/{unitId}/status {active}
func (h *AdminHandler) UnitStatus(w http.ResponseWriter, r *http.Request) {
	var in StatusInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	u, err := h.svc.SetUnitStatus(r.Context(), currentUser(r), chi.URLParam(r, "unitId"), in)
	respond(w, r, http.StatusOK, u, err)
}

// --- Jabatan ---

// Positions menangani GET /api/v1/admin/positions?unitId=&q=&status=&page=&pageSize=
func (h *AdminHandler) Positions(w http.ResponseWriter, r *http.Request) {
	lq, page, ok := pagedQuery(w, r)
	if !ok {
		return
	}
	items, total, err := h.svc.Positions(r.Context(), currentUser(r), lq)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSONPage(w, items, httpx.PageMeta{Page: page.Page, PageSize: page.PageSize, Total: total})
}

// Levels menangani GET /api/v1/admin/position-levels
func (h *AdminHandler) Levels(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Levels(r.Context(), currentUser(r))
	respond(w, r, http.StatusOK, items, err)
}

// CreatePosition menangani POST /api/v1/admin/positions {unitId, name, levelId, isUnitHead}
func (h *AdminHandler) CreatePosition(w http.ResponseWriter, r *http.Request) {
	var in PositionInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	p, err := h.svc.CreatePosition(r.Context(), currentUser(r), in)
	respond(w, r, http.StatusCreated, p, err)
}

// UpdatePosition menangani PUT /api/v1/admin/positions/{positionId}
func (h *AdminHandler) UpdatePosition(w http.ResponseWriter, r *http.Request) {
	var in PositionInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	p, err := h.svc.UpdatePosition(r.Context(), currentUser(r), chi.URLParam(r, "positionId"), in)
	respond(w, r, http.StatusOK, p, err)
}

// PositionStatus menangani PATCH /api/v1/admin/positions/{positionId}/status {active}
func (h *AdminHandler) PositionStatus(w http.ResponseWriter, r *http.Request) {
	var in StatusInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	p, err := h.svc.SetPositionStatus(r.Context(), currentUser(r), chi.URLParam(r, "positionId"), in)
	respond(w, r, http.StatusOK, p, err)
}

// --- User ---

// Users menangani GET /api/v1/admin/users?q=&unitId=&roleId=&status=&page=&pageSize=
func (h *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	lq, page, ok := pagedQuery(w, r)
	if !ok {
		return
	}
	items, total, err := h.svc.Users(r.Context(), currentUser(r), lq)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSONPage(w, items, httpx.PageMeta{Page: page.Page, PageSize: page.PageSize, Total: total})
}

// User menangani GET /api/v1/admin/users/{userId}
func (h *AdminHandler) User(w http.ResponseWriter, r *http.Request) {
	u, err := h.svc.User(r.Context(), currentUser(r), chi.URLParam(r, "userId"))
	respond(w, r, http.StatusOK, u, err)
}

// Roles menangani GET /api/v1/admin/roles
func (h *AdminHandler) Roles(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Roles(r.Context(), currentUser(r))
	respond(w, r, http.StatusOK, items, err)
}

// CreateUser menangani POST /api/v1/admin/users {employeeNumber, name, email, positionId, roleId, password}
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var in UserInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	u, err := h.svc.CreateUser(r.Context(), currentUser(r), in)
	respond(w, r, http.StatusCreated, u, err)
}

// UpdateUser menangani PUT /api/v1/admin/users/{userId} {employeeNumber, name, email, positionId, roleId}
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var in UserInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	u, err := h.svc.UpdateUser(r.Context(), currentUser(r), chi.URLParam(r, "userId"), in)
	respond(w, r, http.StatusOK, u, err)
}

// UserStatus menangani PATCH /api/v1/admin/users/{userId}/status {active}
func (h *AdminHandler) UserStatus(w http.ResponseWriter, r *http.Request) {
	var in StatusInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	u, err := h.svc.SetUserStatus(r.Context(), currentUser(r), chi.URLParam(r, "userId"), in)
	respond(w, r, http.StatusOK, u, err)
}

// ResetPassword menangani POST /api/v1/admin/users/{userId}/reset-password {password}
func (h *AdminHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var in PasswordInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if err := h.svc.ResetPassword(r.Context(), currentUser(r), chi.URLParam(r, "userId"), in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// --- SUPERADMIN ---

// CompanyUnits menangani GET /api/v1/admin/companies/{code}/units?q= (unit aktif perusahaan lain)
func (h *AdminHandler) CompanyUnits(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.CompanyUnits(r.Context(), chi.URLParam(r, "code"), listQuery(r))
	respond(w, r, http.StatusOK, items, err)
}

// CreateCompanyAdmin menangani POST /api/v1/admin/companies/{code}/admins
func (h *AdminHandler) CreateCompanyAdmin(w http.ResponseWriter, r *http.Request) {
	var in CompanyAdminInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	u, err := h.svc.CreateCompanyAdmin(r.Context(), currentUser(r), chi.URLParam(r, "code"), in)
	respond(w, r, http.StatusCreated, u, err)
}
