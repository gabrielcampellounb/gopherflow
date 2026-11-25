package token

import "fmt"

type TokenType string

const (
	EOF TokenType = "EOF"

	Identifier TokenType = "IDENTIFIER"
	Number     TokenType = "NUMBER"
	String     TokenType = "STRING"

	Def   TokenType = "def"
	Let   TokenType = "let"
	In    TokenType = "in"
	If    TokenType = "if"
	Then  TokenType = "then"
	Else  TokenType = "else"
	Fn    TokenType = "fn"
	True  TokenType = "true"
	False TokenType = "false"
	Null  TokenType = "null"
	And   TokenType = "and"
	Or    TokenType = "or"
	Not   TokenType = "not"

	Plus         TokenType = "+"
	Minus        TokenType = "-"
	Star         TokenType = "*"
	Slash        TokenType = "/"
	Percent      TokenType = "%"
	Equal        TokenType = "=="
	NotEqual     TokenType = "!="
	Less         TokenType = "<"
	LessEqual    TokenType = "<="
	Greater      TokenType = ">"
	GreaterEq    TokenType = ">="
	Pipe         TokenType = "|>"
	Assign       TokenType = "="
	Arrow        TokenType = "=>"
	LeftParen    TokenType = "("
	RightParen   TokenType = ")"
	LeftBrace    TokenType = "{"
	RightBrace   TokenType = "}"
	LeftBracket  TokenType = "["
	RightBracket TokenType = "]"
	Comma        TokenType = ","
	Dot          TokenType = "."
	Colon        TokenType = ":"
	Semicolon    TokenType = ";"
)

var keywords = map[string]TokenType{
	"def": Def, "let": Let, "in": In, "if": If, "then": Then, "else": Else,
	"fn": Fn, "true": True, "false": False, "null": Null,
	"and": And, "or": Or, "not": Not,
}

func LookupIdentifier(value string) TokenType {
	if tokenType, ok := keywords[value]; ok {
		return tokenType
	}
	return Identifier
}

func IsIdentifierStart(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}

func IsIdentifierContinue(ch byte) bool {
	return IsIdentifierStart(ch) || ch >= '0' && ch <= '9' || ch == '_'
}

// IsIdentifier reports whether value can be used as an unquoted identifier.
func IsIdentifier(value string) bool {
	if len(value) == 0 || !IsIdentifierStart(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		if !IsIdentifierContinue(value[i]) {
			return false
		}
	}
	return LookupIdentifier(value) == Identifier
}

type Position struct {
	Offset int
	Line   int
	Column int
}

type Token struct {
	Type     TokenType
	Lexeme   string
	Position Position
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%q) at %d:%d", t.Type, t.Lexeme, t.Position.Line, t.Position.Column)
}
