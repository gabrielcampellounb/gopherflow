package evaluator

import (
	"strings"
	"testing"

	"github.com/gabrielcampellounb/gopherflow/internal/lexer"
	"github.com/gabrielcampellounb/gopherflow/internal/parser"
)

func TestDataflowOperations(t *testing.T) {
	for _, tt := range []struct{ source, want string }{
		{"let factor = 2 in [1, 2, 3] |> map(fn(x) => x * factor) |> reduce(0, fn(total, x) => total + x)", "12"},
		{"[1, 2, 3] |> filter(fn(x) => x > 1) |> flatMap(fn(x) => [x, -x]) |> take(3) |> reduce(0, fn(total, x) => total * 10 + x)", "183"},
		{`let groups = [{kind: "a", n: 2}, {kind: "b", n: 9}, {kind: "a", n: 5}] |> keyBy(fn(x) => x.kind) in groups.a |> reduce(0, fn(sum, x) => sum * 10 + x.n)`, "25"},
		{"[] |> reduce(42, fn(a, x) => a + x)", "42"},
		{"[1] |> take(100000000000000000000) |> reduce(0, fn(a, x) => a + x)", "1"},
		{"[1, 2] |> forEach(fn(x) => x * 2)", "null"},
	} {
		program, err := parser.Parse(lexer.New(tt.source))
		if err != nil {
			t.Fatal(err)
		}
		for _, tree := range []bool{false, true} {
			input := program
			if tree {
				input = Optimize(program)
			}
			got, err := Evaluate(input)
			if err != nil || got.Inspect() != tt.want {
				t.Errorf("%s: got %v, %v, want %s", tt.source, got, err, tt.want)
			}
		}
	}
}

func TestDataflowErrors(t *testing.T) {
	for _, tt := range []struct{ source, want string }{
		{"map(1, fn(x) => x)", "map expects a List or Stream"},
		{"[] |> map(1)", "expected a callable callback"},
		{"[] |> map(fn() => 1)", "callback must accept 1 arguments, got 0"},
		{"[1] |> filter(fn(x) => x)", "filter item 1: callback must return Bool"},
		{"[1] |> flatMap(fn(x) => x)", "flatMap item 1: callback must return a List or Stream"},
		{"[1] |> keyBy(fn(x) => x)", "keyBy item 1: callback must return String"},
		{"[1] |> take(-1)", "nonnegative integer"},
		{"[1] |> take(0.5)", "nonnegative integer"},
		{"[1, 0] |> forEach(fn(x) => 1 / x)", "forEach item 2: runtime error"},
		{"[1] |> reduce(0, fn(a, x) => missing)", "reduce item 1: name error"},
		{"source(1)", "source expects a String path"},
		{"sink(1, null)", "sink expects a List or Stream"},
		{"sink([], null)", "sink expects a String path"},
	} {
		program, err := parser.Parse(lexer.New(tt.source))
		if err != nil {
			t.Fatal(err)
		}
		if got, err := Evaluate(Optimize(program)); got != nil || err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: got %v, %v, want %s", tt.source, got, err, tt.want)
		}
	}
}
