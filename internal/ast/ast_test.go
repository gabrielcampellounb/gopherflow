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
	not := &UnaryExpression{Operator: token.Token{Lexeme: "not", Type: token.Not, Position: pos}, Right: &Literal{Value: "true", Kind: token.True, Pos: pos}}
	b := &BinaryExpression{Left: x, Right: n, Operator: token.Token{Lexeme: "+", Type: token.Plus, Position: pos}}
	l := &LetExpression{Name: x, Value: n, Body: b, Pos: pos}
	i := &IfExpression{Condition: &Literal{Value: "true", Kind: token.True, Pos: pos}, Consequence: n, Alternative: x, Pos: pos}
	for _, tt := range []struct {
		node     Node
		text     string
		position token.Position
	}{
		{&Literal{Value: "\r\t", Kind: token.String, Pos: pos}, `"\r\t"`, pos},
		{x, "x", pos}, {n, "2", pos}, {&Literal{Value: "a\n\"\\", Kind: token.String, Pos: pos}, `"a\n\"\\"`, pos},
		{u, "(-2)", pos},
		{not, "(not true)", pos},
		{b, "(x + 2)", pos}, {l, "let x = 2 in (x + 2)", pos},
		{i, "if true then 2 else x", pos},
		{&BinaryExpression{Left: i, Right: n, Operator: token.Token{Lexeme: "+", Position: pos}}, "((if true then 2 else x) + 2)", pos},
		{&UnaryExpression{Operator: token.Token{Lexeme: "-", Type: token.Minus, Position: pos}, Right: i}, "(-(if true then 2 else x))", pos},
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
	opPos := token.Position{Offset: 10, Line: 2, Column: 9}
	pipe := token.Token{Type: token.Pipe, Lexeme: "|>", Position: opPos}

	x := &Identifier{Name: "x", Pos: pos}
	y := &Identifier{Name: "y", Pos: pos}
	n := &Literal{Value: "1", Kind: token.Number, Pos: pos}
	sum := &BinaryExpression{Left: x, Right: n, Operator: token.Token{Lexeme: "+", Type: token.Plus, Position: pos}}

	fn1 := &FunctionLiteral{
		Token:      token.Token{Type: token.Fn, Lexeme: "fn", Position: pos},
		Parameters: []*Identifier{x},
		Body:       sum,
	}
	fn2 := &FunctionLiteral{
		Token:      token.Token{Type: token.Fn, Lexeme: "fn", Position: pos},
		Parameters: []*Identifier{x, y},
		Body:       sum,
	}
	call := &CallExpression{Function: &Identifier{Name: "f", Pos: pos}, Arguments: []Expression{n, x}, Pos: pos}

	for _, tt := range []struct {
		node     Node
		text     string
		position token.Position
	}{
		{fn1, "fn(x) => (x + 1)", pos},
		{&FunctionLiteral{Body: n}, "fn() => 1", token.Position{}},
		{call, "f(1, x)", pos},
		{&CallExpression{Function: &Identifier{Name: "f"}}, "f()", token.Position{}},
		{&CallExpression{Function: fn1, Arguments: []Expression{n}}, "(fn(x) => (x + 1))(1)", token.Position{}},
		{&CallExpression{Function: call}, "f(1, x)()", token.Position{}},
		{&BinaryExpression{Left: n, Right: fn2, Operator: pipe}, "(1 |> (fn(x, y) => (x + 1)))", opPos},
		{&BinaryExpression{Left: fn2, Right: n, Operator: pipe}, "((fn(x, y) => (x + 1)) |> 1)", opPos},
		{&BinaryExpression{Left: n, Right: x, Operator: pipe}, "(1 |> x)", opPos},
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

func TestListNodesRenderingAndPosition(t *testing.T) {
	pos := token.Position{Offset: 4, Line: 2, Column: 3}
	bracket := token.Position{Offset: 9, Line: 2, Column: 8}

	x := &Identifier{Name: "x", Pos: pos}
	one := &Literal{Value: "1", Kind: token.Number, Pos: pos}
	plus := token.Token{Type: token.Plus, Lexeme: "+", Position: pos}
	sum := &BinaryExpression{Left: x, Right: one, Operator: plus}
	fn := &FunctionLiteral{
		Token:      token.Token{Type: token.Fn, Lexeme: "fn", Position: pos},
		Parameters: []*Identifier{x},
		Body:       x,
	}

	call := &CallExpression{Function: &Identifier{Name: "f", Pos: pos}, Arguments: []Expression{one}, Pos: pos}
	list := &ListLiteral{Elements: []Expression{one, x}, Pos: bracket}
	index := &IndexExpression{Target: x, Index: one, Pos: bracket}

	for _, tt := range []struct {
		node     Node
		text     string
		position token.Position
	}{
		{&ListLiteral{Pos: bracket}, "[]", bracket},
		{&ListLiteral{Elements: []Expression{one}, Pos: bracket}, "[1]", bracket},
		{&ListLiteral{Elements: []Expression{one, x}, Pos: bracket}, "[1, x]", bracket},
		{&ListLiteral{Elements: []Expression{list, &ListLiteral{}}}, "[[1, x], []]", token.Position{}},
		{&ListLiteral{Elements: []Expression{sum, fn}}, "[(x + 1), fn(x) => x]", token.Position{}},

		{&IndexExpression{Target: x, Index: one, Pos: bracket}, "x[1]", bracket},
		{&IndexExpression{Target: list, Index: one}, "[1, x][1]", token.Position{}},
		{&IndexExpression{Target: index, Index: one}, "x[1][1]", token.Position{}},
		{&IndexExpression{Target: call, Index: one}, "f(1)[1]", token.Position{}},
		{&IndexExpression{Target: x, Index: sum}, "x[(x + 1)]", token.Position{}},
		{&IndexExpression{Target: fn, Index: one}, "(fn(x) => x)[1]", token.Position{}},

		{&CallExpression{Function: index}, "x[1]()", token.Position{}},
		{&BinaryExpression{Left: index, Right: one, Operator: plus}, "(x[1] + 1)", pos},
	} {
		if got := tt.node.String(); got != tt.text {
			t.Errorf("%T.String = %q, want %q", tt.node, got, tt.text)
		}
		if got := tt.node.Position(); got != tt.position {
			t.Errorf("%T.Position = %+v, want %+v", tt.node, got, tt.position)
		}
	}
}

func TestRecordAndFieldRendering(t *testing.T) {
	recordPos := token.Position{Line: 1, Column: 1}
	fieldPos := token.Position{Line: 1, Column: 7}
	name := &Identifier{Name: "x"}
	record := &RecordLiteral{
		Fields: []RecordField{{Name: name, Value: &Literal{Kind: token.Number, Value: "1"}}},
		Pos:    recordPos,
	}
	field := &FieldExpression{Target: record, Name: name, Pos: fieldPos}
	for _, tt := range []struct {
		node Node
		text string
		pos  token.Position
	}{
		{record, "{x: 1}", recordPos},
		{field, "{x: 1}.x", fieldPos},
		{&RecordLiteral{}, "{}", token.Position{}},
	} {
		if tt.node.String() != tt.text || tt.node.Position() != tt.pos {
			t.Errorf("got %q at %v, want %q at %v", tt.node.String(), tt.node.Position(), tt.text, tt.pos)
		}
	}
}
