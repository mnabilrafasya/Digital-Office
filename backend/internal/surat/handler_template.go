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
	sendFile(w, f, "attachment", "template")
}

// CreateTemplate menangani POST /api/v1/templates (multipart/form-data: file .docx, name,
// description, letterTypeId). Response 201 berisi template baru.
func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	in, release, ok := h.readTemplateForm(w, r, true)
	if !ok {
		return
	}
	defer release()
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
	in, release, ok := h.readTemplateForm(w, r, false)
	if !ok {
		return
	}
	defer release()
	in.ID = chi.URLParam(r, "templateId")
	t, err := h.letters.UpdateTemplate(r.Context(), auth.CurrentUserFrom(r.Context()), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

// readTemplateForm membaca form template. ok=false berarti response error sudah ditulis; selain
// itu release wajib dipanggil setelah template diproses (jatah antrean unggahan).
func (h *Handler) readTemplateForm(w http.ResponseWriter, r *http.Request, fileRequired bool) (TemplateInput, func(), bool) {
	form, ok := h.readUpload(w, r, MaxDocumentBytes, "template", fileRequired)
	if !ok {
		return TemplateInput{}, nil, false
	}
	return TemplateInput{
		Name:         form.value("name"),
		Description:  form.value("description"),
		LetterTypeID: form.value("letterTypeId"),
		FileName:     form.fileName,
		Data:         form.data,
		ClientIP:     middleware.ClientIPFrom(r.Context()),
	}, form.release, true
}
