package evaluator

import (
	"github.com/gabrielcampellounb/gopherflow/internal/lexer"
	"github.com/gabrielcampellounb/gopherflow/internal/parser"
	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
	"testing"
)

func BenchmarkEvaluate(b *testing.B) {
	p, err := parser.Parse(lexer.New("let x = 6 in let y = x * 7 in y == 42"))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		value, err := Evaluate(p)
		if err != nil || value != runtime.Bool(true) {
			b.Fatalf("got %v, %v", value, err)
		}
	}
}
