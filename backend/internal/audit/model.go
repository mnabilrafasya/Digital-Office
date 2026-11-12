package audit

// Entry adalah satu kejadian yang dicatat. Snapshot di-encode menjadi JSON kecil.
type Entry struct {
	CompanyCode string
	EntityType  string
	EntityID    int64
	EntityUUID  string
	Action      string
	ActorID     int64  // 0 = dilakukan sistem
	ActorName   string // disalin, supaya jejak tetap terbaca walau nama user berubah; "" = SystemActor
	IPAddress   string // "" = diambil dari request yang sedang berjalan
	Snapshot    any
}

// SystemActor adalah nama pelaku untuk kejadian yang dilakukan sistem (misalnya pengiriman ke
// inbox saat surat terbit, atau pembuatan PDF resmi).
const SystemActor = "Sistem"

// Jenis data yang dicatat (kolom entity_type).
const (
	EntityLetter     = "LETTER"
	EntityTemplate   = "TEMPLATE"   // template surat
	EntityAudit      = "AUDIT"      // tindakan atas audit log itu sendiri, misalnya ekspor
	EntityCompany    = "COMPANY"    // perusahaan (entity_id 0; kodenya di snapshot)
	EntityUnit       = "UNIT"       // unit kerja
	EntityPosition   = "POSITION"   // jabatan
	EntityUser       = "USER"       // user
	EntityDelegation = "DELEGATION" // jadwal delegasi tugas
)

// Aksi yang dicatat (kolom action). Transisi alur surat memakai nama aksinya sendiri
// (SUBMIT, APPROVE, RETURN, SIGN).
const (
	ActionCreate           = "CREATE"            // draf surat dibuat
	ActionSave             = "SAVE"              // naskah .docx disimpan
	ActionAttachmentAdd    = "ATTACHMENT_ADD"    // lampiran surat diunggah
	ActionAttachmentDelete = "ATTACHMENT_DELETE" // lampiran surat dihapus
	ActionFinalPDF         = "FINAL_PDF"         // PDF resmi ber-QR dibuat
	ActionDownload         = "DOWNLOAD"          // dokumen atau lampiran surat diunduh
	ActionView             = "VIEW"              // surat Rahasia dibuka
	ActionDeliver          = "DELIVER"           // surat terbit dikirim ke inbox penerima
	ActionInboxRead        = "INBOX_READ"        // surat masuk dibuka pertama kali
	ActionDisposition      = "DISPOSITION"       // surat didisposisikan
	ActionRedirect         = "REDIRECT"          // surat dialihkan
	ActionTemplateCreate   = "TEMPLATE_CREATE"   // template surat dibuat
	ActionTemplateUpdate   = "TEMPLATE_UPDATE"   // data atau file template surat diganti
	ActionExport           = "AUDIT_EXPORT"      // audit log diekspor
	ActionTemplateStatus   = "TEMPLATE_STATUS"   // template diaktifkan/dinonaktifkan admin
	ActionTemplateDelete   = "TEMPLATE_DELETE"   // template surat dihapus admin
	ActionSignedUpload     = "UPLOAD_SIGNED"     // scan surat eksternal bertanda tangan basah diunggah
	ActionIncoming         = "INCOMING_RECEIVE"  // surat masuk eksternal dicatat dan didisposisikan
)

// Aksi master data (kolom action). Snapshot berisi data baru, dan data lama untuk perubahan.
const (
	ActionCompanyCreate   = "COMPANY_CREATE"   // perusahaan ditambah (SUPERADMIN)
	ActionCompanyUpdate   = "COMPANY_UPDATE"   // identitas perusahaan diubah (SUPERADMIN)
	ActionCompanyStatus   = "COMPANY_STATUS"   // perusahaan diaktifkan/dinonaktifkan (SUPERADMIN)
	ActionCompanyDelete   = "COMPANY_DELETE"   // perusahaan dihapus (SUPERADMIN)
	ActionCompanySettings = "COMPANY_SETTINGS" // watermark/format nomor diubah (ADMIN)
	ActionUnitCreate      = "UNIT_CREATE"
	ActionUnitUpdate      = "UNIT_UPDATE"
	ActionUnitStatus      = "UNIT_STATUS"
	ActionUnitDelete      = "UNIT_DELETE"
	ActionPositionCreate  = "POSITION_CREATE"
	ActionPositionUpdate  = "POSITION_UPDATE"
	ActionPositionStatus  = "POSITION_STATUS"
	ActionPositionDelete  = "POSITION_DELETE"
	ActionUserCreate      = "USER_CREATE"
	ActionUserUpdate      = "USER_UPDATE"
	ActionUserStatus      = "USER_STATUS"
	ActionUserDelete      = "USER_DELETE"
	ActionUserPassword    = "USER_PASSWORD_RESET"
)

// Aksi delegasi tugas (kolom action). Snapshot berisi jadwal sebelum dan/atau sesudahnya.
const (
	ActionDelegationCreate = "DELEGATION_CREATE"
	ActionDelegationUpdate = "DELEGATION_UPDATE"
	ActionDelegationDelete = "DELEGATION_DELETE"
)
