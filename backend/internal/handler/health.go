package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Pinger adalah komponen yang bisa dicek ketersediaannya (database, storage).
type Pinger interface {
	PingContext(ctx context.Context) error
}

// StoragePinger adalah storage yang bisa dicek ketersediaannya.
type StoragePinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler mengecek apakah database dan storage bisa dihubungi.
type HealthHandler struct {
	db      Pinger
	storage StoragePinger
}

// NewHealthHandler membuat HealthHandler.
func NewHealthHandler(db Pinger, storage StoragePinger) *HealthHandler {
	return &HealthHandler{db: db, storage: storage}
}

// Check menangani GET /health: {"database":"ok|error","storage":"ok|error"}, 503 bila ada yang error.
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	result := map[string]string{"database": "ok", "storage": "ok"}
	code := http.StatusOK

	if err := h.db.PingContext(ctx); err != nil {
		result["database"] = "error"
		code = http.StatusServiceUnavailable
	}
	if err := h.storage.Ping(ctx); err != nil {
		result["storage"] = "error"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(result)
}
