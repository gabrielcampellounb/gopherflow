package stdlib

import (
	"fmt"

	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
)

func Environment() *runtime.Record {
	return &runtime.Record{
		"source": runtime.Builtin{Name: "source", Call: unavailable("source")},
		"map":    runtime.Builtin{Name: "map", Call: unavailable("map")},
		"filter": runtime.Builtin{Name: "filter", Call: unavailable("filter")},
		"sink":   runtime.Builtin{Name: "sink", Call: unavailable("sink")},
	}
}

func unavailable(name string) func([]runtime.Value) (runtime.Value, error) {
	return func([]runtime.Value) (runtime.Value, error) {
		return nil, fmt.Errorf("stdlib function %s is not implemented", name)
	}
}
