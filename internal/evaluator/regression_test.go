package evaluator

import (
	"fmt"
	"github.com/gabrielcampellounb/gopherflow/internal/token"
	"strings"
	"testing"

	"github.com/gabrielcampellounb/gopherflow/internal/ast"
	"github.com/gabrielcampellounb/gopherflow/internal/lexer"
	"github.com/gabrielcampellounb/gopherflow/internal/parser"
	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
)

func TestEvaluateRegression(t *testing.T) {
	for _, tt := range []struct {
		source string
		want   runtime.Value
	}{
		{`"a\nb\tc\r\"\\"`, runtime.String("a\nb\tc\r\"\\")},
		{`(0 - 0) == (0 / (0 - 1))`, runtime.Bool(true)},
		{`(0 - 1) * 2 + 8 / 4`, runtime.Number(0)},
		{`20 - 5 - 3`, runtime.Number(12)}, {`24 / 4 / 2`, runtime.Number(3)},
		{`0.5 + 1.25`, runtime.Number(1.75)}, {`false`, runtime.Bool(false)},
		{`5 % 2`, runtime.Number(1)},
		{`1 == 1`, runtime.Bool(true)}, {`1 == 2`, runtime.Bool(false)},
		{`1 != 2`, runtime.Bool(true)},
		{`true and false`, runtime.Bool(false)}, {`false or true`, runtime.Bool(true)},
		{`if true then 1 else 2`, runtime.Number(1)},
		{`1 < 2`, runtime.Bool(true)},
		{`1 <= 2`, runtime.Bool(true)},
		{`2 > 1`, runtime.Bool(true)},
		{`2 >= 1`, runtime.Bool(true)},
		{`1 == "1"`, runtime.Bool(false)}, {`true == "true"`, runtime.Bool(false)},
		{`null == null`, runtime.Bool(true)}, {`false == null`, runtime.Bool(false)},
		{`"a\n" == "a\n"`, runtime.Bool(true)},
		{`let x = 2 in let x = x + 1 in x * x`, runtime.Number(9)},
		{`let x = 2 in (let x = 3 in x) + x`, runtime.Number(5)},
		{`let x = let y = 2 in y + 1 in x`, runtime.Number(3)},
	} {
		t.Run(tt.source, func(t *testing.T) {
			got := evaluate(t, tt.source)
			if got != tt.want {
				t.Fatalf("got %#v (%s), want %#v", got, got.Type(), tt.want)
			}
		})
	}
}

func TestEvaluationErrorsAndOrder(t *testing.T) {
	for _, tt := range []struct{ source, message string }{
		{"missing", `undefined identifier "missing"`},
		{"let x = missing in x", `undefined identifier "missing"`},
		{"let x = x in x", `undefined identifier "x"`},
		{"(let x = 1 in x) + x", `undefined identifier "x"`},
		{"left + right", `undefined identifier "left"`},
		{"1 + right", `undefined identifier "right"`},
		{"1 / 0", "division by zero"}, {"0 / (0 - 0)", "division by zero"},
		{"1 + true", "operator + is not defined for Number and Bool"},
		{"false - 1", "operator - is not defined for Bool and Number"},
		{`"a" * 2`, "operator * is not defined for String and Number"},
		{"null / 0", "operator / is not defined for Null and Number"},
		{"1 |> 2", "Number is not callable"},
	} {
		t.Run(tt.source, func(t *testing.T) {
			p, err := parser.Parse(lexer.New(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			value, err := Evaluate(p)
			if err == nil || value != nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("got %v, %v; want %q", value, err, tt.message)
			}
		})
	}
}

func TestEnvironmentScopes(t *testing.T) {
	parent := runtime.NewEnvironment(nil)
	parent.Set("x", runtime.Number(10))
	child := runtime.NewEnvironment(parent)
	child.Set("x", runtime.Number(20))
	child.Set("y", runtime.Bool(true))
	grandchild := runtime.NewEnvironment(child)
	for _, tt := range []struct {
		env   *runtime.Environment
		name  string
		want  runtime.Value
		found bool
	}{
		{parent, "x", runtime.Number(10), true}, {parent, "y", nil, false},
		{child, "x", runtime.Number(20), true}, {grandchild, "y", runtime.Bool(true), true}, {grandchild, "missing", nil, false},
	} {
		got, ok := tt.env.Get(tt.name)
		if got != tt.want || ok != tt.found {
			t.Fatalf("Get(%q) = %v, %v", tt.name, got, ok)
		}
	}
	p, err := parser.Parse(lexer.New("let x = x + 1 in x"))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		got, err := EvaluateIn(p, parent)
		if err != nil || got != runtime.Number(11) {
			t.Fatalf("EvaluateIn = %v, %v", got, err)
		}
	}
	if got, _ := parent.Get("x"); got != runtime.Number(10) {
		t.Fatalf("parent mutated: %v", got)
	}
	if got, err := Evaluate(&ast.Program{}); err != nil || got != (runtime.Null{}) {
		t.Fatalf("empty program = %v, %v", got, err)
	}
}

func TestRuntimeErrorPositions(t *testing.T) {
	for _, tt := range []struct{ source, position string }{
		{" \n missing", "2:2"}, {"1 +\n true", "1:3"}, {"1\n / 0", "2:2"},
	} {
		p, err := parser.Parse(lexer.New(tt.source))
		if err != nil {
			t.Fatal(err)
		}
		_, err = Evaluate(p)
		if err == nil || !strings.Contains(err.Error(), "at "+tt.position+":") {
			t.Errorf("%q: got %v, want position %s", tt.source, err, tt.position)
		}
	}
}

func TestEqualityMatrix(t *testing.T) {
	sources := []string{"0", "1", `"0"`, `""`, "true", "false", "null"}
	for i, left := range sources {
		for j, right := range sources {
			t.Run(left+"=="+right, func(t *testing.T) {
				if got := evaluate(t, left+" == "+right); got != runtime.Bool(i == j) {
					t.Fatalf("got %v", got)
				}
			})
		}
	}
}

func TestEvaluateInvalidAST(t *testing.T) {
	for _, tt := range []struct {
		expression ast.Expression
		message    string
	}{
		{nil, "unsupported expression"},
		{&ast.Literal{Kind: token.Number, Value: "invalid"}, "invalid number"},
		{&ast.Literal{Kind: token.Identifier, Value: "x"}, "unsupported literal"},
	} {
		value, err := Evaluate(&ast.Program{Expressions: []ast.Expression{tt.expression}})
		if value != nil || err == nil || !strings.Contains(err.Error(), tt.message) {
			t.Fatalf("got %v, %v", value, err)
		}
	}
}

func TestConcurrentIndependentEvaluation(t *testing.T) {
	p, err := parser.Parse(lexer.New("let x = 2 * 3 in let y = x * (8 - 1) in y"))
	if err != nil {
		t.Fatal(err)
	}
	for range 16 {
		t.Run("isolated", func(t *testing.T) {
			t.Parallel()
			for range 50 {
				got, err := Evaluate(Optimize(p))
				if err != nil || got != runtime.Number(42) {
					t.Fatalf("got %v, %v", got, err)
				}
			}
		})
	}
}

func FuzzArithmetic(f *testing.F) {
	f.Add(int32(0), int32(1))
	f.Add(int32(-7), int32(3))
	f.Add(int32(42), int32(0))
	f.Fuzz(func(t *testing.T, a, b int32) {
		// Representa números com sinal pela subtração já implementada, sem usar menos unário.
		left, right := int64(a), int64(b)
		source := fmt.Sprintf("(%d - %d)", max(left, 0), max(-left, 0))
		other := fmt.Sprintf("(%d - %d)", max(right, 0), max(-right, 0))
		for _, tt := range []struct {
			op   string
			want float64
		}{{"+", float64(a) + float64(b)}, {"-", float64(a) - float64(b)}, {"*", float64(a) * float64(b)}} {
			got := evaluate(t, source+tt.op+other)
			if got != runtime.Number(tt.want) {
				t.Fatalf("%s %s %s = %v, want %v", source, tt.op, other, got, tt.want)
			}
		}
		if b != 0 {
			if got := evaluate(t, source+"/"+other); got != runtime.Number(float64(a)/float64(b)) {
				t.Fatalf("division = %v", got)
			}
		}
	})
}

func FuzzEvaluation(f *testing.F) {
	for _, s := range []string{"1 + 2", "missing", "let x = 1 in x", `"\n"`, "1 / 0", "true and false", "if true then 3 * 2 else missing", "[1 + 2][0]", "(fn(x) => x + (3 * 2))(1)", "def f() = 1 + 2; f", "{a: [1 + 2]}.a[0]", "1 |> (fn(x) => x * 2)"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 2048 {
			t.Skip()
		}
		p, err := parser.Parse(lexer.New(s))
		if err != nil {
			return
		}
		// Keep fuzz inputs isolated from filesystem-backed standard bindings.
		a, firstErr := EvaluateIn(p, runtime.NewEnvironment(nil))
		optimized := Optimize(p)
		if Optimize(optimized) != optimized {
			t.Fatal("optimization did not reach a fixed point")
		}
		concurrent := &execution{workerLimit: 2}
		defer concurrent.close()
		b, secondErr := concurrent.evaluateProgram(optimized, runtime.NewEnvironment(nil))
		if (firstErr == nil) != (secondErr == nil) {
			t.Fatalf("optimization changed error from %v to %v", firstErr, secondErr)
		}
		if firstErr != nil {
			if a != nil || b != nil || firstErr.Error() != secondErr.Error() {
				t.Fatalf("optimization changed failure from %v to %v", firstErr, secondErr)
			}
			return
		}
		if a == nil || b == nil || a.Type() != b.Type() || a.Inspect() != b.Inspect() {
			t.Fatalf("optimization changed result from %v to %v", a, b)
		}
	})
}

func TestNaNIsNotEqualToItself(t *testing.T) {
	// A multiplicação pode gerar overflow; um NaN IEEE-754 não deve ser igual a si
	// mesmo apenas porque os dois valores são exibidos como "NaN".
	huge := "1" + strings.Repeat("0", 308)
	source := "let x = " + huge + " * " + huge + " in (x / x) == (x / x)"
	if got := evaluate(t, source); got != runtime.Bool(false) {
		t.Fatalf("NaN equality = %v", got)
	}
	source = "let x = " + huge + " * " + huge + " in (x / x) != (x / x)"
	if got := evaluate(t, source); got != runtime.Bool(true) {
		t.Fatalf("NaN inequality = %v", got)
	}
}

func TestCallEvaluationErrors(t *testing.T) {
	for _, tt := range []struct{ source, message string }{
		{"1(2)", "type error at 1:2: Number is not callable"},
		{"null()", "Null is not callable"},
		{"let f = fn(x) => x in f()", "arity error at 1:24: function expects 1 arguments, got 0"},
		{"(fn() => 1)(2)", "function expects 0 arguments, got 1"},
		{"missing(1)", `undefined identifier "missing"`},
		{"let f = fn(x) => y in f(1)", `undefined identifier "y"`},
		{"let f = fn(x) => x in f(missing)", `undefined identifier "missing"`},
		{"first(second)", `undefined identifier "first"`},
		{"let f = fn() => secret in let secret = 1 in f()", `undefined identifier "secret"`},
	} {
		t.Run(tt.source, func(t *testing.T) {
			p, err := parser.Parse(lexer.New(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			value, err := Evaluate(p)
			if err == nil || value != nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("got %v, %v; want %q", value, err, tt.message)
			}
		})
	}
}

func TestDefinitionErrors(t *testing.T) {
	for _, tt := range []struct{ source, message string }{
		{"def f() = missing(); f()", `undefined identifier "missing"`},
		{"def f(x) = x; f()", "function expects 1 arguments, got 0"},
		{"def f() = 1; f(1)", "function expects 0 arguments, got 1"},
		{"def f() = 1; f()()", "Number is not callable"},
	} {
		t.Run(tt.source, func(t *testing.T) {
			p, err := parser.Parse(lexer.New(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			value, err := Evaluate(p)
			if err == nil || value != nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("got %v, %v; want %q", value, err, tt.message)
			}
		})
	}
}

func TestIndexErrors(t *testing.T) {
	for _, tt := range []struct{ source, message string }{
		{"[1, 2][2]", "runtime error at 1:7: list index 2 out of range (length 2)"},
		{"[][0]", "list index 0 out of range (length 0)"},
		{"[1, 2][0 - 1]", "list index -1 out of range (length 2)"},
		{"[1, 2][1.5]", "list index must be an integer, got 1.5"},
		{`[1, 2]["a"]`, "type error at 1:7: list index must be a Number, got String"},
		{"[1, 2][null]", "got Null"},
		{"1[0]", "type error at 1:2: Number is not indexable"},
		{`"abc"[0]`, "String is not indexable"},
		{"null[0]", "Null is not indexable"},
		{"(fn(x) => x)[0]", "Function is not indexable"},
		{"missing[0]", `undefined identifier "missing"`},
	} {
		t.Run(tt.source, func(t *testing.T) {
			p, err := parser.Parse(lexer.New(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			value, err := Evaluate(p)
			if err == nil || value != nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("got %v, %v; want %q", value, err, tt.message)
			}
		})
	}
}

func TestFunctionsCompareByIdentity(t *testing.T) {
	for _, tt := range []struct {
		source string
		want   runtime.Bool
	}{
		{`def make(x) = fn() => x; make(1) == make(2)`, false},
		{`def make(x) = fn() => x; make(1) == make(1)`, false},
		{`let f = fn() => 1 + 2 in let alias = f in {a: [f]} == {a: [alias]}`, true},
		{`def f() = 1 + 2; f == f`, true},
		{`(fn() => 1) != (fn() => 1)`, true},
	} {
		program := parallelTestProgram(t, tt.source)
		for _, workers := range []int{1, 4} {
			actual, err := EvaluateConcurrent(program, workers)
			if err != nil || actual != tt.want {
				t.Errorf("%q: workers=%d got %v, %v, want %t", tt.source, workers, actual, err, tt.want)
			}
		}
	}
}

func TestFunctionCallDepthAcrossDataflowCallbacks(t *testing.T) {
	for _, source := range []string{
		"def loop(x) = loop(x); loop(0)",
		"def first(x) = second(x); def second(x) = first(x); first(0)",
		"def loop(x) = [x] |> forEach(loop); loop(0)",
	} {
		t.Run(source, func(t *testing.T) {
			program := parallelTestProgram(t, source)
			value, err := EvaluateConcurrent(program, 4)
			if value != nil || err == nil || !strings.Contains(err.Error(), "maximum function call depth (1024) exceeded") {
				t.Fatalf("got %v, %v", value, err)
			}
		})
	}
	program := parallelTestProgram(t, "def step(x) = if x <= 0 then 42 else [x - 1] |> reduce(0, fn(acc, n) => step(n)); step(50)")
	value, err := EvaluateConcurrent(program, 4)
	if err != nil || value != runtime.Number(42) {
		t.Fatalf("finite recursion: got %v, %v", value, err)
	}
}

func TestEscapedClosuresUseCurrentExecution(t *testing.T) {
	factory := parallelTestProgram(t, "fn(n) => [n] |> reduce(0, fn(sum, x) => sum + x)")
	closure, err := EvaluateConcurrent(factory, 4)
	if err != nil {
		t.Fatal(err)
	}
	environment := runtime.NewEnvironment(nil)
	environment.Set("f", closure)
	program := Optimize(parallelTestProgram(t, "f(42)"))
	for range 8 {
		t.Run("independent evaluation", func(t *testing.T) {
			t.Parallel()
			value, err := EvaluateIn(program, environment)
			if err != nil || value != runtime.Number(42) {
				t.Fatalf("got %v, %v", value, err)
			}
		})
	}
}

func TestEvaluationWithoutEnvironmentAndBuiltinWithoutValue(t *testing.T) {
	program := parallelTestProgram(t, "missing")
	if value, err := EvaluateIn(program, nil); value != nil || err == nil || !strings.Contains(err.Error(), "undefined identifier") {
		t.Fatalf("empty environment returned %v, %v", value, err)
	}
	environment := runtime.NewEnvironment(nil)
	environment.Set("bad", runtime.Builtin{
		Name: "bad",
		Call: func([]runtime.Value) (runtime.Value, error) { return nil, nil },
	})
	program = parallelTestProgram(t, "bad() + 1")
	if value, err := EvaluateIn(program, environment); value != nil || err == nil || err.Error() != `runtime error at 1:4: builtin "bad" returned no value` {
		t.Fatalf("invalid builtin returned %v, %v", value, err)
	}
}

func TestExactFunctionCallDepthBoundary(t *testing.T) {
	program := parallelTestProgram(t, "def count(n) = if n <= 1 then 1 else 1 + count(n - 1); count(limit)")
	for _, workers := range []int{1, 4} {
		run := &execution{workerLimit: workers}
		t.Cleanup(run.close)
		env := runtime.NewEnvironment(nil)
		for _, limit := range []int{maxFunctionCallDepth, maxFunctionCallDepth + 1, 1} {
			env.Set("limit", runtime.Number(limit))
			result, err := run.evaluateProgram(program, env)
			if limit > maxFunctionCallDepth {
				if result != nil || err == nil || !strings.Contains(err.Error(), "maximum function call depth") {
					t.Fatalf("workers=%d, limit=%d: expected depth error, got %v, %v", workers, limit, result, err)
				}
			} else if err != nil || result != runtime.Number(limit) {
				t.Fatalf("workers=%d, limit=%d: got %v, %v", workers, limit, result, err)
			}
		}
	}
}

func TestCallbackRecursionReportsOneSourcePosition(t *testing.T) {
	program := parallelTestProgram(t, "def loop(x) =\n  [x] |> forEach(loop);\nloop(0)")
	value, err := EvaluateConcurrent(program, 4)
	const want = "runtime error at 2:17: maximum function call depth (1024) exceeded"
	if value != nil || err == nil || err.Error() != want {
		t.Fatalf("got value=%v, matching diagnostic=%t", value, err != nil && err.Error() == want)
	}
}

func TestNonFiniteNumbersAtDataflowAndIndexBoundaries(t *testing.T) {
	large := "1" + strings.Repeat("0", 308)
	for _, tt := range []struct{ expression, diagnostic string }{
		{"[1][infinity]", "list index +Inf out of range"},
		{"[1][-infinity]", "list index -Inf out of range"},
		{"[1][nan]", "list index must be an integer, got NaN"},
		{"[1] |> take(infinity)", "nonnegative integer count"},
		{"[1] |> take(nan)", "nonnegative integer count"},
		{"[1][-0]", ""},
		{"[1] |> take(-0) |> reduce(1, fn(sum, x) => sum + x)", ""},
	} {
		program := parallelTestProgram(t, "let infinity = "+large+" * "+large+" in let nan = infinity / infinity in "+tt.expression)
		for _, workers := range []int{1, 4} {
			value, err := EvaluateConcurrent(program, workers)
			if tt.diagnostic == "" {
				if err != nil || value != runtime.Number(1) {
					t.Errorf("%s: got %v, %v, want 1", tt.expression, value, err)
				}
			} else if value != nil || err == nil || !strings.Contains(err.Error(), tt.diagnostic) {
				t.Errorf("%s: got %v, %v, want %q", tt.expression, value, err, tt.diagnostic)
			}
		}
	}
}

func TestNilHostBindingIsLanguageNull(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	env.Set("host", nil)
	value, err := EvaluateIn(parallelTestProgram(t, "host == null"), env)
	if err != nil || value != runtime.Bool(true) {
		t.Fatalf("nil host binding = %v, %v, want true", value, err)
	}
	value, err = EvaluateIn(parallelTestProgram(t, "host + 1"), env)
	if value != nil || err == nil || err.Error() != "type error at 1:6: operator + is not defined for Null and Number" {
		t.Fatalf("nil host binding in arithmetic = %v, %v", value, err)
	}
}
