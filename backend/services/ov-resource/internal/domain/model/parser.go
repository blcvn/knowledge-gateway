package model

type ParserType string

const (
	ParserTypeTreeSitter ParserType = "treesitter"
	ParserTypeMarkdown   ParserType = "markdown"
	ParserTypeDocument   ParserType = "document"
	ParserTypeDefault    ParserType = "default"
)

type ParserConfig struct {
	ChunkSizeTokens    int
	ChunkOverlapTokens int
	TreesitterEnabled  bool
}
