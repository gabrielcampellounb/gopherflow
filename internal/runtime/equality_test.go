package runtime

import (
	"math"
	"testing"
)

func repeatedCollections(leaf Value, levels int, wrap func(Value) Value) Value {
	for range levels {
		leaf = wrap(leaf)
	}
	return leaf
}

func TestEqualSharedCollections(t *testing.T) {
	for _, tt := range []struct {
		name string
		wrap func(Value) Value
	}{
		{"lists", func(value Value) Value { return List{value, value} }},
		{"records", func(value Value) Value { return Record{"a": value, "b": value} }},
		{"streams", func(value Value) Value { return Stream{Values: []Value{value, value}, Columns: []string{"data"}} }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			left := repeatedCollections(Number(1), 64, tt.wrap)
			right := repeatedCollections(Number(1), 64, tt.wrap)
			if !Equal(left, left) || !Equal(left, right) {
				t.Fatal("equal shared collections compared as different")
			}
			if Equal(left, repeatedCollections(Number(2), 64, tt.wrap)) {
				t.Fatal("collections with different leaves compared as equal")
			}
			nan := repeatedCollections(Number(math.NaN()), 64, tt.wrap)
			if Equal(nan, nan) {
				t.Fatal("shared identity bypassed NaN comparison")
			}
		})
	}
}

type inspectedValue struct{ inspections *int }

func (inspectedValue) Type() Type { return StringType }
func (value inspectedValue) Inspect() string {
	*value.inspections++
	return "host value"
}

func TestEqualRepeatsHostInspection(t *testing.T) {
	var inspections int
	item := List{inspectedValue{inspections: &inspections}}
	if !Equal(List{item, item}, List{item, item}) || inspections != 4 {
		t.Fatalf("expected four host inspections, got %d", inspections)
	}
}

func TestEqualDistinguishesSliceViewsAndStreamColumns(t *testing.T) {
	left := List{Number(0), Number(1)}
	right := List{Number(0), Number(2)}
	if Equal(List{left[:1], left}, List{right[:1], right}) {
		t.Fatal("a matching prefix hid unequal elements")
	}
	if Equal(Stream{Values: left, Columns: []string{"a", "b"}}, Stream{Values: left, Columns: []string{"b", "a"}}) {
		t.Fatal("streams with different column order compared as equal")
	}
}
