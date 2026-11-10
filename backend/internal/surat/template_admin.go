package surat

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/audit"
	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/db"
	"office.backend/internal/platform/httpx"
)

// Kelola template oleh admin perusahaan (TEMPLATE_MANAGE, dipasang di router): melihat semua
// template (aktif dan nonaktif) beserta pemakaiannya, lalu mengaktifkan/menonaktifkannya. Template
// tidak pernah dihapus permanen karena surat lama merujuknya. Membuat dan mengganti template tetap
// terbuka untuk semua pemegang TEMPLATE_CREATE (template.go).

// templateStatusSnapshot adalah isi audit_logs.snapshot saat status template diubah.
type templateStatusSnapshot struct {
	Name     string `json:"name"`
	IsActive bool   `json:"isActive"`
}

// AdminTemplates mengambil semua template perusahaan user. status: "" | ACTIVE | INACTIVE.
func (s *Service) AdminTemplates(ctx context.Context, user *auth.CurrentUser, status, search string) ([]AdminTemplate, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "" && status != "ACTIVE" && status != "INACTIVE" {
		return nil, apperr.Invalid("Parameter status wajib ACTIVE atau INACTIVE")
	}
	search, err := db.NormalizeSearch(search)
	if err != nil {
		return nil, err
	}
	items, err := s.repos.Templates.ListAdminTemplates(ctx, user.Company.Code, status, search)
	if err != nil {
		return nil, err
	}
	for i := range items {
		s.localTemplate(&items[i].Template)
	}
	return items, nil
}

// SetTemplateStatus mengaktifkan/menonaktifkan template. Template nonaktif tidak muncul saat membuat
// surat; draf dan surat yang sudah memakainya tidak terpengaruh.
func (s *Service) SetTemplateStatus(ctx context.Context, user *auth.CurrentUser, rawID string, active *bool) (*AdminTemplate, error) {
	id, err := templateID(rawID)
	if err != nil {
		return nil, err
	}
	if active == nil {
		return nil, apperr.Invalid("active wajib diisi (true atau false)")
	}
	err = s.tx.InTx(ctx, func(q db.DBTX) error {
		w := s.repos.TemplateWriter(q)
		t, err := w.LockAdminTemplate(ctx, user.Company.Code, id)
		if err != nil {
			return templateNotFoundAs(err)
		}
		if t.IsActive == *active {
			return nil
		}
		if err := w.SetTemplateActive(ctx, id, *active); err != nil {
			return err
		}
		return s.repos.Audit(q).Record(ctx, audit.Entry{
			CompanyCode: user.Company.Code, EntityType: audit.EntityTemplate, EntityID: id,
			Action: audit.ActionTemplateStatus, ActorID: user.ID, ActorName: user.Name,
			Snapshot: templateStatusSnapshot{Name: t.Name, IsActive: *active},
		})
	})
	if err != nil {
		return nil, err
	}
	t, err := s.repos.Templates.FindAdminTemplate(ctx, user.Company.Code, id, false)
	if err != nil {
		return nil, templateNotFoundAs(err)
	}
	s.localTemplate(&t.Template)
	return t, nil
}

// manageableTemplate mencari template nonaktif untuk pemegang TEMPLATE_MANAGE (supaya admin bisa
// mengunduh template yang sudah dinonaktifkan). Selain itu errTemplateNotFound.
func (s *Service) manageableTemplate(ctx context.Context, user *auth.CurrentUser, rawID string) (*Template, error) {
	if !user.HasPermission(auth.TemplateManage) {
		return nil, errTemplateNotFound
	}
	id, err := templateID(rawID)
	if err != nil {
		return nil, err
	}
	t, err := s.repos.Templates.FindAdminTemplate(ctx, user.Company.Code, id, false)
	if errors.Is(err, db.ErrNotFound) {
		return nil, errTemplateNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t.Template, nil
}

// AdminTemplates menangani GET /api/v1/admin/templates?status=&q= (TEMPLATE_MANAGE)
func (h *Handler) AdminTemplates(w http.ResponseWriter, r *http.Request) {
	items, err := h.letters.AdminTemplates(r.Context(), auth.CurrentUserFrom(r.Context()),
		r.URL.Query().Get("status"), r.URL.Query().Get("q"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// TemplateStatus menangani PATCH /api/v1/admin/templates/{templateId}/status {active} (TEMPLATE_MANAGE)
func (h *Handler) TemplateStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Active *bool `json:"active"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	t, err := h.letters.SetTemplateStatus(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "templateId"), in.Active)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}
