package audit

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
)

// Handler menangani Lacak Surat, Audit Log, dan Pantauan Surat. Permission AUDIT_VIEW untuk
// audit log dan pantauan dipasang di router.
type Handler struct {
	svc *Service
}

// NewHandler membuat Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Tracking menangani GET /api/v1/letters/{letterId}/tracking
func (h *Handler) Tracking(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.Tracking(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "letterId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

// LetterLogs menangani GET /api/v1/letters/{letterId}/audit (AUDIT_VIEW)
func (h *Handler) LetterLogs(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.LetterLogs(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "letterId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// Logs menangani GET /api/v1/audit-logs?from=&to=&actorId=&action=&entityType=&q=&page=&pageSize= (AUDIT_VIEW)
func (h *Handler) Logs(w http.ResponseWriter, r *http.Request) {
	page, err := httpx.ParsePage(r.URL.Query())
	if err != nil {
		httpx.Validation(w, err.Error())
		return
	}
	lq := logQuery(r)
	lq.Page, lq.PageSize = page.Page, page.PageSize
	items, total, err := h.svc.Logs(r.Context(), auth.CurrentUserFrom(r.Context()), lq)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSONPage(w, items, httpx.PageMeta{Page: page.Page, PageSize: page.PageSize, Total: total})
}

// Log menangani GET /api/v1/audit-logs/{logId} (AUDIT_VIEW)
func (h *Handler) Log(w http.ResponseWriter, r *http.Request) {
	e, err := h.svc.Log(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "logId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, e)
}

// Export menangani GET /api/v1/audit-logs/export?from=&to=&... (AUDIT_VIEW). CSV disusun di memori
// dulu, supaya error validasi masih bisa dijawab sebagai JSON.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	lq := logQuery(r)
	var buf bytes.Buffer
	if err := h.svc.Export(r.Context(), auth.CurrentUserFrom(r.Context()), lq, &buf); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="audit-log_%s_%s.csv"`, lq.From, lq.To))
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// Monitor menangani GET /api/v1/monitoring/letters?status=&minWaitingHours=&page=&pageSize= (AUDIT_VIEW)
func (h *Handler) Monitor(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, err := httpx.ParsePage(query)
	if err != nil {
		httpx.Validation(w, err.Error())
		return
	}
	items, total, err := h.svc.Monitor(r.Context(), auth.CurrentUserFrom(r.Context()), MonitorQuery{
		Status:          query.Get("status"),
		MinWaitingHours: query.Get("minWaitingHours"),
		Page:            page.Page,
		PageSize:        page.PageSize,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSONPage(w, items, httpx.PageMeta{Page: page.Page, PageSize: page.PageSize, Total: total})
}

func logQuery(r *http.Request) LogQuery {
	q := r.URL.Query()
	return LogQuery{
		From: q.Get("from"), To: q.Get("to"), ActorID: q.Get("actorId"), Action: q.Get("action"),
		EntityType: q.Get("entityType"), Search: q.Get("q"),
	}
}
