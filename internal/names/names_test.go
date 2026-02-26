package names

import "testing"

func TestSanitize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feature-x", "feature-x"},
		{"fix/bug-123", "fix-bug-123"},
		{"feature//double", "feature-double"},
		{"/leading-slash", "leading-slash"},
		{"release/v2.0", "release-v2.0"},
		{"trailing/", "trailing"},
		{"a/b/c/d", "a-b-c-d"},
		{"simple", "simple"},
		{"dots.and-dashes", "dots.and-dashes"},
		{"special@chars#here!", "special-chars-here"},
		{"///", ""},
		{"a", "a"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
