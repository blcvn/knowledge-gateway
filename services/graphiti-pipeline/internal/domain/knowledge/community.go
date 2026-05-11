package knowledge

type CommunityLevel int

const (
	LevelLow    CommunityLevel = 1
	LevelMedium CommunityLevel = 2
	LevelHigh   CommunityLevel = 3
)

type CommunityNode struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Summary string         `json:"summary"`
	Level   CommunityLevel `json:"level"`
}

type CommunityEdge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"`
}
