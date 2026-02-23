package prd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/blcvn/ba-shared-libs/pkg/domain"
	v32 "github.com/blcvn/ba-shared-libs/pkg/domain/v3.2"
)

// Parser handles PRD parsing
type Parser struct {
	llm domain.LLMService
}

// NewParser creates a new PRD parser
func NewParser(llm domain.LLMService) *Parser {
	return &Parser{
		llm: llm,
	}
}

// ParsePRD parses a PRD document into structured format
func (p *Parser) ParsePRD(ctx context.Context, prdContent string) (*v32.StructuredPRD, error) {
	log.Printf("[PRDParser] Starting ParsePRD (Content Length: %d)", len(prdContent))

	systemPrompt := `You are a Senior Business Analyst expert in parsing Product Requirement Documents (PRD).

Your task is to parse the provided PRD content and extract structured information according to the complete PRD schema.

CRITICAL RULES:
1. Return ONLY valid JSON - no explanations, no markdown, no preamble
2. Extract ONLY information explicitly stated in the PRD - do NOT invent or hallucinate
3. If a section is missing, return empty arrays/objects for optional fields
4. Preserve all original IDs if present, or generate sequential IDs
5. Maintain all details exactly as written in the source document
6. Parse Vietnamese content accurately and map to English field names
7. Normalize enums to match allowed values:
   - Priority: "P0", "P1", "P2", "P3"
   - Status: "draft", "review", "approved", "archived"
   - Role: "technical", "business", "domain" (for Glossary)
   - Frequency: "realtime", "daily", "weekly", "monthly" (for Analytics)

REQUIRED FIELDS:
- Metadata: product_name, version, status
- Personas: id, name, role, goals, technical_level
- ProductOverview: name, description, objectives
- Features: id, name, description, priority
- UserStories: id, feature_id, as_a, i_want, so_that, priority

DATA TYPE ENFORCEMENT:
- PermissionMatrix.roles must be an ARRAY of STRINGS: ["Role A", "Role B"]
- AnalyticsRequirements.properties must be a MAP (Key-Value pairs): {"prop1": "val1", "prop2": "val2"}
- Integration.endpoints must be an ARRAY of STRINGS: ["/api/v1/resource"]

Output must be valid JSON matching the StructuredPRD schema.`

	userPrompt := fmt.Sprintf(`Parse this PRD document and extract all structured information:

%s

Return a JSON object with this EXACT structure:
{
  "Metadata": {
    "product_name": "...",
    "version": "1.0",
    "status": "draft",
    "author": "...",
    "last_updated": "YYYY-MM-DD",
    "prd_id": "generated-uuid"
  },
  "Glossary": [
    {
      "id": "TERM-001",
      "term": "...",
      "meaning": "...",
        "english": "...",
        "related_terms": ["..."],
      "category": "business"
    }
  ],
  "Personas": [
    {
      "id": "P001",
      "name": "...",
      "role": "...",
      "goals": ["..."],
      "pain_points": ["..."],
      "behaviors": ["..."],
      "motivations": ["..."],
      "barriers": ["..."],
      "technical_level": "intermediate",
      "usage_frequency": "daily"
    }
  ],
  "ProductOverview": {
    "name": "...",
    "description": "...",
    "vision": "...",
    "target_release": "...",
    "objectives": ["..."],
    "success_metrics": [
      {
        "metric_name": "...",
        "target": "...",
        "measurement_method": "..."
      }
    ]
  },
  "Features": [
    {
      "id": "F001",
      "name": "...",
      "description": "...",
      "priority": "P0",
      "status": "planned",
      "category": "...",
      "dependencies": ["F002"],
      "acceptance_criteria": ["..."],
      "technical_notes": "..."
    }
  ],
  "PermissionMatrix": {
    "roles": ["Role A", "Role B"],
    "permissions": [
      {
        "action": "View Dashboard",
        "allowed_roles": ["Role A"],
        "description": "..."
      }
    ]
  },
  "Integrations": [
    {
      "id": "INT-001",
      "system_name": "...",
      "type": "rest_api",
      "purpose": "...",
      "direction": "inbound",
      "status": "planned",
      "authentication": "OAuth2",
      "endpoints": ["/api/v1/resource"],
      "data_flow": "...",
      "error_scenarios": ["Total failure"]
    }
  ],
  "UserFlows": [
    {
      "id": "UF-001",
      "name": "Login Flow",
      "description": "...",
      "steps": [
        {
          "step_number": 1,
          "actor": "User",
          "action": "Enter credentials",
          "result": "System validates",
          "system_behavior": "...",
          "screen": "Login Screen"
        }
      ],
      "involved_personas": ["P001"],
      "related_features": ["F001"],
      "related_user_stories": ["US-001"],
      "alternative_paths": [
        {
          "condition": "Invalid credentials",
          "outcome": "Show error",
          "steps": [{"step_number": 1, "action": "...", "actor": "..."}]
        }
      ]
    }
  ],
  "UserStories": [
     {
       "id": "US-001",
       "feature_id": "F001",
       "as_a": "User",
       "i_want": "to login",
       "so_that": "I can access the system",
       "priority": "P0",
       "size": "S",
       "acceptance_criteria": ["..."],
       "dependencies": ["US-002"],
       "status": "ready"
     }
  ],
  "UserStoryMap": {
    "user_type": "...",
    "activity_backbone": [
      {
        "activity_name": "...",
        "user_tasks": [
          {
            "task_name": "...",
            "user_stories": ["US-001"],
            "priority": 1
          }
        ]
      }
    ],
    "releases": [
      {
        "release_name": "MVP",
        "included_stories": ["US-001"]
      }
    ]
  },
  "BusinessRules": [
    {
      "id": "BR-01",
      "name": "...",
      "description": "...",
      "rule_logic": "...",
      "applies_to": ["US-001"],
      "validation_logic": "...",
      "error_message": "...",
      "severity": "high"
    }
  ],
  "AnalyticsRequirements": [
    {
      "metric_id": "METRIC-001",
      "metric_name": "...",
      "event": "login_success",
      "properties": {"user_id": "string"},
      "tool": "Google Analytics",
      "reporting_frequency": "daily",
      "dashboard": "Main Dashboard"
    }
  ],
  "ScopeDefinition": {
    "in_scope": ["Feature A", "Feature B"],
    "out_of_scope": [
      {
        "feature": "Feature C",
        "reason": "Deferred to Phase 2",
        "planned_phase": "Phase 2"
      }
    ]
  }
}`, prdContent)

	log.Printf("[PRDParser] Sending request to LLM...")
	response, err := p.llm.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[PRDParser] LLM Chat failed: %v", err)
		return nil, fmt.Errorf("failed to parse PRD: %w", err)
	}

	log.Printf("[PRDParser] LLM Response received. Length: %d", len(response))

	// Clean JSON response (remove markdown code blocks if present)
	jsonStr := cleanJSON(response)

	var prd v32.StructuredPRD
	if err := json.Unmarshal([]byte(jsonStr), &prd); err != nil {
		log.Printf("[PRDParser] JSON Unmarshal failed: %v", err)
		return nil, fmt.Errorf("failed to unmarshal PRD JSON: %w", err)
	}

	log.Printf("[PRDParser] ParsePRD successful. Extracted %d features, %d user stories", len(prd.Features), len(prd.UserStories))
	return &prd, nil
}

// cleanJSON removes markdown code blocks and cleans JSON string
func cleanJSON(s string) string {
	// Remove markdown code blocks
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
