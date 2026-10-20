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
