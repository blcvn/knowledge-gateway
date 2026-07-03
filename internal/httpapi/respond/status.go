package respond

import "net/http"

const (
	CodeBadRequest             = "BAD_REQUEST"
	CodeUnauthorized           = "INVALID_API_KEY"
	CodeForbidden              = "FORBIDDEN"
	CodeNotFound               = "NOT_FOUND"
	CodeValidationFailed       = "VALIDATION_FAILED"
	CodeProjectionInconsistent = "PROJECTION_INCONSISTENT"
	CodeSyncScopeLocked        = "SYNC_SCOPE_LOCKED"
	CodeTooManyRequests        = "TOO_MANY_REQUESTS"
	CodeRequestTimedOut        = "REQUEST_TIMEOUT"
	CodeInternal               = "INTERNAL_ERROR"
)

func StatusFor(kind string) int {
	switch kind {
	case CodeBadRequest:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeValidationFailed:
		return http.StatusUnprocessableEntity
	case CodeProjectionInconsistent:
		return http.StatusConflict
	case CodeSyncScopeLocked:
		return http.StatusConflict
	case CodeTooManyRequests:
		return http.StatusTooManyRequests
	case CodeRequestTimedOut:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}
