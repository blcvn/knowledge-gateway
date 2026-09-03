package crypto

// EnvelopeEncryption holds the metadata for an OVE1 envelope.
type EnvelopeEncryption struct {
	Version      string `json:"version"` // "OVE1"
	ProviderType string `json:"provider_type"`
	WrappedDEK   []byte `json:"wrapped_dek"`
	IV           []byte `json:"iv"`
}
