package biz

import kerrors "github.com/go-kratos/kratos/v2/errors"

var (
	ErrDepthExceeded  = kerrors.BadRequest("ERR_DEPTH_EXCEEDED", "requested query depth exceeds the maximum allowed limit")
	ErrNodesExceeded  = kerrors.BadRequest("ERR_NODES_EXCEEDED", "requested query node count exceeds the maximum allowed limit")
	ErrAPIKeyNotFound = kerrors.Unauthorized("ERR_UNAUTHORIZED", "api key not found")
	ErrAPIKeyRevoked  = kerrors.Unauthorized("ERR_UNAUTHORIZED", "api key revoked")
	ErrAPIKeyExpired  = kerrors.Unauthorized("ERR_UNAUTHORIZED", "api key expired")
)

func ErrForbiddenWithMetadata(message string, metadata map[string]string) error {
	return kerrors.Forbidden("ERR_FORBIDDEN", message).WithMetadata(metadata)
}

func ErrNotConfigured(message string, metadata map[string]string) error {
	return kerrors.InternalServer("ERR_NOT_CONFIGURED", message).WithMetadata(metadata)
}

func ErrSchemaInvalid(message string, metadata map[string]string) error {
	return kerrors.BadRequest("ERR_SCHEMA_INVALID", message).WithMetadata(metadata)
}
