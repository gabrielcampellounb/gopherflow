package runtime

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

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

func (List) Type() Type { return ListType }
func (l List) Inspect() string {
	parts := make([]string, len(l))
	for i, value := range l {
		parts[i] = value.Inspect()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

type Record map[string]Value

func (Record) Type() Type { return RecordType }
func (r Record) Inspect() string {
	keys := make([]string, 0, len(r))
	for key := range r {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, key := range keys {
		parts[i] = fmt.Sprintf("%s: %s", key, r[key].Inspect())
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

type Function struct {
	Parameters []string
	Body       any
}

func (Function) Type() Type      { return FunctionType }
func (Function) Inspect() string { return "<function>" }

type Builtin struct {
	Name string
	Call func([]Value) (Value, error)
}

func (Builtin) Type() Type        { return BuiltinType }
func (b Builtin) Inspect() string { return "<builtin " + b.Name + ">" }

type Stream struct {
	Values []Value
}

func (Stream) Type() Type        { return StreamType }
func (s Stream) Inspect() string { return fmt.Sprintf("<stream len=%d>", len(s.Values)) }
