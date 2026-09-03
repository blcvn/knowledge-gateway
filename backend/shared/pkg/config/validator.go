package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

// ValidationRule defines a single validation check against a Viper instance.
type ValidationRule struct {
	Field   string
	Check   func(v *viper.Viper) bool
	Message string
}

// Validate runs all rules against v and returns a combined error if any fail.
func Validate(v *viper.Viper, rules []ValidationRule) error {
	var errs []string
	for _, rule := range rules {
		if !rule.Check(v) {
			errs = append(errs, fmt.Sprintf("  - %s: %s", rule.Field, rule.Message))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n%s", strings.Join(errs, "\n"))
	}
	return nil
}

// RequireNonEmpty returns a rule that fails if the field is an empty string.
func RequireNonEmpty(field string) ValidationRule {
	return ValidationRule{
		Field:   field,
		Check:   func(v *viper.Viper) bool { return strings.TrimSpace(v.GetString(field)) != "" },
		Message: "must not be empty",
	}
}

// RequirePositiveInt returns a rule that fails if the field is not a positive integer.
func RequirePositiveInt(field string) ValidationRule {
	return ValidationRule{
		Field:   field,
		Check:   func(v *viper.Viper) bool { return v.GetInt(field) > 0 },
		Message: "must be a positive integer",
	}
}

// RequireValidURL returns a rule that fails if the field is not a valid URL.
func RequireValidURL(field string) ValidationRule {
	return ValidationRule{
		Field: field,
		Check: func(v *viper.Viper) bool {
			raw := v.GetString(field)
			if raw == "" {
				return false
			}
			u, err := url.ParseRequestURI(raw)
			return err == nil && u.Scheme != "" && u.Host != ""
		},
		Message: "must be a valid URL (e.g. http://host:port)",
	}
}

// RequireOneOf returns a rule that fails if the field value is not in the allowed set.
func RequireOneOf(field string, allowed ...string) ValidationRule {
	allowedSet := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = true
	}
	return ValidationRule{
		Field: field,
		Check: func(v *viper.Viper) bool {
			return allowedSet[v.GetString(field)]
		},
		Message: fmt.Sprintf("must be one of [%s]", strings.Join(allowed, ", ")),
	}
}
