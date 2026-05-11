package grpc

import (
	"vnp-memory/services/graphiti-knowledge/internal/domain"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/dto"
)

// Mapper maps between protobuf structures and domain/DTO structures
type Mapper struct{}

func NewMapper() *Mapper {
	return &Mapper{}
}

func (m *Mapper) ToDomainExtractedEntity(name, label, summary string) domain.ExtractedEntity {
	return domain.ExtractedEntity{
		Name:    name,
		Label:   label,
		Summary: summary,
	}
}

func (m *Mapper) ToDTOExtractEntitiesRequest(content string) dto.ExtractEntitiesRequest {
	return dto.ExtractEntitiesRequest{
		Content: content,
	}
}
