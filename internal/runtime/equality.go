package runtime

import (
	"maps"
	"reflect"
	"slices"
)

// Equal compares language values. Collections compare their contents.
// Interpreter-created functions compare by identity. Host-provided values may
// still use their inspection representation.
func Equal(left, right Value) bool {
	return (&comparison{}).equal(left, right)
}

type collectionPair struct {
	left, right any
	length      int
}

type comparison struct {
	equalPairs      map[collectionPair]struct{}
	hostComparisons int
}

func (c *comparison) equal(left, right Value) bool {
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
			return c.equalSequence(left, right)
		}
	case Record:
		if right, ok := right.(Record); ok {
			return c.equalRecord(left, right)
		}
	case Function:
		if right, ok := right.(Function); ok && (left.identity != nil || right.identity != nil) {
			return left.identity == right.identity
		}
	case Builtin:
		if right, ok := right.(Builtin); ok {
			return left.Name == right.Name
		}
	case Stream:
		if right, ok := right.(Stream); ok {
			return slices.Equal(left.Columns, right.Columns) && c.equalSequence(left.Values, right.Values)
		}
	}
	// Host inspection can have observable effects or mutate host-owned data.
	c.hostComparisons++
	clear(c.equalPairs)
	return left.Type() == right.Type() && left.Inspect() == right.Inspect()
}

func (c *comparison) equalSequence(left, right []Value) bool {
	if len(left) != len(right) {
		return false
	}
	if len(left) == 0 {
		return true
	}
	// The first element and length identify a nonempty slice view.
	pair := collectionPair{left: &left[0], right: &right[0], length: len(left)}
	if _, seen := c.equalPairs[pair]; seen {
		return true
	}
	before := c.hostComparisons
	for i, value := range left {
		if !c.equal(value, right[i]) {
			return false
		}
	}
	c.remember(pair, before)
	return true
}

func (c *comparison) equalRecord(left, right Record) bool {
	if len(left) != len(right) {
		return false
	}
	if len(left) == 0 {
		return true
	}
	pair := collectionPair{left: reflect.ValueOf(left).UnsafePointer(), right: reflect.ValueOf(right).UnsafePointer()}
	if _, seen := c.equalPairs[pair]; seen {
		return true
	}
	before := c.hostComparisons
	// Keep inspection order deterministic for nested host values.
	for _, key := range slices.Sorted(maps.Keys(left)) {
		value, ok := right[key]
		if !ok || !c.equal(left[key], value) {
			return false
		}
	}
	c.remember(pair, before)
	return true
}

func (c *comparison) remember(pair collectionPair, before int) {
	if c.hostComparisons != before {
		return
	}
	if c.equalPairs == nil {
		c.equalPairs = make(map[collectionPair]struct{})
	}
	c.equalPairs[pair] = struct{}{}
}
