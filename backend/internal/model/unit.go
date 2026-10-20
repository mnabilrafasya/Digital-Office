package model

// Unit adalah unit kerja beserta nama jabatan pimpinannya.
type Unit struct {
	ID               int64   `json:"id,string"`
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	UnitType         string  `json:"unitType"`
	ParentID         *int64  `json:"parentId,string"`
	HeadPositionName *string `json:"headPositionName"`
}

// UnitRef adalah unit untuk validasi referensi dan label tampilan (milik perusahaan user).
type UnitRef struct {
	ID       int64
	Code     string
	Name     string
	IsActive bool
}

// UnitLabel adalah unit yang ditampilkan di detail/daftar surat.
type UnitLabel struct {
	ID   int64  `json:"id,string"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// TemplateRef adalah template surat untuk validasi dan label tampilan.
type TemplateRef struct {
	ID       int64  `json:"id,string"`
	Name     string `json:"name"`
	IsActive bool   `json:"-"`
}
