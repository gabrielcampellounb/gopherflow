package token

import "testing"

func TestEveryKeywordAndIdentifierBoundary(t *testing.T) {
	for _, kind := range []TokenType{Def, Let, In, If, Then, Else, Fn, True, False, Null, And, Or, Not} {
		if got := LookupIdentifier(string(kind)); got != kind {
			t.Errorf("keyword %s: %s", kind, got)
		}
		for _, s := range []string{string(kind) + "x", "x" + string(kind), string(kind) + "_", string(kind) + "1"} {
			if got := LookupIdentifier(s); got != Identifier {
				t.Errorf("identifier %q: %s", s, got)
			}
		}
	}
	for _, s := range []string{"", "Let", "TRUE", "source", "map", "reduce", "sink"} {
		if got := LookupIdentifier(s); got != Identifier {
			t.Errorf("%q: %s", s, got)
		}
	}
}

func TestTokenString(t *testing.T) {
	tok := Token{Type: String, Lexeme: "a\n\"", Position: Position{Offset: 20, Line: 3, Column: 4}}
	if got := tok.String(); got != `STRING("a\n\"") at 3:4` {
		t.Fatal(got)
	}
}
