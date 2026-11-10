package notifikasi

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"office.backend/internal/auth"
	"office.backend/internal/platform/apperr"
	"office.backend/internal/platform/httpx"
)

// employeeChecker memeriksa NIK pegawai aktif (dipenuhi Repository).
type employeeChecker interface {
	activeEmployee(ctx context.Context, companyCode, employeeNumber string) (bool, error)
}

// Handler menangani pesan uji WhatsApp. Permission AUDIT_VIEW dipasang di router.
type Handler struct {
	employees employeeChecker
	sender    WhatsAppSender // nil = layanan WhatsApp belum dipasang
}

// NewHandler membuat Handler.
func NewHandler(r *Repository, sender WhatsAppSender) *Handler {
	return &Handler{employees: r, sender: sender}
}

type testRequest struct {
	EmployeeNumber string `json:"employeeNumber"`
	Message        string `json:"message"`
}

// Test menangani POST /api/v1/notifications/test {employeeNumber, message} (AUDIT_VIEW): mengirim
// satu pesan langsung untuk memeriksa sambungan layanan WhatsApp.
func (h *Handler) Test(w http.ResponseWriter, r *http.Request) {
	var req testRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if err := h.sendTest(r.Context(), auth.CurrentUserFrom(r.Context()), req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "SENT"})
}

func (h *Handler) sendTest(ctx context.Context, user *auth.CurrentUser, req testRequest) error {
	if h.sender == nil {
		return &apperr.UnavailableError{Message: "Layanan WhatsApp belum terpasang"}
	}
	nik := strings.TrimSpace(req.EmployeeNumber)
	if nik == "" {
		return apperr.Invalid("employeeNumber wajib diisi")
	}
	text := strings.TrimSpace(req.Message)
	if text == "" {
		text = "*Digital Office PT Pupuk Sriwidjaja Palembang*\nPesan uji dari " + user.Name + "."
	}
	if utf8.RuneCountInString(text) > maxMessageRunes {
		return apperr.Invalid("message maksimal 2000 karakter")
	}
	ok, err := h.employees.activeEmployee(ctx, user.Company.Code, nik)
	if err != nil {
		return err
	}
	if !ok {
		return apperr.Invalid("NIK tidak ditemukan atau pegawai tidak aktif")
	}
	sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	if err := h.sender.Send(sendCtx, Outgoing{EmployeeNumber: nik, Event: "UJI", Text: text}); err != nil {
		return &apperr.UnavailableError{Message: "Layanan WhatsApp gagal mengirim pesan", Cause: err}
	}
	return nil
}
