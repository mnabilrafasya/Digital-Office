package audit

// Entry adalah satu kejadian yang dicatat. Snapshot di-encode menjadi JSON kecil.
type Entry struct {
	CompanyCode string
	EntityType  string
	EntityID    int64
	EntityUUID  string
	Action      string
	ActorID     int64
	ActorName   string // disalin, supaya jejak tetap terbaca walau nama user berubah
	IPAddress   string
	Snapshot    any
}
