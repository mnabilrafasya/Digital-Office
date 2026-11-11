package masterdata

// Lookup adalah satu pilihan dropdown (jenis surat, sifat, kecepatan, klasifikasi, status).
type Lookup struct {
	ID        int64   `json:"id,string"`
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Color     *string `json:"color"`
	Note      *string `json:"note"`
	SortOrder int     `json:"sortOrder"`
}

// Kode jenis surat (lookup LETTER_TYPE) yang mengubah alur surat.
const (
	// TypeExternalOut: surat keluar ke perusahaan/instansi luar. Setelah approver terakhir setuju
	// statusnya APPROVED (menunggu tanda tangan basah), lalu SENT saat scan diunggah; tidak masuk
	// inbox penerima, tidak memakai QR.
	TypeExternalOut = "SURAT_EKSTERNAL"
	// TypeExternalIn: surat masuk dari luar, dicatat lewat form surat masuk (status RECEIVED).
	TypeExternalIn = "SURAT_MASUK_EKSTERNAL"
)

// Arah surat, dihitung dari jenis surat.
const (
	DirectionInternal = "INTERNAL"
	DirectionOutgoing = "OUTGOING" // Pusri ke luar
	DirectionIncoming = "INCOMING" // luar ke Pusri
)

// Direction mengembalikan arah surat dari kode jenis suratnya.
func Direction(letterTypeCode string) string {
	switch letterTypeCode {
	case TypeExternalOut:
		return DirectionOutgoing
	case TypeExternalIn:
		return DirectionIncoming
	}
	return DirectionInternal
}
