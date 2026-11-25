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
		// Encode signed numbers using implemented subtraction rather than unary minus.
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
	// Overflow is possible using existing multiplication; IEEE-754 NaN must not
	// become equal to itself merely because both values render as "NaN".
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
