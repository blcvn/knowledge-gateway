package model

import "time"

// Enterprise Domain Models for sm-mcp
type MCPTool struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type MCPResource struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type MCPRequest struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type MCPResponse struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


