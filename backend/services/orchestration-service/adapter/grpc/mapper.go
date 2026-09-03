package grpc

import (
	pb "github.com/vnp-memory/api/proto/orchestration/v1"
	"vnp-memory/services/orchestration-service/internal/domain"
)

func mapAction(a domain.Action) *pb.ActionProto {
	return &pb.ActionProto{
		Id:     a.ID,
		Status: string(a.Status),
	}
}

func mapListActionsResponse(actions []domain.Action) *pb.ListActionsResponse {
	resp := &pb.ListActionsResponse{}
	for _, a := range actions {
		resp.Actions = append(resp.Actions, mapAction(a))
	}
	return resp
}

func mapListSignalsResponse(signals []domain.Signal) *pb.ListSignalsResponse {
    resp := &pb.ListSignalsResponse{}
    return resp
}

func mapCrystalResponse(c *domain.Crystal) *pb.GetCrystalResponse {
    return &pb.GetCrystalResponse{}
}
