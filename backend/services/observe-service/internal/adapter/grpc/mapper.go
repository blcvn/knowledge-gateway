package grpc

import (
	pb "github.com/vnp-memory/api/proto/observe/v1"
	"github.com/vnp-memory/services/observe-service/internal/domain"
	"github.com/vnp-memory/services/observe-service/internal/observe"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapObserveResponse(resp *observe.ObserveResponse) *pb.ObserveResponse {
	return &pb.ObserveResponse{
		ObservationId:   resp.ObservationID,
		Deduplicated:    resp.Deduplicated,
		Compressed:      mapCompressedObs(resp.Compressed),
		InjectedContext: resp.InjectedContext,
		ContextTokens:   int32(resp.ContextTokens),
	}
}

func mapCompressedObs(obs domain.CompressedObservation) *pb.CompressedObservationProto {
	return &pb.CompressedObservationProto{
		Id:         obs.ID,
		ObsType:    obs.ObsType,
		Title:      obs.Title,
		Subtitle:   obs.Subtitle,
		Facts:      obs.Facts,
		Narrative:  obs.Narrative,
		Concepts:   obs.Concepts,
		Files:      obs.Files,
		Importance: obs.Importance,
		Confidence: obs.Confidence,
		AgentId:    obs.AgentID,
		Timestamp:  timestamppb.New(obs.Timestamp),
	}
}

func mapSession(session *domain.Session) *pb.SessionProto {
	var endedAt *timestamppb.Timestamp
	if session.EndedAt != nil {
		endedAt = timestamppb.New(*session.EndedAt)
	}
	return &pb.SessionProto{
		SessionId:        session.ID,
		TenantId:         session.TenantID,
		Project:          session.Project,
		Cwd:              session.CWD,
		Model:            session.Model,
		AgentId:          session.AgentID,
		Status:           session.Status,
		Summary:          session.Summary,
		ObservationCount: int32(session.ObservationCount),
		Tags:             session.Tags,
		StartedAt:        timestamppb.New(session.StartedAt),
		EndedAt:          endedAt,
		LastActiveAt:     timestamppb.New(session.LastActiveAt),
	}
}

func mapListSessionsResponse(sessions []domain.Session) *pb.ListSessionsResponse {
	resp := &pb.ListSessionsResponse{}
	for _, s := range sessions {
		resp.Sessions = append(resp.Sessions, mapSession(&s))
	}
	return resp
}

func mapObservationsResponse(obs []domain.CompressedObservation) *pb.GetObservationsResponse {
	resp := &pb.GetObservationsResponse{}
	for _, o := range obs {
		resp.Observations = append(resp.Observations, mapCompressedObs(o))
	}
	return resp
}
