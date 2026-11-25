package evaluator

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gabrielcampellounb/gopherflow/internal/ast"
	"github.com/gabrielcampellounb/gopherflow/internal/lexer"
	"github.com/gabrielcampellounb/gopherflow/internal/parser"
	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
)

func parallelTestProgram(t testing.TB, source string) *ast.Program {
	t.Helper()
	program, err := parser.Parse(lexer.New(source))
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func TestConcurrentEvaluationOrdersListsAndRecords(t *testing.T) {
	count := 4*nonliteralItemsPerWorker + 1
	items := make([]string, count)
	fields := make([]string, count)
	for i := range count {
		items[i] = fmt.Sprintf("let x = %d in x * 2", i)
		fields[i] = fmt.Sprintf("field%d: %s", i, items[i])
	}
	program := Optimize(parallelTestProgram(t, "["+strings.Join(items, ",")+"]"))
	records := Optimize(parallelTestProgram(t, "{"+strings.Join(fields, ",")+"}"))
	run := &execution{workerLimit: 3}
	defer run.close()
	list, err := run.evaluateProgram(program, runtime.NewEnvironment(nil))
	if err != nil {
		t.Fatal(err)
	}
	pool := run.pool
	if pool == nil || pool.size != 3 {
		t.Fatalf("expected three workers, got %+v", pool)
	}
	record, err := run.evaluateProgram(records, runtime.NewEnvironment(nil))
	if err != nil {
		t.Fatal(err)
	}
	if run.pool != pool {
		t.Fatal("record evaluation did not reuse the worker pool")
	}
	for i := range count {
		want := runtime.Number(i * 2)
		if list.(runtime.List)[i] != want || record.(runtime.Record)[fmt.Sprintf("field%d", i)] != want {
			t.Fatalf("wrong result for item %d: %v, %v", i, list, record)
		}
	}
}

func TestConcurrentEvaluationReturnsFirstSourceError(t *testing.T) {
	items := make([]string, 4*nonliteralItemsPerWorker)
	for i := range items {
		items[i] = "let x = 1 in x + 2"
	}
	items[1] = "2 / 0"
	items[len(items)-2] = "not 5"
	program := parallelTestProgram(t, "[\n"+strings.Join(items, ",\n")+"\n]")
	_, want := Evaluate(program)
	for _, workers := range []int{0, 1, 2, 4, 100} {
		result, err := EvaluateConcurrent(program, workers)
		if result != nil || err == nil || want == nil || err.Error() != want.Error() {
			t.Errorf("workers=%d: got %v, %v, want %v", workers, result, err, want)
		}
	}
	exec := &execution{workerLimit: 4}
	defer exec.close()
	_, err := exec.evaluateProgram(Optimize(program), runtime.NewEnvironment(nil))
	if err == nil || exec.pool == nil {
		t.Fatalf("expected worker execution to return an error, got %v", err)
	}
}

func TestConcurrentEvaluationKeepsEffectsAndExternalBindingsSequential(t *testing.T) {
	items := make([]string, 2*nonliteralItemsPerWorker+3)
	for i := range items {
		items[i] = "let x = 1 in x + 2"
	}
	items[0], items[1], items[2] = "mark(1)", "mark(outside)", "1 / 0"
	items[len(items)-1] = "mark(3)"
	program := Optimize(parallelTestProgram(t, "["+strings.Join(items, ",")+"]"))
	env := runtime.NewEnvironment(nil)
	env.Set("outside", runtime.Number(2))
	var effects []string
	env.Set("mark", runtime.Builtin{Name: "mark", Call: func(args []runtime.Value) (runtime.Value, error) {
		effects = append(effects, args[0].Inspect())
		return args[0], nil
	}})
	run := &execution{workerLimit: 4}
	defer run.close()
	_, err := run.evaluateProgram(program, env)
	if err == nil || !strings.Contains(err.Error(), "division by zero") || strings.Join(effects, ",") != "1,2" || run.pool != nil {
		t.Fatalf("error=%v effects=%v pool=%v", err, effects, run.pool)
	}

	items[0], items[1], items[2], items[len(items)-1] = "outside + 1", items[3], items[3], items[3]
	program = Optimize(parallelTestProgram(t, "["+strings.Join(items, ",")+"]"))
	result, err := run.evaluateProgram(program, env)
	if err != nil || result.(runtime.List)[0] != runtime.Number(3) || run.pool != nil {
		t.Fatalf("external binding: result=%v error=%v pool=%v", result, err, run.pool)
	}
}

func TestConcurrentEvaluationDoesNotStartWorkersForFoldedLiterals(t *testing.T) {
	source := "[" + strings.Repeat("1+2,", 2*nonliteralItemsPerWorker-1) + "1+2]"
	run := &execution{workerLimit: 8}
	defer run.close()
	result, err := run.evaluateProgram(Optimize(parallelTestProgram(t, source)), runtime.NewEnvironment(nil))
	if err != nil || len(result.(runtime.List)) != 2*nonliteralItemsPerWorker || run.pool != nil {
		t.Fatalf("folded list: %v, %v, pool=%v", result, err, run.pool)
	}
}

func TestWorkerPoolRunsJobsSimultaneously(t *testing.T) {
	exec := &execution{pool: &workerPool{jobs: make(chan func())}}
	defer exec.close()
	exec.pool.grow(2)
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	defer close(release)
	job := func() {
		started <- struct{}{}
		<-release
	}
	for range 2 {
		select {
		case exec.pool.jobs <- job:
		case <-time.After(5 * time.Second):
			t.Fatal("no idle worker")
		}
	}
	for range 2 {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("jobs did not run simultaneously")
		}
	}
}

func BenchmarkEvaluateIndependentItems(b *testing.B) {
	const count = 512
	const item = "let x = 7 in x*x + x*x + x*x + x*x"
	program := Optimize(parallelTestProgram(b, "["+strings.Repeat(item+",", count-1)+item+"]"))
	for _, workers := range []int{1, 2, 4} {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				run := &execution{workerLimit: workers}
				_, err := run.evaluateProgram(program, runtime.NewEnvironment(nil))
				run.close()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
