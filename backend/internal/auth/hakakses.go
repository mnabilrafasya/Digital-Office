package auth

// Kode permission (hak akses) yang dipakai di seluruh aplikasi.
// Nilainya harus sama dengan kolom permissions.code di database.

const (
	LetterCreate   = "LETTER_CREATE"
	LetterReview   = "LETTER_REVIEW"
	LetterApprove  = "LETTER_APPROVE"
	InboxDispose   = "INBOX_DISPOSE"
	TemplateCreate = "TEMPLATE_CREATE"
	AuditView      = "AUDIT_VIEW"
)
