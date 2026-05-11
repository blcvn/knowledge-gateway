package client

import (
	"context"
)

type FSClient interface {
	WriteArchive(ctx context.Context, accountID, userID, content string) (string, error)
	WriteMemory(ctx context.Context, accountID, category, content string) (string, error)
}

type fsClientImpl struct{}

func NewFSClient() FSClient {
	return &fsClientImpl{}
}

func (c *fsClientImpl) WriteArchive(ctx context.Context, accountID, userID, content string) (string, error) {
	return "viking://" + accountID + "/" + userID + "/sessions/archives/session.json.gz", nil
}

func (c *fsClientImpl) WriteMemory(ctx context.Context, accountID, category, content string) (string, error) {
	return "viking://" + accountID + "/memories/" + category + "/memory.json", nil
}
