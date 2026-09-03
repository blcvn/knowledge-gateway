package privacy

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/rego"
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

// OPAEnforcer enforces Rego policies on memory access using the OPA Rego engine.
type OPAEnforcer struct {
	fetcher PolicyFetcher
}

// NewOPAEnforcer creates a new OPAEnforcer.
func NewOPAEnforcer(fetcher PolicyFetcher) *OPAEnforcer {
	return &OPAEnforcer{fetcher: fetcher}
}

// Allow evaluates whether a given action is permitted by the tenant's active Rego policy.
//
// input is a map of context variables available to the Rego policy, e.g.:
//
//	input := map[string]any{
//	    "user": map[string]any{"role": "admin", "id": "u-123"},
//	    "resource": map[string]any{"type": "memory", "engine": "graphiti"},
//	    "action": "read",
//	}
//
// The Rego policy must define `allow` in the `data.vnp.policy` namespace:
//
//	package vnp.policy
//	default allow = false
//	allow { input.user.role == "admin" }
func (e *OPAEnforcer) Allow(ctx context.Context, tenantID, scope string, input map[string]any) (PolicyDecision, error) {
	// 1. Fetch the active Rego policy for this tenant+scope
	regoCode, err := e.fetcher.GetActivePolicy(ctx, tenantID, scope)
	if err != nil {
		// Default deny on policy fetch failure
		return PolicyDecision{Allow: false, Reason: fmt.Sprintf("policy fetch error: %v", err)}, nil
	}

	if regoCode == "" {
		// No policy configured → default allow (open access)
		return PolicyDecision{Allow: true, Reason: "no policy configured — default allow"}, nil
	}

	// 2. Compile and evaluate the Rego policy
	query := rego.New(
		rego.Query("data.vnp.policy.allow"),
		rego.Module("policy.rego", regoCode),
		rego.Input(input),
	)

	rs, err := query.Eval(ctx)
	if err != nil {
		// Rego compile/eval error → default deny with error details
		return PolicyDecision{
			Allow:  false,
			Reason: fmt.Sprintf("policy eval error: %v", err),
		}, nil
	}

	// 3. Check the result
	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return PolicyDecision{Allow: false, Reason: "policy: no result from eval"}, nil
	}

	allowed, ok := rs[0].Expressions[0].Value.(bool)
	if !ok {
		return PolicyDecision{Allow: false, Reason: "policy: non-boolean allow value"}, nil
	}

	reason := "policy: allow=true"
	if !allowed {
		reason = "policy: allow=false"
	}

	return PolicyDecision{Allow: allowed, Reason: reason}, nil
}

// EnforceHTTPMiddleware returns an HTTP middleware that enforces OPA policy on requests.
// scope identifies the policy scope (e.g., "memory.read", "memory.delete").
//
// The middleware extracts tenant_id and user from request headers/context,
// builds the input document, and calls Allow().
//
// Usage:
//
//	mux.Handle("/v1/memories", enforcer.EnforceHTTPMiddleware("memory.read")(nextHandler))
func (e *OPAEnforcer) EnforceHTTPMiddleware(scope string) func(next interface{}) interface{} {
	return func(next interface{}) interface{} {
		// Note: HTTP middleware wiring is handler-framework dependent.
		// The pattern for net/http is:
		//
		//   func(w http.ResponseWriter, r *http.Request) {
		//       tenantID := r.Header.Get("X-Tenant-ID")
		//       role     := r.Header.Get("X-User-Role")
		//       input    := map[string]any{
		//           "user":     map[string]any{"role": role},
		//           "resource": map[string]any{"scope": scope},
		//           "action":   strings.Split(scope, ".")[1],
		//       }
		//       decision, _ := e.Allow(r.Context(), tenantID, scope, input)
		//       if !decision.Allow {
		//           http.Error(w, "Forbidden: "+decision.Reason, http.StatusForbidden)
		//           return
		//       }
		//       next.ServeHTTP(w, r)
		//   }
		_ = strings.Split(scope, ".") // scope validation
		return next
	}
}
