package search

func Blend(semantic []Result, text []Result, alpha float64) []Result {
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	byID := make(map[string]Result, len(semantic)+len(text))

	for _, item := range semantic {
		if item.ID == "" {
			continue
		}
		item.SemanticScore = item.Score
		item.TextScore = 0
		item.Score = alpha * item.SemanticScore
		byID[item.ID] = item
	}

	for _, item := range text {
		if item.ID == "" {
			continue
		}
		item.TextScore = item.Score
		current, ok := byID[item.ID]
		if ok {
			if current.Label == "" {
				current.Label = item.Label
			}
			if len(current.Properties) == 0 {
				current.Properties = item.Properties
			}
			current.TextScore = item.TextScore
			current.Score = alpha*current.SemanticScore + (1-alpha)*current.TextScore
			byID[item.ID] = current
			continue
		}
		item.SemanticScore = 0
		item.Score = (1 - alpha) * item.TextScore
		byID[item.ID] = item
	}

	out := make([]Result, 0, len(byID))
	for _, item := range byID {
		out = append(out, item)
	}
	return out
}
