// Package repository berisi query SQL saja, tanpa aturan bisnis.
// Semua input dikirim sebagai parameter SQL (sql.Named), tidak pernah disambung ke teks query.
package repository

import (
	"errors"
	"strings"
)

// ErrNotFound dikembalikan bila data yang dicari tidak ada
var ErrNotFound = errors.New("data tidak ditemukan")

// likeEscaper meng-escape karakter khusus LIKE di SQL Server. "\" harus di-escape lebih dulu
// (strings.NewReplacer memproses setiap karakter input sekali, jadi urutannya aman).
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`, `[`, `\[`)

// ContainsPattern mengubah teks pencarian menjadi pola LIKE "%teks%" yang aman.
// Pola ini wajib dipakai bersama `ESCAPE '\'` di query.
func ContainsPattern(q string) string {
	return "%" + likeEscaper.Replace(q) + "%"
}
