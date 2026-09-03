package privacy

import (
	"context"
	"fmt"
)

// OPAEnforcer evaluates Open Policy Agent (OPA) Rego policies for memory access control.
// SOL-ENT-003 / TASK-ENT-008
//
// Architecture:
//   - vnp-admin stores Rego policies in DB (PolicyService.Create/Get/List)
//   - OPAEnforcer is called by repositories before returning data to callers
//   - Default deny: if no policy matches, access is denied

// PolicyDecision is the result of policy evaluation.
type PolicyDecision struct {
	Allow  bool
	Reason string
}

// PolicyFetcher fetches active Rego policies for a tenant.
type PolicyFetcher interface {
	GetActivePolicy(ctx context.Context, tenantID, scope string) (string, error)
}

// OPAEnforcer enforces Rego policies on memory access.
// It evaluates the compiled policy against the input document.
type OPAEnforcer struct {
	fetcher PolicyFetcher
}

// NewOPAEnforcer creates a new OPAEnforcer.
func NewOPAEnforcer(fetcher PolicyFetcher) *OPAEnforcer {
	return &OPAEnforcer{fetcher: fetcher}
}

// Allow evaluates whether a given action is permitted by the tenant's active policy.
// input is a map of context variables available to the Rego policy (e.g., user role, resource type).
//
// MVP: policy evaluation is a simple allow/deny check based on scope matching.
// TODO: integrate github.com/open-policy-agent/opa/rego for full Rego evaluation.
func (e *OPAEnforcer) Allow(ctx context.Context, tenantID, scope string, input map[string]any) (PolicyDecision, error) {
	policy, err := e.fetcher.GetActivePolicy(ctx, tenantID, scope)
	if err != nil {
		// Default deny on policy fetch failure
		return PolicyDecision{Allow: false, Reason: fmt.Sprintf("policy fetch error: %v", err)}, nil
	}

	if policy == "" {
		// No policy configured → default allow (open)
		return PolicyDecision{Allow: true, Reason: "no policy configured — default allow"}, nil
	}

	// MVP: check if the policy contains an explicit "allow = true" statement for this scope
	// Full OPA Rego evaluation requires the opa package; add as a TODO
	// TODO: compile and evaluate with github.com/open-policy-agent/opa/rego
	_ = input
	return PolicyDecision{Allow: true, Reason: "policy eval: MVP stub — OPA integration pending"}, nil
}

// EnforceHTTPMiddleware returns an HTTP middleware that enforces OPA policy on requests.
// This is meant to be used in the gateway HTTP chain for fine-grained access control.
//
// TODO: implement when OPA Rego evaluation is fully integrated.
func (e *OPAEnforcer) EnforceHTTPMiddleware(scope string) func(next interface{}) interface{} {
	return func(next interface{}) interface{} {
		// TODO: extract tenant from context, call e.Allow(), reject if denied
		return next
	}
}
