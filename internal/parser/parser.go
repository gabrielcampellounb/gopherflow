package parser

import (
	"fmt"
	"strconv"

	"github.com/gabrielcampellounb/gopherflow/internal/ast"
	"github.com/gabrielcampellounb/gopherflow/internal/lexer"
	"github.com/gabrielcampellounb/gopherflow/internal/token"
)

type Error struct {
	Position token.Position
	Message  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("parser error at %d:%d: %s", e.Position.Line, e.Position.Column, e.Message)
}

type Parser struct {
	lexer   *lexer.Lexer
	current token.Token
}

func New(l *lexer.Lexer) *Parser {
	return &Parser{lexer: l}
}

func Parse(l *lexer.Lexer) (*ast.Program, error) {
	return New(l).Parse()
}

func (p *Parser) Parse() (*ast.Program, error) {
	if err := p.advance(); err != nil {
		return nil, err
	}
	var definitions []*ast.Definition
	defined := make(map[string]bool)
	for p.current.Type == token.Def {
		definition, err := p.parseDefinition()
		if err != nil {
			return nil, err
		}
		if defined[definition.Name.Name] {
			return nil, &Error{
				Position: definition.Name.Pos,
				Message:  fmt.Sprintf("duplicate definition %q", definition.Name.Name),
			}
		}
		defined[definition.Name.Name] = true
		definitions = append(definitions, definition)
		if p.current.Type != token.Semicolon {
			return nil, p.errorf("expected ';' after definition")
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
	}
	expression, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if p.current.Type == token.Semicolon {
		if err := p.advance(); err != nil {
			return nil, err
		}
	}
	if p.current.Type != token.EOF {
		return nil, p.errorf("expected end of input, got %s", p.current.Type)
	}
	return &ast.Program{Definitions: definitions, Expressions: []ast.Expression{expression}}, nil
}

func (p *Parser) parseExpression(precedence int) (ast.Expression, error) {
	var left ast.Expression
	var err error
	switch p.current.Type {
	case token.Identifier:
		left = &ast.Identifier{Name: p.current.Lexeme, Pos: p.current.Position}
	case token.Number:
		if _, err = strconv.ParseFloat(p.current.Lexeme, 64); err != nil {
			return nil, p.errorf("invalid number %q", p.current.Lexeme)
		}
		left = &ast.Literal{Value: p.current.Lexeme, Kind: token.Number, Pos: p.current.Position}
	case token.String, token.True, token.False, token.Null:
		left = &ast.Literal{Value: p.current.Lexeme, Kind: p.current.Type, Pos: p.current.Position}
	case token.LeftParen:
		if err = p.advance(); err != nil {
			return nil, err
		}
		left, err = p.parseExpression(0)
		if err == nil && p.current.Type != token.RightParen {
			err = p.errorf("expected ')'")
		}
	default:
		switch p.current.Type {
		case token.Let:
			return p.parseLet()
		case token.Fn:
			return p.parseFunction()
		}
		return nil, p.errorf("expected expression, got %s", p.current.Type)
	}
	if err != nil {
		return nil, err
	}
	if err = p.advance(); err != nil {
		return nil, err
	}
	for p.current.Type == token.LeftParen {
		left, err = p.parseCall(left)
		if err != nil {
			return nil, err
		}
	}

	for precedence < precedenceOf(p.current.Type) {
		operator := p.current
		operatorPrecedence := precedenceOf(operator.Type)
		if err = p.advance(); err != nil {
			return nil, err
		}
		right, rightErr := p.parseExpression(operatorPrecedence)
		if rightErr != nil {
			return nil, rightErr
		}
		left = &ast.BinaryExpression{Left: left, Operator: operator, Right: right}
	}
	return left, nil
}

func (p *Parser) parseDefinition() (*ast.Definition, error) {
	position := p.current.Position
	if err := p.advance(); err != nil {
		return nil, err
	}
	if p.current.Type != token.Identifier {
		return nil, p.errorf("expected identifier after def")
	}
	name := &ast.Identifier{Name: p.current.Lexeme, Pos: p.current.Position}
	if err := p.advance(); err != nil {
		return nil, err
	}
	parameters, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	if p.current.Type != token.Assign {
		return nil, p.errorf("expected '=' after parameters")
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	body, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return &ast.Definition{Name: name, Parameters: parameters, Body: body, Pos: position}, nil
}

func (p *Parser) parseLet() (ast.Expression, error) {
	position := p.current.Position
	if err := p.advance(); err != nil {
		return nil, err
	}
	if p.current.Type != token.Identifier {
		return nil, p.errorf("expected identifier after let")
	}
	name := &ast.Identifier{Name: p.current.Lexeme, Pos: p.current.Position}
	if err := p.advance(); err != nil {
		return nil, err
	}
	if p.current.Type != token.Assign {
		return nil, p.errorf("expected '=' after identifier")
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	value, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if p.current.Type != token.In {
		return nil, p.errorf("expected 'in' after let value")
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	body, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return &ast.LetExpression{Name: name, Value: value, Body: body, Pos: position}, nil
}

func (p *Parser) parseCall(callee ast.Expression) (ast.Expression, error) {
	position := p.current.Position
	if err := p.advance(); err != nil {
		return nil, err
	}
	var arguments []ast.Expression
	if p.current.Type != token.RightParen {
		for {
			argument, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			arguments = append(arguments, argument)
			if p.current.Type != token.Comma {
				break
			}
			if err := p.advance(); err != nil {
				return nil, err
			}
		}
	}
	if p.current.Type != token.RightParen {
		return nil, p.errorf("expected ')' after arguments")
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	return &ast.CallExpression{Function: callee, Arguments: arguments, Pos: position}, nil
}

func (p *Parser) parseFunction() (ast.Expression, error) {
	start := p.current
	if err := p.advance(); err != nil {
		return nil, err
	}
	parameters, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	if p.current.Type != token.Arrow {
		return nil, p.errorf("expected '=>' after parameters")
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	body, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return &ast.FunctionLiteral{Token: start, Parameters: parameters, Body: body}, nil
}

func (p *Parser) parseParameters() ([]*ast.Identifier, error) {
	if p.current.Type != token.LeftParen {
		return nil, p.errorf("expected '(' before parameters")
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	var parameters []*ast.Identifier
	seen := make(map[string]bool)
	if p.current.Type != token.RightParen {
		for {
			if p.current.Type != token.Identifier {
				return nil, p.errorf("expected parameter name, got %s", p.current.Type)
			}
			if seen[p.current.Lexeme] {
				return nil, p.errorf("duplicate parameter %q", p.current.Lexeme)
			}
			seen[p.current.Lexeme] = true
			parameters = append(parameters, &ast.Identifier{Name: p.current.Lexeme, Pos: p.current.Position})
			if err := p.advance(); err != nil {
				return nil, err
			}
			if p.current.Type != token.Comma {
				break
			}
			if err := p.advance(); err != nil {
				return nil, err
			}
		}
	}
	if p.current.Type != token.RightParen {
		return nil, p.errorf("expected ')' after parameters")
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	return parameters, nil
}

func precedenceOf(kind token.TokenType) int {
	switch kind {
	case token.Pipe:
		return 1
	case token.Or:
		return 2
	case token.And:
		return 3
	case token.Equal, token.NotEqual:
		return 4
	case token.Less, token.LessEqual, token.Greater, token.GreaterEq:
		return 5
	case token.Plus, token.Minus:
		return 6
	case token.Star, token.Slash, token.Percent:
		return 7
	default:
		return 0
	}
}

func (p *Parser) advance() error {
	next, err := p.lexer.Next()
	if err != nil {
		return err
	}
	p.current = next
	return nil
}

func (p *Parser) errorf(format string, args ...any) error {
	return &Error{Position: p.current.Position, Message: fmt.Sprintf(format, args...)}
}
