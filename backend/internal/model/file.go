package model

import "time"

// Jenis unggahan.
const (
	UploadAttachment = "ATTACHMENT"
	UploadVersion    = "VERSION"
)

// Attachment adalah satu elemen JSON di kolom letters.attachments. ID sama dengan uuid object.
type Attachment struct {
	ID          string    `json:"id"`
	FileName    string    `json:"fileName"`
	Size        int64     `json:"size"`
	ContentType string    `json:"contentType"`
	Checksum    string    `json:"checksum"`
	UploadedAt  time.Time `json:"uploadedAt"`
	UploadedBy  string    `json:"uploadedBy"`
}

// Version adalah satu baris letter_versions.
type Version struct {
	ID            int64
	LetterID      int64
	VersionNumber int
	VersionType   string
	ObjectKey     string
	FileName      string
	FileExtension string
	FileSize      int64
	Checksum      string
	CreatedBy     PersonRef
	CreatedByRole string
	CreatedAt     time.Time
}

// NewVersion adalah versi dokumen yang akan disimpan.
type NewVersion struct {
	LetterID      int64
	VersionNumber int
	ObjectKey     string
	FileName      string
	FileExtension string
	FileSize      int64
	Checksum      string
	CreatedBy     int64
	CreatedByRole string
}

// StoredObject adalah hasil pemeriksaan object di storage.
type StoredObject struct {
	Size        int64
	ContentType string
}

// ---- Bentuk JSON untuk klien ----

// AttachmentView adalah lampiran untuk klien.
type AttachmentView struct {
	ID          string    `json:"id"`
	FileName    string    `json:"fileName"`
	Size        int64     `json:"size"`
	ContentType string    `json:"contentType"`
	Checksum    string    `json:"checksum"`
	UploadedAt  string    `json:"uploadedAt"`
	UploadedBy  PersonRef `json:"uploadedBy"`
}

// VersionView adalah versi dokumen untuk klien.
type VersionView struct {
	VersionNumber int       `json:"versionNumber"`
	VersionType   string    `json:"versionType"`
	FileName      string    `json:"fileName"`
	FileExtension string    `json:"fileExtension"`
	Size          int64     `json:"size"`
	Checksum      string    `json:"checksum"`
	CreatedBy     PersonRef `json:"createdBy"`
	CreatedByRole string    `json:"createdByRole"`
	CreatedAt     string    `json:"createdAt"`
}

// UploadTicket adalah izin unggah langsung ke storage (presigned POST).
type UploadTicket struct {
	UploadURL    string            `json:"uploadUrl"`
	Method       string            `json:"method"`
	Fields       map[string]string `json:"fields"`
	UploadToken  string            `json:"uploadToken"`
	ExpiresAt    string            `json:"expiresAt"`
	MaxSizeBytes int64             `json:"maxSizeBytes"`
}

// DownloadLink adalah URL unduh sementara (presigned GET).
type DownloadLink struct {
	URL       string `json:"url"`
	FileName  string `json:"fileName"`
	ExpiresAt string `json:"expiresAt"`
}
