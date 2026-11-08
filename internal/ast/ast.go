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
	Expressions []Expression
}

func (p *Program) String() string {
	parts := make([]string, len(p.Expressions))
	for i, expression := range p.Expressions {
		parts[i] = expression.String()
	}
	return strings.Join(parts, "; ")
}

func (p *Program) Position() token.Position {
	if len(p.Expressions) == 0 {
		return token.Position{}
	}
	return p.Expressions[0].Position()
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

type BinaryExpression struct {
	Left     Expression
	Operator token.Token
	Right    Expression
}

func (b *BinaryExpression) expressionNode() {}
func (b *BinaryExpression) String() string {
	return "(" + operandString(b.Left) + " " + b.Operator.Lexeme + " " + operandString(b.Right) + ")"
}

// A let body extends to the end of its expression. Preserve explicit grouping
// when rendering it as a binary operand so parsing the result keeps its scope.
func operandString(expression Expression) string {
	if _, ok := expression.(*LetExpression); ok {
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
