package ast

import (
	"github.com/gabrielcampellounb/gopherflow/internal/token"
	"testing"
)

func TestNodeRenderingAndPosition(t *testing.T) {
	pos := token.Position{Offset: 4, Line: 2, Column: 3}
	x := &Identifier{Name: "x", Pos: pos}
	n := &Literal{Value: "2", Kind: token.Number, Pos: pos}
	u := &UnaryExpression{Operator: token.Token{Lexeme: "-", Type: token.Minus, Position: pos}, Right: n}
	b := &BinaryExpression{Left: x, Right: n, Operator: token.Token{Lexeme: "+", Type: token.Plus, Position: pos}}
	l := &LetExpression{Name: x, Value: n, Body: b, Pos: pos}
	for _, tt := range []struct {
		node     Node
		text     string
		position token.Position
	}{
		{&Literal{Value: "\r\t", Kind: token.String, Pos: pos}, `"\r\t"`, pos},
		{x, "x", pos}, {n, "2", pos}, {&Literal{Value: "a\n\"\\", Kind: token.String, Pos: pos}, `"a\n\"\\"`, pos},
		{u, "(-2)", pos},
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
