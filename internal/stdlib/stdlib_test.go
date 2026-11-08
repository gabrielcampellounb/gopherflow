package stdlib

import (
	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
	"testing"
)

func TestPlaceholdersAreExplicitAndIndependent(t *testing.T) {
	env := Environment()
	if env == nil || len(*env) != 4 {
		t.Fatalf("unexpected environment: %v", env)
	}
	for _, name := range []string{"source", "map", "filter", "sink"} {
		b, ok := (*env)[name].(runtime.Builtin)
		if !ok || b.Name != name || b.Call == nil {
			t.Fatalf("invalid builtin %q: %#v", name, (*env)[name])
		}
		for _, args := range [][]runtime.Value{nil, {}, {runtime.Number(1), runtime.String("x")}} {
			got, err := b.Call(args)
			if got != nil || err == nil || err.Error() != "stdlib function "+name+" is not implemented" {
				t.Fatalf("%s: got %v, %v", name, got, err)
			}
		}
	}
	delete(*env, "map")
	if _, ok := (*Environment())["map"]; !ok {
		t.Fatal("environments share mutable bindings")
	}
}
