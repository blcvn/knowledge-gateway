package pb

type ExtractedEntity struct {
	Name    string
	Label   string
	Summary string
}

type ExtractedEdge struct {
	Source   string
	Target   string
	Relation string
	Fact     string
	Temporal []string
}

type ExtractEntitiesRequest struct {
	Content          string
	PreviousEpisodes []string
	EntityTypes      []string
}

type ExtractEntitiesResponse struct {
	Entities []*ExtractedEntity
}

type ResolveEntitiesRequest struct {
	ExtractedEntities []*ExtractedEntity
	GroupId           string
}

type Resolution struct {
	ExistingEntityId string
	ExtractedEntity  *ExtractedEntity
	Decision         string
	Confidence       float64
}

type ResolveEntitiesResponse struct {
	Resolutions []*Resolution
}

type ExtractEdgesRequest struct{}
type ExtractEdgesResponse struct{}

type ResolveEdgesRequest struct{}
type ResolveEdgesResponse struct{}

type GenerateEmbeddingRequest struct{}
type GenerateEmbeddingResponse struct{}

type GenerateEmbeddingBulkRequest struct{}
type GenerateEmbeddingBulkResponse struct{}

type RerankRequest struct{}
type RerankResponse struct{}

type UpdateCommunityRequest struct{}
type UpdateCommunityResponse struct{}
