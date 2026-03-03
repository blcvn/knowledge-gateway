package service

import (
	"context"

	pb "kgs-platform/api/ontology/v1"
)

type OntologyService struct {
	pb.UnimplementedOntologyServer
}

func NewOntologyService() *OntologyService {
	return &OntologyService{}
}

func (s *OntologyService) CreateEntityType(ctx context.Context, req *pb.CreateEntityTypeRequest) (*pb.CreateEntityTypeReply, error) {
    return &pb.CreateEntityTypeReply{}, nil
}
func (s *OntologyService) CreateRelationType(ctx context.Context, req *pb.CreateRelationTypeRequest) (*pb.CreateRelationTypeReply, error) {
    return &pb.CreateRelationTypeReply{}, nil
}
func (s *OntologyService) ListEntityTypes(ctx context.Context, req *pb.ListEntityTypesRequest) (*pb.ListEntityTypesReply, error) {
    return &pb.ListEntityTypesReply{}, nil
}
func (s *OntologyService) ListRelationTypes(ctx context.Context, req *pb.ListRelationTypesRequest) (*pb.ListRelationTypesReply, error) {
    return &pb.ListRelationTypesReply{}, nil
}
