// Package search provides the hybrid BM25 + vector + RRF search engine
// for the AgentMemory observe-search service.
//
// Entry points:
//   - NewBM25Index()      — create an in-memory BM25 inverted index
//   - NewVectorIndex()    — create a dense cosine-similarity vector index
//   - RRFFuse()           — combine BM25 and vector results via Reciprocal Rank Fusion
//   - NewIndexPersister() — debounced gob-file persistence for both indexes
package search
