package runtime

import (
	"maps"
	"slices"
)

// Equal compares language values. Collections compare their contents.
// Interpreter-created functions compare by identity. Host-provided values may
// still use their inspection representation.
func Equal(left, right Value) bool {
	switch left := left.(type) {
	case Number:
		if right, ok := right.(Number); ok {
			return left == right
		}
	case String:
		if right, ok := right.(String); ok {
			return left == right
		}
	case Bool:
		if right, ok := right.(Bool); ok {
			return left == right
		}
	case Null:
		if _, ok := right.(Null); ok {
			return true
		}
	case List:
		if right, ok := right.(List); ok {
			return slices.EqualFunc(left, right, Equal)
		}
	case Record:
		if right, ok := right.(Record); ok {
			if len(left) != len(right) {
				return false
			}
			// A stable order also preserves deterministic inspection for nested host values.
			for _, key := range slices.Sorted(maps.Keys(left)) {
				value, ok := right[key]
				if !ok || !Equal(left[key], value) {
					return false
				}
			}
			return true
		}
	case Function:
		if right, ok := right.(Function); ok && (left.identity != nil || right.identity != nil) {
			return left.identity == right.identity
		}
	case Stream:
		if right, ok := right.(Stream); ok {
			return slices.Equal(left.Columns, right.Columns) && slices.EqualFunc(left.Values, right.Values, Equal)
		}
	}
	return left.Type() == right.Type() && left.Inspect() == right.Inspect()
}
