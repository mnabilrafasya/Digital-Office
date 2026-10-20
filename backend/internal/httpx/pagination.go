package httpx

import (
	"errors"
	"net/url"
	"strconv"
)

// Batas pagination.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Error pagination; pesannya aman ditampilkan ke pengguna.
var (
	ErrInvalidPage     = errors.New("Parameter page harus angka mulai dari 1")
	ErrInvalidPageSize = errors.New("Parameter pageSize harus angka antara 1 dan 100")
)

// Page adalah halaman yang diminta klien.
type Page struct {
	Page     int
	PageSize int
}

// Offset adalah jumlah baris yang dilewati untuk halaman ini.
func (p Page) Offset() int { return (p.Page - 1) * p.PageSize }

// PageMeta adalah informasi halaman di response daftar.
type PageMeta struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

// ParsePage membaca page (default 1) dan pageSize (default 20, maksimal 100) dari query string.
func ParsePage(q url.Values) (Page, error) {
	p := Page{Page: 1, PageSize: DefaultPageSize}
	if raw := q.Get("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return p, ErrInvalidPage
		}
		p.Page = n
	}
	if raw := q.Get("pageSize"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxPageSize {
			return p, ErrInvalidPageSize
		}
		p.PageSize = n
	}
	return p, nil
}
