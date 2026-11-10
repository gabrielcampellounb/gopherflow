package parser

import (
	"strings"
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

func TestParserLogicalPrecedence(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"true or false and false", "(true or (false and false))"},
		{"not false and true", "((not false) and true)"},
		{"1 < 2 and 3 > 2 or false", "(((1 < 2) and (3 > 2)) or false)"},
		{"true and false and true", "((true and false) and true)"},
		{"false or true or false", "((false or true) or false)"},
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

func TestParserIfExpression(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"if true then 10 else 20", "if true then 10 else 20"},
		{"if 5 > 3 then 100 else 200", "if (5 > 3) then 100 else 200"},
		{"if true and not false then 1 else 2", "if (true and (not false)) then 1 else 2"},
		{"if true then if false then 1 else 2 else 3", "if true then if false then 1 else 2 else 3"},
		{"if true then 1 else if false then 2 else 3", "if true then 1 else if false then 2 else 3"},
		{"if true then 1 else 2 + 3", "if true then 1 else (2 + 3)"},
		{"(if true then 1 else 2) + 3", "((if true then 1 else 2) + 3)"},
		{"2 * (if true then 3 else 4)", "(2 * (if true then 3 else 4))"},
		{"not (if true then false else true)", "(not (if true then false else true))"},
		{"let x = if true then 1 else 2 in x", "let x = if true then 1 else 2 in x"},
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

func TestParserIfErrors(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"if", "expected expression, got EOF"},
		{"if true", "expected 'then' after if condition"},
		{"if true then", "expected expression, got EOF"},
		{"if true then 1", "expected 'else' after then branch"},
		{"if true then 1 else", "expected expression, got EOF"},
		{"1 + if true then 2 else 3", "expected expression, got if"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			program, err := Parse(lexer.New(tt.input))
			if program != nil || err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Parse(%q) = %v, %v; want %q", tt.input, program, err, tt.want)
			}
		})
	}
	program, err := Parse(lexer.New("\n if true then 1"))
	if program != nil || err == nil || err.Error() != "parser error at 2:16: expected 'else' after then branch" {
		t.Fatalf("unexpected syntax error: %v, %v", program, err)
	}
}
