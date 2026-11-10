package organisasi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Inisiasi unit kerja dari API HC PUSRI (hc/organization). HC hanya dipakai untuk mengisi struktur
// organisasi: sinkron hanya MENAMBAH unit yang belum ada. Unit yang sudah ada tidak diubah dan tidak
// dinonaktifkan, karena setelah inisiasi unit dikelola admin perusahaan (kelola_unit.go). Unit
// dicocokkan lewat UID HC yang disimpan di units.code; kode numerik HC tidak dipakai karena dipakai
// ulang antar-reorganisasi.

// HCUnit adalah satu unit dari HC (isi field "data" setelah base64 di-decode).
type HCUnit struct {
	UID       string `json:"uid"`
	ParentUID string `json:"uid_parent"` // "#" = unit paling atas
	Text      string `json:"text"`
	Level     int    `json:"level"`
	DescLevel string `json:"desc_level"` // DIREKTORAT, KOMPARTEMEN, DEPARTEMEN, BAGIAN
	Status    string `json:"status"`     // active atau in-active
}

// SyncResult adalah ringkasan sinkronisasi.
type SyncResult struct {
	Added    int // unit baru dari HC
	Existing int // unit yang sudah ada (tidak diubah)
}

// ParseHCUnits membaca snapshot unit HC berupa array JSON (boleh diawali BOM).
func ParseHCUnits(raw []byte) ([]HCUnit, error) {
	var units []HCUnit
	if err := json.Unmarshal(bytes.TrimPrefix(raw, []byte("\xEF\xBB\xBF")), &units); err != nil {
		return nil, fmt.Errorf("membaca data unit HC: %w", err)
	}
	return units, validateHCUnits(units)
}

// DecodeHCPage membaca satu halaman response API HC: {"data": "<base64 array JSON>", "meta": {"total": n}}.
func DecodeHCPage(body []byte) ([]HCUnit, int, error) {
	var page struct {
		Data string `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, 0, fmt.Errorf("membaca response HC: %w", err)
	}
	raw, err := base64.StdEncoding.DecodeString(page.Data)
	if err != nil {
		return nil, 0, fmt.Errorf("decode base64 data HC: %w", err)
	}
	var units []HCUnit
	if err := json.Unmarshal(raw, &units); err != nil {
		return nil, 0, fmt.Errorf("membaca data unit HC: %w", err)
	}
	return units, page.Meta.Total, nil
}

func validateHCUnits(units []HCUnit) error {
	seen := make(map[string]bool, len(units))
	for _, u := range units {
		switch {
		case len(u.UID) != 36:
			return fmt.Errorf("unit HC %q: uid tidak valid", u.Text)
		case seen[u.UID]:
			return fmt.Errorf("unit HC %q: uid dobel", u.Text)
		case strings.TrimSpace(u.Text) == "":
			return fmt.Errorf("unit HC %s: nama kosong", u.UID)
		}
		seen[u.UID] = true
	}
	for _, u := range units {
		if u.ParentUID != "#" && !seen[u.ParentUID] {
			return fmt.Errorf("unit HC %q: induk %s tidak ada di data", u.Text, u.ParentUID)
		}
	}
	return nil
}

// Aturan penyeragaman nama unit, dijalankan berurutan.
var (
	unitPrefixes = []struct {
		re   *regexp.Regexp
		repl string
	}{
		{regexp.MustCompile(`(?i)^(departemen|dept\.?|dep\.?)\s+`), "Departemen "},
		{regexp.MustCompile(`(?i)^(kompartemen|komp\.)\s+`), "Kompartemen "},
		{regexp.MustCompile(`(?i)^(bagian|bag\.)\s+`), "Bagian "},
		{regexp.MustCompile(`(?i)^(fungsional|fungs\.)\s+`), "Fungsional "},
		{regexp.MustCompile(`(?i)^(kelompok|kel\.)\s+`), "Kelompok "},
	}
	unitAbbreviations = []struct {
		re   *regexp.Regexp
		repl string
	}{
		{regexp.MustCompile(`\bTK & Manrisk\b`), "Tata Kelola & Manajemen Risiko"},
		{regexp.MustCompile(`\bManrisk\b`), "Manajemen Risiko"},
		{regexp.MustCompile(`\bRendal &`), "Perencanaan, Pengendalian &"}, // hindari "&" berurutan
		{regexp.MustCompile(`\bRendal\b`), "Perencanaan & Pengendalian"},
		{regexp.MustCompile(`\bPerenc\b\.?`), "Perencanaan"},
		{regexp.MustCompile(`\bAdm\.`), "Administrasi"},
		{regexp.MustCompile(`\bAdmin\b`), "Administrasi"},
		{regexp.MustCompile(`\b(Manaj|Manj)\b\.?`), "Manajemen"},
		{regexp.MustCompile(`\bSisman\b`), "Sistem Manajemen"},
		{regexp.MustCompile(`\bLH\b`), "Lingkungan Hidup"},
	}
	ampersand = regexp.MustCompile(`\s*&\s*`)
	comma     = regexp.MustCompile(`\s*,\s*`)
	spaces    = regexp.MustCompile(`\s+`)
)

// NormalizeUnitName menyeragamkan nama unit HC: awalan ditulis lengkap (Dep./Dept. -> Departemen,
// Komp. -> Kompartemen, Bag. -> Bagian, Fungs. -> Fungsional, Kel. -> Kelompok), nama berhuruf
// besar semua diubah ke huruf kapital di awal kata, singkatan umum diperpanjang, dan spasi di
// sekitar "&" serta "," dirapikan. Istilah resmi (TJSL, HSSE, K3LH, P-IIIB, ...) dibiarkan.
func NormalizeUnitName(name string) string {
	s := spaces.ReplaceAllString(strings.TrimSpace(name), " ")
	if s == strings.ToUpper(s) && s != strings.ToLower(s) {
		s = titleCase(s)
	}
	for _, p := range unitPrefixes {
		if p.re.MatchString(s) {
			s = p.re.ReplaceAllString(s, p.repl)
			break
		}
	}
	for _, a := range unitAbbreviations {
		s = a.re.ReplaceAllString(s, a.repl)
	}
	s = ampersand.ReplaceAllString(s, " & ")
	s = comma.ReplaceAllString(s, ", ")
	return strings.TrimSpace(spaces.ReplaceAllString(s, " "))
}

// titleCase mengubah "DEPARTEMEN HUKUM" menjadi "Departemen Hukum".
func titleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// existingUnit adalah unit perusahaan yang sudah ada di database.
type existingUnit struct {
	ID       int64
	ParentID sql.NullInt64
	Name     string
	Type     string
	Active   bool
}

// SyncHCUnits menambah unit HC yang belum ada di companyCode; unit yang sudah ada tidak diubah. Wajib
// dipanggil di dalam transaksi (Repository dibuat di atas transaksi), supaya berhasil seluruhnya atau
// tidak sama sekali.
func (r *Repository) SyncHCUnits(ctx context.Context, companyCode string, units []HCUnit) (SyncResult, error) {
	var res SyncResult
	if err := validateHCUnits(units); err != nil {
		return res, err
	}
	existing, err := r.unitsByCode(ctx, companyCode)
	if err != nil {
		return res, err
	}

	// Induk lebih dulu (level kecil), supaya urutan ID mengikuti struktur
	sorted := append([]HCUnit(nil), units...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Level < sorted[j].Level })

	ids := make(map[string]int64, len(units))
	added := map[string]bool{}
	for _, u := range sorted {
		if e, ok := existing[u.UID]; ok {
			ids[u.UID] = e.ID
			res.Existing++
			continue
		}
		name, typ, active := NormalizeUnitName(u.Text), strings.ToUpper(u.DescLevel), u.Status == "active"
		const q = `
INSERT INTO units (company_code, parent_id, unit_type, code, name, is_active)
OUTPUT inserted.id
VALUES (@companyCode, NULL, @type, @code, @name, @active)`
		var id int64
		if err := r.q.QueryRowContext(ctx, q, sql.Named("companyCode", companyCode), sql.Named("type", typ),
			sql.Named("code", u.UID), sql.Named("name", name), sql.Named("active", active)).Scan(&id); err != nil {
			return res, fmt.Errorf("menambah unit %q: %w", name, err)
		}
		ids[u.UID] = id
		added[u.UID] = true
		res.Added++
	}

	// Induk unit baru disusun setelah semua unit punya ID (induknya bisa unit lama atau baru)
	for _, u := range units {
		if !added[u.UID] || u.ParentUID == "#" {
			continue
		}
		const q = `UPDATE units SET parent_id = @parentId, updated_at = SYSDATETIMEOFFSET() WHERE id = @id`
		if _, err := r.q.ExecContext(ctx, q, sql.Named("parentId", ids[u.ParentUID]), sql.Named("id", ids[u.UID])); err != nil {
			return res, fmt.Errorf("menyusun induk unit %q: %w", u.Text, err)
		}
	}
	return res, nil
}

func (r *Repository) unitsByCode(ctx context.Context, companyCode string) (map[string]existingUnit, error) {
	const q = `
SELECT id, code, parent_id, name, unit_type, is_active
FROM units
WHERE company_code = @companyCode AND deleted_at IS NULL`
	rows, err := r.q.QueryContext(ctx, q, sql.Named("companyCode", companyCode))
	if err != nil {
		return nil, fmt.Errorf("membaca unit %s: %w", companyCode, err)
	}
	defer rows.Close()
	out := map[string]existingUnit{}
	for rows.Next() {
		var (
			e    existingUnit
			code string
		)
		if err := rows.Scan(&e.ID, &code, &e.ParentID, &e.Name, &e.Type, &e.Active); err != nil {
			return nil, fmt.Errorf("membaca unit %s: %w", companyCode, err)
		}
		out[strings.ToLower(code)] = e
	}
	return out, rows.Err()
}
