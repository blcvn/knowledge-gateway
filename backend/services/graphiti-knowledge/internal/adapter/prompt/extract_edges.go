package prompt

import (
	"fmt"
	"strings"
)

func extractEdgesPrompt() PromptTemplate {
	return PromptTemplate{
		Name: "extract_edges",
		SystemPrompt: `You are an expert knowledge graph builder. Extract factual relationships between named entities.

For each relationship extract:
- source_entity: name of the source entity (must be an extracted entity)
- target_entity: name of the target entity (must be an extracted entity)  
- relation_type: UPPERCASE relationship type (e.g. WORKS_AT, REPORTS_TO, FOUNDED, ACQUIRED)
- fact: natural language statement of the specific fact
- valid_at: ISO8601 when fact became true (null if unknown)
- invalid_at: ISO8601 when fact ceased to be true (null if still valid)

Rules:
- Only extract relationships between entities present in the provided entity list
- Be specific: "Alice joined Engineering team at Acme in March 2024" not "Alice works somewhere"
- For temporal facts: extract valid_at/invalid_at when explicitly mentioned
- Return as JSON object with "edges" array`,

		BuildUser: func(ctx PromptContext) string {
			var sb strings.Builder

			sb.WriteString("Text:\n```\n")
			for _, chunk := range ctx.Chunks {
				sb.WriteString(Sanitize(chunk) + "\n\n")
			}
			sb.WriteString("```\n")

			if len(ctx.ExistingNodes) > 0 {
				sb.WriteString("\nEntities to use (source/target must be from this list):\n")
				for _, n := range ctx.ExistingNodes {
					sb.WriteString("- ")
					sb.WriteString(n)
					sb.WriteString("\n")
				}
			}

			if len(ctx.EdgeTypes) > 0 {
				sb.WriteString("\nIMPORTANT: Extract ONLY relationships of these prescribed types:\n")
				for name, schema := range ctx.EdgeTypes {
					sb.WriteString(fmt.Sprintf("- **%s**: %s", name, schema.Description))
					if len(schema.SourceTypes) > 0 {
						sb.WriteString(fmt.Sprintf(" (from %s to %s)",
							strings.Join(schema.SourceTypes, "/"),
							strings.Join(schema.TargetTypes, "/")))
					}
					sb.WriteString("\n")
				}
			}

			if ctx.ReferenceTime != "" {
				sb.WriteString(fmt.Sprintf("\nReference time for temporal context: %s\n", ctx.ReferenceTime))
			}

			return sb.String()
		},

		Schema: ExtractedEdgeListSchema,
	}
}
