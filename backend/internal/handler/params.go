package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/httpx"
)

// letterID membaca {letterId} (uuid) dari URL dalam bentuk baku huruf kecil.
// uuid yang tidak valid dijawab 404, sama seperti surat yang tidak ada.
func letterID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "letterId"))
	if err != nil {
		httpx.NotFoundMessage(w, "Surat tidak ditemukan")
		return "", false
	}
	return id.String(), true
}

// ifMatch membaca header If-Match yang wajib untuk PUT/DELETE surat.
func ifMatch(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	rv, present, err := httpx.ParseIfMatch(r.Header.Get("If-Match"))
	switch {
	case !present:
		httpx.PreconditionRequired(w)
		return nil, false
	case err != nil:
		httpx.Validation(w, "Header If-Match tidak valid, kirim nilai ETag apa adanya")
		return nil, false
	}
	return rv, true
}

// idParser mengubah ID string dari JSON menjadi angka dan mengumpulkan field yang tidak valid.
type idParser struct{ bad []string }

// optional mengubah ID opsional; null atau "" berarti tidak diisi.
func (p *idParser) optional(field string, s *string) *int64 {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	n, err := strconv.ParseInt(strings.TrimSpace(*s), 10, 64)
	if err != nil || n < 1 {
		p.bad = append(p.bad, field)
		return nil
	}
	return &n
}

// list mengubah daftar ID; ID kosong atau tidak valid dicatat sebagai kesalahan.
func (p *idParser) list(field string, ids []string) []int64 {
	out := make([]int64, 0, len(ids))
	for i, s := range ids {
		n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil || n < 1 {
			p.bad = append(p.bad, fmt.Sprintf("%s[%d]", field, i))
			continue
		}
		out = append(out, n)
	}
	return out
}

// check menulis 400 bila ada ID yang tidak valid.
func (p *idParser) check(w http.ResponseWriter) bool {
	if len(p.bad) == 0 {
		return true
	}
	httpx.Validation(w, "ID tidak valid pada field: "+strings.Join(p.bad, ", "))
	return false
}
