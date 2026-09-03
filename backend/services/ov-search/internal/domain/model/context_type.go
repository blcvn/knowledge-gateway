package model

type ContextType string

const (
	ContextTypeCode          ContextType = "code"
	ContextTypeDocumentation ContextType = "documentation"
	ContextTypeMemory        ContextType = "memory"
	ContextTypeFileSearch    ContextType = "file_search"
)

type TypedQuery struct {
	RawQuery string
	Type     ContextType
}

type QueryPlan struct {
	TypedQueries  []TypedQuery
	TargetDomains []string
}
