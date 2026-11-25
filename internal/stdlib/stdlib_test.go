package stdlib

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
)

func TestStandardBindings(t *testing.T) {
	env := Environment()
	if len(*env) != 9 {
		t.Fatalf("unexpected environment: %v", env)
	}
	for _, name := range []string{"source", "map", "filter", "flatMap", "take", "keyBy", "reduce", "forEach", "sink"} {
		builtin, ok := (*env)[name].(runtime.Builtin)
		if !ok || builtin.Name != name || builtin.CallWith == nil {
			t.Fatalf("invalid builtin %q: %#v", name, (*env)[name])
		}
		if result, err := builtin.Invoke(nil, nil); result != nil || err == nil {
			t.Fatalf("%s accepted zero arguments: %v, %v", name, result, err)
		}
	}
	delete(*env, "map")
	if _, ok := (*Environment())["map"]; !ok {
		t.Fatal("environments share mutable bindings")
	}
}

func TestCSVSourceAndSink(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.csv")
	output := filepath.Join(dir, "output.csv")
	data := "temperature,city\n31.5,\"Brasilia, DF\"\n-2,\"Two\nlines\"\n"
	if err := os.WriteFile(input, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	value, err := source([]runtime.Value{runtime.String(input)})
	if err != nil {
		t.Fatal(err)
	}
	want := runtime.Stream{Columns: []string{"temperature", "city"}, Values: []runtime.Value{
		runtime.Record{"temperature": runtime.Number(31.5), "city": runtime.String("Brasilia, DF")},
		runtime.Record{"temperature": runtime.Number(-2), "city": runtime.String("Two\nlines")},
	}}
	if !reflect.DeepEqual(value, want) {
		t.Fatalf("source = %#v, want %#v", value, want)
	}
	if _, err := sink([]runtime.Value{value, runtime.String(output)}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(output)
	if err != nil || string(got) != data {
		t.Fatalf("sink wrote %q, %v, want %q", got, err, data)
	}
}

func TestCSVSinkPreservesDestinationOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.csv")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, rows := range []runtime.List{
		{runtime.Record{"a": runtime.Number(1)}, runtime.Record{"b": runtime.Number(2)}},
		{runtime.Record{"a": runtime.List{}}},
		{runtime.Record{"a": runtime.Number(1)}, runtime.Number(2)},
	} {
		if _, err := sink([]runtime.Value{rows, runtime.String(path)}); err == nil {
			t.Fatalf("sink accepted %v", rows)
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != "original" {
			t.Fatalf("destination changed: %q, %v", got, err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files left over: %v, %v", entries, err)
	}
}

func TestCSVValidationAndEmptyRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.csv")
	for _, tt := range []struct{ data, errorText string }{
		{"a,a\n1,2\n", "duplicate CSV column"},
		{"a,\n1,2\n", "empty or duplicate CSV column"},
		{"a,b\n1\n", "wrong number of fields"},
		{"\"open\n", "source header"},
	} {
		if err := os.WriteFile(path, []byte(tt.data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := source([]runtime.Value{runtime.String(path)}); err == nil || !strings.Contains(err.Error(), tt.errorText) {
			t.Errorf("source(%q): %v", tt.data, err)
		}
	}
	rows := runtime.List{runtime.Record{"a": runtime.Null{}}, runtime.Record{"a": runtime.String("")}}
	if _, err := sink([]runtime.Value{rows, runtime.String(path)}); err != nil {
		t.Fatal(err)
	}
	value, err := source([]runtime.Value{runtime.String(path)})
	if err != nil || len(value.(runtime.Stream).Values) != 2 {
		t.Fatalf("empty cells disappeared: %#v, %v", value, err)
	}
}

func TestCSVSourceUTF8BOM(t *testing.T) {
	for _, tt := range []struct {
		name string
		data string
		want runtime.Stream
	}{
		{"quoted header with CRLF", "\ufeff\"city\",temperature\r\n\ufeffBrasília,30\r\n", runtime.Stream{
			Columns: []string{"city", "temperature"},
			Values: []runtime.Value{
				runtime.Record{"city": runtime.String("\ufeffBrasília"), "temperature": runtime.Number(30)},
			},
		}},
		{"unquoted header", "\ufeffcity\n", runtime.Stream{Columns: []string{"city"}}},
		{"BOM only", "\ufeff", runtime.Stream{}},
		{"empty file", "", runtime.Stream{}},
		{"short header", "x", runtime.Stream{Columns: []string{"x"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "data.csv")
			if err := os.WriteFile(path, []byte(tt.data), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := source([]runtime.Value{runtime.String(path)})
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("source = %#v, %v, want %#v", got, err, tt.want)
			}
		})
	}
}
