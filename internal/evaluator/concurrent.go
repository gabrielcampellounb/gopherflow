package evaluator

import (
	"sync"

	"github.com/gabrielcampellounb/gopherflow/internal/ast"
	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
	"github.com/gabrielcampellounb/gopherflow/internal/token"
)

type execution struct {
	workerLimit int
	callDepth   int
	pool        *workerPool
}

// EvaluateConcurrent optimizes the AST and evaluates independent batches with
// up to workers goroutines. Counts below two use sequential evaluation.
func EvaluateConcurrent(program *ast.Program, workers int) (runtime.Value, error) {
	exec := &execution{workerLimit: workers}
	defer exec.close()
	return exec.evaluateProgram(Optimize(program), standardEnvironment())
}

func (exec *execution) close() {
	if exec.pool != nil {
		close(exec.pool.jobs)
		exec.pool.workers.Wait()
	}
}

type workerPool struct {
	jobs    chan func()
	workers sync.WaitGroup
	size    int
}

func (pool *workerPool) grow(size int) {
	for pool.size < size {
		pool.workers.Go(func() {
			for job := range pool.jobs {
				job()
			}
		})
		pool.size++
	}
}

// Require several nonliteral items per worker so constant-folded collections
// and small batches avoid goroutine scheduling overhead.
const nonliteralItemsPerWorker = 16

func (exec *execution) parallelItems(count int, item func(int) ast.Expression) ([]runtime.Value, bool, error) {
	if exec.workerLimit < 2 || count < 2*nonliteralItemsPerWorker {
		return nil, false, nil
	}
	nonliteral := 0
	for i := range count {
		expression := item(i)
		if _, literal := expression.(*ast.Literal); literal {
			continue
		}
		nonliteral++
		if !isClosedPure(expression, nil) {
			return nil, false, nil
		}
	}

	chunks := min(exec.workerLimit, nonliteral/nonliteralItemsPerWorker)
	if chunks < 2 {
		return nil, false, nil
	}
	if exec.pool == nil {
		exec.pool = &workerPool{jobs: make(chan func())}
	}
	exec.pool.grow(chunks)

	values := make([]runtime.Value, count)
	chunkErrors := make([]error, chunks)
	var done sync.WaitGroup
	for chunk := range chunks {
		first := chunk*(count/chunks) + min(chunk, count%chunks)
		last := first + count/chunks
		if chunk < count%chunks {
			last++
		}
		done.Add(1)
		exec.pool.jobs <- func() {
			defer done.Done()
			// Worker jobs use no caller environment and never submit nested jobs.
			serial := &execution{}
			for i := first; i < last; i++ {
				value, err := serial.evaluateExpression(item(i), nil)
				if err != nil {
					chunkErrors[chunk] = err
					return
				}
				values[i] = value
			}
		}
	}
	done.Wait()
	for _, err := range chunkErrors {
		if err != nil {
			return nil, true, err
		}
	}
	return values, true, nil
}

type localBinding struct {
	name   string
	parent *localBinding
}

// A closed pure expression can only construct interpreter-native values and
// read bindings introduced inside the job. Impure or unknown nodes stay serial.
func isClosedPure(expression ast.Expression, scope *localBinding) bool {
	switch node := expression.(type) {
	case *ast.Literal:
		return true
	case *ast.Identifier:
		for binding := scope; binding != nil; binding = binding.parent {
			if binding.name == node.Name {
				return true
			}
		}
		return false
	case *ast.LetExpression:
		return isClosedPure(node.Value, scope) &&
			isClosedPure(node.Body, &localBinding{name: node.Name.Name, parent: scope})
	case *ast.UnaryExpression:
		return isClosedPure(node.Right, scope)
	case *ast.BinaryExpression:
		return node.Operator.Type != token.Pipe && isClosedPure(node.Left, scope) && isClosedPure(node.Right, scope)
	case *ast.IfExpression:
		return isClosedPure(node.Condition, scope) &&
			isClosedPure(node.Consequence, scope) && isClosedPure(node.Alternative, scope)
	case *ast.ListLiteral:
		for _, element := range node.Elements {
			if !isClosedPure(element, scope) {
				return false
			}
		}
		return true
	case *ast.RecordLiteral:
		for _, field := range node.Fields {
			if !isClosedPure(field.Value, scope) {
				return false
			}
		}
		return true
	case *ast.IndexExpression:
		return isClosedPure(node.Target, scope) && isClosedPure(node.Index, scope)
	case *ast.FieldExpression:
		return isClosedPure(node.Target, scope)
	default:
		return false
	}
}
