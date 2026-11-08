package evaluator

import (
	"testing"

	"github.com/gabrielcampellounb/gopherflow/internal/lexer"
	"github.com/gabrielcampellounb/gopherflow/internal/parser"
	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
)

func evaluate(t *testing.T, input string) runtime.Value {
	t.Helper()
	program, err := parser.Parse(lexer.New(input))
	if err != nil {
		t.Fatal(err)
	}
	value, err := Evaluate(program)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestEvaluateExpressions(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1 + 2 * 3", "7"},
		{"let answer = 41 in answer + 1", "42"},
		{`"hello"`, `"hello"`},
		{"true", "true"},
		{"null", "null"},
	}
	for _, tt := range tests {
		if got := evaluate(t, tt.input).Inspect(); got != tt.want {
			t.Errorf("Evaluate(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEvaluateErrors(t *testing.T) {
	for _, input := range []string{"missing", "1 + true", "1 / 0"} {
		program, err := parser.Parse(lexer.New(input))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Evaluate(program); err == nil {
			t.Errorf("Evaluate(%q) returned nil error", input)
		}
	}
}
