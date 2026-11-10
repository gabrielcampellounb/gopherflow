package parser

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gabrielcampellounb/gopherflow/internal/ast"
	"github.com/gabrielcampellounb/gopherflow/internal/lexer"
	"github.com/gabrielcampellounb/gopherflow/internal/token"
)

func TestPrecedenceAndAssociativity(t *testing.T) {
	for _, tt := range []struct{ source, want string }{
		{"1 - 2 - 3", "((1 - 2) - 3)"}, {"8 / 4 * 2 % 3", "(((8 / 4) * 2) % 3)"},
		{"(1 + 2) * 3", "((1 + 2) * 3)"}, {"1 + (2 * (3 + 4))", "(1 + (2 * (3 + 4)))"},
		{"1 |> 2 or 3 and 4 == 5 < 6 + 7 * 8", "(1 |> (2 or (3 and (4 == (5 < (6 + (7 * 8)))))))"},
		{"1 != 2 == 3", "((1 != 2) == 3)"},
		{"1 <= 2", "(1 <= 2)"}, {"1 >= 2", "(1 >= 2)"}, {"1 > 2", "(1 > 2)"},
		{"let x = 1 + 2 in let y = x * 3 in y", "let x = (1 + 2) in let y = (x * 3) in y"},
		{"let x = let y = 1 in y in x", "let x = let y = 1 in y in x"},
		{"null; \n", "null"},
	} {
		t.Run(tt.source, func(t *testing.T) {
			p, err := Parse(lexer.New(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Expressions) != 1 || p.String() != tt.want {
				t.Fatalf("got %s, want %s", p.String(), tt.want)
			}
		})
	}
}

func TestLiteralNodes(t *testing.T) {
	for _, tt := range []struct {
		source, value string
		kind          token.TokenType
	}{
		{"12.25", "12.25", token.Number}, {`"hi\n"`, "hi\n", token.String},
		{"true", "true", token.True}, {"false", "false", token.False}, {"null", "null", token.Null},
	} {
		p, err := Parse(lexer.New(" \n " + tt.source))
		if err != nil {
			t.Fatal(err)
		}
		got, ok := p.Expressions[0].(*ast.Literal)
		if !ok || got.Kind != tt.kind || got.Value != tt.value || got.Pos != (token.Position{Offset: 3, Line: 2, Column: 2}) {
			t.Fatalf("%q: %#v", tt.source, p.Expressions[0])
		}
	}
}

func TestLetAST(t *testing.T) {
	p, err := Parse(lexer.New("let x = 2 in x + 1"))
	if err != nil {
		t.Fatal(err)
	}
	want := &ast.Program{Expressions: []ast.Expression{&ast.LetExpression{
		Name:  &ast.Identifier{Name: "x", Pos: token.Position{Offset: 4, Line: 1, Column: 5}},
		Value: &ast.Literal{Value: "2", Kind: token.Number, Pos: token.Position{Offset: 8, Line: 1, Column: 9}},
		Body: &ast.BinaryExpression{
			Left:     &ast.Identifier{Name: "x", Pos: token.Position{Offset: 13, Line: 1, Column: 14}},
			Operator: token.Token{Type: token.Plus, Lexeme: "+", Position: token.Position{Offset: 15, Line: 1, Column: 16}},
			Right:    &ast.Literal{Value: "1", Kind: token.Number, Pos: token.Position{Offset: 17, Line: 1, Column: 18}},
		}, Pos: token.Position{Line: 1, Column: 1},
	}}}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("unexpected let AST: %#v", p.Expressions[0])
	}
}

func TestUnaryMinusASTPosition(t *testing.T) {
	program, err := Parse(lexer.New("\n -5"))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := program.Expressions[0].(*ast.UnaryExpression)
	if !ok || got.Operator.Type != token.Minus || got.Position() != (token.Position{Offset: 2, Line: 2, Column: 2}) {
		t.Fatalf("unexpected unary expression: %#v", program.Expressions[0])
	}
	right, ok := got.Right.(*ast.Literal)
	if !ok || right.Position() != (token.Position{Offset: 3, Line: 2, Column: 3}) {
		t.Fatalf("unexpected unary operand: %#v", got.Right)
	}
}

func TestSyntaxErrorDetails(t *testing.T) {
	for _, tt := range []struct {
		source, message      string
		offset, line, column int
	}{
		{"", "expected expression, got EOF", 0, 1, 1},
		{" \n", "expected expression, got EOF", 2, 2, 1},
		{"1 +", "expected expression, got EOF", 3, 1, 4},
		{"-", "expected expression, got EOF", 1, 1, 2},
		{"2 * -", "expected expression, got EOF", 5, 1, 6},
		{"let = 1 in 2", "expected identifier after let", 4, 1, 5},
		{"let x 1 in x", "expected '=' after identifier", 6, 1, 7},
		{"let x = 1 x", "expected 'in' after let value", 10, 1, 11},
		{"let x = 1 in", "expected expression, got EOF", 12, 1, 13},
		{"(1", "expected ')'", 2, 1, 3},
		{"1 2", "expected end of input, got NUMBER", 2, 1, 3},
		{"1;;", "expected end of input, got ;", 2, 1, 3},
		{"1.", "expected end of input, got .", 1, 1, 2},
	} {
		t.Run(tt.source, func(t *testing.T) {
			p, err := Parse(lexer.New(tt.source))
			var pe *Error
			if p != nil || !errors.As(err, &pe) {
				t.Fatalf("got %v, %v", p, err)
			}
			if pe.Message != tt.message || pe.Position != (token.Position{Offset: tt.offset, Line: tt.line, Column: tt.column}) {
				t.Fatalf("got %#v", pe)
			}
			if !strings.Contains(err.Error(), tt.message) {
				t.Fatal(err)
			}
		})
	}
}

func TestLexicalErrorsPropagate(t *testing.T) {
	for _, source := range []string{"@", "1 @", "1; @", "( @", "(1 @", "1 + @", "let @", "let x @", "let x = @", "let x = 1 in @", `"\q"`} {
		p, err := Parse(lexer.New(source))
		var le *lexer.Error
		if p != nil || !errors.As(err, &le) {
			t.Errorf("%q: got %v, %v; want lexical error", source, p, err)
		}
	}
}

func TestInvalidAndUnimplementedSyntax(t *testing.T) {
	for _, source := range []string{"()", "(1))", "1 + ;", "let true = 1 in 2", "let x = in x", "let x = 1", "let x = 1 in", "[1]", "{x: 1}", "fn(x) => x", "if true then 1 else 2", "def f() = 1; f()", "x(1)", "x.y", "x[0]", "-let x = 1 in x", "not true", "1; 2", strings.Repeat("9", 400)} {
		p, err := Parse(lexer.New(source))
		if err == nil || p != nil {
			t.Errorf("accepted %q: %v", source, p)
		}
	}
}

func TestProgramStringRoundTrip(t *testing.T) {
	for _, source := range []string{"\"\x00\a\b\f\v\xff\"", `"a\n\"\\"`, "1 + 2 * 3", "let x = 1 in x + 2", "(let x = 1 in x) + 2", "2 * (let x = 3 in x) + 1", "-5", "--5", "-(3 + 2)", "2 * -3"} {
		p, err := Parse(lexer.New(source))
		if err != nil {
			t.Fatal(err)
		}
		q, err := Parse(lexer.New(p.String()))
		if err != nil {
			t.Fatalf("%q: %v", source, err)
		}
		if p.String() != q.String() {
			t.Errorf("%q: round trip changed %q to %q", source, p.String(), q.String())
		}
	}
}

func FuzzParser(f *testing.F) {
	for _, s := range []string{"", "null", "let x = 1 in x + 2", "(1 + 2) * 3", "1 +", `"\n"`, "(let x = 1 in x) + 2"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		// Bound recursive grammar depth and test throughput; arbitrary bytes remain valid inputs.
		if len(s) > 4096 {
			t.Skip()
		}
		p, err := Parse(lexer.New(s))
		if err != nil {
			if p != nil {
				t.Fatal("partial program returned on error")
			}
			return
		}
		if p == nil || len(p.Expressions) != 1 {
			t.Fatal("invalid successful parse")
		}
		q, err := Parse(lexer.New(p.String()))
		if err != nil {
			t.Fatalf("cannot parse AST rendering %q: %v", p.String(), err)
		}
		if p.String() != q.String() {
			t.Fatalf("unstable AST rendering: %q -> %q", p.String(), q.String())
		}
	})
}

func TestStringRenderingPreservesEveryByte(t *testing.T) {
	var bytes []byte
	for i := 0; i < 256; i++ {
		bytes = append(bytes, byte(i))
	}
	literal := &ast.Literal{Kind: token.String, Value: string(bytes)}
	p, err := Parse(lexer.New(literal.String()))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.Expressions[0].(*ast.Literal)
	if !ok || got.Kind != token.String || got.Value != literal.Value {
		t.Fatal("string rendering changed bytes")
	}
}
