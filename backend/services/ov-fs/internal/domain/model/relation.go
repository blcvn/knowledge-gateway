package model

import "time"

type RelationType string

const (
	RelationReferences   RelationType = "references"
	RelationExtractedFrom RelationType = "extracted_from"
	RelationSummarizes   RelationType = "summarizes"
)

type FileRelation struct {
	ID           string
	SourceFileID string
	TargetFileID string
	RelationType RelationType
	AccountID    string
	Metadata     map[string]interface{}
	CreatedAt    time.Time
}
