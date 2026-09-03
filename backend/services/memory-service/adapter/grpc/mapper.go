package grpc

import (
    "vnp-memory/services/memory-service/internal/domain/agentmemory"
    memorypb "github.com/vnp-memory/api/proto/memory/v1"
)

func mapProceduralResponse(items []agentmemory.ProceduralMemory) *memorypb.ListProceduralResponse {
    resp := &memorypb.ListProceduralResponse{}
    for _, i := range items {
        resp.Items = append(resp.Items, &memorypb.ProceduralProto{
            Id:               i.ID,
            Name:             i.Name,
            Steps:            i.Steps,
            TriggerCondition: i.TriggerCondition,
            ExpectedOutcome:  i.ExpectedOutcome,
            Frequency:        int32(i.Frequency),
        })
    }
    return resp
}

func mapLessonsResponse(items []agentmemory.Lesson) *memorypb.ListLessonsResponse {
    resp := &memorypb.ListLessonsResponse{}
    for _, i := range items {
        resp.Items = append(resp.Items, &memorypb.LessonProto{
            Id:         i.ID,
            Content:    i.Content,
            Categories: i.Categories,
            Source:     i.Source,
        })
    }
    return resp
}

func mapInsightsResponse(items []agentmemory.Insight) *memorypb.ListInsightsResponse {
    resp := &memorypb.ListInsightsResponse{}
    for _, i := range items {
        resp.Items = append(resp.Items, &memorypb.InsightProto{
            Id:        i.ID,
            Content:   i.Content,
            LessonIds: i.LessonIDs,
        })
    }
    return resp
}
