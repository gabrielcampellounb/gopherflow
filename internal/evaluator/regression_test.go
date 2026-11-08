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
		{`1 == 1`, runtime.Bool(true)}, {`1 == 2`, runtime.Bool(false)},
		{`1 < 2`, runtime.Bool(true)},
		{`1 <= 2`, runtime.Bool(true)},
		{`2 > 1`, runtime.Bool(true)},
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
		{"1 % 2", "operator % is not defined"}, {"1 != 2", "operator != is not defined"},
		{"1 >= 2", "operator >= is not defined"},
		{"true and false", "operator and is not defined"}, {"true or false", "operator or is not defined"},
		{"1 |> 2", "operator |> is not defined"},
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
	parent := NewEnvironment(nil)
	parent.Set("x", runtime.Number(10))
	child := NewEnvironment(parent)
	child.Set("x", runtime.Number(20))
	child.Set("y", runtime.Bool(true))
	grandchild := NewEnvironment(child)
	for _, tt := range []struct {
		env   *Environment
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
	p, err := parser.Parse(lexer.New("let x = 6 in let y = x * 7 in y"))
	if err != nil {
		t.Fatal(err)
	}
	for range 16 {
		t.Run("isolated", func(t *testing.T) {
			t.Parallel()
			for range 50 {
				got, err := Evaluate(p)
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
	for _, s := range []string{"1 + 2", "missing", "let x = 1 in x", `"\n"`, "1 / 0", "true and false"} {
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
		a, firstErr := Evaluate(p)
		b, secondErr := Evaluate(p)
		if (firstErr == nil) != (secondErr == nil) {
			t.Fatal("nondeterministic error")
		}
		if firstErr != nil {
			if a != nil || b != nil || firstErr.Error() != secondErr.Error() {
				t.Fatal("inconsistent failure")
			}
			return
		}
		if a == nil || b == nil || a.Type() != b.Type() || a.Inspect() != b.Inspect() {
			t.Fatal("inconsistent result")
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
}
