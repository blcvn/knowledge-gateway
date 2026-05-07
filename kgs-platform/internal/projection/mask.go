package projection

import (
	"fmt"
	"strings"
)

func MaskEmail(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "@")
	if len(parts) != 2 {
		return value
	}
	local := parts[0]
	domain := parts[1]
	prefix := "*"
	if local != "" {
		prefix = string(local[0])
	}
	suffix := ".com"
	if idx := strings.LastIndex(domain, "."); idx >= 0 && idx < len(domain)-1 {
		suffix = domain[idx:]
	}
	return fmt.Sprintf("%s***@***%s", prefix, suffix)
}

func MaskPhone(value string) string {
	digits := make([]rune, 0, len(value))
	for _, r := range value {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) == 0 {
		return ""
	}
	last := ""
	if len(digits) >= 4 {
		last = string(digits[len(digits)-4:])
	} else {
		last = string(digits)
	}
	return "***-***-" + last
}

func MaskPIIValue(field string, value any) any {
	field = strings.ToLower(strings.TrimSpace(field))
	str, ok := value.(string)
	if !ok {
		return value
	}
	switch {
	case strings.Contains(field, "email"):
		return MaskEmail(str)
	case strings.Contains(field, "phone"), strings.Contains(field, "mobile"):
		return MaskPhone(str)
	default:
		if strings.Contains(str, "@") {
			return MaskEmail(str)
		}
		return MaskPhone(str)
	}
}
