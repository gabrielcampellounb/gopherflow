package evaluator

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/gabrielcampellounb/gopherflow/internal/ast"
	"github.com/gabrielcampellounb/gopherflow/internal/lexer"
	"github.com/gabrielcampellounb/gopherflow/internal/parser"
	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
	"github.com/gabrielcampellounb/gopherflow/internal/token"
)

func TestOptimizeFixedPoint(t *testing.T) {
	program, err := parser.Parse(lexer.New("def f() = if true then 4 + 1 else missing; let x = if not false then 2 * 3 else missing in (fn(y) => y + (1 + 1))(x)"))
	if err != nil {
		t.Fatal(err)
	}
	original := program.String()
	optimized := Optimize(program)
	if optimized == program || Optimize(optimized) != optimized {
		t.Fatal("expected a new AST at a fixed point")
	}
	if got, want := optimized.String(), "def f() = if true then (4 + 1) else missing; let x = 6 in (fn(y) => (y + (1 + 1)))(x)"; got != want {
		t.Fatalf("optimized source = %q, want %q", got, want)
	}
	if got := optimized.Definitions[0].OptimizedBody.String(); got != "5" {
		t.Fatalf("optimized definition body = %q", got)
	}
	call := optimized.Expressions[0].(*ast.LetExpression).Body.(*ast.CallExpression)
	function := call.Function.(*ast.FunctionLiteral)
	if got := function.OptimizedBody.String(); got != "(y + 2)" {
		t.Fatalf("optimized function body = %q", got)
	}
	if program.String() != original || program.Definitions[0].OptimizedBody != nil {
		t.Fatal("optimization mutated the input tree")
	}
	empty := &ast.Program{}
	if Optimize(empty) != empty {
		t.Fatal("empty program changed")
	}
}

func TestOptimizeNestedSelectedBranchesInOnePass(t *testing.T) {
	const depth = 128
	source := strings.Repeat("if true then ", depth) + "1 + 2" + strings.Repeat(" else missing()", depth)
	program, err := parser.Parse(lexer.New("[" + source + ", 4 * 5]"))
	if err != nil {
		t.Fatal(err)
	}
	expression := optimizeExpression(program.Expressions[0])
	if got, want := expression.String(), "[3, 20]"; got != want {
		t.Fatalf("one pass produced %q, want %q", got, want)
	}
}

func TestOptimizePreservesEvaluation(t *testing.T) {
	huge := "1" + strings.Repeat("0", 308)
	for _, tt := range []struct{ source, want string }{
		{"-(2 - 2)", "-0"},
		{"1 / (2 - 2)", "runtime error at 1:3: division by zero"},
		{"1 % (2 - 2)", "runtime error at 1:3: division by zero"},
		{"true and (1 + 2)", "type error at 1:6: operator and is not defined for Bool and Number"},
		{"false or (1 / 0)", "runtime error at 1:13: division by zero"},
		{"true or missing()", "true"},
		{"false and missing()", "false"},
		{"if (1 + 2) then missing else missing", "type error at 1:1: if condition must be Bool, got Number"},
		{"if not false then\n missing else 0", "name error at 2:2: undefined identifier \"missing\""},
		{"let x = missing in 1 + 2", "name error at 1:9: undefined identifier \"missing\""},
		{"let x = 1 in let x = x + 2 in x", "3"},
		{"(if true then fn(x) => x + (1 + 2) else missing)(4)", "7"},
		{"fn() => 1 + 2", "fn() => (1 + 2)"},
		{"(fn() => 1 + 2) == (fn() => 3)", "false"},
		{"def f(n) = if n <= 0 then 1 + 1 else f(n - 1); f(3)", "2"},
		{"def f() = 1 + 2; f", "fn() => (1 + 2)"},
		{"[1 + 2, (1 / 0)][0]", "runtime error at 1:12: division by zero"},
		{"[[1 + 2]][1 - 1][0]", "3"},
		{"(-2)[0]", "type error at 1:5: Number is not indexable"},
		{huge + " * " + huge, "+Inf"},
		{"let x = " + huge + " * " + huge + " in x / x", "NaN"},
	} {
		t.Run(tt.source, func(t *testing.T) {
			program, err := parser.Parse(lexer.New(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			optimized := Optimize(program)
			for _, tree := range []*ast.Program{program, optimized} {
				value, err := Evaluate(tree)
				got := fmt.Sprint(err)
				if err == nil {
					got = value.Inspect()
				}
				if got != tt.want {
					t.Fatalf("evaluating %q got %q, want %q", tree, got, tt.want)
				}
			}
		})
	}
}

func TestOptimizedNegativePostfixTargetRoundTrip(t *testing.T) {
	for _, source := range []string{"(-2)[0]", "(-2)()"} {
		program, err := parser.Parse(lexer.New(source))
		if err != nil {
			t.Fatal(err)
		}
		rendered := Optimize(program).String()
		reparsed, err := parser.Parse(lexer.New(rendered))
		if err != nil {
			t.Fatal(err)
		}
		if rendered != source || reparsed.String() != rendered {
			t.Fatalf("rendering %q produced %q and reparsed as %q", source, rendered, reparsed)
		}
	}
}

func TestOptimizeMalformedLiteral(t *testing.T) {
	eager := &ast.LetExpression{
		Name:  &ast.Identifier{Name: "x"},
		Value: &ast.Literal{Kind: token.Number, Value: "bad"},
		Body:  &ast.Literal{Kind: token.True, Value: "true"},
	}
	program := &ast.Program{Expressions: []ast.Expression{eager}}
	if Optimize(program) != program {
		t.Fatal("optimization removed a failing binding")
	}
}

type observedValue struct {
	name string
	log  *[]string
}

func (v observedValue) Type() runtime.Type { return runtime.StringType }
func (v observedValue) Inspect() string {
	*v.log = append(*v.log, v.name)
	return v.name
}

func TestOptimizePreservesEvaluationOrderAndEffects(t *testing.T) {
	source := "def f(x, y) = (1 == 1) and x; f((if true then a == b else c == d), [c == d, e == fValue][0])"
	program, err := parser.Parse(lexer.New(source))
	if err != nil {
		t.Fatal(err)
	}
	var log []string
	env := runtime.NewEnvironment(nil)
	for _, name := range []string{"a", "b", "c", "d", "e", "fValue"} {
		env.Set(name, observedValue{name: name, log: &log})
	}
	optimized := Optimize(program)
	if len(log) != 0 {
		t.Fatalf("optimization triggered effects: %v", log)
	}
	for _, tree := range []*ast.Program{program, optimized} {
		log = nil
		value, err := EvaluateIn(tree, env)
		if err != nil || value != runtime.Bool(false) || !slices.Equal(log, []string{"a", "b", "c", "d", "e", "fValue"}) {
			t.Fatalf("evaluating %q got %v, %v and effects %v", tree, value, err, log)
		}
	}
}

func TestOptimizeLiteralListIndex(t *testing.T) {
	for _, tt := range []struct{ source, want string }{
		{`[1 + 2, "a\n\"b", null][1]`, `"a\n\"b"`},
		{"[false, null][1]", "null"},
		{"[4, 8][2 - 1] * 2", "16"},
		{"[false, true][1]", "true"},
		{"[1, missing()][0]", "[1, missing()][0]"},
		{"[1, (2 / 0)][0]", "[1, (2 / 0)][0]"},
		{"[1][1]", "[1][1]"},
		{"[1][0.5]", "[1][0.5]"},
	} {
		program, err := parser.Parse(lexer.New(tt.source))
		if err != nil {
			t.Fatal(err)
		}
		optimized := Optimize(program)
		if got := optimized.String(); got != tt.want {
			t.Errorf("Optimize(%q) = %q, want %q", tt.source, got, tt.want)
		}
	}
}
