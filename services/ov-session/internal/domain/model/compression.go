package model

type CompressionVersion string

const (
	CompressionVersionV1 CompressionVersion = "v1"
	CompressionVersionV2 CompressionVersion = "v2"
)

type ExtractionStats struct {
	TotalExtracted int            `json:"total_extracted"`
	ByCategory     map[string]int `json:"by_category"`
	TokensUsed     int            `json:"tokens_used"`
}

type SessionCompression struct {
	ArchivePath        string
	CompressionVersion CompressionVersion
	MemoriesCount      int
	ExtractionStats    ExtractionStats
}
