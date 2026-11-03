package audit

import (
	"encoding/json"
	"time"
)

// Semua ID numerik dikirim ke klien sebagai string (tag `json:",string"`).

// PersonRef adalah ringkasan user beserta jabatannya.
type PersonRef struct {
	ID           int64  `json:"id,string"`
	Name         string `json:"name"`
	PositionName string `json:"positionName"`
}

// ---- Lacak Surat ----

// Tracking adalah posisi dan perjalanan satu surat.
type Tracking struct {
	LetterID     string             `json:"letterId"` // uuid
	LetterNumber *string            `json:"letterNumber"`
	Title        *string            `json:"title"`
	Status       string             `json:"status"`
	Position     Position           `json:"position"`
	Timeline     []TimelineEvent    `json:"timeline"`
	Distribution []DistributionNode `json:"distribution"` // kosong sampai surat terbit
	Scope        string             `json:"scope"`        // FULL (drafter/peserta) atau RECIPIENT (penerima inbox)
}

// Position adalah tempat surat saat ini. Holder nil bila surat tidak sedang dipegang siapa pun
// (sudah terbit atau dibatalkan).
type Position struct {
	Label        string     `json:"label"` // contoh "Sedang diperiksa Reviewer 2"
	Holder       *PersonRef `json:"holder"`
	Role         *string    `json:"role"` // DRAFTER, REVIEWER, atau APPROVER
	Step         *int       `json:"step"` // urutan reviewer
	Since        time.Time  `json:"since"`
	WaitingHours *int       `json:"waitingHours"` // lama surat di tangan Holder
	Stuck        bool       `json:"stuck"`        // menunggu >= StuckAfterHours
}

// TimelineEvent adalah satu kejadian pada surat, urut dari yang paling lama.
type TimelineEvent struct {
	At     time.Time  `json:"at"`
	Action string     `json:"action"`
	Label  string     `json:"label"` // kalimat siap tampil
	Actor  PersonRef  `json:"actor"`
	Role   *string    `json:"role"`
	Target *PersonRef `json:"target"`
	Note   *string    `json:"note"`
}

// DistributionNode adalah satu penerima surat terbit beserta penerusan darinya (berjenjang).
type DistributionNode struct {
	InboxItemID int64              `json:"inboxItemId,string"`
	Recipient   PersonRef          `json:"recipient"`
	Via         string             `json:"via"` // TO, CC, DISPOSITION, atau REDIRECT
	From        *PersonRef         `json:"from"`
	Note        *string            `json:"note"`
	Status      string             `json:"status"` // UNREAD, READ, DISPOSED, REDIRECTED
	ReceivedAt  time.Time          `json:"receivedAt"`
	ReadAt      *time.Time         `json:"readAt"`
	Forwards    []DistributionNode `json:"forwards"`
}

// ---- Audit Log ----

// LogEntry adalah satu baris audit log. Snapshot hanya terisi di detail.
type LogEntry struct {
	ID         int64           `json:"id,string"`
	At         time.Time       `json:"at"`
	Action     string          `json:"action"`
	EntityType string          `json:"entityType"`
	EntityID   *string         `json:"entityId"` // uuid surat
	Actor      LogActor        `json:"actor"`
	Letter     *LogLetter      `json:"letter"`
	IPAddress  *string         `json:"ipAddress"`
	RequestID  *string         `json:"requestId"`
	Snapshot   json.RawMessage `json:"snapshot,omitempty"`
}

// LogActor adalah pelaku. ID nil berarti sistem.
type LogActor struct {
	ID   *int64 `json:"id,string"`
	Name string `json:"name"`
}

// LogLetter adalah ringkasan surat terkait (data terbaru). Title disamarkan untuk surat Terbatas
// atau Rahasia bila pembaca bukan drafter/peserta surat itu.
type LogLetter struct {
	Number *string `json:"number"`
	Title  *string `json:"title"`
	Masked bool    `json:"masked"`
}

// ---- Pantauan ----

// MonitoredLetter adalah surat yang sedang diproses beserta pemegang dan lama menunggunya.
type MonitoredLetter struct {
	LetterID     string     `json:"letterId"`
	Title        *string    `json:"title"`
	TitleMasked  bool       `json:"titleMasked"`
	Status       string     `json:"status"`
	SenderUnit   string     `json:"senderUnit"`
	Drafter      PersonRef  `json:"drafter"`
	Holder       *PersonRef `json:"holder"`
	Step         *int       `json:"step"`
	Since        time.Time  `json:"since"`
	WaitingHours int        `json:"waitingHours"`
	Stuck        bool       `json:"stuck"`
}
