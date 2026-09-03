package model

const (
	// OVE1Magic is the magic string indicating an OpenViking Envelope Version 1 format
	OVE1Magic = "OVE1"
)

// ProviderType represents the KMS provider used to encrypt the file key.
type ProviderType byte

const (
	ProviderTypeUnknown    ProviderType = 0x00
	ProviderTypeLocal      ProviderType = 0x01
	ProviderTypeVault      ProviderType = 0x02
	ProviderTypeCloud      ProviderType = 0x03 // AWS or GCP
)

// EnvelopeHeader represents the header of the OVE1 envelope.
type EnvelopeHeader struct {
	Magic        string
	Version      byte
	ProviderType ProviderType
	EFKLen       uint16 // Encrypted File Key Length
	KIVLen       uint16 // Key Initialization Vector Length
	DIVLen       uint16 // Data Initialization Vector Length
}

// Envelope represents the complete OpenViking encrypted envelope structure.
// Format: Magic(4) | Version(1) | ProviderType(1) | EFKLen(2) | KIVLen(2) | DIVLen(2) | EFK | KIV | DIV | Ciphertext + AuthTag
type Envelope struct {
	Header     EnvelopeHeader
	EFK        []byte // Encrypted File Key
	KIV        []byte // Key Initialization Vector
	DIV        []byte // Data Initialization Vector
	Ciphertext []byte // The encrypted content including AES-GCM Auth Tag
}
