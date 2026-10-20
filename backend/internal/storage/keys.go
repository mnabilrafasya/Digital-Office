package storage

// stagingPrefix adalah lokasi sementara unggahan sebelum dikonfirmasi.
const stagingPrefix = "staging/"

// StagingKey adalah lokasi unggahan sementara: staging/{objectID}.
func StagingKey(objectID string) string {
	return stagingPrefix + objectID
}

// AttachmentKey adalah lokasi lampiran: letters/{letterUUID}/attachments/{objectID}.
func AttachmentKey(letterUUID, objectID string) string {
	return "letters/" + letterUUID + "/attachments/" + objectID
}

// VersionKey adalah lokasi versi dokumen surat: letters/{letterUUID}/versions/{objectID}.
func VersionKey(letterUUID, objectID string) string {
	return "letters/" + letterUUID + "/versions/" + objectID
}
