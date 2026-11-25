package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	goruntime "runtime"

	"github.com/gabrielcampellounb/gopherflow/internal/evaluator"
	"github.com/gabrielcampellounb/gopherflow/internal/lexer"
	"github.com/gabrielcampellounb/gopherflow/internal/parser"
	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
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
	value, err := evaluator.EvaluateConcurrent(program, goruntime.GOMAXPROCS(0))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	output := bufio.NewWriter(stdout)
	err = runtime.WriteValue(output, value)
	if err == nil {
		err = output.WriteByte('\n')
	}
	if err == nil {
		err = output.Flush()
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
