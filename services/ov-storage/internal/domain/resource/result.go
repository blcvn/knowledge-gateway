package resource

// ParseResult represents the output of a parsing operation.
type ParseResult struct {
	SourcePath string    `json:"source_path"`
	ParserType ParserType `json:"parser_type"`
	Sections   []Section  `json:"sections"`
}
