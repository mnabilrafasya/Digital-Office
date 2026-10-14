package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"
)

// mengecek apakah database dan storage bisa dihubungi
type HealthHandler struct {
	DB    *sql.DB
	Minio *minio.Client
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	result := map[string]string{"database": "ok", "storage": "ok"}
	code := http.StatusOK

	if err := h.DB.PingContext(ctx); err != nil {
		result["database"] = "error"
		code = http.StatusServiceUnavailable
	}
	if _, err := h.Minio.ListBuckets(ctx); err != nil {
		result["storage"] = "error"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(result)
}