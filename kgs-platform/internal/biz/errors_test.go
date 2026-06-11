package biz

import (
	"encoding/json"
	"testing"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrSchemaInvalid_HasExpectedCodeReasonAndMetadata(t *testing.T) {
	err := ErrSchemaInvalid("invalid schema", map[string]string{
		"label": "Requirement",
	})
	kerr := kerrors.FromError(err)
	if kerr == nil {
		t.Fatalf("expected kratos error, got %T", err)
	}
	if kerr.Code != 400 {
		t.Fatalf("expected code 400, got %d", kerr.Code)
	}
	if kerr.Reason != "ERR_SCHEMA_INVALID" {
		t.Fatalf("expected reason ERR_SCHEMA_INVALID, got %s", kerr.Reason)
	}
	if kerr.Metadata["label"] != "Requirement" {
		t.Fatalf("expected metadata label=Requirement, got %#v", kerr.Metadata)
	}
}

func TestErrSchemaInvalid_SerializesForGRPCAndHTTP(t *testing.T) {
	err := ErrSchemaInvalid("invalid schema", map[string]string{
		"label": "Requirement",
	})

	grpcStatus := status.Convert(err)
	if grpcStatus.Code() != codes.InvalidArgument {
		t.Fatalf("expected grpc code InvalidArgument, got %s", grpcStatus.Code())
	}

	kerr := kerrors.FromError(err)
	raw, marshalErr := json.Marshal(kerr)
	if marshalErr != nil {
		t.Fatalf("failed to marshal kratos error as HTTP payload: %v", marshalErr)
	}
	var decoded map[string]any
	if unmarshalErr := json.Unmarshal(raw, &decoded); unmarshalErr != nil {
		t.Fatalf("failed to unmarshal serialized payload: %v", unmarshalErr)
	}
	if int(decoded["code"].(float64)) != 400 {
		t.Fatalf("expected HTTP code 400 in serialized payload, got %v", decoded["code"])
	}
	if decoded["reason"].(string) != "ERR_SCHEMA_INVALID" {
		t.Fatalf("expected HTTP reason ERR_SCHEMA_INVALID, got %v", decoded["reason"])
	}
}
