package evaluator

import (
	"fmt"
	"math"
	"strconv"

	"github.com/gabrielcampellounb/gopherflow/internal/ast"
	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
	"github.com/gabrielcampellounb/gopherflow/internal/stdlib"
	"github.com/gabrielcampellounb/gopherflow/internal/token"
)

func Evaluate(program *ast.Program) (runtime.Value, error) {
	return EvaluateIn(program, standardEnvironment())
}

func standardEnvironment() *runtime.Environment {
	environment := runtime.NewEnvironment(nil)
	// Standard bindings may outlive this evaluation through returned closures.
	// Their callbacks use a sequential evaluator without a worker pool.
	for name, value := range *stdlib.Environment((&execution{}).invokeFunction) {
		environment.Set(name, value)
	}
	return environment
}

func EvaluateIn(program *ast.Program, environment *runtime.Environment) (runtime.Value, error) {
	return (&execution{}).evaluateProgram(program, environment)
}

func (exec *execution) evaluateProgram(program *ast.Program, environment *runtime.Environment) (runtime.Value, error) {
	if len(program.Expressions) == 0 {
		return runtime.Null{}, nil
	}
	scope := environment
	if len(program.Definitions) > 0 {
		scope = runtime.NewEnvironment(environment)
		for _, definition := range program.Definitions {
			names := make([]string, len(definition.Parameters))
			for i, parameter := range definition.Parameters {
				names[i] = parameter.Name
			}
			scope.Set(definition.Name.Name, runtime.Function{
				Parameters:    names,
				Body:          definition.Body,
				OptimizedBody: definition.OptimizedBody,
				Env:           scope,
			})
		}
	}
	return exec.evaluateExpression(program.Expressions[0], scope)
}

func (exec *execution) evaluateExpression(expression ast.Expression, environment *runtime.Environment) (runtime.Value, error) {
	switch node := expression.(type) {
	case *ast.Identifier:
		value, ok := environment.Get(node.Name)
		if !ok {
			return nil, fmt.Errorf("name error at %d:%d: undefined identifier %q", node.Pos.Line, node.Pos.Column, node.Name)
		}
		return value, nil
	case *ast.Literal:
		return evaluateLiteral(node)
	case *ast.LetExpression:
		value, err := exec.evaluateExpression(node.Value, environment)
		if err != nil {
			return nil, err
		}
		child := runtime.NewEnvironment(environment)
		child.Set(node.Name.Name, value)
		return exec.evaluateExpression(node.Body, child)
	case *ast.IfExpression:
		condition, err := exec.evaluateExpression(node.Condition, environment)
		if err != nil {
			return nil, err
		}
		boolean, ok := condition.(runtime.Bool)
		if !ok {
			return nil, fmt.Errorf("type error at %d:%d: if condition must be Bool, got %s", node.Pos.Line, node.Pos.Column, condition.Type())
		}
		if boolean {
			return exec.evaluateExpression(node.Consequence, environment)
		}
		return exec.evaluateExpression(node.Alternative, environment)
	case *ast.UnaryExpression:
		right, err := exec.evaluateExpression(node.Right, environment)
		if err != nil {
			return nil, err
		}
		if number, ok := right.(runtime.Number); ok && node.Operator.Type == token.Minus {
			return -number, nil
		}
		if boolean, ok := right.(runtime.Bool); ok && node.Operator.Type == token.Not {
			return !boolean, nil
		}
		return nil, fmt.Errorf("type error at %d:%d: operator %s is not defined for %s", node.Operator.Position.Line, node.Operator.Position.Column, node.Operator.Lexeme, right.Type())
	case *ast.BinaryExpression:
		if node.Operator.Type == token.Pipe {
			return exec.evaluateCall(pipelineCall(node), environment)
		}
		return exec.evaluateBinary(node, environment)
	case *ast.FunctionLiteral:
		parameters := make([]string, len(node.Parameters))
		for i, parameter := range node.Parameters {
			parameters[i] = parameter.Name
		}
		return runtime.Function{
			Parameters:    parameters,
			Body:          node.Body,
			OptimizedBody: node.OptimizedBody,
			Env:           environment,
		}, nil
	case *ast.CallExpression:
		return exec.evaluateCall(node, environment)
	case *ast.ListLiteral:
		if values, scheduled, err := exec.parallelItems(len(node.Elements), func(i int) ast.Expression {
			return node.Elements[i]
		}); scheduled {
			if err != nil {
				return nil, err
			}
			return runtime.List(values), nil
		}
		elements := make(runtime.List, len(node.Elements))
		for i, element := range node.Elements {
			value, err := exec.evaluateExpression(element, environment)
			if err != nil {
				return nil, err
			}
			elements[i] = value
		}
		return elements, nil
	case *ast.RecordLiteral:
		if values, scheduled, err := exec.parallelItems(len(node.Fields), func(i int) ast.Expression {
			return node.Fields[i].Value
		}); scheduled {
			if err != nil {
				return nil, err
			}
			record := make(runtime.Record, len(node.Fields))
			for i, field := range node.Fields {
				record[field.Name.Name] = values[i]
			}
			return record, nil
		}
		record := make(runtime.Record, len(node.Fields))
		for _, field := range node.Fields {
			value, err := exec.evaluateExpression(field.Value, environment)
			if err != nil {
				return nil, err
			}
			record[field.Name.Name] = value
		}
		return record, nil
	case *ast.FieldExpression:
		target, err := exec.evaluateExpression(node.Target, environment)
		if err != nil {
			return nil, err
		}
		record, ok := target.(runtime.Record)
		if !ok {
			return nil, fmt.Errorf("type error at %d:%d: %s has no fields", node.Pos.Line, node.Pos.Column, target.Type())
		}
		value, ok := record[node.Name.Name]
		if !ok {
			return nil, fmt.Errorf("runtime error at %d:%d: unknown record field %q", node.Name.Pos.Line, node.Name.Pos.Column, node.Name.Name)
		}
		return value, nil
	case *ast.IndexExpression:
		return exec.evaluateIndex(node, environment)
	default:
		return nil, fmt.Errorf("unsupported expression %T", expression)
	}
}

// pipelineCall lowers x |> f to f(x) and x |> f(a) to f(x, a).
// Keep the original right operand's form before optimization can change it.
func pipelineCall(expression *ast.BinaryExpression) *ast.CallExpression {
	call := &ast.CallExpression{
		Function:  expression.Right,
		Arguments: []ast.Expression{expression.Left},
		Pos:       expression.Operator.Position,
	}
	if right, ok := expression.Right.(*ast.CallExpression); ok {
		call.Function = right.Function
		call.Arguments = append(call.Arguments, right.Arguments...)
		call.Pos = right.Pos
	}
	return call
}

func (exec *execution) evaluateCall(call *ast.CallExpression, environment *runtime.Environment) (runtime.Value, error) {
	callee, err := exec.evaluateExpression(call.Function, environment)
	if err != nil {
		return nil, err
	}
	arguments := make([]runtime.Value, len(call.Arguments))
	for i, argument := range call.Arguments {
		arguments[i], err = exec.evaluateExpression(argument, environment)
		if err != nil {
			return nil, err
		}
	}
	position := call.Pos
	switch function := callee.(type) {
	case runtime.Function:
		if len(arguments) != len(function.Parameters) {
			return nil, fmt.Errorf("arity error at %d:%d: function expects %d arguments, got %d",
				position.Line, position.Column, len(function.Parameters), len(arguments))
		}
		return exec.invokeFunction(function, arguments)
	case runtime.Builtin:
		if function.Call == nil {
			return nil, fmt.Errorf("runtime error at %d:%d: builtin %q has no implementation", position.Line, position.Column, function.Name)
		}
		value, err := function.Call(arguments)
		if err != nil {
			return nil, fmt.Errorf("runtime error at %d:%d: %w", position.Line, position.Column, err)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("type error at %d:%d: %s is not callable",
			position.Line, position.Column, callee.Type())
	}
}

func (exec *execution) invokeFunction(function runtime.Function, arguments []runtime.Value) (runtime.Value, error) {
	scope := runtime.NewEnvironment(function.Env)
	for i, name := range function.Parameters {
		scope.Set(name, arguments[i])
	}
	if function.OptimizedBody != nil {
		return exec.evaluateExpression(function.OptimizedBody, scope)
	}
	return exec.evaluateExpression(function.Body, scope)
}

func (exec *execution) evaluateIndex(expression *ast.IndexExpression, environment *runtime.Environment) (runtime.Value, error) {
	target, err := exec.evaluateExpression(expression.Target, environment)
	if err != nil {
		return nil, err
	}
	index, err := exec.evaluateExpression(expression.Index, environment)
	if err != nil {
		return nil, err
	}
	position := expression.Pos
	list, ok := target.(runtime.List)
	if !ok {
		return nil, fmt.Errorf("type error at %d:%d: %s is not indexable",
			position.Line, position.Column, target.Type())
	}
	number, ok := index.(runtime.Number)
	if !ok {
		return nil, fmt.Errorf("type error at %d:%d: list index must be a Number, got %s",
			position.Line, position.Column, index.Type())
	}
	value := float64(number)
	if value != math.Trunc(value) {
		return nil, fmt.Errorf("runtime error at %d:%d: list index must be an integer, got %s",
			position.Line, position.Column, number.Inspect())
	}
	if value < 0 || value >= float64(len(list)) {
		return nil, fmt.Errorf("runtime error at %d:%d: list index %s out of range (length %d)",
			position.Line, position.Column, number.Inspect(), len(list))
	}
	return list[int(value)], nil
}

func evaluateLiteral(literal *ast.Literal) (runtime.Value, error) {
	switch literal.Kind {
	case token.Number:
		number, err := strconv.ParseFloat(literal.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q: %w", literal.Value, err)
		}
		return runtime.Number(number), nil
	case token.String:
		return runtime.String(literal.Value), nil
	case token.True:
		return runtime.Bool(true), nil
	case token.False:
		return runtime.Bool(false), nil
	case token.Null:
		return runtime.Null{}, nil
	default:
		return nil, fmt.Errorf("unsupported literal %s", literal.Kind)
	}
}

func (exec *execution) evaluateBinary(expression *ast.BinaryExpression, environment *runtime.Environment) (runtime.Value, error) {
	left, err := exec.evaluateExpression(expression.Left, environment)
	if err != nil {
		return nil, err
	}
	leftBool, leftIsBool := left.(runtime.Bool)
	if leftIsBool {
		switch expression.Operator.Type {
		case token.And:
			if !leftBool {
				return leftBool, nil
			}
		case token.Or:
			if leftBool {
				return leftBool, nil
			}
		}
	}
	right, err := exec.evaluateExpression(expression.Right, environment)
	if err != nil {
		return nil, err
	}
	if leftIsBool && (expression.Operator.Type == token.And || expression.Operator.Type == token.Or) {
		if rightBool, ok := right.(runtime.Bool); ok {
			return rightBool, nil
		}
	}
	leftNumber, leftOK := left.(runtime.Number)
	rightNumber, rightOK := right.(runtime.Number)
	if leftOK && rightOK {
		switch expression.Operator.Type {
		case token.Plus:
			return leftNumber + rightNumber, nil
		case token.Minus:
			return leftNumber - rightNumber, nil
		case token.Star:
			return leftNumber * rightNumber, nil
		case token.Slash:
			if rightNumber == 0 {
				return nil, fmt.Errorf("runtime error at %d:%d: division by zero", expression.Operator.Position.Line, expression.Operator.Position.Column)
			}
			return leftNumber / rightNumber, nil
		case token.Percent:
			if rightNumber == 0 {
				return nil, fmt.Errorf("runtime error at %d:%d: division by zero", expression.Operator.Position.Line, expression.Operator.Position.Column)
			}
			return runtime.Number(math.Mod(float64(leftNumber), float64(rightNumber))), nil
		case token.Equal:
			return runtime.Bool(leftNumber == rightNumber), nil
		case token.NotEqual:
			return runtime.Bool(leftNumber != rightNumber), nil
		case token.Less:
			return runtime.Bool(leftNumber < rightNumber), nil
		case token.LessEqual:
			return runtime.Bool(leftNumber <= rightNumber), nil
		case token.Greater:
			return runtime.Bool(leftNumber > rightNumber), nil
		case token.GreaterEq:
			return runtime.Bool(leftNumber >= rightNumber), nil
		}
	}
	if expression.Operator.Type == token.Equal {
		return runtime.Bool(left.Type() == right.Type() && left.Inspect() == right.Inspect()), nil
	}
	if expression.Operator.Type == token.NotEqual {
		return runtime.Bool(!(left.Type() == right.Type() && left.Inspect() == right.Inspect())), nil
	}
	return nil, fmt.Errorf("type error at %d:%d: operator %s is not defined for %s and %s", expression.Operator.Position.Line, expression.Operator.Position.Column, expression.Operator.Lexeme, left.Type(), right.Type())
}
