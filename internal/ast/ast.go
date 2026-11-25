package ast

import (
	"strings"

	"github.com/gabrielcampellounb/gopherflow/internal/token"
)

type Node interface {
	String() string
	Position() token.Position
}

type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Definitions []*Definition
	Expressions []Expression
}

func (p *Program) String() string {
	parts := make([]string, 0, len(p.Definitions)+len(p.Expressions))
	for _, definition := range p.Definitions {
		parts = append(parts, definition.String())
	}
	for _, expression := range p.Expressions {
		parts = append(parts, expression.String())
	}
	return strings.Join(parts, "; ")
}

func (p *Program) Position() token.Position {
	if len(p.Definitions) > 0 {
		return p.Definitions[0].Position()
	}
	if len(p.Expressions) == 0 {
		return token.Position{}
	}
	return p.Expressions[0].Position()
}

type ListLiteral struct {
	Elements []Expression
	Pos      token.Position
}

func (l *ListLiteral) expressionNode() {}

func (l *ListLiteral) String() string {
	elements := make([]string, len(l.Elements))
	for i, element := range l.Elements {
		elements[i] = element.String()
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

func (l *ListLiteral) Position() token.Position { return l.Pos }

type IndexExpression struct {
	Target Expression
	Index  Expression
	Pos    token.Position
}

func (i *IndexExpression) expressionNode() {}

func (i *IndexExpression) String() string {
	return groupedOperand(i.Target) + "[" + i.Index.String() + "]"
}

func (i *IndexExpression) Position() token.Position { return i.Pos }

type Definition struct {
	Name       *Identifier
	Parameters []*Identifier
	Body       Expression
	Pos        token.Position
}

func (d *Definition) String() string {
	return "def " + d.Name.String() + "(" + identifierList(d.Parameters) + ") = " + d.Body.String()
}

func (d *Definition) Position() token.Position { return d.Pos }

type CallExpression struct {
	Function  Expression
	Arguments []Expression
	Pos       token.Position
}

func (c *CallExpression) expressionNode() {}

func (c *CallExpression) String() string {
	args := make([]string, len(c.Arguments))
	for i, argument := range c.Arguments {
		args[i] = argument.String()
	}
	return groupedOperand(c.Function) + "(" + strings.Join(args, ", ") + ")"
}

func (c *CallExpression) Position() token.Position { return c.Pos }

type FunctionLiteral struct {
	Token      token.Token
	Parameters []*Identifier
	Body       Expression
}

func (f *FunctionLiteral) expressionNode() {}

func (f *FunctionLiteral) String() string {
	return "fn(" + identifierList(f.Parameters) + ") => " + f.Body.String()
}

func (f *FunctionLiteral) Position() token.Position {
	return f.Token.Position
}

type Identifier struct {
	Name string
	Pos  token.Position
}

func (i *Identifier) expressionNode()          {}
func (i *Identifier) String() string           { return i.Name }
func (i *Identifier) Position() token.Position { return i.Pos }

type Literal struct {
	Value string
	Kind  token.TokenType
	Pos   token.Position
}

func (l *Literal) expressionNode() {}
func (l *Literal) String() string {
	if l.Kind == token.String {
		return quoteString(l.Value)
	}
	return l.Value
}
func (l *Literal) Position() token.Position { return l.Pos }

type UnaryExpression struct {
	Operator token.Token
	Right    Expression
}

func (u *UnaryExpression) expressionNode() {}
func (u *UnaryExpression) String() string {
	right := groupedOperand(u.Right)
	if u.Operator.Type == token.Not {
		return "(" + u.Operator.Lexeme + " " + right + ")"
	}
	return "(" + u.Operator.Lexeme + right + ")"
}
func (u *UnaryExpression) Position() token.Position { return u.Operator.Position }

type BinaryExpression struct {
	Left     Expression
	Operator token.Token
	Right    Expression
}

func (b *BinaryExpression) expressionNode() {}
func (b *BinaryExpression) String() string {
	return "(" + groupedOperand(b.Left) + " " + b.Operator.Lexeme + " " + groupedOperand(b.Right) + ")"
}

// Preserve the scope of expressions whose bodies extend to the next delimiter.
func groupedOperand(expression Expression) string {
	switch expression.(type) {
	case *LetExpression, *FunctionLiteral, *IfExpression:
		return "(" + expression.String() + ")"
	}
	return expression.String()
}

func (b *BinaryExpression) Position() token.Position { return b.Operator.Position }

type LetExpression struct {
	Name  *Identifier
	Value Expression
	Body  Expression
	Pos   token.Position
}

func (l *LetExpression) expressionNode() {}
func (l *LetExpression) String() string {
	return "let " + l.Name.String() + " = " + l.Value.String() + " in " + l.Body.String()
}
func (l *LetExpression) Position() token.Position { return l.Pos }

type IfExpression struct {
	Condition   Expression
	Consequence Expression
	Alternative Expression
	Pos         token.Position
}

func (i *IfExpression) expressionNode() {}
func (i *IfExpression) String() string {
	return "if " + i.Condition.String() + " then " + i.Consequence.String() + " else " + i.Alternative.String()
}
func (i *IfExpression) Position() token.Position { return i.Pos }

// quoteString uses only escapes accepted by the lexer, preserving other bytes.
// Go's strconv.Quote also emits escapes such as \xNN that GopherFlow does not support.
func quoteString(value string) string {
	var out strings.Builder
	out.Grow(len(value) + 2)
	out.WriteByte('"')
	for i := 0; i < len(value); i++ {
		switch ch := value[i]; ch {
		case '"', '\\':
			out.WriteByte('\\')
			out.WriteByte(ch)
		case '\n':
			out.WriteString(`\n`)
		case '\r':
			out.WriteString(`\r`)
		case '\t':
			out.WriteString(`\t`)
		default:
			out.WriteByte(ch)
		}
	}
	out.WriteByte('"')
	return out.String()
}

func identifierList(identifiers []*Identifier) string {
	names := make([]string, len(identifiers))
	for i, identifier := range identifiers {
		names[i] = identifier.String()
	}
	return strings.Join(names, ", ")
}
