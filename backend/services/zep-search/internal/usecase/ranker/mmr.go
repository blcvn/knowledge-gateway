package ranker

import (
	"math"
	"sort"
)

// Document represents an item with an embedded vector.
type Document struct {
	ID        string
	Vector    []float64
	InitScore float64
}

// MaximalMarginalRelevance computes diversity-aware ranking.
// Lambda (0.0 to 1.0) controls the trade-off:
// 1.0 = Maximize relevance to query (standard search)
// 0.0 = Maximize diversity among selected documents
func MaximalMarginalRelevance(queryVector []float64, docs []Document, lambda float64, topK int) []string {
	var selected []string
	var unselected []Document
	unselected = append(unselected, docs...)

	for i := 0; i < topK && len(unselected) > 0; i++ {
		bestIdx := -1
		bestScore := -math.MaxFloat64

		for j, candidate := range unselected {
			// Relevance to query is pre-computed as InitScore (or cosine similarity here)
			rel := candidate.InitScore 

			// Penalty: Max cosine similarity to any already-selected document
			maxSim := 0.0
			for _, selID := range selected {
				selVector := getVectorByID(docs, selID)
				sim := cosineSimilarity(candidate.Vector, selVector)
				if sim > maxSim {
					maxSim = sim
				}
			}

			// MMR Equation: Lambda * Relevance - (1 - Lambda) * Diversity_Penalty
			mmrScore := (lambda * rel) - ((1.0 - lambda) * maxSim)

			if mmrScore > bestScore {
				bestScore = mmrScore
				bestIdx = j
			}
		}

		if bestIdx != -1 {
			selected = append(selected, unselected[bestIdx].ID)
			// Remove from unselected
			unselected = append(unselected[:bestIdx], unselected[bestIdx+1:]...)
		}
	}

	return selected
}

func getVectorByID(docs []Document, id string) []float64 {
	for _, d := range docs {
		if d.ID == id {
			return d.Vector
		}
	}
	return nil
}

func cosineSimilarity(a, b []float64) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
