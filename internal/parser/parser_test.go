package parser

import (
	"testing"

	"github.com/gabrielcampellounb/gopherflow/internal/ast"
	"github.com/gabrielcampellounb/gopherflow/internal/lexer"
)

func TestParserPrecedence(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1 + 2 * 3", "(1 + (2 * 3))"},
		{"1 |> f |> g", "((1 |> f) |> g)"},
		{"let x = 2 in x + 1", "let x = 2 in (x + 1)"},
	}
	for _, tt := range tests {
		program, err := Parse(lexer.New(tt.input))
		if err != nil {
			t.Fatal(err)
		}
		if got := program.String(); got != tt.want {
			t.Errorf("Parse(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParserErrors(t *testing.T) {
	for _, input := range []string{"1 +", "let = 1 in 2", "(1"} {
		if _, err := Parse(lexer.New(input)); err == nil {
			t.Errorf("Parse(%q) returned nil error", input)
		}
	}
}

func TestParserLiteralKind(t *testing.T) {
	program, err := Parse(lexer.New(`"hello"`))
	if err != nil {
		t.Fatal(err)
	}
	literal, ok := program.Expressions[0].(*ast.Literal)
	if !ok || literal.Value != "hello" {
		t.Fatalf("unexpected AST: %#v", program.Expressions[0])
	}
}

func TestParserUnaryMinusPrecedence(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"-5", "(-5)"},
		{"--5", "(-(-5))"},
		{"-(3 + 2)", "(-(3 + 2))"},
		{"2 * -3", "(2 * (-3))"},
		{"-2 * 3", "((-2) * 3)"},
		{"5 - -2", "(5 - (-2))"},
		{"-2 + 3 * -4", "((-2) + (3 * (-4)))"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			program, err := Parse(lexer.New(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			if got := program.String(); got != tt.want {
				t.Errorf("Parse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParserNotPrecedence(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"not true", "(not true)"},
		{"not not true", "(not (not true))"},
		{"not (1 < 2)", "(not (1 < 2))"},
		{"not true == false", "((not true) == false)"},
		{"not false == true", "((not false) == true)"},
		{"-not true", "(-(not true))"},
		{"not -1", "(not (-1))"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			program, err := Parse(lexer.New(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			if got := program.String(); got != tt.want {
				t.Errorf("Parse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
