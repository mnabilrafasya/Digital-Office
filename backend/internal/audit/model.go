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
	EntityLetter   = "LETTER"
	EntityTemplate = "TEMPLATE" // template surat
	EntityAudit    = "AUDIT"    // tindakan atas audit log itu sendiri, misalnya ekspor
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
)
