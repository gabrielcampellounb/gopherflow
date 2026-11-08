package runtime

import (
	"math"
	"reflect"
	"testing"
)

func TestAllValueRepresentations(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value Value
		typ   Type
		text  string
	}{
		{"zero", Number(0), NumberType, "0"}, {"negative zero", Number(math.Copysign(0, -1)), NumberType, "-0"},
		{"negative", Number(-2.5), NumberType, "-2.5"}, {"false", Bool(false), BoolType, "false"},
		{"escaped", String("\"\n\r\t\\"), StringType, `"\"\n\r\t\\"`},
		{"unicode", String("ação 😀"), StringType, `"ação 😀"`},
		{"nil list", List(nil), ListType, "[]"}, {"empty list", List{}, ListType, "[]"},
		{"nil record", Record(nil), RecordType, "{}"}, {"empty record", Record{}, RecordType, "{}"},
		{"nested", List{Record{"z": List{Null{}, Bool(true)}, "a": Number(1)}}, ListType, "[{a: 1, z: [null, true]}]"},
		{"function", Function{}, FunctionType, "<function>"},
		{"builtin", Builtin{Name: "map"}, BuiltinType, "<builtin map>"},
		{"empty stream", Stream{}, StreamType, "<stream len=0>"},
		{"stream", Stream{Values: []Value{Number(1), Null{}}}, StreamType, "<stream len=2>"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value.Type() != tt.typ || tt.value.Inspect() != tt.text {
				t.Fatalf("got %s %q, want %s %q", tt.value.Type(), tt.value.Inspect(), tt.typ, tt.text)
			}
		})
	}
}

func TestInspectionDeterminismAndPurity(t *testing.T) {
	r := Record{"z": Number(3), "a": String("x"), "m": Bool(false)}
	want := Record{"z": Number(3), "a": String("x"), "m": Bool(false)}
	list := List{Number(2), r}
	wantList := List{Number(2), want}
	for range 100 {
		if got := r.Inspect(); got != `{a: "x", m: false, z: 3}` {
			t.Fatal(got)
		}
		if got := list.Inspect(); got != `[2, {a: "x", m: false, z: 3}]` {
			t.Fatal(got)
		}
	}
	if !reflect.DeepEqual(r, want) || !reflect.DeepEqual(list, wantList) {
		t.Fatal("inspection mutated values")
	}
}
