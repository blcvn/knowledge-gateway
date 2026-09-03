package overlay

import "testing"

func TestNormalizeOptionalUUIDValue(t *testing.T) {
	t.Parallel()

	const valid = "3367b110-29d2-49df-b503-6eeae83700fa"
	tests := []struct {
		name string
		in   any
		want string
	}{
		{name: "nil", in: nil, want: ""},
		{name: "nilLike", in: "<nil>", want: ""},
		{name: "nullLike", in: "null", want: ""},
		{name: "empty", in: "", want: ""},
		{name: "invalid", in: "abc", want: ""},
		{name: "valid", in: valid, want: valid},
		{name: "validTrim", in: " " + valid + " ", want: valid},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeOptionalUUIDValue(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeOptionalUUIDValue(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
