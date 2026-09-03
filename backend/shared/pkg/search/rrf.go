package search

import "sort"

const rrfK = 60

func RRFFuse(bm25 []BM25Result, vector []VectorResult, graph []GraphResult,
    weights ScoreWeights, limit int) []HybridResult {

    scores := map[string]*HybridResult{}

    for rank, r := range bm25 {
        h := getOrCreate(scores, r.DocID, r.SessionID)
        h.CombinedScore += weights.BM25 * (1.0 / float64(rrfK+rank+1))
        h.BM25Rank = rank + 1
        h.BM25Score = r.Score
    }
    for rank, r := range vector {
        h := getOrCreate(scores, r.DocID, r.SessionID)
        h.CombinedScore += weights.Vector * (1.0 / float64(rrfK+rank+1))
        h.VectorRank = rank + 1
        h.VectorScore = r.Score
    }
    for rank, r := range graph {
        h := getOrCreate(scores, r.DocID, "")
        h.CombinedScore += weights.Graph * (1.0 / float64(rrfK+rank+1))
        h.GraphScore = r.Score
    }

    results := make([]HybridResult, 0, len(scores))
    for _, v := range scores { results = append(results, *v) }
    sort.Slice(results, func(i, j int) bool { return results[i].CombinedScore > results[j].CombinedScore })
    if limit > len(results) { limit = len(results) }
    return results[:limit]
}

func getOrCreate(scores map[string]*HybridResult, docID, sessionID string) *HybridResult {
    if scores[docID] == nil {
        scores[docID] = &HybridResult{DocID: docID, SessionID: sessionID}
    }
    return scores[docID]
}
