package domain

type ContentWritten struct {
	Path      string
	AccountID string
	SizeBytes int64
	Checksum  string
}

type ContentDeleted struct {
	Path      string
	AccountID string
}
