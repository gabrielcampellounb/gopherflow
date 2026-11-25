package lexer

import (
	"github.com/gabrielcampellounb/gopherflow/internal/token"
	"testing"
)

func BenchmarkTokenize(b *testing.B) {
	const source = `let answer = (12.5 + 2) * 3 in answer == 43.5;`
	b.ReportAllocs()
	b.SetBytes(int64(len(source)))
	for b.Loop() {
		l := New(source)
		for {
			tok, err := l.Next()
			if err != nil {
				b.Fatal(err)
			}
			if tok.Type == token.EOF {
				break
			}
		}
	}
}

var stringToken token.Token

func BenchmarkStringToken(b *testing.B) {
	for _, tc := range []struct{ name, source string }{
		{"plain", `"a plain string with enough text to grow a buffer"`},
		{"escaped", `"a string with\nnewlines and \"quotes\""`},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var err error
				stringToken, err = New(tc.source).Next()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
