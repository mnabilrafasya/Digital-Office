package db

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"office.backend/internal/platform/apperr"
)

// Helper query yang dipakai repository semua fitur. Semua input dikirim sebagai parameter SQL
// (sql.Named), tidak pernah disambung ke teks query.

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

// maxSearchLength adalah panjang maksimal teks pencarian q (karakter).
const maxSearchLength = 100

// NormalizeSearch merapikan teks pencarian dari klien: trim spasi, "" berarti tidak mencari.
func NormalizeSearch(q string) (string, error) {
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) > maxSearchLength {
		return "", apperr.Invalid("Teks pencarian maksimal 100 karakter")
	}
	return q, nil
}

// JSONIDs mengubah daftar ID menjadi array JSON untuk dibaca query dengan OPENJSON:
//
//	WHERE id IN (SELECT CAST([value] AS BIGINT) FROM OPENJSON(@ids))
//
// Satu parameter untuk berapa pun jumlah ID, tanpa menyusun teks query secara dinamis.
func JSONIDs(ids []int64) string {
	b, _ := json.Marshal(ids) // []int64 selalu berhasil di-marshal
	return string(b)
}

// NullID mengubah ID 0 menjadi NULL untuk parameter SQL.
func NullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// NullString mengubah "" menjadi NULL untuk parameter SQL.
func NullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
