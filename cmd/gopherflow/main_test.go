package main

import (
	"bytes"
	"fmt"
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
