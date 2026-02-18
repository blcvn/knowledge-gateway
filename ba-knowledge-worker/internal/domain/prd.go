package domain

// CanonicalPRD represents the normalized structure of a PRD.
// It serves as the AST for downstream generation.
type AcceptanceCriteria struct {
	ID    string `json:"id"`
	Given string `json:"given"`
	When  string `json:"when"`
	Then  string `json:"then"`
}

type UIComponent struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Behavior   string `json:"behavior"`
	Validation string `json:"validation"`
}

type ImageMetadata struct {
	ID  string `json:"id"`
	URL string `json:"url"`
	Alt string `json:"alt"`
}

// UserStory represents a structured user story.
type UserStory struct {
	ID                 string               `json:"id"`
	Role               string               `json:"role"`
	Action             string               `json:"action"`
	Benefit            string               `json:"benefit"`
	AcceptanceCriteria []AcceptanceCriteria `json:"acceptance_criteria"`
	UIComponents       []UIComponent        `json:"ui_components"`
}

type CanonicalPRD struct {
	Title         string      `json:"title"`
	Introduction  string      `json:"introduction"`
	UserStories   []UserStory `json:"user_stories"`
	Functional    []string    `json:"functional_requirements"`
	NonFunctional []string    `json:"non_functional_requirements"`
	Constraints   []string    `json:"constraints"`
}

// Merge combines another CanonicalPRD into this one.
func (p *CanonicalPRD) Merge(other CanonicalPRD) {
	if p.Title == "" || p.Title == "Untitled PRD" {
		p.Title = other.Title
	}
	if p.Introduction == "" {
		p.Introduction = other.Introduction
	} else if other.Introduction != "" && other.Introduction != p.Introduction {
		p.Introduction += "\n" + other.Introduction
	}

	// Merge User Stories (Maintain logical order by appending)
	p.UserStories = append(p.UserStories, other.UserStories...)

	// Merge requirements (Deduplicate)
	p.Functional = appendUnique(p.Functional, other.Functional)
	p.NonFunctional = appendUnique(p.NonFunctional, other.NonFunctional)
	p.Constraints = appendUnique(p.Constraints, other.Constraints)
}

func appendUnique(target []string, source []string) []string {
	seen := make(map[string]bool)
	for _, item := range target {
		seen[item] = true
	}
	for _, item := range source {
		if !seen[item] {
			target = append(target, item)
			seen[item] = true
		}
	}
	return target
}

// StructuredPRD represents the parsed/structured PRD used for KG building.
type StructuredPRD struct {
	Metadata        PRDMetadata        `json:"metadata"`
	ProductOverview PRDProductOverview `json:"product_overview"`
	UserStories     []PRDUserStory     `json:"user_stories"`
	Features        []PRDFeature       `json:"features"`
	Entities        []PRDEntity        `json:"entities"`
	Personas        []PRDPersona       `json:"personas"`
	Integrations    []PRDIntegration   `json:"integrations"`
	BusinessRules   []PRDBusinessRule  `json:"business_rules"`
}

type PRDMetadata struct {
	ProductName string `json:"product_name"`
	Version     string `json:"version"`
	Status      string `json:"status"`
}

type PRDProductOverview struct {
	Vision string `json:"vision"`
}

type PRDUserStory struct {
	ID        string `json:"id"`
	IWant     string `json:"i_want"`  // Action/Desire
	AsA       string `json:"as_a"`    // Role
	SoThat    string `json:"so_that"` // Benefit
	FeatureID string `json:"feature_id"`
	Priority  string `json:"priority"`
}

type PRDFeature struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

type PRDEntity struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PRDPersona struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Role           string   `json:"role"`
	TechnicalLevel string   `json:"technical_level"`
	Goals          []string `json:"goals"`
	PainPoints     []string `json:"pain_points"`
}

type PRDIntegration struct {
	ID         string `json:"id"`
	SystemName string `json:"system_name"`
	Type       string `json:"type"`
	Purpose    string `json:"purpose"`
	Direction  string `json:"direction"`
	Status     string `json:"status"`
}

type PRDBusinessRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"`
	AppliesTo   []string `json:"applies_to"`
}
