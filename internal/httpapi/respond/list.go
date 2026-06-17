package respond

type Page struct {
	Limit  int
	Cursor string
}

func NormalizePage(limit int, cursor string, defaultLimit, maxLimit int) Page {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return Page{Limit: limit, Cursor: cursor}
}
