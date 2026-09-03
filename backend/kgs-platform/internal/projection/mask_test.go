package projection

import "testing"

func TestMaskEmailEdgeCases(t *testing.T) {
	if MaskEmail("") != "" {
		t.Fatalf("empty email should stay empty")
	}
	if MaskEmail("invalid-email") != "invalid-email" {
		t.Fatalf("invalid email should stay unchanged")
	}
	if got := MaskEmail("alice@example.org"); got != "a***@***.org" {
		t.Fatalf("unexpected masked email: %s", got)
	}
}

func TestMaskPhoneEdgeCases(t *testing.T) {
	if got := MaskPhone(""); got != "" {
		t.Fatalf("empty phone should stay empty, got %q", got)
	}
	if got := MaskPhone("123"); got != "***-***-123" {
		t.Fatalf("partial phone mask unexpected: %s", got)
	}
	if got := MaskPhone("+1 (415) 555-1234"); got != "***-***-1234" {
		t.Fatalf("unexpected masked phone: %s", got)
	}
}

func TestMaskPIIValueHandlesNonString(t *testing.T) {
	in := 12345
	if got := MaskPIIValue("phone", in); got != in {
		t.Fatalf("non-string value should stay unchanged, got %#v", got)
	}
}
