package main

import (
	"fmt"
	"io"
	"os"

	"github.com/gabrielcampellounb/gopherflow/internal/evaluator"
	"github.com/gabrielcampellounb/gopherflow/internal/lexer"
	"github.com/gabrielcampellounb/gopherflow/internal/parser"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		fmt.Fprintln(stderr, "file execution is not implemented yet")
		return 2
	}

	program, err := parser.Parse(lexer.New("null"))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	value, err := evaluator.Evaluate(program)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, value.Inspect())
	return 0
}
