package lexer

import (
	"fmt"

	"github.com/gabrielcampellounb/gopherflow/internal/token"
)

type Lexer struct {
	source string
	offset int
	line   int
	column int
}

type Error struct {
	Position token.Position
	Message  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("lexer error at %d:%d: %s", e.Position.Line, e.Position.Column, e.Message)
}

func New(source string) *Lexer {
	return &Lexer{source: source, line: 1, column: 1}
}

func (l *Lexer) Next() (token.Token, error) {
	l.skipWhitespace()
	start := l.position()
	if l.atEnd() {
		return token.Token{Type: token.EOF, Position: start}, nil
	}

	ch := l.advance()
	if isLetter(ch) || ch == '_' {
		lexeme := l.readIdentifier()
		return token.Token{Type: token.LookupIdentifier(lexeme), Lexeme: lexeme, Position: start}, nil
	}
	if isDigit(ch) {
		lexeme := l.readNumber()
		return token.Token{Type: token.Number, Lexeme: lexeme, Position: start}, nil
	}
	if ch == '"' {
		lexeme, err := l.readString(start)
		if err != nil {
			return token.Token{}, err
		}
		return token.Token{Type: token.String, Lexeme: lexeme, Position: start}, nil
	}

	var kind token.TokenType
	switch ch {
	case '+':
		kind = token.Plus
	case '-':
		kind = token.Minus
	case '*':
		kind = token.Star
	case '/':
		kind = token.Slash
	case '%':
		kind = token.Percent
	case '(':
		kind = token.LeftParen
	case ')':
		kind = token.RightParen
	case '{':
		kind = token.LeftBrace
	case '}':
		kind = token.RightBrace
	case '[':
		kind = token.LeftBracket
	case ']':
		kind = token.RightBracket
	case ',':
		kind = token.Comma
	case '.':
		kind = token.Dot
	case ':':
		kind = token.Colon
	case ';':
		kind = token.Semicolon
	case '=':
		kind = token.Assign
		if l.match('=') {
			kind = token.Equal
		} else if l.match('>') {
			kind = token.Arrow
		}
	case '!':
		if l.match('=') {
			kind = token.NotEqual
		}
	case '<':
		kind = token.Less
		if l.match('=') {
			kind = token.LessEqual
		}
	case '>':
		kind = token.Greater
		if l.match('=') {
			kind = token.GreaterEq
		}
	case '|':
		if l.match('>') {
			kind = token.Pipe
		}
	}
	if kind != "" {
		return token.Token{Type: kind, Lexeme: l.source[start.Offset:l.offset], Position: start}, nil
	}

	return token.Token{}, &Error{Position: start, Message: fmt.Sprintf("unexpected character %q", ch)}
}

func (l *Lexer) skipWhitespace() {
	for !l.atEnd() {
		switch l.source[l.offset] {
		case ' ', '\t', '\r':
			l.advance()
		case '\n':
			l.advance()
		default:
			return
		}
	}
}

func (l *Lexer) readIdentifier() string {
	start := l.offset - 1
	for !l.atEnd() {
		ch := l.source[l.offset]
		if !isLetter(ch) && !isDigit(ch) && ch != '_' {
			break
		}
		l.advance()
	}
	return l.source[start:l.offset]
}

func (l *Lexer) readNumber() string {
	start := l.offset - 1
	for !l.atEnd() && isDigit(l.peek()) {
		l.advance()
	}
	if l.peek() == '.' && l.offset+1 < len(l.source) && isDigit(l.source[l.offset+1]) {
		l.advance()
		for !l.atEnd() && isDigit(l.peek()) {
			l.advance()
		}
	}
	return l.source[start:l.offset]
}

func isLetter(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func (l *Lexer) readString(start token.Position) (string, error) {
	var value []byte
	for !l.atEnd() {
		ch := l.advance()
		if ch == '"' {
			return string(value), nil
		}
		if ch == '\\' {
			if l.atEnd() {
				break
			}
			escapePosition := l.position()
			escaped := l.advance()
			switch escaped {
			case '"', '\\':
				value = append(value, escaped)
			case 'n':
				value = append(value, '\n')
			case 'r':
				value = append(value, '\r')
			case 't':
				value = append(value, '\t')
			default:
				return "", &Error{Position: escapePosition, Message: fmt.Sprintf("invalid escape \\%c", escaped)}
			}
			continue
		}
		if ch == '\n' {
			return "", &Error{Position: start, Message: "unterminated string"}
		}
		value = append(value, ch)
	}
	return "", &Error{Position: start, Message: "unterminated string"}
}

func (l *Lexer) position() token.Position {
	return token.Position{Offset: l.offset, Line: l.line, Column: l.column}
}

func (l *Lexer) advance() byte {
	ch := l.source[l.offset]
	l.offset++
	if ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	return ch
}

func (l *Lexer) peek() byte {
	if l.atEnd() {
		return 0
	}
	return l.source[l.offset]
}

func (l *Lexer) atEnd() bool {
	return l.offset >= len(l.source)
}

func (l *Lexer) match(expected byte) bool {
	if l.atEnd() || l.peek() != expected {
		return false
	}
	l.advance()
	return true
}
