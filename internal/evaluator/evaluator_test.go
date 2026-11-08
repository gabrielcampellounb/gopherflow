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

func TestEvaluateLessThan(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  runtime.Bool
	}{
		{"1 < 2", runtime.Bool(true)},
		{"2 < 1", runtime.Bool(false)},
		{"2 < 2", runtime.Bool(false)},
		{"1.5 < 2", runtime.Bool(true)},
		{"1 + 2 < 2 * 2", runtime.Bool(true)},
	} {
		t.Run(tt.input, func(t *testing.T) {
			if got := evaluate(t, tt.input); got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvaluateLessThanTypeErrors(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"1 < true", "type error at 1:3: operator < is not defined for Number and Bool"},
		{"\n\"a\" < 2", "type error at 2:5: operator < is not defined for String and Number"},
		{"null < null", "type error at 1:6: operator < is not defined for Null and Null"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			program, err := parser.Parse(lexer.New(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			value, err := Evaluate(program)
			if value != nil || err == nil || err.Error() != tt.want {
				t.Fatalf("Evaluate(%q) = %v, %v; want %q", tt.input, value, err, tt.want)
			}
		})
	}
}
