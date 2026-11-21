package evaluator

import (
	"testing"

	"github.com/gabrielcampellounb/gopherflow/internal/lexer"
	"github.com/gabrielcampellounb/gopherflow/internal/parser"
	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
)

func evaluate(t *testing.T, input string) runtime.Value {
	t.Helper()
	program, err := parser.Parse(lexer.New(input))
	if err != nil {
		t.Fatal(err)
	}
	value, err := Evaluate(program)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func evaluateFunction(t *testing.T, source string) runtime.Function {
	t.Helper()
	value := evaluate(t, source)
	function, ok := value.(runtime.Function)
	if !ok {
		t.Fatalf("Evaluate(%q) = %T, want runtime.Function", source, value)
	}
	return function
}

func TestEvaluateExpressions(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1 + 2 * 3", "7"},
		{"let answer = 41 in answer + 1", "42"},
		{`"hello"`, `"hello"`},
		{"true", "true"},
		{"null", "null"},
	}
	for _, tt := range tests {
		if got := evaluate(t, tt.input).Inspect(); got != tt.want {
			t.Errorf("Evaluate(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEvaluateErrors(t *testing.T) {
	for _, input := range []string{"missing", "1 + true", "1 / 0"} {
		program, err := parser.Parse(lexer.New(input))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Evaluate(program); err == nil {
			t.Errorf("Evaluate(%q) returned nil error", input)
		}
	}
}

func TestEvaluateFunctions(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"fn(x) => x + 1", "fn(x) => (x + 1)"},
		{"fn() => 1", "fn() => 1"},
		{"fn(a, b) => a", "fn(a, b) => a"},
		{"fn(x) => y", "fn(x) => y"},
	}
	for _, tt := range tests {
		if got := evaluate(t, tt.input).Inspect(); got != tt.want {
			t.Errorf("Evaluate(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEvaluateCall(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"(fn(x) => x + 1)(41)", "42"},
		{"(fn() => 7)()", "7"},
		{`let add = fn(a, b) => a + b in add(1, 2)`, `3`},
		{`let id = fn(x) => x in id("hi")`, `"hi"`},
		{"let apply = fn(f) => f(10) in apply(fn(x) => x - 1)", "9"},
		{"let apply = fn(f) => f(10) in apply(fn(x) => x * 5)", "50"},
	}
	for _, tt := range tests {
		if got := evaluate(t, tt.input).Inspect(); got != tt.want {
			t.Errorf("Evaluate(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEvaluateDefinition(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"def square(x) = x * x; square(7)", "49"},
		{"def five() = 5; five()", "5"},
		{"def twice(f, x) = f(f(x)); twice(fn(n) => n + 3, 1)", "7"},
		{"def inc(x) = x + 1; def twice(f, x) = f(f(x)); twice(inc, 1)", "3"},
		{"def f(x) = g(x); def g(x) = x; f(5)", "5"},
		{"def f(x) = g(x) + 1; def g(x) = x * 2; f(5)", "11"},
		{"def f(x) = x; f", "fn(x) => x"},
		{"def f() = 1; f() + f()", "2"},
		{"def f() = 1; def g(f) = f; g(5)", "5"},
		{"def x() = 1; let x = 2 in x", "2"},
		{"def make(n) = fn(x) => x + n; make(1)(2)	", "3"},
		{"def f() = 1; let f = 2 in f", "2"},
	}

	for _, tt := range tests {
		if got := evaluate(t, tt.input).Inspect(); got != tt.want {
			t.Errorf("Evaluate(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEvaluateList(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"[]", "[]"},
		{"[1, 2, 3]", "[1, 2, 3]"},
		{`["a", true, null]`, `["a", true, null]`},
		{"[[1], [], [2, 3]]", "[[1], [], [2, 3]]"},
		{"let x = 5 in [x, x + 1]", "[5, 6]"},
		{"[let a = 1 in a, 2]", "[1, 2]"},
		{"(fn(x) => [x, x])(7)", "[7, 7]"},
		{"def f(x) = [x]; [f(1), f(2)]", "[[1], [2]]"},
	}
	for _, tt := range tests {
		if got := evaluate(t, tt.input).Inspect(); got != tt.want {
			t.Errorf("Evaluate(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEvaluateIndex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"[10, 20, 30][0]", "10"},
		{"[10, 20, 30][2]", "30"},
		{"[10, 20, 30][1 + 1]", "30"},
		{"[10, 20][1.0]", "20"},
		{"let xs = [1, 2] in xs[1]", "2"},
		{"[[1, 2], [3]][1]", "[3]"},
		{"[[1, 2], [3]][0][1]", "2"},
		{"[0, 1][[1][0]]", "1"},
		{`["a", true, null][2]`, "null"},
		{"def first(xs) = xs[0]; first([7, 8])", "7"},
		{"(fn(xs) => xs[1])([5, 6])", "6"},
		{"[fn(x) => x + 1, fn(x) => x * 2][1](10)", "20"},
		{"[1, 2][0] + [3, 4][1]", "5"},
		{"1 + [2, 3][0] * 4", "9"},
		{"[1, 2][0] == 1", "true"},
	}
	for _, tt := range tests {
		if got := evaluate(t, tt.input).Inspect(); got != tt.want {
			t.Errorf("Evaluate(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFunctionCapturesEnvironment(t *testing.T) {
	t.Run("captures n's let", func(t *testing.T) {
		function := evaluateFunction(t, "let n = 1 in fn(x) => x + n")
		n, found := function.Env.Get("n")
		if !found || n != runtime.Number(1) {
			t.Fatalf("n = %v, %v; want 1, true", n, found)
		}
	})

	t.Run("keep the same environment", func(t *testing.T) {
		program, err := parser.Parse(lexer.New("fn(x) => x"))
		if err != nil {
			t.Fatal(err)
		}
		env := runtime.NewEnvironment(nil)
		value, err := EvaluateIn(program, env)
		if err != nil {
			t.Fatal(err)
		}
		function, ok := value.(runtime.Function)
		if !ok {
			t.Fatalf("got %T, want runtime.Function", value)
		}
		if function.Env != env {
			t.Fatal("Env is not the environment in which the fn was evaluated")
		}
	})
}
