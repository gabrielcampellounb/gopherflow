package evaluator

import (
	"fmt"
	"math"
	"strconv"

	"github.com/gabrielcampellounb/gopherflow/internal/ast"
	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
	"github.com/gabrielcampellounb/gopherflow/internal/token"
)

type Environment struct {
	values map[string]runtime.Value
	parent *Environment
}

func NewEnvironment(parent *Environment) *Environment {
	return &Environment{values: make(map[string]runtime.Value), parent: parent}
}

func (e *Environment) Get(name string) (runtime.Value, bool) {
	value, ok := e.values[name]
	if ok {
		return value, true
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return nil, false
}

func (e *Environment) Set(name string, value runtime.Value) {
	e.values[name] = value
}

func Evaluate(program *ast.Program) (runtime.Value, error) {
	return EvaluateIn(program, NewEnvironment(nil))
}

func EvaluateIn(program *ast.Program, environment *Environment) (runtime.Value, error) {
	if len(program.Expressions) == 0 {
		return runtime.Null{}, nil
	}
	return evaluateExpression(program.Expressions[0], environment)
}

func evaluateExpression(expression ast.Expression, environment *Environment) (runtime.Value, error) {
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
		child := NewEnvironment(environment)
		child.Set(node.Name.Name, value)
		return evaluateExpression(node.Body, child)
	case *ast.IfExpression:
		condition, err := evaluateExpression(node.Condition, environment)
		if err != nil {
			return nil, err
		}
		boolean, ok := condition.(runtime.Bool)
		if !ok {
			return nil, fmt.Errorf("type error at %d:%d: if condition must be Bool, got %s", node.Pos.Line, node.Pos.Column, condition.Type())
		}
		if boolean {
			return evaluateExpression(node.Consequence, environment)
		}
		return evaluateExpression(node.Alternative, environment)
	case *ast.UnaryExpression:
		right, err := evaluateExpression(node.Right, environment)
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
		return evaluateBinary(node, environment)
	default:
		return nil, fmt.Errorf("unsupported expression %T", expression)
	}
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

func evaluateBinary(expression *ast.BinaryExpression, environment *Environment) (runtime.Value, error) {
	left, err := evaluateExpression(expression.Left, environment)
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
	right, err := evaluateExpression(expression.Right, environment)
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
