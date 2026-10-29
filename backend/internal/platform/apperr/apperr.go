// Package apperr berisi jenis error aplikasi yang dipakai semua fitur. Service mengembalikan error
// ini tanpa mengenal HTTP; httpx.WriteError yang menerjemahkannya menjadi response.
// Message setiap error aman ditampilkan ke pengguna.
package apperr

import "errors"

// ValidationError menandakan input dari klien tidak valid (400).
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// Invalid membuat ValidationError.
func Invalid(message string) error { return &ValidationError{Message: message} }

// NotFoundError menandakan data tidak ada atau user tidak berhak melihatnya (404).
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string { return e.Message }

// ErrForbidden: user boleh melihat data, tetapi tidak boleh melakukan tindakan ini (403).
var ErrForbidden = errors.New("Anda tidak memiliki akses untuk tindakan ini")

// InvalidStateError menandakan tindakan tidak bisa dilakukan pada status data saat ini (409).
type InvalidStateError struct {
	Message string
}

func (e *InvalidStateError) Error() string { return e.Message }

// UnavailableError menandakan layanan pendukung (mis. pembuat PDF) sedang tidak bisa dihubungi
// (503). Cause adalah penyebab teknisnya: dicatat di log, tidak dikirim ke klien.
type UnavailableError struct {
	Message string
	Cause   error
}

func (e *UnavailableError) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return e.Message + ": " + e.Cause.Error()
}

func (e *UnavailableError) Unwrap() error { return e.Cause }

// VersionConflictError menandakan If-Match tidak sama dengan versi terbaru (412).
// RowVersion adalah versi terbaru, dikirim ke klien sebagai ETag.
type VersionConflictError struct {
	RowVersion []byte
}

func (e *VersionConflictError) Error() string {
	return "Surat sudah diubah di tempat lain. Muat ulang untuk melihat versi terbaru."
}
