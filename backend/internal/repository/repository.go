// Package repository berisi query SQL saja, tanpa aturan bisnis.
// Semua input dikirim sebagai parameter SQL (sql.Named), tidak pernah disambung ke teks query.
// Daftar ID dikirim sebagai satu parameter JSON lalu dibaca dengan OPENJSON.
package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrNotFound dikembalikan bila data yang dicari tidak ada.
var ErrNotFound = errors.New("data tidak ditemukan")

// likeEscaper meng-escape karakter khusus LIKE di SQL Server. "\" harus di-escape lebih dulu
// (strings.NewReplacer memproses setiap karakter input sekali, jadi urutannya aman).
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`, `[`, `\[`)

// ContainsPattern mengubah teks pencarian menjadi pola LIKE "%teks%" yang aman.
// Pola ini wajib dipakai bersama `ESCAPE '\'` di query.
func ContainsPattern(q string) string {
	return "%" + likeEscaper.Replace(q) + "%"
}

// jsonParam mengubah nilai (biasanya slice) menjadi parameter JSON untuk OPENJSON.
func jsonParam(name string, v any) (sql.NamedArg, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return sql.NamedArg{}, fmt.Errorf("menyusun parameter %s: %w", name, err)
	}
	return sql.Named(name, string(b)), nil
}

// idsParam mengubah daftar ID menjadi parameter JSON array. Slice nil dikirim sebagai "[]".
func idsParam(name string, ids []int64) sql.NamedArg {
	if ids == nil {
		ids = []int64{}
	}
	p, _ := jsonParam(name, ids) // []int64 selalu bisa di-marshal
	return p
}

// idsIn adalah potongan SQL untuk "kolom IN (daftar ID dari parameter JSON)".
func idsIn(param string) string {
	return "(SELECT CAST([value] AS BIGINT) FROM OPENJSON(@" + param + "))"
}
