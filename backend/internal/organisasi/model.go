package organisasi

// Unit adalah unit kerja beserta nama jabatan pimpinannya.
type Unit struct {
	ID               int64   `json:"id,string"`
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	UnitType         string  `json:"unitType"`
	ParentID         *int64  `json:"parentId,string"`
	HeadPositionName *string `json:"headPositionName"`
}

// Candidate adalah user yang bisa dipilih sebagai reviewer atau approver.
type Candidate struct {
	ID             int64  `json:"id,string"`
	Name           string `json:"name"`
	EmployeeNumber string `json:"employeeNumber"`
	PositionName   string `json:"positionName"`
	UnitName       string `json:"unitName"`
}

// Alias adalah alias penerima berdasarkan level jabatan, contoh "Seluruh VP".
type Alias struct {
	LevelID     int64  `json:"levelId,string"`
	Code        string `json:"code"`
	Label       string `json:"label"`
	MemberCount int    `json:"memberCount"`
}
