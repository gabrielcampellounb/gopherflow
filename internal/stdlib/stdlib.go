package stdlib

import (
	"fmt"
	"math"

	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
)

// Environment builds standard bindings without retaining an evaluator's call state.
func Environment() *runtime.Record {
	bindings := make(runtime.Record)
	register := func(name string, arity int, call func([]runtime.Value, runtime.FunctionInvoker) (runtime.Value, error)) {
		bindings[name] = runtime.Builtin{Name: name, CallWith: func(args []runtime.Value, invoke runtime.FunctionInvoker) (runtime.Value, error) {
			if len(args) != arity {
				return nil, fmt.Errorf("%s expects %d arguments, got %d", name, arity, len(args))
			}
			return call(args, invoke)
		}}
	}
	registerSimple := func(name string, arity int, call func([]runtime.Value) (runtime.Value, error)) {
		register(name, arity, func(args []runtime.Value, _ runtime.FunctionInvoker) (runtime.Value, error) {
			return call(args)
		})
	}

	registerSimple("source", 1, source)
	registerSimple("sink", 2, sink)
	registerSimple("take", 2, take)
	for _, name := range []string{"map", "filter", "flatMap", "keyBy", "reduce", "forEach"} {
		arity, callbackArity := 2, 1
		if name == "reduce" {
			arity, callbackArity = 3, 2
		}
		register(name, arity, func(args []runtime.Value, invoke runtime.FunctionInvoker) (runtime.Value, error) {
			stream, ok := asStream(args[0])
			if !ok {
				return nil, fmt.Errorf("%s expects a List or Stream", name)
			}
			call, err := prepareCallback(args[arity-1], callbackArity, invoke)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			return transform(name, stream, args[1], call)
		})
	}
	return &bindings
}

func asStream(value runtime.Value) (runtime.Stream, bool) {
	switch value := value.(type) {
	case runtime.List:
		return runtime.Stream{Values: value}, true
	case runtime.Stream:
		return value, true
	default:
		return runtime.Stream{}, false
	}
}

func prepareCallback(value runtime.Value, arity int, invoke runtime.FunctionInvoker) (func([]runtime.Value) (runtime.Value, error), error) {
	switch function := value.(type) {
	case runtime.Function:
		if len(function.Parameters) != arity {
			return nil, fmt.Errorf("callback must accept %d arguments, got %d", arity, len(function.Parameters))
		}
		if invoke == nil {
			return nil, fmt.Errorf("no evaluator available for function callback")
		}
		return func(args []runtime.Value) (runtime.Value, error) { return invoke(function, args) }, nil
	case runtime.Builtin:
		if function.Call != nil || function.CallWith != nil {
			return func(args []runtime.Value) (runtime.Value, error) { return function.Invoke(args, invoke) }, nil
		}
	}
	return nil, fmt.Errorf("expected a callable callback")
}

func take(args []runtime.Value) (runtime.Value, error) {
	stream, ok := asStream(args[0])
	if !ok {
		return nil, fmt.Errorf("take expects a List or Stream")
	}
	count, ok := args[1].(runtime.Number)
	n := float64(count)
	if !ok || math.IsInf(n, 0) || n < 0 || n != math.Trunc(n) {
		return nil, fmt.Errorf("take expects a nonnegative integer count")
	}
	if n < float64(len(stream.Values)) {
		stream.Values = stream.Values[:int(n):int(n)]
	}
	return stream, nil
}

func transform(name string, input runtime.Stream, initial runtime.Value, call func([]runtime.Value) (runtime.Value, error)) (runtime.Value, error) {
	var output runtime.Stream
	if name == "filter" {
		output.Columns = input.Columns
	}
	var groups runtime.Record
	if name == "keyBy" {
		groups = make(runtime.Record)
	}
	accumulator := initial
	for i, item := range input.Values {
		args := []runtime.Value{item}
		if name == "reduce" {
			args = []runtime.Value{accumulator, item}
		}
		value, err := call(args)
		if err != nil {
			return nil, fmt.Errorf("%s item %d: %w", name, i+1, err)
		}
		switch name {
		case "map":
			output.Values = append(output.Values, value)
		case "filter":
			keep, ok := value.(runtime.Bool)
			if !ok {
				return nil, fmt.Errorf("filter item %d: callback must return Bool", i+1)
			}
			if keep {
				output.Values = append(output.Values, item)
			}
		case "flatMap":
			items, ok := asStream(value)
			if !ok {
				return nil, fmt.Errorf("flatMap item %d: callback must return a List or Stream", i+1)
			}
			output.Values = append(output.Values, items.Values...)
		case "keyBy":
			key, ok := value.(runtime.String)
			if !ok {
				return nil, fmt.Errorf("keyBy item %d: callback must return String", i+1)
			}
			group, _ := groups[string(key)].(runtime.Stream)
			group.Columns = input.Columns
			group.Values = append(group.Values, item)
			groups[string(key)] = group
		case "reduce":
			accumulator = value
		}
	}
	switch name {
	case "keyBy":
		return groups, nil
	case "reduce":
		return accumulator, nil
	case "forEach":
		return runtime.Null{}, nil
	default:
		return output, nil
	}
}
