package main

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/gabrielcampellounb/gopherflow/internal/evaluator"
	"github.com/gabrielcampellounb/gopherflow/internal/lexer"
	"github.com/gabrielcampellounb/gopherflow/internal/parser"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: gopherflow [file]")
		return 2
	}
	source := "null"
	if len(args) == 1 {
		data, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		source = string(data)
	}
	program, err := parser.Parse(lexer.New(source))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	value, err := evaluator.EvaluateConcurrent(program, runtime.GOMAXPROCS(0))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, value.Inspect())
	return 0
}
