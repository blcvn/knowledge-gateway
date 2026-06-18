package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kg-service/internal/access"
	"kg-service/internal/write"
)

func TestWriteHandlerIgnoresCallerSuppliedIdentityFields(t *testing.T) {
	fixture := newIntegrationFixture(t)
	handler := write.NewHandler(fixture.writeSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/kg/write/nodes", strings.NewReader(`{"tenant_id":"22222222-2222-2222-2222-222222222222","app_id":"22222222-bbbb-2222-bbbb-222222222222","domain_id":"integration-domain","node_type":"Doc","properties":{"title":"Spoofed Identity","status":"active"}}`))
	req = req.WithContext(access.ContextWithIdentity(req.Context(), fixture.actor))
	rec := httptest.NewRecorder()

	handler.CreateNode(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("CreateNode() status = %d body=%s", rec.Code, rec.Body.String())
	}

	var payload write.NodeCreateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.NodeID == "" {
		t.Fatal("CreateNode() returned empty node_id")
	}

	record, ok := fixture.store.GetNodeByID(payload.NodeID)
	if !ok {
		t.Fatalf("GetNodeByID(%q) returned no record", payload.NodeID)
	}
	if record.OwnerTenantID != fixture.actor.TenantID {
		t.Fatalf("owner_tenant_id = %q, want %q", record.OwnerTenantID, fixture.actor.TenantID)
	}
	if record.OwnerAppID != fixture.actor.AppID {
		t.Fatalf("owner_app_id = %q, want %q", record.OwnerAppID, fixture.actor.AppID)
	}
}
