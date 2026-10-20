package handler

import (
	"net/http"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/httpx"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/middleware"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/service"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/workflow"
)

// LetterHandler menangani draf surat dan alur persetujuan.
type LetterHandler struct {
	letters *service.LetterService
}

// NewLetterHandler membuat LetterHandler.
func NewLetterHandler(letters *service.LetterService) *LetterHandler {
	return &LetterHandler{letters: letters}
}

// letterRequest adalah body POST/PUT surat. Semua ID berupa string.
type letterRequest struct {
	LetterTypeID      *string  `json:"letterTypeId"`
	TemplateID        *string  `json:"templateId"`
	ClassificationID  *string  `json:"classificationId"`
	ConfidentialityID *string  `json:"confidentialityId"`
	UrgencyID         *string  `json:"urgencyId"`
	Title             *string  `json:"title"`
	Summary           *string  `json:"summary"`
	SpecialNote       *string  `json:"specialNote"`
	ReviewerIDs       []string `json:"reviewerIds"`
	ApproverID        *string  `json:"approverId"`
	Recipients        struct {
		To []recipientRequest `json:"to"`
		CC []recipientRequest `json:"cc"`
	} `json:"recipients"`
}

type recipientRequest struct {
	UnitID       *string `json:"unitId"`
	AliasLevelID *string `json:"aliasLevelId"`
}

func (req letterRequest) toInput(w http.ResponseWriter) (service.LetterInput, bool) {
	var p idParser
	in := service.LetterInput{
		LetterTypeID:      p.optional("letterTypeId", req.LetterTypeID),
		TemplateID:        p.optional("templateId", req.TemplateID),
		ClassificationID:  p.optional("classificationId", req.ClassificationID),
		ConfidentialityID: p.optional("confidentialityId", req.ConfidentialityID),
		UrgencyID:         p.optional("urgencyId", req.UrgencyID),
		Title:             req.Title,
		Summary:           req.Summary,
		SpecialNote:       req.SpecialNote,
		ReviewerIDs:       p.list("reviewerIds", req.ReviewerIDs),
		ApproverID:        p.optional("approverId", req.ApproverID),
	}
	recipients := func(field string, list []recipientRequest) []service.RecipientInput {
		out := make([]service.RecipientInput, 0, len(list))
		for _, rc := range list {
			out = append(out, service.RecipientInput{
				UnitID:       p.optional(field+".unitId", rc.UnitID),
				AliasLevelID: p.optional(field+".aliasLevelId", rc.AliasLevelID),
			})
		}
		return out
	}
	in.To = recipients("recipients.to", req.Recipients.To)
	in.CC = recipients("recipients.cc", req.Recipients.CC)
	return in, p.check(w)
}

func writeLetter(w http.ResponseWriter, status int, res *service.LetterResult) {
	httpx.SetETag(w, res.RowVersion)
	httpx.JSON(w, status, res.Detail)
}

// Create menangani POST /api/v1/letters (201 + Location + ETag).
func (h *LetterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req letterRequest
	if !httpx.DecodeOptionalJSON(w, r, &req) {
		return
	}
	in, ok := req.toInput(w)
	if !ok {
		return
	}
	res, err := h.letters.Create(r.Context(), middleware.CurrentUser(r.Context()), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/letters/"+res.Detail.ID)
	writeLetter(w, http.StatusCreated, res)
}

// List menangani GET /api/v1/letters?scope=mine|tasks&status=...&q=...&page=...&pageSize=...
func (h *LetterHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, err := httpx.ParsePage(q)
	if err != nil {
		httpx.Validation(w, err.Error())
		return
	}
	items, total, err := h.letters.List(r.Context(), middleware.CurrentUser(r.Context()), service.ListQuery{
		Scope: q.Get("scope"), Statuses: q.Get("status"), Search: q.Get("q"),
		Offset: page.Offset(), Limit: page.PageSize,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.JSONPage(w, items, httpx.PageMeta{Page: page.Page, PageSize: page.PageSize, Total: total})
}

// Get menangani GET /api/v1/letters/{letterId} (detail + ETag).
func (h *LetterHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := letterID(w, r)
	if !ok {
		return
	}
	res, err := h.letters.Get(r.Context(), middleware.CurrentUser(r.Context()), id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeLetter(w, http.StatusOK, res)
}

// Update menangani PUT /api/v1/letters/{letterId} (wajib If-Match).
func (h *LetterHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := letterID(w, r)
	if !ok {
		return
	}
	version, ok := ifMatch(w, r)
	if !ok {
		return
	}
	var req letterRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	in, ok := req.toInput(w)
	if !ok {
		return
	}
	res, err := h.letters.Update(r.Context(), middleware.CurrentUser(r.Context()), id, version, in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeLetter(w, http.StatusOK, res)
}

// Delete menangani DELETE /api/v1/letters/{letterId} (wajib If-Match, 204).
func (h *LetterHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := letterID(w, r)
	if !ok {
		return
	}
	version, ok := ifMatch(w, r)
	if !ok {
		return
	}
	if err := h.letters.Delete(r.Context(), middleware.CurrentUser(r.Context()), id, version); err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// History menangani GET /api/v1/letters/{letterId}/history.
func (h *LetterHandler) History(w http.ResponseWriter, r *http.Request) {
	id, ok := letterID(w, r)
	if !ok {
		return
	}
	items, err := h.letters.History(r.Context(), middleware.CurrentUser(r.Context()), id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

type noteRequest struct {
	Note *string `json:"note"`
}

// Submit menangani POST /api/v1/letters/{letterId}/submit.
func (h *LetterHandler) Submit(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, workflow.ActionSubmit)
}

// Approve menangani POST /api/v1/letters/{letterId}/approve.
func (h *LetterHandler) Approve(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, workflow.ActionApprove)
}

// Return menangani POST /api/v1/letters/{letterId}/return.
func (h *LetterHandler) Return(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, workflow.ActionReturn)
}

// Sign menangani POST /api/v1/letters/{letterId}/sign.
func (h *LetterHandler) Sign(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, workflow.ActionSign)
}

func (h *LetterHandler) transition(w http.ResponseWriter, r *http.Request, action workflow.Action) {
	id, ok := letterID(w, r)
	if !ok {
		return
	}
	var req noteRequest
	if !httpx.DecodeOptionalJSON(w, r, &req) {
		return
	}
	res, err := h.letters.Transition(r.Context(), middleware.CurrentUser(r.Context()), id, service.TransitionInput{
		Action: action, Note: req.Note, ClientIP: middleware.ClientIPFrom(r.Context()),
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeLetter(w, http.StatusOK, res)
}
