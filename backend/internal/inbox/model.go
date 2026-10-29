package inbox

import "time"

// Semua ID numerik dikirim ke klien sebagai string (tag `json:",string"`).

// Item adalah satu baris inbox: satu surat terkirim di inbox satu orang.
type Item struct {
	ID         int64      `json:"id,string"`
	Status     string     `json:"status"`
	ReceivedAt time.Time  `json:"receivedAt"`
	ReadAt     *time.Time `json:"readAt"`
	Source     Source     `json:"source"`
	Letter     Letter     `json:"letter"`
}

// Source adalah asal surat masuk ke inbox, untuk kolom Keterangan: "Surat dari" (DIRECT TO),
// "CC dari" (DIRECT CC), atau "Disposisi/Dialihkan dari" (FORWARD).
type Source struct {
	Type          string     `json:"type"`          // DIRECT atau FORWARD
	RecipientType *string    `json:"recipientType"` // TO/CC, hanya untuk DIRECT
	ForwardType   *string    `json:"forwardType"`   // DISPOSITION/REDIRECT, hanya untuk FORWARD
	ForwardedBy   *PersonRef `json:"forwardedBy"`   // hanya untuk FORWARD
	Note          *string    `json:"note"`          // catatan/instruksi disposisi, hanya untuk FORWARD
}

// ItemDetail adalah satu item inbox yang dibuka penerimanya: item, penerusan yang sudah dibuat
// dari item ini, dan aksi yang boleh dilakukan (DISPOSE, REDIRECT).
type ItemDetail struct {
	Item
	Forwards       []Forward `json:"forwards"`
	AllowedActions []string  `json:"allowedActions"`
}

// Forward adalah satu penerusan (disposisi/pengalihan) yang dibuat dari item inbox.
type Forward struct {
	ID         int64      `json:"id,string"`
	Type       string     `json:"type"`       // DISPOSITION atau REDIRECT
	TargetType string     `json:"targetType"` // USER atau UNIT
	TargetUser *PersonRef `json:"targetUser"` // untuk USER
	TargetUnit *UnitRef   `json:"targetUnit"` // untuk UNIT
	Recipient  *PersonRef `json:"recipient"`  // orang yang menerima di inbox (untuk UNIT: pimpinannya)
	Note       *string    `json:"note"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// Letter adalah ringkasan surat yang tampil di inbox.
type Letter struct {
	ID              string     `json:"id"` // uuid
	LetterNumber    *string    `json:"letterNumber"`
	Title           *string    `json:"title"`
	LetterDate      *time.Time `json:"letterDate"` // tanggal surat = waktu disetujui
	LetterType      *LookupRef `json:"letterType"`
	Confidentiality *LookupRef `json:"confidentiality"`
	Urgency         *LookupRef `json:"urgency"`
	SenderUnit      UnitRef    `json:"senderUnit"`
	Approver        *PersonRef `json:"approver"` // penandatangan surat
}

// LookupRef adalah ringkasan data acuan (jenis surat, sifat, kecepatan).
type LookupRef struct {
	ID   int64  `json:"id,string"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// UnitRef adalah ringkasan unit.
type UnitRef struct {
	ID   int64  `json:"id,string"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// PersonRef adalah ringkasan user beserta jabatannya.
type PersonRef struct {
	ID           int64  `json:"id,string"`
	Name         string `json:"name"`
	PositionName string `json:"positionName"`
}

// Counts adalah jumlah item inbox per status, untuk badge tab.
type Counts map[string]int
