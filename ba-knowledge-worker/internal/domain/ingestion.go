package domain

// Block definition matching the shared UI contract
type Block struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Position int    `json:"position"`
}

type IngestionData struct {
	Blocks []Block `json:"blocks"`
}

type AgentInput struct {
	JobID     string
	InputType string
	Source    string
	Payload   []byte
}

type AgentOutput struct {
	Success bool
	Payload []byte
	Error   string
}
