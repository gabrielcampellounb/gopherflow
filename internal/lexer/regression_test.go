package lexer

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gabrielcampellounb/gopherflow/internal/token"
)

func TestStringEscapes(t *testing.T) {
	for _, tt := range []struct{ source, want string }{
		{`""`, ""}, {`"plain"`, "plain"}, {`"a\nb\rc\td"`, "a\nb\rc\td"},
		{`"\"\\"`, "\"\\"}, {`"ação 😀"`, "ação 😀"}, {`"\\n"`, `\n`},
	} {
		t.Run(tt.source, func(t *testing.T) {
			got, err := New(tt.source).Next()
			if err != nil {
				t.Fatal(err)
			}
			if got.Type != token.String || got.Lexeme != tt.want {
				t.Fatalf("got %#v, want STRING %q", got, tt.want)
			}
		})
	}
}

func TestNumberBoundaries(t *testing.T) {
	for _, tt := range []struct {
		source string
		want   []token.Token
	}{
		{"12.5", []token.Token{{Type: token.Number, Lexeme: "12.5"}}},
		{"12.", []token.Token{{Type: token.Number, Lexeme: "12"}, {Type: token.Dot, Lexeme: "."}}},
		{"1.2.3", []token.Token{{Type: token.Number, Lexeme: "1.2"}, {Type: token.Dot, Lexeme: "."}, {Type: token.Number, Lexeme: "3"}}},
		{"0.001 x2", []token.Token{{Type: token.Number, Lexeme: "0.001"}, {Type: token.Identifier, Lexeme: "x2"}}},
	} {
		t.Run(tt.source, func(t *testing.T) {
			l := New(tt.source)
			for _, want := range tt.want {
				got, err := l.Next()
				if err != nil {
					t.Fatal(err)
				}
				if got.Type != want.Type || got.Lexeme != want.Lexeme {
					t.Fatalf("got %#v, want %#v", got, want)
				}
			}
			got, err := l.Next()
			if err != nil || got.Type != token.EOF {
				t.Fatalf("trailing token: %#v, %v", got, err)
			}
		})
	}
}

func TestRejectNonASCIIIdentifiers(t *testing.T) {
	for _, source := range []string{"é", "ÿ", "²", "\xff", "😀"} {
		if _, err := New(source).Next(); err == nil {
			t.Errorf("accepted %q", source)
		}
	}
}

func TestAllTokens(t *testing.T) {
	for _, kind := range []token.TokenType{
		token.Def, token.Let, token.In, token.If, token.Then, token.Else, token.Fn, token.True, token.False, token.Null, token.And, token.Or, token.Not,
		token.Plus, token.Minus, token.Star, token.Slash, token.Percent, token.Equal, token.NotEqual, token.Less, token.LessEqual, token.Greater, token.GreaterEq, token.Pipe, token.Assign, token.Arrow,
		token.LeftParen, token.RightParen, token.LeftBrace, token.RightBrace, token.LeftBracket, token.RightBracket, token.Comma, token.Dot, token.Colon, token.Semicolon,
	} {
		t.Run(string(kind), func(t *testing.T) {
			l := New(string(kind))
			got, err := l.Next()
			if err != nil || got.Type != kind || got.Lexeme != string(kind) {
				t.Fatalf("got %#v, %v", got, err)
			}
			end, err := l.Next()
			if err != nil || end.Type != token.EOF {
				t.Fatalf("trailing token: %#v, %v", end, err)
			}
		})
	}
	for _, name := range []string{"a", "A0_b", "letx", "True", "source", "map", "filter", "sink"} {
		got, err := New(name).Next()
		if err != nil || got.Type != token.Identifier || got.Lexeme != name {
			t.Errorf("identifier %q: %#v, %v", name, got, err)
		}
	}
}

func TestPositionsAndStableEOF(t *testing.T) {
	l := New(" \tlet\r\n x + 2")
	want := []token.Token{
		{Type: token.Let, Lexeme: "let", Position: token.Position{Offset: 2, Line: 1, Column: 3}},
		{Type: token.Identifier, Lexeme: "x", Position: token.Position{Offset: 8, Line: 2, Column: 2}},
		{Type: token.Plus, Lexeme: "+", Position: token.Position{Offset: 10, Line: 2, Column: 4}},
		{Type: token.Number, Lexeme: "2", Position: token.Position{Offset: 12, Line: 2, Column: 6}},
		{Type: token.EOF, Position: token.Position{Offset: 13, Line: 2, Column: 7}},
	}
	for _, expected := range append(want, want[len(want)-1]) {
		got, err := l.Next()
		if err != nil || !reflect.DeepEqual(got, expected) {
			t.Fatalf("got %#v, %v; want %#v", got, err, expected)
		}
	}
}

func TestLexicalErrorDetails(t *testing.T) {
	for _, tt := range []struct {
		source, message string
		pos             token.Position
	}{
		{" \n@", "unexpected character '@'", token.Position{Offset: 2, Line: 2, Column: 1}},
		{`"abc`, "unterminated string", token.Position{Line: 1, Column: 1}},
		{"\"abc\n", "unterminated string", token.Position{Line: 1, Column: 1}},
		{"\"abc\\", "unterminated string", token.Position{Line: 1, Column: 1}},
		{`"\q"`, `invalid escape \q`, token.Position{Offset: 2, Line: 1, Column: 3}},
	} {
		t.Run(tt.source, func(t *testing.T) {
			_, err := New(tt.source).Next()
			var lexErr *Error
			if !errors.As(err, &lexErr) {
				t.Fatalf("expected *Error, got %v", err)
			}
			if lexErr.Position != tt.pos || lexErr.Message != tt.message {
				t.Fatalf("got %#v, want %+v / %s", lexErr, tt.pos, tt.message)
			}
			if !strings.Contains(err.Error(), tt.message) {
				t.Fatal(err)
			}
		})
	}
}

func FuzzLexerProgress(f *testing.F) {
	for _, s := range []string{"", "let x = 1 in x", `"\n"`, "\xff", "12.3.4", "\"\\", "\x00"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		l := New(s)
		for i := 0; i <= len(s); i++ {
			got, err := l.Next()
			if err != nil {
				return
			}
			if got.Position.Offset < 0 || got.Position.Offset > len(s) || got.Position.Line < 1 || got.Position.Column < 1 {
				t.Fatalf("invalid position: %#v", got)
			}
			if got.Type == token.EOF {
				again, err := l.Next()
				if err != nil || again != got {
					t.Fatalf("unstable EOF: %#v %v", again, err)
				}
				return
			}
		}
		t.Fatal("lexer failed to terminate")
	})
}

func TestCompoundOperatorBoundaries(t *testing.T) {
	l := New("=== =>= !== <== >== |>=")
	for _, want := range []token.TokenType{token.Equal, token.Assign, token.Arrow, token.Assign, token.NotEqual, token.Assign, token.LessEqual, token.Assign, token.GreaterEq, token.Assign, token.Pipe, token.Assign, token.EOF} {
		got, err := l.Next()
		if err != nil || got.Type != want {
			t.Fatalf("got %#v, %v; want %s", got, err, want)
		}
	}
	for _, source := range []string{"!", "|", "!x", "|x"} {
		if _, err := New(source).Next(); err == nil {
			t.Errorf("accepted incomplete operator %q", source)
		}
	}
}

func TestIdentifiersStartWithASCIILetter(t *testing.T) {
	for _, source := range []string{"_", "_x"} {
		_, err := New(source).Next()
		var lexicalError *Error
		if !errors.As(err, &lexicalError) || lexicalError.Position.Column != 1 {
			t.Errorf("Next(%q) = %v, want lexical error at column 1", source, err)
		}
	}
}

func TestLineEndingPositions(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		offset       int
		line         int
	}{
		{"initial LF", "\nx", 1, 2},
		{"CR only", "a\rb", 2, 2},
		{"LF only", "a\nb", 2, 2},
		{"CRLF", "a\r\nb", 3, 2},
		{"mixed", "a\r\r\n\nb", 5, 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			lexer := New(tt.source)
			if tt.source[0] == 'a' {
				if _, err := lexer.Next(); err != nil {
					t.Fatal(err)
				}
			}
			got, err := lexer.Next()
			want := token.Position{Offset: tt.offset, Line: tt.line, Column: 1}
			if err != nil || got.Position != want || got.Type != token.Identifier {
				t.Fatalf("got %v, %v, want identifier at %+v", got, err, want)
			}
		})
	}
}

func TestStringsRejectLiteralLineEndings(t *testing.T) {
	for _, lineEnding := range []string{"\r", "\n", "\r\n"} {
		_, err := New("\"first" + lineEnding + "second\"").Next()
		var lexicalError *Error
		if !errors.As(err, &lexicalError) || lexicalError.Message != "unterminated string" ||
			lexicalError.Position != (token.Position{Line: 1, Column: 1}) {
			t.Errorf("literal line ending %q returned %v", lineEnding, err)
		}
	}
	got, err := New(`"first\r\nsecond"`).Next()
	if err != nil || got.Lexeme != "first\r\nsecond" {
		t.Fatalf("escaped line ending: got %v, %v", got, err)
	}
}

func TestUTF8PositionsAndUnexpectedCharacters(t *testing.T) {
	lexer := New("\"é😀e\u0301\" + 1")
	first, err := lexer.Next()
	if err != nil || first.Lexeme != "é😀e\u0301" {
		t.Fatalf("got %v, %v", first, err)
	}
	operator, err := lexer.Next()
	want := token.Position{Offset: 12, Line: 1, Column: 8}
	if err != nil || operator.Type != token.Plus || operator.Position != want {
		t.Fatalf("operator = %v, %v, want position %+v", operator, err, want)
	}

	for _, tt := range []struct {
		source, message string
		position        token.Position
	}{
		{"é", "unexpected character 'é'", token.Position{Line: 1, Column: 1}},
		{"😀", "unexpected character '😀'", token.Position{Line: 1, Column: 1}},
		{"�", "unexpected character '�'", token.Position{Line: 1, Column: 1}},
		{"\xff", "invalid UTF-8 byte 0xff", token.Position{Line: 1, Column: 1}},
		{`"é\😀"`, `invalid escape \😀`, token.Position{Offset: 4, Line: 1, Column: 4}},
	} {
		l := New(tt.source)
		_, err := l.Next()
		var lexicalError *Error
		if !errors.As(err, &lexicalError) || lexicalError.Message != tt.message || lexicalError.Position != tt.position {
			t.Errorf("Next(%q) = %v, want %q at %+v", tt.source, err, tt.message, tt.position)
		}
	}

	lexer = New("é+")
	_, _ = lexer.Next()
	got, err := lexer.Next()
	if err != nil || got.Position != (token.Position{Offset: 2, Line: 1, Column: 2}) || got.Type != token.Plus {
		t.Fatalf("unexpected rune was not consumed exactly once: %v, %v", got, err)
	}
}
