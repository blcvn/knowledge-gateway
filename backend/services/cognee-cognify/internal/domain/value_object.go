package domain

// JobStatus represents the lifecycle of a CognifyJob.
type JobStatus string

const (
	JobPending   JobStatus = "PENDING"
	JobRunning   JobStatus = "RUNNING"
	JobCompleted JobStatus = "COMPLETED"
	JobFailed    JobStatus = "FAILED"
)

// IsTerminal returns true if the job is in a final state.
func (s JobStatus) IsTerminal() bool {
	return s == JobCompleted || s == JobFailed
}

func (s JobStatus) String() string { return string(s) }

// StageType identifies which pipeline stage is currently executing.
type StageType string

const (
	StageNone                 StageType = ""
	StageClassify             StageType = "classify"
	StageChunk                StageType = "chunk"
	StageExtractEntities      StageType = "extract_entities"
	StageExtractRelationships StageType = "extract_relationships"
	StageDeduplicate          StageType = "deduplicate"
	StageBuildGraph           StageType = "build_graph"
	StageEmbed                StageType = "embed"
	StageSummarize            StageType = "summarize"
)

// AllStages returns the ordered list of all 8 pipeline stages.
func AllStages() []StageType {
	return []StageType{
		StageClassify,
		StageChunk,
		StageExtractEntities,
		StageExtractRelationships,
		StageDeduplicate,
		StageBuildGraph,
		StageEmbed,
		StageSummarize,
	}
}

// StageProgress returns the progress (0.0–1.0) for a given stage index (0-based).
func StageProgress(index int) float64 {
	total := len(AllStages())
	if total == 0 || index < 0 {
		return 0.0
	}
	return float64(index+1) / float64(total)
}

func (s StageType) String() string { return string(s) }

// EntityType classifies the kind of named entity extracted.
type EntityType string

const (
	EntityPerson       EntityType = "PERSON"
	EntityOrganization EntityType = "ORGANIZATION"
	EntityLocation     EntityType = "LOCATION"
	EntityEvent        EntityType = "EVENT"
	EntityConcept      EntityType = "CONCEPT"
	EntityTechnology   EntityType = "TECHNOLOGY"
	EntityProduct      EntityType = "PRODUCT"
	EntityDate         EntityType = "DATE"
	EntityOther        EntityType = "OTHER"
)

func (t EntityType) String() string { return string(t) }

// ChunkingStrategy determines how text is split into chunks.
type ChunkingStrategy string

const (
	StrategyRecursive ChunkingStrategy = "recursive"   // recursive character splitting
	StrategySemantic  ChunkingStrategy = "semantic"     // embedding-based semantic chunking
	StrategySentence  ChunkingStrategy = "sentence"     // sentence-level splitting
	StrategyParagraph ChunkingStrategy = "paragraph"    // paragraph-level splitting
)

func (s ChunkingStrategy) String() string { return string(s) }

// ClassificationResult holds the output of the classify stage.
type ClassificationResult struct {
	ContentType string           `json:"content_type"` // technical, narrative, tabular, etc.
	Language    string           `json:"language"`      // ISO 639-1
	Topics      []string         `json:"topics"`
	Strategy    ChunkingStrategy `json:"strategy"`
}

// DefaultCognifyConfig returns the default pipeline configuration.
func DefaultCognifyConfig() CognifyConfig {
	return CognifyConfig{
		Template:  "STANDARD",
		ChunkSize: 512,
	}
}
