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

func TestEvaluateLessThan(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  runtime.Bool
	}{
		{"1 < 2", runtime.Bool(true)},
		{"2 < 1", runtime.Bool(false)},
		{"2 < 2", runtime.Bool(false)},
		{"1.5 < 2", runtime.Bool(true)},
		{"1 + 2 < 2 * 2", runtime.Bool(true)},
	} {
		t.Run(tt.input, func(t *testing.T) {
			if got := evaluate(t, tt.input); got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvaluateLessThanTypeErrors(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"1 < true", "type error at 1:3: operator < is not defined for Number and Bool"},
		{"\n\"a\" < 2", "type error at 2:5: operator < is not defined for String and Number"},
		{"null < null", "type error at 1:6: operator < is not defined for Null and Null"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			program, err := parser.Parse(lexer.New(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			value, err := Evaluate(program)
			if value != nil || err == nil || err.Error() != tt.want {
				t.Fatalf("Evaluate(%q) = %v, %v; want %q", tt.input, value, err, tt.want)
			}
		})
	}
}

func TestEvaluateLessThanOrEqual(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  runtime.Bool
	}{
		{"1 <= 2", runtime.Bool(true)},
		{"2 <= 2", runtime.Bool(true)},
		{"2 <= 1", runtime.Bool(false)},
		{"1.5 <= 2", runtime.Bool(true)},
		{"1 + 2 <= 2 * 2", runtime.Bool(true)},
	} {
		t.Run(tt.input, func(t *testing.T) {
			if got := evaluate(t, tt.input); got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvaluateLessThanOrEqualTypeErrors(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"1 <= true", "type error at 1:3: operator <= is not defined for Number and Bool"},
		{"\n\"a\" <= 2", "type error at 2:5: operator <= is not defined for String and Number"},
		{"null <= null", "type error at 1:6: operator <= is not defined for Null and Null"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			program, err := parser.Parse(lexer.New(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			value, err := Evaluate(program)
			if value != nil || err == nil || err.Error() != tt.want {
				t.Fatalf("Evaluate(%q) = %v, %v; want %q", tt.input, value, err, tt.want)
			}
		})
	}
}

func TestEvaluateGreaterThan(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  runtime.Bool
	}{
		{"2 > 1", runtime.Bool(true)},
		{"1 > 2", runtime.Bool(false)},
		{"2 > 2", runtime.Bool(false)},
		{"2 > 1.5", runtime.Bool(true)},
		{"2 * 2 > 1 + 2", runtime.Bool(true)},
	} {
		t.Run(tt.input, func(t *testing.T) {
			if got := evaluate(t, tt.input); got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvaluateGreaterThanTypeErrors(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"1 > true", "type error at 1:3: operator > is not defined for Number and Bool"},
		{"\n\"a\" > 2", "type error at 2:5: operator > is not defined for String and Number"},
		{"null > null", "type error at 1:6: operator > is not defined for Null and Null"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			program, err := parser.Parse(lexer.New(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			value, err := Evaluate(program)
			if value != nil || err == nil || err.Error() != tt.want {
				t.Fatalf("Evaluate(%q) = %v, %v; want %q", tt.input, value, err, tt.want)
			}
		})
	}
}

func TestEvaluateGreaterThanOrEqual(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  runtime.Bool
	}{
		{"2 >= 1", runtime.Bool(true)},
		{"2 >= 2", runtime.Bool(true)},
		{"1 >= 2", runtime.Bool(false)},
		{"2 >= 1.5", runtime.Bool(true)},
		{"2 * 2 >= 1 + 2", runtime.Bool(true)},
	} {
		t.Run(tt.input, func(t *testing.T) {
			if got := evaluate(t, tt.input); got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvaluateGreaterThanOrEqualTypeErrors(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"1 >= true", "type error at 1:3: operator >= is not defined for Number and Bool"},
		{"\n\"a\" >= 2", "type error at 2:5: operator >= is not defined for String and Number"},
		{"null >= null", "type error at 1:6: operator >= is not defined for Null and Null"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			program, err := parser.Parse(lexer.New(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			value, err := Evaluate(program)
			if value != nil || err == nil || err.Error() != tt.want {
				t.Fatalf("Evaluate(%q) = %v, %v; want %q", tt.input, value, err, tt.want)
			}
		})
	}
}

func TestEvaluateRemainder(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  runtime.Number
	}{
		{"5 % 2", runtime.Number(1)},
		{"5.5 % 2", runtime.Number(1.5)},
		{"(0 - 5) % 2", runtime.Number(-1)},
		{"5 % (0 - 2)", runtime.Number(1)},
		{"(0 - 5) % (0 - 2)", runtime.Number(-1)},
		{"4 % 2", runtime.Number(0)},
		{"1 + 5 % 2", runtime.Number(2)},
	} {
		t.Run(tt.input, func(t *testing.T) {
			if got := evaluate(t, tt.input); got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvaluateRemainderErrors(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"5 % 0", "runtime error at 1:3: division by zero"},
		{"\n5 % (2 - 2)", "runtime error at 2:3: division by zero"},
		{"1 % true", "type error at 1:3: operator % is not defined for Number and Bool"},
		{"\n\"a\" % 2", "type error at 2:5: operator % is not defined for String and Number"},
		{"null % null", "type error at 1:6: operator % is not defined for Null and Null"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			program, err := parser.Parse(lexer.New(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			value, err := Evaluate(program)
			if value != nil || err == nil || err.Error() != tt.want {
				t.Fatalf("Evaluate(%q) = %v, %v; want %q", tt.input, value, err, tt.want)
			}
		})
	}
}

func TestEvaluateNotEqual(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  runtime.Bool
	}{
		{"1 != 1", runtime.Bool(false)},
		{"1 != 2", runtime.Bool(true)},
		{"1.5 != 1.5", runtime.Bool(false)},
		{"true != true", runtime.Bool(false)},
		{"true != false", runtime.Bool(true)},
		{`"a" != "a"`, runtime.Bool(false)},
		{`"a" != "b"`, runtime.Bool(true)},
		{"1 != true", runtime.Bool(true)},
		{`"1" != 1`, runtime.Bool(true)},
		{"null != null", runtime.Bool(false)},
		{"1 + 1 != 3", runtime.Bool(true)},
	} {
		t.Run(tt.input, func(t *testing.T) {
			if got := evaluate(t, tt.input); got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvaluateNotEqualOperandErrors(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"\nmissing != 1", "name error at 2:1: undefined identifier \"missing\""},
		{"1 != missing", "name error at 1:6: undefined identifier \"missing\""},
		{"1 != (1 / 0)", "runtime error at 1:9: division by zero"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			program, err := parser.Parse(lexer.New(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			value, err := Evaluate(program)
			if value != nil || err == nil || err.Error() != tt.want {
				t.Fatalf("Evaluate(%q) = %v, %v; want %q", tt.input, value, err, tt.want)
			}
		})
	}
}

func TestEvaluateUnaryMinus(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  runtime.Number
	}{
		{"-5", runtime.Number(-5)},
		{"--5", runtime.Number(5)},
		{"-(0 - 5)", runtime.Number(5)},
		{"-0", runtime.Number(0)},
		{"-1.5", runtime.Number(-1.5)},
		{"-(3 + 2)", runtime.Number(-5)},
		{"2 * -3", runtime.Number(-6)},
		{"-2 * 3", runtime.Number(-6)},
		{"5 - -2", runtime.Number(7)},
		{"-5 % 2", runtime.Number(-1)},
	} {
		t.Run(tt.input, func(t *testing.T) {
			if got := evaluate(t, tt.input); got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvaluateUnaryMinusErrors(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"-true", "type error at 1:1: operator - is not defined for Bool"},
		{"\n -\"a\"", "type error at 2:2: operator - is not defined for String"},
		{"-null", "type error at 1:1: operator - is not defined for Null"},
		{"--true", "type error at 1:2: operator - is not defined for Bool"},
		{"-missing", "name error at 1:2: undefined identifier \"missing\""},
		{"-(1 / 0)", "runtime error at 1:5: division by zero"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			program, err := parser.Parse(lexer.New(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			value, err := Evaluate(program)
			if value != nil || err == nil || err.Error() != tt.want {
				t.Fatalf("Evaluate(%q) = %v, %v; want %q", tt.input, value, err, tt.want)
			}
		})
	}
}

func TestEvaluateNot(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  runtime.Bool
	}{
		{"not true", runtime.Bool(false)},
		{"not false", runtime.Bool(true)},
		{"not not true", runtime.Bool(true)},
		{"not (1 < 2)", runtime.Bool(false)},
		{"not (2 < 1)", runtime.Bool(true)},
		{"not true == false", runtime.Bool(true)},
		{"not false == false", runtime.Bool(false)},
		{"not (2 * -3 < -5)", runtime.Bool(false)},
	} {
		t.Run(tt.input, func(t *testing.T) {
			if got := evaluate(t, tt.input); got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvaluateNotErrors(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"not 1", "type error at 1:1: operator not is not defined for Number"},
		{"\n not \"a\"", "type error at 2:2: operator not is not defined for String"},
		{"not null", "type error at 1:1: operator not is not defined for Null"},
		{"not not 1", "type error at 1:5: operator not is not defined for Number"},
		{"not missing", "name error at 1:5: undefined identifier \"missing\""},
		{"not (1 / 0)", "runtime error at 1:8: division by zero"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			program, err := parser.Parse(lexer.New(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			value, err := Evaluate(program)
			if value != nil || err == nil || err.Error() != tt.want {
				t.Fatalf("Evaluate(%q) = %v, %v; want %q", tt.input, value, err, tt.want)
			}
		})
	}
}
