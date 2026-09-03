package cogneetools

import (
	"github.com/vnp-community/vnp-memory/gateway/internal/adapter/mcp"
	ingestionpb "github.com/vnp-memory/api/proto/cognee/ingestion/v1"
	cognifypb   "github.com/vnp-memory/api/proto/cognee/cognify/v1"
	searchpb    "github.com/vnp-memory/api/proto/cognee/search/v1"
	memorypb    "github.com/vnp-memory/api/proto/memory/v1"
)

// Dependencies — all gRPC clients needed by cognee tools
type Dependencies struct {
	IngestionClient ingestionpb.IngestionServiceClient
	CognifyClient   cognifypb.CognifyServiceClient
	SearchClient    searchpb.SearchServiceClient
	MemoryClient    memorypb.AgentMemoryServiceClient
}

// RegisterCogneeStandardTools registers 6 Cognee-standard MCP tools.
// These coexist with existing 16 tools — no tool is removed.
// After: total = 22 tools.
func RegisterCogneeStandardTools(reg *mcp.ToolRegistry, deps *Dependencies) {
	tools := []mcp.Tool{
		cognifyTool(deps),
		searchTool(deps),
		saveInteractionTool(deps),
		listDataTool(deps),
		deleteDatasetTool(deps),
		cognifyStatusTool(deps),
	}
	for _, t := range tools {
		reg.Register(t)
	}
}

func cognifyTool(deps *Dependencies) mcp.Tool {
	return mcp.Tool{
		Name:        "cognify",
		Description: "Transform data into a knowledge graph. Ingests text/URL/file and builds semantic graph with entity extraction and relationship mapping. Use this to add knowledge to the memory system.",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"data": {
					Type:        "string",
					Description: "Text content, URL, or file path to ingest into the knowledge graph",
				},
				"dataset_name": {
					Type:        "string",
					Description: "Target dataset name (created if not exists). Defaults to 'default'.",
				},
				"node_sets": {
					Type:        "array",
					Items:       &mcp.Property{Type: "string"},
					Description: "Optional NodeSet tags for memory scoping (e.g. ['user_123', 'project_alpha'])",
				},
				"template": {
					Type:        "string",
					Enum:        []string{"STANDARD", "EMBED_ONLY", "FAST_INDEX", "TEMPORAL", "GRAPH_ONLY"},
					Description: "Pipeline template. 'EMBED_ONLY' is fastest (no LLM). Default: 'STANDARD'.",
				},
			},
			Required: []string{"data"},
		},
		Handler: NewCognifyHandler(deps.IngestionClient, deps.CognifyClient).Handle,
	}
}

func searchTool(deps *Dependencies) mcp.Tool {
	return mcp.Tool{
		Name:        "search",
		Description: "Query the knowledge graph with multiple search strategies. Returns semantically relevant information from ingested documents and memories.",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"query": {
					Type:        "string",
					Description: "Natural language search query",
				},
				"query_type": {
					Type: "string",
					Enum: []string{
						"SIMILARITY", "GRAPH_COMPLETION", "GRAPH_SUMMARY",
						"KEYWORD", "CHUNKS", "TEMPORAL", "MULTI_HOP",
						"HYBRID", "FEELING_LUCKY",
					},
					Description: "Search strategy. Default: 'GRAPH_COMPLETION'.",
				},
				"dataset_name": {Type: "string", Description: "Restrict search to a specific dataset"},
				"node_sets":    {Type: "array", Items: &mcp.Property{Type: "string"}, Description: "Filter by NodeSet tags"},
				"top_k":        {Type: "integer", Default: 10},
				"save_interaction": {
					Type:        "boolean",
					Description: "Log this search for feedback tracking. Returns interaction_id.",
				},
			},
			Required: []string{"query"},
		},
		Handler: NewSearchHandler(deps.SearchClient).Handle,
	}
}

func saveInteractionTool(deps *Dependencies) mcp.Tool {
	return mcp.Tool{
		Name:        "save_interaction",
		Description: "Log a user-agent interaction (query + answer) for memory and feedback. Use after important Q&A exchanges to improve future retrieval quality.",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"query":         {Type: "string", Description: "The user's question or query"},
				"answer":        {Type: "string", Description: "The agent's answer or response"},
				"session_id":    {Type: "string", Description: "Session identifier for grouping related interactions"},
				"score":         {Type: "number", Description: "Quality score (-1.0 to 1.0)"},
				"feedback_text": {Type: "string", Description: "Optional text comment about interaction quality"},
			},
			Required: []string{"query", "answer"},
		},
		Handler: NewSaveInteractionHandler(deps.SearchClient, deps.MemoryClient).Handle,
	}
}

func listDataTool(deps *Dependencies) mcp.Tool {
	return mcp.Tool{
		Name:        "list_data",
		Description: "List available datasets or items within a specific dataset.",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"dataset_id":   {Type: "string", Description: "If provided, lists entries within this dataset"},
				"dataset_name": {Type: "string", Description: "Alternative to dataset_id"},
				"limit":        {Type: "integer", Default: 20},
				"offset":       {Type: "integer", Default: 0},
			},
		},
		Handler: NewListDataHandler(deps.IngestionClient).Handle,
	}
}

func deleteDatasetTool(deps *Dependencies) mcp.Tool {
	return mcp.Tool{
		Name:        "delete_dataset",
		Description: "Delete a dataset and ALL associated data (graph nodes in Neo4j, vectors in Qdrant, metadata in PostgreSQL). Irreversible.",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"dataset_id":   {Type: "string", Description: "UUID of the dataset to delete"},
				"dataset_name": {Type: "string", Description: "Alternative: use dataset name"},
			},
		},
		Handler: NewDeleteDatasetHandler(deps.IngestionClient).Handle,
	}
}

func cognifyStatusTool(deps *Dependencies) mcp.Tool {
	return mcp.Tool{
		Name:        "cognify_status",
		Description: "Check the status of a background cognify or memify pipeline job.",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"pipeline_run_id": {Type: "string", Description: "Pipeline run ID returned by cognify"},
				"dataset_id":      {Type: "string", Description: "Alternative: check latest pipeline run for this dataset"},
			},
		},
		Handler: NewCognifyStatusHandler(deps.CognifyClient).Handle,
	}
}
