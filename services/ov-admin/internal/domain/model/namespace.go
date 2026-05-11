package model

import "fmt"

type NamespaceURI struct {
	AccountID string
	UserID    string
	AgentID   string
}

func (n NamespaceURI) String() string {
	return fmt.Sprintf("viking://%s/%s/%s/", n.AccountID, n.UserID, n.AgentID)
}
