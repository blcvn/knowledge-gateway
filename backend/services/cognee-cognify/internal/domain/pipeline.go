// Package domain defines pipeline step constants and template configuration.
// TASK-CE-010: Custom Pipelines Orchestration (SOL-006 §2.1)
//
// PipelineConfig.Resolve() is the single source of truth for step ordering.
// Backward compatible: empty config → STANDARD (7 steps).
package domain

// PipelineStep — identifies each step in the cognify pipeline.
type PipelineStep string

const (
	StepClassify             PipelineStep = "CLASSIFY"
	StepChunk                PipelineStep = "CHUNK"
	StepExtractGraph         PipelineStep = "EXTRACT_GRAPH"
	StepBuildOntology        PipelineStep = "BUILD_ONTOLOGY"
	StepAddDatapoints        PipelineStep = "ADD_DATAPOINTS"
	StepDetectCommunity      PipelineStep = "DETECT_COMMUNITY"
	StepSummarizeCommunity   PipelineStep = "SUMMARIZE_COMMUNITY"
	StepExtractTemporalGraph PipelineStep = "EXTRACT_TEMPORAL_GRAPH"
)

// PipelineTemplateName — named presets for common pipeline configurations.
type PipelineTemplateName string

const (
	TemplateStandard  PipelineTemplateName = "STANDARD"   // all 7 steps (default)
	TemplateEmbedOnly PipelineTemplateName = "EMBED_ONLY" // fastest: no LLM
	TemplateFastIndex PipelineTemplateName = "FAST_INDEX" // classify + chunk + embed
	TemplateTemporal  PipelineTemplateName = "TEMPORAL"   // temporal extraction variant
	TemplateGraphOnly PipelineTemplateName = "GRAPH_ONLY" // graph without embeddings
)

// templateSteps maps each template to its ordered step list.
var templateSteps = map[PipelineTemplateName][]PipelineStep{
	TemplateStandard: {
		StepClassify, StepChunk, StepExtractGraph,
		StepBuildOntology, StepAddDatapoints,
		StepDetectCommunity, StepSummarizeCommunity,
	},
	TemplateEmbedOnly: {StepChunk, StepAddDatapoints},
	TemplateFastIndex: {StepClassify, StepChunk, StepAddDatapoints},
	TemplateTemporal:  {StepClassify, StepChunk, StepExtractTemporalGraph, StepAddDatapoints},
	TemplateGraphOnly: {StepClassify, StepChunk, StepExtractGraph, StepBuildOntology},
}

// PipelineTemplateInfo is returned by GetPipelineTemplates RPC.
type PipelineTemplateInfo struct {
	Name  string
	Steps []string
}

// ListTemplates returns all available pipeline templates (for GetPipelineTemplates RPC).
func ListTemplates() []PipelineTemplateInfo {
	infos := make([]PipelineTemplateInfo, 0, len(templateSteps))
	for name, steps := range templateSteps {
		strSteps := make([]string, len(steps))
		for i, s := range steps {
			strSteps[i] = string(s)
		}
		infos = append(infos, PipelineTemplateInfo{Name: string(name), Steps: strSteps})
	}
	return infos
}
