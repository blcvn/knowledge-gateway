package persistence

import (
	"context"
	"fmt"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"vnp-memory/ov-search/internal/domain/model"
	"vnp-memory/ov-search/internal/domain/repository"
)

type qdrantRepo struct {
	client     pb.PointsClient
	collection string
	conn       *grpc.ClientConn
}

func NewQdrantRepo(url, collection string) (repository.VectorRepository, error) {
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to qdrant: %w", err)
	}

	client := pb.NewPointsClient(conn)
	return &qdrantRepo{
		client:     client,
		collection: collection,
		conn:       conn,
	}, nil
}

func (r *qdrantRepo) Upsert(ctx context.Context, vector model.EmbeddingVector, payload model.UpsertPayload) error {
	pMap := map[string]*pb.Value{
		"path":          {Kind: &pb.Value_StringValue{StringValue: payload.Path}},
		"account_id":    {Kind: &pb.Value_StringValue{StringValue: payload.AccountID}},
		"user_id":       {Kind: &pb.Value_StringValue{StringValue: payload.UserID}},
		"content_hash":  {Kind: &pb.Value_StringValue{StringValue: payload.ContentHash}},
		"context_level": {Kind: &pb.Value_StringValue{StringValue: payload.ContextLevel}},
		"chunk_index":   {Kind: &pb.Value_IntegerValue{IntegerValue: int64(payload.ChunkIndex)}},
		"parent_dir":    {Kind: &pb.Value_StringValue{StringValue: payload.ParentDir}},
		"mime_type":     {Kind: &pb.Value_StringValue{StringValue: payload.MimeType}},
		"updated_at":    {Kind: &pb.Value_StringValue{StringValue: payload.UpdatedAt.Format("2006-01-02T15:04:05Z")}},
	}

	point := &pb.PointStruct{
		Id: &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: vector.ID}},
		Vectors: &pb.Vectors{
			VectorsOptions: &pb.Vectors_Vector{
				Vector: &pb.Vector{Data: vector.Vector},
			},
		},
		Payload: pMap,
	}

	_, err := r.client.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: r.collection,
		Points:         []*pb.PointStruct{point},
	})
	return err
}

func (r *qdrantRepo) Search(ctx context.Context, queryVector []float32, sparseVector []float32, accountID string, maxResults int) ([]model.SearchResult, error) {
	req := &pb.SearchPoints{
		CollectionName: r.collection,
		Vector:         queryVector,
		Limit:          uint64(maxResults),
		Filter: &pb.Filter{
			Must: []*pb.Condition{
				{
					ConditionOneOf: &pb.Condition_Field{
						Field: &pb.FieldCondition{
							Key: "account_id",
							Match: &pb.Match{
								MatchValue: &pb.Match_Keyword{Keyword: accountID},
							},
						},
					},
				},
			},
		},
		WithPayload: &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	}

	res, err := r.client.Search(ctx, req)
	if err != nil {
		return nil, err
	}

	var results []model.SearchResult
	for _, p := range res.GetResult() {
		payload := p.GetPayload()
		path := payload["path"].GetStringValue()
		ctxLevel := payload["context_level"].GetStringValue()

		results = append(results, model.SearchResult{
			ID:            p.GetId().GetUuid(),
			Path:          path,
			SemanticScore: model.Score(p.GetScore()),
			MatchedContext: model.MatchedContext{
				DepthLevel: ctxLevel,
			},
		})
	}
	return results, nil
}

func (r *qdrantRepo) Delete(ctx context.Context, path string, accountID string) error {
	_, err := r.client.Delete(ctx, &pb.DeletePoints{
		CollectionName: r.collection,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Filter{
				Filter: &pb.Filter{
					Must: []*pb.Condition{
						{
							ConditionOneOf: &pb.Condition_Field{
								Field: &pb.FieldCondition{
									Key: "account_id",
									Match: &pb.Match{MatchValue: &pb.Match_Keyword{Keyword: accountID}},
								},
							},
						},
						{
							ConditionOneOf: &pb.Condition_Field{
								Field: &pb.FieldCondition{
									Key: "path",
									Match: &pb.Match{MatchValue: &pb.Match_Keyword{Keyword: path}},
								},
							},
						},
					},
				},
			},
		},
	})
	return err
}
