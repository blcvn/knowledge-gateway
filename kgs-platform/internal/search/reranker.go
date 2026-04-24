package search

func RerankWithCentrality(in []Result, centrality map[string]float64, beta float64) []Result {
	if len(in) == 0 || len(centrality) == 0 || beta == 0 {
		return in
	}
	maxScore := 0.0
	for _, score := range centrality {
		if score > maxScore {
			maxScore = score
		}
	}
	if maxScore <= 0 {
		maxScore = 1
	}

	out := make([]Result, 0, len(in))
	for _, item := range in {
		c := centrality[item.ID] / maxScore
		item.Centrality = c
		item.Score = item.Score * (1 + beta*c)
		out = append(out, item)
	}
	return out
}
