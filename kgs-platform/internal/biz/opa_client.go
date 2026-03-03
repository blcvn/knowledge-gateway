package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kratos/kratos/v2/log"
)

// OPAClient interfaces with the sidecar Open Policy Agent container
type OPAClient struct {
	baseURL string
	log     *log.Helper
}

// NewOPAClient initializes the OPA client
// For KGS, OPA is typically deployed as a sidecar running at localhost:8181
func NewOPAClient(logger log.Logger) *OPAClient {
	return &OPAClient{
		baseURL: "http://localhost:8181/v1/data/kgs/allow",
		log:     log.NewHelper(logger),
	}
}

// OPARequest represents the input data evaluated by OPA
type OPARequest struct {
	Input map[string]interface{} `json:"input"`
}

// OPAResponse represents the evaluation result from OPA
type OPAResponse struct {
	Result bool `json:"result"`
}

// EvaluatePolicy sends the context and action to OPA and returns whether it is allowed.
func (c *OPAClient) EvaluatePolicy(ctx context.Context, appID, action, resource string) (bool, error) {
	reqData := OPARequest{
		Input: map[string]interface{}{
			"app_id":   appID,
			"action":   action,
			"resource": resource,
		},
	}

	payload, err := json.Marshal(reqData)
	if err != nil {
		return false, fmt.Errorf("failed to marshal OPA request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(payload))
	if err != nil {
		return false, fmt.Errorf("failed to create OPA request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.log.Errorf("OPA request failed: %v", err)
		// If OPA is unreachable, fail closed
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Errorf("OPA returned non-200 status: %d", resp.StatusCode)
		return false, nil
	}

	var opaResp OPAResponse
	if err := json.NewDecoder(resp.Body).Decode(&opaResp); err != nil {
		return false, fmt.Errorf("failed to decode OPA response: %v", err)
	}

	return opaResp.Result, nil
}

// PutPolicy uploads a raw Rego policy string to OPA's /v1/policies API
func (c *OPAClient) PutPolicy(ctx context.Context, policyID string, regoContent string) error {
	// e.g. "http://localhost:8181/v1/policies/{policyID}"
	// We need to construct the URL from baseURL dynamically since baseURL is currently /v1/data/kgs/allow
	// A quick hack is replacing the suffix or using a fixed config. We'll assume localhost:8181.
	url := fmt.Sprintf("http://localhost:8181/v1/policies/%s", policyID)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer([]byte(regoContent)))
	if err != nil {
		return fmt.Errorf("failed to create OPA PUT request: %v", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.log.Errorf("OPA request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Errorf("OPA PUT policy returned non-200 status: %d", resp.StatusCode)
		return fmt.Errorf("OPA PUT returned status %d", resp.StatusCode)
	}

	return nil
}
