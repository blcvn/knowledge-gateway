package domain

type SearchStrategy string

const (
	Similarity                   SearchStrategy = "SIMILARITY"
	GraphCompletion              SearchStrategy = "GRAPH_COMPLETION"
	RAGCompletion                SearchStrategy = "RAG_COMPLETION"
	NaturalLanguage              SearchStrategy = "NATURAL_LANGUAGE"
	Chunks                       SearchStrategy = "CHUNKS"
	ChunksLexical                SearchStrategy = "CHUNKS_LEXICAL"
	Summaries                    SearchStrategy = "SUMMARIES"
	TripletCompletion            SearchStrategy = "TRIPLET_COMPLETION"
	GraphCompletionCoT           SearchStrategy = "GRAPH_COMPLETION_COT"
	GraphCompletionDecomposition SearchStrategy = "GRAPH_COMPLETION_DECOMPOSITION"
	GraphCompletionContextExt    SearchStrategy = "GRAPH_COMPLETION_CONTEXT_EXTENSION"
	GraphSummaryCompletion       SearchStrategy = "GRAPH_SUMMARY_COMPLETION"
	Cypher                       SearchStrategy = "CYPHER"
	Temporal                     SearchStrategy = "TEMPORAL"
	FeelingLucky                 SearchStrategy = "FEELING_LUCKY"
)

type ResultType string

const (
	ResultTypeChunk    ResultType = "CHUNK"
	ResultTypeGraph    ResultType = "GRAPH"
	ResultTypeHybrid   ResultType = "HYBRID"
	ResultTypeCypher   ResultType = "CYPHER"
	ResultTypeSummary  ResultType = "SUMMARY"
	ResultTypeTemporal ResultType = "TEMPORAL"
)

type SearchScope struct {
	TenantID  string
	DatasetID string
}

type SearchFilters struct {
	TenantID    string
	DatasetID   string
	TimeRange   *TimeRange
	EntityTypes []string
}

type TimeRange struct {
	Start int64
	End   int64
}
