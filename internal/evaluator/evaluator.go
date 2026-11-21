package evaluator

import (
	"fmt"
	"math"
	"strconv"

	"github.com/gabrielcampellounb/gopherflow/internal/ast"
	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
	"github.com/gabrielcampellounb/gopherflow/internal/token"
)

func Evaluate(program *ast.Program) (runtime.Value, error) {
	return EvaluateIn(program, runtime.NewEnvironment(nil))
}

func EvaluateIn(program *ast.Program, environment *runtime.Environment) (runtime.Value, error) {
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
				Parameters: names,
				Body:       definition.Body,
				Env:        scope,
			})
		}
	}
	return evaluateExpression(program.Expressions[0], scope)
}

func evaluateExpression(expression ast.Expression, environment *runtime.Environment) (runtime.Value, error) {
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
		value, err := evaluateExpression(node.Value, environment)
		if err != nil {
			return nil, err
		}
		child := runtime.NewEnvironment(environment)
		child.Set(node.Name.Name, value)
		return evaluateExpression(node.Body, child)
	case *ast.BinaryExpression:
		return evaluateBinary(node, environment)
	case *ast.FunctionLiteral:
		parameters := make([]string, len(node.Parameters))
		for i, parameter := range node.Parameters {
			parameters[i] = parameter.Name
		}
		return runtime.Function{
			Parameters: parameters,
			Body:       node.Body,
			Env:        environment,
		}, nil
	case *ast.CallExpression:
		return evaluateCall(node, environment)
	case *ast.ListLiteral:
		elements := make(runtime.List, len(node.Elements))
		for i, element := range node.Elements {
			value, err := evaluateExpression(element, environment)
			if err != nil {
				return nil, err
			}
			elements[i] = value
		}
		return elements, nil
	case *ast.IndexExpression:
		return evaluateIndex(node, environment)
	default:
		return nil, fmt.Errorf("unsupported expression %T", expression)
	}
}

func evaluateCall(call *ast.CallExpression, environment *runtime.Environment) (runtime.Value, error) {
	callee, err := evaluateExpression(call.Function, environment)
	if err != nil {
		return nil, err
	}
	arguments := make([]runtime.Value, len(call.Arguments))
	for i, argument := range call.Arguments {
		arguments[i], err = evaluateExpression(argument, environment)
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
		scope := runtime.NewEnvironment(function.Env)
		for i, name := range function.Parameters {
			scope.Set(name, arguments[i])
		}
		return evaluateExpression(function.Body, scope)
	//case runtime.Builtin:
	default:
		return nil, fmt.Errorf("type error at %d:%d: %s is not callable",
			position.Line, position.Column, callee.Type())
	}
}

func evaluateIndex(expression *ast.IndexExpression, environment *runtime.Environment) (runtime.Value, error) {
	target, err := evaluateExpression(expression.Target, environment)
	if err != nil {
		return nil, err
	}
	index, err := evaluateExpression(expression.Index, environment)
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

func evaluateBinary(expression *ast.BinaryExpression, environment *runtime.Environment) (runtime.Value, error) {
	left, err := evaluateExpression(expression.Left, environment)
	if err != nil {
		return nil, err
	}
	right, err := evaluateExpression(expression.Right, environment)
	if err != nil {
		return nil, err
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
		case token.Equal:
			return runtime.Bool(leftNumber == rightNumber), nil
		}
	}
	if expression.Operator.Type == token.Equal {
		return runtime.Bool(left.Type() == right.Type() && left.Inspect() == right.Inspect()), nil
	}
	return nil, fmt.Errorf("type error at %d:%d: operator %s is not defined for %s and %s", expression.Operator.Position.Line, expression.Operator.Position.Column, expression.Operator.Lexeme, left.Type(), right.Type())
}
