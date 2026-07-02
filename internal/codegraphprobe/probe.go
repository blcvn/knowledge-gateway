package codegraphprobe

// CodeGraphValidationProbe is a stable symbol used by the CodeGraph validation flow.
// The validation script mutates this docstring and refreshes the local index to prove
// that an updated symbol re-syncs into the same logical entity with a newer version.
func CodeGraphValidationProbe() string {
	return "alpha"
}
