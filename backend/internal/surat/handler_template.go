package surat

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"office.backend/internal/auth"
	"office.backend/internal/platform/httpx"
	"office.backend/internal/platform/middleware"
)

// Templates menangani GET /api/v1/templates?letterTypeId= (template aktif perusahaan user).
func (h *Handler) Templates(w http.ResponseWriter, r *http.Request) {
	items, err := h.letters.Templates(r.Context(), auth.CurrentUserFrom(r.Context()), r.URL.Query().Get("letterTypeId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// TemplateFile menangani GET /api/v1/templates/{templateId}/file (unduh .docx).
func (h *Handler) TemplateFile(w http.ResponseWriter, r *http.Request) {
	f, err := h.letters.OpenTemplate(r.Context(), auth.CurrentUserFrom(r.Context()), chi.URLParam(r, "templateId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	sendFile(w, f, "template")
}

// CreateTemplate menangani POST /api/v1/templates (multipart/form-data: file .docx, name,
// description, letterTypeId). Response 201 berisi template baru.
func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	in, ok := readTemplateForm(w, r, true)
	if !ok {
		return
	}
	t, err := h.letters.CreateTemplate(r.Context(), auth.CurrentUserFrom(r.Context()), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/templates/"+strconv.FormatInt(t.ID, 10))
	httpx.JSON(w, http.StatusCreated, t)
}

// UpdateTemplate menangani PUT /api/v1/templates/{templateId} (multipart/form-data). Field teks
// mengganti seluruh data template (yang tidak dikirim berarti kosong); file .docx opsional,
// tanpa file berarti file lama tetap dipakai.
func (h *Handler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	in, ok := readTemplateForm(w, r, false)
	if !ok {
		return
	}
	in.ID = chi.URLParam(r, "templateId")
	t, err := h.letters.UpdateTemplate(r.Context(), auth.CurrentUserFrom(r.Context()), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

// readTemplateForm membaca form template. ok=false berarti response error sudah ditulis.
func readTemplateForm(w http.ResponseWriter, r *http.Request, fileRequired bool) (TemplateInput, bool) {
	data, fileName, ok := readUpload(w, r, MaxDocumentBytes, "template", fileRequired)
	if !ok {
		return TemplateInput{}, false
	}
	return TemplateInput{
		Name:         formValue(r, "name"),
		Description:  formValue(r, "description"),
		LetterTypeID: formValue(r, "letterTypeId"),
		FileName:     fileName,
		Data:         data,
		ClientIP:     middleware.ClientIPFrom(r.Context()),
	}, true
}
