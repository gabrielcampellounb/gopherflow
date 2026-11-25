package stdlib

import (
	"fmt"

	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
)

func Environment() *runtime.Record {
	bindings := make(runtime.Record)
	for _, name := range []string{"source", "map", "filter", "flatMap", "take", "keyBy", "reduce", "forEach", "sink"} {
		bindings[name] = runtime.Builtin{Name: name, Call: unavailable(name)}
	}
	return &bindings
}

func unavailable(name string) func([]runtime.Value) (runtime.Value, error) {
	return func([]runtime.Value) (runtime.Value, error) {
		return nil, fmt.Errorf("stdlib function %s is not implemented", name)
	}
}
