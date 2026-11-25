package runtime

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestWriteValuePreservesInspectionFormat(t *testing.T) {
	value := Record{"z": List{}, "a": List{Number(3), Record{"b": String("x\n")}, Null{}}}
	const want = `{a: [3, {b: "x\n"}, null], z: []}`
	var output strings.Builder
	if err := WriteValue(&output, value); err != nil || output.String() != want || value.Inspect() != want {
		t.Fatalf("write=%q inspect=%q error=%v", output.String(), value.Inspect(), err)
	}
}

type limitedOutput struct {
	remaining int
	written   strings.Builder
}

func (out *limitedOutput) Write(p []byte) (int, error) {
	n := min(len(p), out.remaining)
	out.remaining -= n
	_, _ = out.written.Write(p[:n])
	if n != len(p) {
		return n, io.ErrClosedPipe
	}
	return n, nil
}

type shortOutput struct{}

func (shortOutput) Write([]byte) (int, error) { return 0, nil }

func TestWriteValueStopsTraversingOnOutputError(t *testing.T) {
	for _, tt := range []struct {
		value  func(Value) Value
		budget int
		prefix string
	}{
		{func(item Value) Value { return List{Number(1), item} }, 2, "[1"},
		{func(item Value) Value { return Record{"a": Number(1), "b": item} }, 5, "{a: 1"},
	} {
		var inspections int
		value := tt.value(inspectedValue{inspections: &inspections})
		output := &limitedOutput{remaining: tt.budget}
		if err := WriteValue(output, value); !errors.Is(err, io.ErrClosedPipe) ||
			output.written.String() != tt.prefix || inspections != 0 {
			t.Fatalf("output=%q inspections=%d error=%v", output.written.String(), inspections, err)
		}
	}
	if err := WriteValue(shortOutput{}, List{Number(1)}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write returned %v", err)
	}
}

func TestRecordFormattingQuotesNonIdentifierKeys(t *testing.T) {
	record := Record{
		"": Number(0), "_name": Number(1), "a: 1, b": Number(2),
		"name_2": Number(3), "true": Number(4), "á\r\n\"": Number(5),
	}
	const want = `{"": 0, "_name": 1, "a: 1, b": 2, name_2: 3, "true": 4, "á\r\n\"": 5}`
	var output strings.Builder
	if err := WriteValue(&output, record); err != nil || output.String() != want || record.Inspect() != want {
		t.Fatalf("write=%q inspect=%q error=%v, want %q", output.String(), record.Inspect(), err, want)
	}
}
