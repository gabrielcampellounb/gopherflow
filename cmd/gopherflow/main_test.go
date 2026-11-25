package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeProgram(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "program.gf")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCLI(t *testing.T) {
	for _, tt := range []struct {
		name           string
		args           []string
		status         int
		stdout, stderr string
	}{
		{"default", nil, 0, "null\n", ""},
		{"example", []string{"../../examples/hello.gf"}, 0, "\"hello\"\n", ""},
		{"grammar", []string{writeProgram(t, "def add(x, y) = x + y; {items: [2, 3]}.items[1] |> add(4)")}, 0, "7\n", ""},
		{"missing file", []string{filepath.Join(t.TempDir(), "missing.gf")}, 1, "", "missing.gf"},
		{"invalid program", []string{writeProgram(t, "1 +")}, 1, "", "parser error at 1:4"},
		{"runtime error", []string{writeProgram(t, "4 / (2 - 2)")}, 1, "", "runtime error at 1:3"},
		{"multiple arguments", []string{"one", "two"}, 2, "", "usage: gopherflow [file]"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := run(tt.args, &stdout, &stderr)
			gotError := stderr.String()
			if status != tt.status || stdout.String() != tt.stdout ||
				(tt.stderr == "" && gotError != "") || !strings.Contains(gotError, tt.stderr) {
				t.Fatalf("got status=%d stdout=%q stderr=%q", status, stdout.String(), gotError)
			}
		})
	}
}

func TestCLIProcess(t *testing.T) {
	if os.Getenv("GOPHERFLOW_TEST_PROCESS") == "1" {
		os.Args = append([]string{"gopherflow"}, os.Args[3:]...)
		main()
		return
	}
	for _, tt := range []struct {
		args           []string
		status         int
		stdout, stderr string
	}{
		{nil, 0, "null\n", ""},
		{[]string{writeProgram(t, "{value: 21 * 2}.value")}, 0, "42\n", ""},
		{[]string{writeProgram(t, "1 / 0")}, 1, "", "runtime error at 1:3: division by zero\n"},
	} {
		cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestCLIProcess$", "--"}, tt.args...)...)
		cmd.Env = append(os.Environ(), "GOPHERFLOW_TEST_PROCESS=1")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		status := 0
		if err := cmd.Run(); err != nil {
			exit, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatal(err)
			}
			status = exit.ExitCode()
		}
		if status != tt.status || stdout.String() != tt.stdout || stderr.String() != tt.stderr {
			t.Fatalf("CLI: %d, %q, %q", status, stdout.String(), stderr.String())
		}
	}
}

func TestCSVDataflowFromFile(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "data.csv")
	output := filepath.Join(dir, "result.csv")
	if err := os.WriteFile(input, []byte("city,temperature\nBrasilia,30\nCold,-1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	program := fmt.Sprintf(`def valid(x) = x.temperature > 0;
source(%q)
|> filter(valid)
|> map(fn(x) => {city: x.city, temperature: x.temperature * 1.8 + 32})
|> sink(%q);`, input, output)
	var stdout, stderr bytes.Buffer
	if status := run([]string{writeProgram(t, program)}, &stdout, &stderr); status != 0 || stdout.String() != "null\n" || stderr.Len() != 0 {
		t.Fatalf("pipeline exited %d, stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "city,temperature\nBrasilia,86\n" {
		t.Fatalf("output=%q, %v", data, err)
	}
}

func TestCLIConcurrentAST(t *testing.T) {
	items := make([]string, 64)
	for i := range items {
		items[i] = fmt.Sprintf("let x = %d in x * 2", i)
	}
	source := "[" + strings.Join(items, ",") + "] |> reduce(0, fn(sum, x) => sum + x)"
	var stdout, stderr bytes.Buffer
	if status := run([]string{writeProgram(t, source)}, &stdout, &stderr); status != 0 || stdout.String() != "4032\n" || stderr.Len() != 0 {
		t.Fatalf("evaluation exited %d, stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestCSVComputedFieldFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.csv")
	if err := os.WriteFile(path, []byte("air temperature\n30\n-2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	program := fmt.Sprintf(`source(%q) |> map(fn(row) => row["air temperature"]) |> reduce(0, fn(sum, n) => sum + n)`, path)
	var stdout, stderr bytes.Buffer
	if status := run([]string{writeProgram(t, program)}, &stdout, &stderr); status != 0 || stdout.String() != "28\n" || stderr.Len() != 0 {
		t.Fatalf("pipeline exited %d, stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

type failedOutput struct{}

func (failedOutput) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestCLIReportsOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	if status := run(nil, failedOutput{}, &stderr); status != 1 || !strings.Contains(stderr.String(), io.ErrClosedPipe.Error()) {
		t.Fatalf("write failure exited %d, stderr=%q", status, stderr.String())
	}
}

func TestCSVEmptyPipelinePreservesHeader(t *testing.T) {
	for _, tt := range []struct{ name, data, stage, want string }{
		{"header only", "z,a\n", "", "z,a\n"},
		{"take zero", "z,a\n1,2\n", " |> take(0)", "z,a\n"},
		{"filter all", "z,a\n1,2\n", " |> filter(fn(row) => false)", "z,a\n"},
		{"empty file", "", " |> take(0)", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "input.csv")
			output := filepath.Join(dir, "output.csv")
			if err := os.WriteFile(path, []byte(tt.data), 0600); err != nil {
				t.Fatal(err)
			}
			program := fmt.Sprintf("source(%q)%s |> sink(%q)", path, tt.stage, output)
			var stdout, stderr bytes.Buffer
			if status := run([]string{writeProgram(t, program)}, &stdout, &stderr); status != 0 || stdout.String() != "null\n" || stderr.Len() != 0 {
				t.Fatalf("pipeline exited %d, stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
			got, err := os.ReadFile(output)
			if err != nil || string(got) != tt.want {
				t.Fatalf("output=%q, %v, want %q", got, err, tt.want)
			}
		})
	}
}

type failedCollectionOutput struct {
	maxChunk int
}

func (out *failedCollectionOutput) Write(p []byte) (int, error) {
	out.maxChunk = max(out.maxChunk, len(p))
	return 0, io.ErrClosedPipe
}

func TestCLIStopsWhenStreamingCollectionFails(t *testing.T) {
	program := writeProgram(t, "def nest(n, x) = if n == 0 then x else nest(n - 1, [x, x]); nest(16, 0)")
	stdout := &failedCollectionOutput{}
	var stderr bytes.Buffer
	status := run([]string{program}, stdout, &stderr)
	if status != 1 || !strings.Contains(stderr.String(), io.ErrClosedPipe.Error()) ||
		stdout.maxChunk == 0 || stdout.maxChunk >= 64*1024 {
		t.Fatalf("status=%d first write size=%d stderr=%q", status, stdout.maxChunk, stderr.String())
	}
}

func TestCLIShowsDistinctCSVRecordKeysUnambiguously(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.csv")
	if err := os.WriteFile(path, []byte("\"a: 1, b\"\n2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	program := fmt.Sprintf(`let a = {a: 1, b: 2}
in let b = source(%q) |> reduce(null, fn(acc, row) => row)
in [a == b, a, b]`, path)
	var stdout, stderr bytes.Buffer
	status := run([]string{writeProgram(t, program)}, &stdout, &stderr)
	if status != 0 || stdout.String() != "[false, {a: 1, b: 2}, {\"a: 1, b\": 2}]\n" || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestCLIReportsLocationsInFilesWithCRLineEndings(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := run([]string{writeProgram(t, "if true then\rmissing\relse 1")}, &stdout, &stderr)
	if status != 1 || stdout.Len() != 0 || stderr.String() != "name error at 2:1: undefined identifier \"missing\"\n" {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}
