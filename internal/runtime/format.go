package runtime

import (
	"io"
	"maps"
	"slices"
	"strings"
)

// WriteValue writes the same representation as Inspect, traversing collections
// in order and stopping when the destination rejects a write.
func WriteValue(output io.Writer, value Value) error {
	writer := valueWriter{output: output}
	writer.write(value)
	return writer.err
}

func inspectCollection(value Value) string {
	var output strings.Builder
	_ = WriteValue(&output, value) // strings.Builder cannot fail.
	return output.String()
}

type valueWriter struct {
	output io.Writer
	err    error
}

func (w *valueWriter) text(text string) {
	if w.err != nil {
		return
	}
	n, err := io.WriteString(w.output, text)
	if err == nil && n != len(text) {
		err = io.ErrShortWrite
	}
	w.err = err
}

func (w *valueWriter) write(value Value) {
	if w.err != nil {
		return
	}
	switch value := value.(type) {
	case List:
		w.text("[")
		for i, item := range value {
			if i > 0 {
				w.text(", ")
			}
			w.write(item)
			if w.err != nil {
				return
			}
		}
		w.text("]")
	case Record:
		w.text("{")
		if w.err != nil {
			return
		}
		for i, name := range slices.Sorted(maps.Keys(value)) {
			if i > 0 {
				w.text(", ")
			}
			w.text(name)
			w.text(": ")
			w.write(value[name])
			if w.err != nil {
				return
			}
		}
		w.text("}")
	default:
		w.text(value.Inspect())
	}
}
