package ast

import (
	"github.com/gabrielcampellounb/gopherflow/internal/token"
	"testing"
)

func TestNodeRenderingAndPosition(t *testing.T) {
	pos := token.Position{Offset: 4, Line: 2, Column: 3}
	x := &Identifier{Name: "x", Pos: pos}
	n := &Literal{Value: "2", Kind: token.Number, Pos: pos}
	b := &BinaryExpression{Left: x, Right: n, Operator: token.Token{Lexeme: "+", Type: token.Plus, Position: pos}}
	l := &LetExpression{Name: x, Value: n, Body: b, Pos: pos}
	for _, tt := range []struct {
		node     Node
		text     string
		position token.Position
	}{
		{&Literal{Value: "\r\t", Kind: token.String, Pos: pos}, `"\r\t"`, pos},
		{x, "x", pos}, {n, "2", pos}, {&Literal{Value: "a\n\"\\", Kind: token.String, Pos: pos}, `"a\n\"\\"`, pos},
		{b, "(x + 2)", pos}, {l, "let x = 2 in (x + 2)", pos},
		{&Program{}, "", token.Position{}}, {&Program{Expressions: []Expression{x, n}}, "x; 2", pos},
		{&BinaryExpression{Left: l, Right: l, Operator: token.Token{Lexeme: "+", Position: pos}}, "((let x = 2 in (x + 2)) + (let x = 2 in (x + 2)))", pos},
	} {
		if got := tt.node.String(); got != tt.text {
			t.Errorf("%T.String = %q, want %q", tt.node, got, tt.text)
		}
		if got := tt.node.Position(); got != tt.position {
			t.Errorf("%T.Position = %+v, want %+v", tt.node, got, tt.position)
		}
	}
}

func TestFunctionCallRenderingAndPosition(t *testing.T) {
	pos := token.Position{Offset: 4, Line: 2, Column: 3}
	x := &Identifier{Name: "x", Pos: pos}
	n := &Literal{Value: "1", Kind: token.Number, Pos: pos}
	sum := &BinaryExpression{Left: x, Right: n, Operator: token.Token{Lexeme: "+", Type: token.Plus, Position: pos}}

	fn := &FunctionLiteral{
		Token:      token.Token{Type: token.Fn, Lexeme: "fn", Position: pos},
		Parameters: []*Identifier{x},
		Body:       sum,
	}
	call := &CallExpression{Function: &Identifier{Name: "f", Pos: pos}, Arguments: []Expression{n, x}, Pos: pos}

	for _, tt := range []struct {
		node     Node
		text     string
		position token.Position
	}{
		{fn, "fn(x) => (x + 1)", pos},
		{&FunctionLiteral{Body: n}, "fn() => 1", token.Position{}},
		{call, "f(1, x)", pos},
		{&CallExpression{Function: &Identifier{Name: "f"}}, "f()", token.Position{}},
		{&CallExpression{Function: fn, Arguments: []Expression{n}}, "(fn(x) => (x + 1))(1)", token.Position{}},
		{&CallExpression{Function: call}, "f(1, x)()", token.Position{}},
	} {
		if got := tt.node.String(); got != tt.text {
			t.Errorf("%T.String = %q, want %q", tt.node, got, tt.text)
		}
		if got := tt.node.Position(); got != tt.position {
			t.Errorf("%T.Position = %+v, want %+v", tt.node, got, tt.position)
		}
	}
}

func TestDefinitionRenderingAndPosition(t *testing.T) {
	pos := token.Position{Offset: 4, Line: 2, Column: 3}
	other := token.Position{Offset: 20, Line: 3, Column: 1}

	x := &Identifier{Name: "x", Pos: pos}
	y := &Identifier{Name: "y", Pos: pos}
	one := &Literal{Value: "1", Kind: token.Number, Pos: pos}
	plus := token.Token{Lexeme: "+", Type: token.Plus, Position: pos}
	xPlusOne := &BinaryExpression{Left: x, Right: one, Operator: plus}
	xPlusY := &BinaryExpression{Left: x, Right: y, Operator: plus}

	g := &Definition{Name: &Identifier{Name: "g", Pos: pos}, Parameters: []*Identifier{x}, Body: xPlusOne, Pos: pos}
	add := &Definition{Name: &Identifier{Name: "add", Pos: pos}, Parameters: []*Identifier{x, y}, Body: xPlusY, Pos: pos}
	constant := &Definition{Name: &Identifier{Name: "c"}, Body: one}
	z := &Identifier{Name: "z", Pos: other}

	for _, tt := range []struct {
		node     Node
		text     string
		position token.Position
	}{
		{g, "def g(x) = (x + 1)", pos},
		{add, "def add(x, y) = (x + y)", pos},
		{constant, "def c() = 1", token.Position{}},
		{&Program{Definitions: []*Definition{g}, Expressions: []Expression{z}}, "def g(x) = (x + 1); z", pos},
		{&Program{Definitions: []*Definition{g, add}, Expressions: []Expression{z}}, "def g(x) = (x + 1); def add(x, y) = (x + y); z", pos},
		{&Program{Expressions: []Expression{z}}, "z", other},
	} {
		if got := tt.node.String(); got != tt.text {
			t.Errorf("%T.String = %q, want %q", tt.node, got, tt.text)
		}
		if got := tt.node.Position(); got != tt.position {
			t.Errorf("%T.Position = %+v, want %+v", tt.node, got, tt.position)
		}
	}
}
