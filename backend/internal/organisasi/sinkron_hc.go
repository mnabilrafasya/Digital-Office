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

// Sinkronisasi unit kerja dari API HC PUSRI (hc/organization). Unit dicocokkan lewat UID HC yang
// disimpan di units.code; kode numerik HC tidak dipakai karena dipakai ulang antar-reorganisasi.
// Unit tidak pernah dihapus: unit yang hilang dari HC atau berstatus in-active dinonaktifkan,
// karena surat lama bisa merujuknya.

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
	Added, Updated, Deactivated, Unchanged int
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

// SyncHCUnits menyamakan unit companyCode dengan data HC. Wajib dipanggil di dalam transaksi
// (Repository dibuat di atas transaksi), supaya sinkronisasi berhasil seluruhnya atau tidak sama sekali.
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
	changed := map[string]bool{}
	for _, u := range sorted {
		name, typ, active := NormalizeUnitName(u.Text), strings.ToUpper(u.DescLevel), u.Status == "active"
		if e, ok := existing[u.UID]; ok {
			ids[u.UID] = e.ID
			if e.Name != name || e.Type != typ || e.Active != active {
				const q = `UPDATE units SET name = @name, unit_type = @type, is_active = @active, updated_at = SYSDATETIMEOFFSET() WHERE id = @id`
				if _, err := r.q.ExecContext(ctx, q, sql.Named("name", name), sql.Named("type", typ),
					sql.Named("active", active), sql.Named("id", e.ID)); err != nil {
					return res, fmt.Errorf("memperbarui unit %q: %w", name, err)
				}
				changed[u.UID] = true
			}
			continue
		}
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
		res.Added++
	}

	// Induk disusun setelah semua unit punya ID
	for _, u := range units {
		var want sql.NullInt64
		if u.ParentUID != "#" {
			want = sql.NullInt64{Int64: ids[u.ParentUID], Valid: true}
		}
		if e, ok := existing[u.UID]; ok && e.ParentID == want {
			continue
		}
		const q = `UPDATE units SET parent_id = @parentId, updated_at = SYSDATETIMEOFFSET() WHERE id = @id`
		if _, err := r.q.ExecContext(ctx, q, sql.Named("parentId", want), sql.Named("id", ids[u.UID])); err != nil {
			return res, fmt.Errorf("menyusun induk unit %q: %w", u.Text, err)
		}
		if _, isOld := existing[u.UID]; isOld {
			changed[u.UID] = true
		}
	}
	res.Updated = len(changed)
	res.Unchanged = len(units) - res.Added - res.Updated

	// Unit yang tidak ada lagi di HC dinonaktifkan, tidak dihapus
	for code, e := range existing {
		if _, still := ids[code]; still || !e.Active {
			continue
		}
		const q = `UPDATE units SET is_active = 0, updated_at = SYSDATETIMEOFFSET() WHERE id = @id`
		if _, err := r.q.ExecContext(ctx, q, sql.Named("id", e.ID)); err != nil {
			return res, fmt.Errorf("menonaktifkan unit %q: %w", e.Name, err)
		}
		res.Deactivated++
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
