package domain

type CommunityLevel int

type CommunityNode struct {
	ID        string
	Name      string
	Summary   string
	Level     CommunityLevel
	MemberIDs []string
}

type CommunityMember struct {
	ID   string
	Type string
}
