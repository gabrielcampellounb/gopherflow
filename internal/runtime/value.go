package runtime

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gabrielcampellounb/gopherflow/internal/ast"
)

type Environment struct {
	values map[string]Value
	parent *Environment
}

func NewEnvironment(parent *Environment) *Environment {
	return &Environment{values: make(map[string]Value), parent: parent}
}

func (e *Environment) Get(name string) (Value, bool) {
	for scope := e; scope != nil; scope = scope.parent {
		if value, ok := scope.values[name]; ok {
			return value, true
		}
	}
	return nil, false
}

// Set treats a nil Go value as the language's null value.
func (e *Environment) Set(name string, value Value) {
	if value == nil {
		value = Null{}
	}
	if e.values == nil {
		e.values = make(map[string]Value)
	}
	e.values[name] = value
}

type Type string

const (
	NumberType   Type = "Number"
	StringType   Type = "String"
	BoolType     Type = "Bool"
	NullType     Type = "Null"
	ListType     Type = "List"
	RecordType   Type = "Record"
	FunctionType Type = "Function"
	BuiltinType  Type = "Builtin"
	StreamType   Type = "Stream"
)

type Value interface {
	Type() Type
	Inspect() string
}

type Number float64

func (Number) Type() Type        { return NumberType }
func (n Number) Inspect() string { return strconv.FormatFloat(float64(n), 'g', -1, 64) }

type String string

func (String) Type() Type        { return StringType }
func (s String) Inspect() string { return strconv.Quote(string(s)) }

type Bool bool

func (Bool) Type() Type        { return BoolType }
func (b Bool) Inspect() string { return strconv.FormatBool(bool(b)) }

type Null struct{}

func (Null) Type() Type      { return NullType }
func (Null) Inspect() string { return "null" }

type List []Value

func (List) Type() Type        { return ListType }
func (l List) Inspect() string { return inspectCollection(l) }

type Record map[string]Value

func (Record) Type() Type        { return RecordType }
func (r Record) Inspect() string { return inspectCollection(r) }

type Function struct {
	Parameters    []string
	Body          ast.Expression
	OptimizedBody ast.Expression // Optional execution body, invisible to Inspect.
	Env           *Environment
	identity      *byte
}

func NewFunction(parameters []string, body, optimizedBody ast.Expression, env *Environment) Function {
	return Function{
		Parameters: parameters, Body: body, OptimizedBody: optimizedBody, Env: env, identity: new(byte),
	}
}

func (Function) Type() Type { return FunctionType }
func (f Function) Inspect() string {
	body := "<nil>"
	if f.Body != nil {
		body = f.Body.String()
	}
	return escapeNonPrinting(fmt.Sprintf("fn(%s) => %s", strings.Join(f.Parameters, ", "), body))
}

type FunctionInvoker func(Function, []Value) (Value, error)

type Builtin struct {
	Name     string
	Arity    *int // Nil when an externally provided builtin has no fixed arity.
	Call     func([]Value) (Value, error)
	CallWith func([]Value, FunctionInvoker) (Value, error)
}

func (b Builtin) Invoke(args []Value, invoke FunctionInvoker) (Value, error) {
	if b.Arity != nil && len(args) != *b.Arity {
		return nil, fmt.Errorf("%s expects %d arguments, got %d", b.Name, *b.Arity, len(args))
	}
	var value Value
	var err error
	switch {
	case b.CallWith != nil:
		value, err = b.CallWith(args, invoke)
	case b.Call != nil:
		value, err = b.Call(args)
	default:
		return nil, fmt.Errorf("builtin %q has no implementation", b.Name)
	}
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("builtin %q returned no value", b.Name)
	}
	return value, nil
}

func (Builtin) Type() Type        { return BuiltinType }
func (b Builtin) Inspect() string { return "<builtin " + b.Name + ">" }

type Stream struct {
	Values  []Value
	Columns []string // Optional CSV column order, retained by row-preserving operations.
}

func (Stream) Type() Type        { return StreamType }
func (s Stream) Inspect() string { return fmt.Sprintf("<stream len=%d>", len(s.Values)) }
