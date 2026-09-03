package privacy

import (
	"regexp"
	"strings"
)

// PIIFilter filters personally identifiable information from text.
type PIIFilter struct {
	emailRe    *regexp.Regexp
	phoneRe    *regexp.Regexp
	creditCardRe *regexp.Regexp
}

func NewPIIFilter() *PIIFilter {
	return &PIIFilter{
		emailRe:      regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		phoneRe:      regexp.MustCompile(`(\+?\d[\d\s\-\(\)]{8,}\d)`),
		creditCardRe: regexp.MustCompile(`\b(?:\d[ -]?){13,16}\b`),
	}
}

// Filter removes PII from text, replacing with redaction markers.
func (f *PIIFilter) Filter(text string) string {
	text = f.emailRe.ReplaceAllString(text, "[EMAIL]")
	text = f.phoneRe.ReplaceAllString(text, "[PHONE]")
	text = f.creditCardRe.ReplaceAllString(text, "[CARD]")
	return text
}

// ContainsPII reports whether the text likely contains PII.
func (f *PIIFilter) ContainsPII(text string) bool {
	return f.emailRe.MatchString(text) ||
		f.phoneRe.MatchString(text) ||
		f.creditCardRe.MatchString(text)
}

// MaskField masks a sensitive field value with asterisks.
func MaskField(value string) string {
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}
