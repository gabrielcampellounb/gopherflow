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
