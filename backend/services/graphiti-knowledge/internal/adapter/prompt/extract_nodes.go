package prompt

import (
	"fmt"
	"strings"
)

func extractNodesPrompt() PromptTemplate {
	return PromptTemplate{
		Name: "extract_nodes",
		SystemPrompt: `You are an expert knowledge graph builder. Extract named entities from the provided text.

Extract entities that are clearly identifiable and semantically meaningful. For each entity provide:
- name: the entity's canonical name (normalized, not a pronoun)
- label: entity type classification (e.g. Person, Organization, Location, Concept, Event, Product)
- summary: brief 1-2 sentence description based only on what's stated in the context

Rules:
- Do NOT extract generic nouns (e.g. "meeting", "issue") unless they are specific named things
- Do NOT extract pronouns
- Normalize entity names (e.g. "John" and "John Smith" in same text → use "John Smith")
- Return entities as a JSON object with "entities" array`,

		BuildUser: func(ctx PromptContext) string {
			var sb strings.Builder

			sb.WriteString("Text to analyze:\n```\n")
			for _, chunk := range ctx.Chunks {
				sb.WriteString(Sanitize(chunk))
				sb.WriteString("\n\n")
			}
			sb.WriteString("```\n")

			if len(ctx.PrevEpisodes) > 0 {
				sb.WriteString("\nPrevious context (for disambiguation):\n")
				for _, ep := range ctx.PrevEpisodes {
					sb.WriteString("- ")
					sb.WriteString(Sanitize(ep))
					sb.WriteString("\n")
				}
			}

			if len(ctx.EntityTypes) > 0 {
				sb.WriteString("\nIMPORTANT: Extract ONLY entities matching these prescribed types:\n")
				for name, schema := range ctx.EntityTypes {
					sb.WriteString(fmt.Sprintf("- **%s**: %s\n", name, schema.Description))
					if len(schema.Examples) > 0 {
						sb.WriteString(fmt.Sprintf("  Examples: %s\n", strings.Join(schema.Examples, ", ")))
					}
				}
				sb.WriteString("Do NOT extract entities that don't match these types.\n")
			}

			if ctx.Language != "" && ctx.Language != "en" {
				sb.WriteString(fmt.Sprintf("\nNote: Text is in %s language. Keep entity names in their original language.\n", ctx.Language))
			}

			if ctx.ReferenceTime != "" {
				sb.WriteString(fmt.Sprintf("\nReference time: %s\n", ctx.ReferenceTime))
			}

			return sb.String()
		},

		Schema: ExtractedNodeListSchema,
	}
}
