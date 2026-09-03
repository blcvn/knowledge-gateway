package domain

type ResourceIngested struct {
	Path       string `json:"path"`
	AccountID  string `json:"account_id"`
	Chunks     int    `json:"chunks"`
	ParserType string `json:"parser_type"`
}
