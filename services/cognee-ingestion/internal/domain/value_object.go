package domain

// DatasetStatus represents the lifecycle state of a dataset.
type DatasetStatus string

const (
	DatasetPending    DatasetStatus = "PENDING"
	DatasetReady      DatasetStatus = "READY"
	DatasetCognifying DatasetStatus = "COGNIFYING"
	DatasetError      DatasetStatus = "ERROR"
)

// IsTerminal returns true if the status is a final state.
func (s DatasetStatus) IsTerminal() bool {
	return s == DatasetReady || s == DatasetError
}

// String implements fmt.Stringer.
func (s DatasetStatus) String() string { return string(s) }

// DataSource identifies the origin type of ingested data.
type DataSource string

const (
	SourceFile DataSource = "FILE"
	SourceText DataSource = "TEXT"
	SourceURL  DataSource = "URL"
)

// String implements fmt.Stringer.
func (s DataSource) String() string { return string(s) }

// MimeType represents supported content types for ingestion.
type MimeType string

const (
	MimePDF       MimeType = "application/pdf"
	MimeDOCX      MimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	MimePPTX      MimeType = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	MimeCSV       MimeType = "text/csv"
	MimeTSV       MimeType = "text/tab-separated-values"
	MimeHTML      MimeType = "text/html"
	MimePlainText MimeType = "text/plain"
	MimeMarkdown  MimeType = "text/markdown"
	MimeJSON      MimeType = "application/json"
)

// supportedMimeTypes lists all MimeTypes the ingestion service can extract text from.
var supportedMimeTypes = map[MimeType]bool{
	MimePDF:       true,
	MimeDOCX:      true,
	MimePPTX:      true,
	MimeCSV:       true,
	MimeTSV:       true,
	MimeHTML:      true,
	MimePlainText: true,
	MimeMarkdown:  true,
	MimeJSON:      true,
}

// IsSupported returns true if the MimeType is supported for text extraction.
func (m MimeType) IsSupported() bool {
	return supportedMimeTypes[m]
}

// String implements fmt.Stringer.
func (m MimeType) String() string { return string(m) }

// SupportedMimeTypes returns all supported MIME types.
func SupportedMimeTypes() []MimeType {
	types := make([]MimeType, 0, len(supportedMimeTypes))
	for mt := range supportedMimeTypes {
		types = append(types, mt)
	}
	return types
}
