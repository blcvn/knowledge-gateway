package model

// EventTagDef defines a tag rule for event extraction.
type EventTagDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ProfileConfig represents project-specific configurations.
type ProfileConfig struct {
	Language              string         `json:"language"`
	ProfileStrictMode     bool           `json:"profile_strict_mode"`
	ProfileValidateMode   bool           `json:"profile_validate_mode"`
	AdditionalProfiles    []ProfileTopic `json:"additional_profiles"`
	OverwriteProfiles     []ProfileTopic `json:"overwrite_profiles"`
	EventTags             []EventTagDef  `json:"event_tags"`
	EventThemeRequirement string         `json:"event_theme_requirement"`
}
