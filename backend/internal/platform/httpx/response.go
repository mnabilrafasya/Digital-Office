// Package httpx berisi helper HTTP dengan format response seragam:
//
//	sukses:  {"data": ...}
//	daftar:  {"data": [...], "meta": {"page", "pageSize", "total"}}, opsional "counts" per tab
//	error:   {"error": {"code": "KODE", "message": "Pesan untuk pengguna"}}
package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

// JSON menulis response sukses {"data": data}.
func JSON(w http.ResponseWriter, status int, data any) {
	write(w, status, map[string]any{"data": data})
}

// JSONPage menulis response daftar berhalaman {"data": data, "meta": meta}.
func JSONPage(w http.ResponseWriter, data any, meta PageMeta) {
	write(w, http.StatusOK, map[string]any{"data": data, "meta": meta})
}

// JSONPageWithCounts menulis response daftar berhalaman beserta jumlah per kelompok (mis. per
// tab): {"data": data, "meta": meta, "counts": counts}.
func JSONPageWithCounts(w http.ResponseWriter, data any, meta PageMeta, counts any) {
	write(w, http.StatusOK, map[string]any{"data": data, "meta": meta, "counts": counts})
}

// NoContent menulis 204 tanpa body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func write(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("menulis response JSON: %v", err)
	}
}
