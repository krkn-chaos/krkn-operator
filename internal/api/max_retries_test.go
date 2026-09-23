package api

import "testing"

func TestResolveMaxRetries(t *testing.T) {
	zero := 0
	positive := 5
	negative := -1

	tests := []struct {
		name      string
		value     *int
		want      int
		wantError bool
	}{
		{name: "omitted uses default", want: 3},
		{name: "zero disables retries", value: &zero, want: 0},
		{name: "positive value is preserved", value: &positive, want: 5},
		{name: "negative value is rejected", value: &negative, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveMaxRetries(tt.value)
			if (err != nil) != tt.wantError {
				t.Fatalf("resolveMaxRetries() error = %v, want error: %v", err, tt.wantError)
			}
			if err == nil && got != tt.want {
				t.Fatalf("resolveMaxRetries() = %d, want %d", got, tt.want)
			}
		})
	}
}
