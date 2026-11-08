package runtime

import "testing"

func TestValueTypesAndInspection(t *testing.T) {
	tests := []struct {
		value Value
		typ   Type
		text  string
	}{
		{Number(2.5), NumberType, "2.5"},
		{String("hello"), StringType, `"hello"`},
		{Bool(true), BoolType, "true"},
		{Null{}, NullType, "null"},
		{List{Number(1), Bool(false)}, ListType, "[1, false]"},
		{Record{"b": Number(2), "a": String("x")}, RecordType, `{a: "x", b: 2}`},
	}
	for _, tt := range tests {
		if tt.value.Type() != tt.typ || tt.value.Inspect() != tt.text {
			t.Errorf("value = %s/%s, want %s/%s", tt.value.Type(), tt.value.Inspect(), tt.typ, tt.text)
		}
	}
}
