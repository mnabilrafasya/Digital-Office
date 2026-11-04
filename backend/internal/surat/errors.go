package surat

import (
	"office.backend/internal/platform/apperr"
)

// ErrLetterNotFound: surat tidak ada, sudah dihapus, atau user tidak berhak melihatnya (404,
// supaya keberadaan surat tidak bocor).
var ErrLetterNotFound error = &apperr.NotFoundError{Message: "Surat tidak ditemukan"}

// errTemplateNotFound: template tidak ada, tidak aktif, atau milik perusahaan lain (404).
var errTemplateNotFound error = &apperr.NotFoundError{Message: "Template tidak ditemukan"}

// errTemplateNameTaken: nama template aktif harus unik per perusahaan (409).
var errTemplateNameTaken error = &apperr.InvalidStateError{Message: "Nama template sudah dipakai. Gunakan nama lain."}

// errAttachmentNotFound: lampiran tidak ada di surat ini (404).
var errAttachmentNotFound error = &apperr.NotFoundError{Message: "Lampiran tidak ditemukan"}
