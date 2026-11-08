package lexer

import (
	"testing"

	"github.com/gabrielcampellounb/gopherflow/internal/token"
)

func TestLexerTokens(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []token.TokenType
	}{
		{"keywords and operators", `let x = 12.5 |> fn(y) => y + 1`, []token.TokenType{
			token.Let, token.Identifier, token.Assign, token.Number, token.Pipe,
			token.Fn, token.LeftParen, token.Identifier, token.RightParen,
			token.Arrow, token.Identifier, token.Plus, token.Number, token.EOF,
		}},
		{"string and punctuation", `"hello\nworld"; [true, null]`, []token.TokenType{
			token.String, token.Semicolon, token.LeftBracket, token.True, token.Comma,
			token.Null, token.RightBracket, token.EOF,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.src)
			for i, want := range tt.want {
				got, err := l.Next()
				if err != nil {
					t.Fatal(err)
				}
				if got.Type != want {
					t.Fatalf("token %d = %s, want %s", i, got.Type, want)
				}
			}
		})
	}
}

func TestLexerPosition(t *testing.T) {
	l := New("let\nx")
	_, _ = l.Next()
	got, err := l.Next()
	if err != nil {
		t.Fatal(err)
	}
	if got.Position.Line != 2 || got.Position.Column != 1 {
		t.Fatalf("position = %+v, want line 2 column 1", got.Position)
	}
}

func TestLexerErrors(t *testing.T) {
	for _, src := range []string{`"unterminated`, "@"} {
		if _, err := New(src).Next(); err == nil {
			t.Errorf("Next(%q) returned nil error", src)
		}
	}
}
