package token

import "testing"

func TestLookupIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  TokenType
	}{
		{"let", Let}, {"map", Identifier}, {"true", True}, {"not", Not},
	}
	for _, tt := range tests {
		if got := LookupIdentifier(tt.input); got != tt.want {
			t.Errorf("LookupIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
