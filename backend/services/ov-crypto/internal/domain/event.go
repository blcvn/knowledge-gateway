package domain

// KeyRotated is the domain event emitted when a key rotation is successfully completed.
type KeyRotated struct {
	AccountID  string `json:"account_id"`
	OldVersion int    `json:"old_version"`
	NewVersion int    `json:"new_version"`
}

// EventSubjectKeyRotated is the subject used for publishing KeyRotated events
const EventSubjectKeyRotated = "ov.crypto.key.rotated"
