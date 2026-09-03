package model

type Chunk struct {
	ID       string
	Content  string
	Metadata ChunkMetadata
}

type ChunkMetadata struct {
	StartLine   int
	EndLine     int
	TotalTokens int
	ASTNodeType string
	ASTNodePath string
}
