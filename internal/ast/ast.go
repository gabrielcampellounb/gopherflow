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

type UnaryExpression struct {
	Operator token.Token
	Right    Expression
}

func (u *UnaryExpression) expressionNode() {}
func (u *UnaryExpression) String() string {
	right := u.Right.String()
	if _, ok := u.Right.(*IfExpression); ok {
		right = "(" + right + ")"
	}
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
	return "(" + operandString(b.Left) + " " + b.Operator.Lexeme + " " + operandString(b.Right) + ")"
}

// Preserve grouping of let and if expressions used as binary operands.
func operandString(expression Expression) string {
	if _, ok := expression.(*LetExpression); ok {
		return "(" + expression.String() + ")"
	}
	if _, ok := expression.(*IfExpression); ok {
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
