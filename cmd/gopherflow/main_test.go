package main

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

func TestCLI(t *testing.T) {
	for _, tt := range []struct {
		name           string
		args           []string
		status         int
		stdout, stderr string
	}{
		{"default", nil, 0, "null\n", ""},
		{"file", []string{"examples/hello.gf"}, 2, "", "file execution is not implemented yet\n"},
		{"flag", []string{"--help"}, 2, "", "file execution is not implemented yet\n"},
		{"multiple arguments", []string{"one", "two"}, 2, "", "file execution is not implemented yet\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := run(tt.args, &stdout, &stderr)
			if status != tt.status || stdout.String() != tt.stdout || stderr.String() != tt.stderr {
				t.Fatalf("got status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

// Exercise the actual entry point and process status, including os.Exit.
func TestCLIProcess(t *testing.T) {
	if os.Getenv("GOPHERFLOW_TEST_PROCESS") == "1" {
		os.Args = append([]string{"gopherflow"}, os.Args[3:]...)
		main()
		return
	}
	for _, args := range [][]string{nil, {"file.gf"}} {
		cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
		cmd.Env = append(os.Environ(), "GOPHERFLOW_TEST_PROCESS=1")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if len(args) == 0 {
			if err != nil || stdout.String() != "null\n" || stderr.Len() != 0 {
				t.Fatalf("CLI: %v, %q, %q", err, stdout.String(), stderr.String())
			}
		} else {
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 2 || stdout.Len() != 0 || stderr.String() != "file execution is not implemented yet\n" {
				t.Fatalf("CLI: %v, %q, %q", err, stdout.String(), stderr.String())
			}
		}
	}
}
