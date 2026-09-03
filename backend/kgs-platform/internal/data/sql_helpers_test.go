package data

import "testing"

func TestNullableUUID(t *testing.T) {
	t.Parallel()

	const valid = "3367b110-29d2-49df-b503-6eeae83700fa"
	tests := []struct {
		name string
		in   string
		want any
	}{
		{name: "empty", in: "", want: nil},
		{name: "spaces", in: "   ", want: nil},
		{name: "nilLike", in: "<nil>", want: nil},
		{name: "nilWord", in: "nil", want: nil},
		{name: "nullWord", in: "null", want: nil},
		{name: "invalidUUID", in: "not-a-uuid", want: nil},
		{name: "validUUID", in: valid, want: valid},
		{name: "validUUIDTrim", in: " " + valid + " ", want: valid},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := nullableUUID(tt.in)
			if got != tt.want {
				t.Fatalf("nullableUUID(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}
