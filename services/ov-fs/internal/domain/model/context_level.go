package model

type ContextLevel string

const (
	ContextLevelL0 ContextLevel = "L0" // Abstract summary (~100 tokens)
	ContextLevelL1 ContextLevel = "L1" // Section overview (~2K tokens)
	ContextLevelL2 ContextLevel = "L2" // Full content
)
