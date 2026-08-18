package vector

import (
	"errors"
	"fmt"
	"net/http"
)

// Embedding failures a caller must be able to tell apart.
//
// Before these existed every failure was one indistinguishable string, which made two different
// operational responses impossible to trigger apart: an expired token needs a human to rotate a
// credential, while a timeout needs nothing but patience. Both arrived as
// "embedding provider status …" and both were retried three times.
var (
	// ErrEmbeddingUnauthorized means the provider rejected the credential — 401 or 403.
	//
	// This is the one embedding failure that never resolves on its own, and impl-02 §R2 records it
	// as an incident that WILL happen rather than one that might: the VNPAY gateway issues tokens
	// through an SSO login, so a token eventually stops working and no amount of retrying helps.
	ErrEmbeddingUnauthorized = errors.New("embedding provider rejected the credential")

	// ErrEmbeddingRejectedInput means the provider refused the request itself — a 4xx that is not
	// about credentials, typically input past the model's context window.
	//
	// Retrying is equally pointless here, and not free: the oversized-input case sent the same
	// rejected payload three times before failing, tripling the cost of a request that could never
	// have succeeded.
	ErrEmbeddingRejectedInput = errors.New("embedding provider rejected the input")
)

// classifyStatus maps an HTTP status onto one of the sentinels above, or nil when the status is
// worth retrying.
//
// 429 is deliberately NOT terminal: rate limiting is the textbook case for backing off, and the
// retry middleware is exactly the right response to it.
func classifyStatus(status int) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ErrEmbeddingUnauthorized
	case status == http.StatusTooManyRequests:
		return nil
	case status >= 400 && status < 500:
		return ErrEmbeddingRejectedInput
	default:
		return nil
	}
}

// statusError builds the error for a non-2xx response, attaching a sentinel when one applies so
// that callers can branch with errors.Is instead of matching on message text.
func statusError(status int, body string) error {
	if sentinel := classifyStatus(status); sentinel != nil {
		return fmt.Errorf("embedding provider status %d: %s: %w", status, body, sentinel)
	}
	return fmt.Errorf("embedding provider status %d: %s", status, body)
}

// IsTerminal reports whether repeating a request could not possibly change the outcome.
//
// Retry middleware consults this so a credential failure surfaces on the first attempt rather than
// after three, and so the log line an operator reads is the first thing that happened rather than
// the third copy of it.
func IsTerminal(err error) bool {
	return errors.Is(err, ErrEmbeddingUnauthorized) || errors.Is(err, ErrEmbeddingRejectedInput)
}
