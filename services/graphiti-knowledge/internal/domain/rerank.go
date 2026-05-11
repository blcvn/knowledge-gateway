package domain

type RerankRequest struct {
	Query     string
	Documents []string
	Model     string
}

type CrossEncoderScore struct {
	DocumentIndex int
	Score         float32
}

type RerankResult struct {
	Scores []CrossEncoderScore
	Usage  TokenUsage
}
