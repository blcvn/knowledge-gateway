package ranker

import (
\t"sort"
)

// DocumentScore represents an individual document's score from a single search system.
type DocumentScore struct {
\tDocumentID string
\tScore      float64
\tRank       int
}

// RRFResult represents the final fused score for a document.
type RRFResult struct {
\tDocumentID string
\tScore      float64
}

// RRFParameters holds tuning parameters for Reciprocal Rank Fusion.
type RRFParameters struct {
\tK int // Commonly set to 60 as per standard RRF literature
}

// DefaultRRFParameters returns the standard configuration for RRF.
func DefaultRRFParameters() RRFParameters {
\treturn RRFParameters{
\t\tK: 60,
\t}
}

// Fuse applies the Reciprocal Rank Fusion algorithm across multiple ranked lists.
// formula: RRF_Score = sum( 1 / (K + rank(d)) ) for each ranked list.
func Fuse(rankedLists [][]DocumentScore, params RRFParameters) []RRFResult {
\tfusionMap := make(map[string]float64)

\tfor _, list := range rankedLists {
\t\t// Ensure the list is sorted by rank if not already. We assume lower rank number = better.
\t\tsort.Slice(list, func(i, j int) bool {
\t\t\treturn list[i].Rank < list[j].Rank
\t\t})

\t\tfor i, doc := range list {
\t\t\t// If the rank isn't explicitly set, use the list index + 1
\t\t\tactualRank := doc.Rank
\t\t\tif actualRank == 0 {
\t\t\t\tactualRank = i + 1
\t\t\t}

\t\t\tscore := 1.0 / float64(params.K+actualRank)
\t\t\tfusionMap[doc.DocumentID] += score
\t\t}
\t}

\tresults := make([]RRFResult, 0, len(fusionMap))
\tfor docID, score := range fusionMap {
\t\tresults = append(results, RRFResult{
\t\t\tDocumentID: docID,
\t\t\tScore:      score,
\t\t})
\t}

\t// Sort final results descending by score
\tsort.Slice(results, func(i, j int) bool {
\t\treturn results[i].Score > results[j].Score
\t})

\treturn results
}
