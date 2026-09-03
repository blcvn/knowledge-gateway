package prompt

import (
	"fmt"
	"strings"
)

func dedupeNodesPrompt() PromptTemplate {
	return PromptTemplate{
		Name: "dedupe_nodes",
		SystemPrompt: `You are an expert at entity disambiguation. Determine whether a newly extracted entity refers to the same real-world entity as any existing entity in the knowledge graph.

Decide:
- "merge": the extracted entity IS the same as an existing entity (return existing_uuid)
- "new": the extracted entity is a genuinely new entity

Be conservative — only merge when you are very confident it's the same entity.
Distinguish between different people with the same name (use context clues, roles, organizations).`,

		BuildUser: func(ctx PromptContext) string {
			if len(ctx.Chunks) == 0 || len(ctx.ExistingNodes) == 0 {
				return "No candidates to compare."
			}

			newEntity := ctx.Chunks[0]
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("New entity: **%s**\n\n", newEntity))

			if len(ctx.PrevEpisodes) > 0 {
				sb.WriteString("Context where entity appears:\n")
				for _, ep := range ctx.PrevEpisodes {
					sb.WriteString("  " + Sanitize(ep) + "\n")
				}
				sb.WriteString("\n")
			}

			sb.WriteString("Existing entities in knowledge graph:\n")
			for _, existing := range ctx.ExistingNodes {
				sb.WriteString("- ")
				sb.WriteString(existing) // format: "UUID | Name | Summary"
				sb.WriteString("\n")
			}

			sb.WriteString("\nIs the new entity the same as any existing entity? Respond with decision='merge' (with existing_uuid) or decision='new'.")
			return sb.String()
		},

		Schema: EntityResolutionSchema,
	}
}

func dedupeEdgesPrompt() PromptTemplate {
	return PromptTemplate{
		Name: "dedupe_edges",
		SystemPrompt: `You are a knowledge graph expert. Determine the relationship between a new fact and existing facts in the graph.

Classify the new edge as:
- "DUPLICATE": same fact as an existing edge (do not add)
- "NEW": genuinely new fact that doesn't contradict existing edges
- "CONTRADICTION": directly contradicts an existing fact (the existing one must be invalidated)
- "UPDATE": supersedes an existing fact (existing one should be marked invalid_at=now)

Return invalidated_edge_uuids for CONTRADICTION or UPDATE cases.`,

		BuildUser: func(ctx PromptContext) string {
			if len(ctx.Chunks) < 2 {
				return "Insufficient data."
			}

			var sb strings.Builder
			sb.WriteString("New fact:\n")
			sb.WriteString(ctx.Chunks[0] + "\n\n")

			sb.WriteString("Existing facts about the same entity pair:\n")
			for _, existing := range ctx.ExistingNodes {
				sb.WriteString("- ")
				sb.WriteString(existing) // format: "UUID | source→target | fact | valid_at | invalid_at"
				sb.WriteString("\n")
			}

			if ctx.ReferenceTime != "" {
				sb.WriteString(fmt.Sprintf("\nCurrent time: %s\n", ctx.ReferenceTime))
			}

			return sb.String()
		},

		Schema: EdgeResolutionSchema,
	}
}

func summarizeNodesPrompt() PromptTemplate {
	return PromptTemplate{
		Name: "summarize_node",
		SystemPrompt: `You are a knowledge graph expert. Update the summary of an entity based on new information.

Write a concise, factual summary (2-4 sentences) that captures the most important and current attributes of this entity.
Blend the existing summary with new facts — do not simply append, but write a coherent updated summary.
Focus on facts, not opinions. Use present tense for current facts, past tense for historical.`,

		BuildUser: func(ctx PromptContext) string {
			var sb strings.Builder
			if len(ctx.ExistingNodes) > 0 {
				sb.WriteString("Current summary:\n")
				sb.WriteString(ctx.ExistingNodes[0] + "\n\n")
			}
			sb.WriteString("New information:\n")
			for _, chunk := range ctx.Chunks {
				sb.WriteString(Sanitize(chunk) + "\n")
			}
			return sb.String()
		},

		Schema: NodeSummarySchema,
	}
}

func summarizeSagasPrompt() PromptTemplate {
	return PromptTemplate{
		Name: "summarize_saga",
		SystemPrompt: `You are an expert at summarizing narrative threads. Create or update a summary of a saga (a sequential series of related episodes).

Write a concise narrative summary (3-5 sentences) that:
- Captures the key events and their sequence
- Notes major changes or outcomes
- Preserves important temporal relationships
- Reads as a coherent story, not a bullet list

Also provide a short descriptive title (max 8 words) for the saga.`,

		BuildUser: func(ctx PromptContext) string {
			var sb strings.Builder

			if len(ctx.ExistingNodes) > 0 {
				sb.WriteString("Existing saga summary:\n")
				sb.WriteString(ctx.ExistingNodes[0] + "\n\n")
			}

			sb.WriteString("New episodes to incorporate:\n")
			for i, ep := range ctx.PrevEpisodes {
				sb.WriteString(fmt.Sprintf("[%d] %s\n\n", i+1, Sanitize(ep)))
			}

			if ctx.ReferenceTime != "" {
				sb.WriteString(fmt.Sprintf("Reference time: %s\n", ctx.ReferenceTime))
			}

			return sb.String()
		},

		Schema: SagaSummarySchema,
	}
}
