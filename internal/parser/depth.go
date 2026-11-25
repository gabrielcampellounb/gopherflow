package parser

import (
	"fmt"

	"github.com/gabrielcampellounb/gopherflow/internal/ast"
)

const (
	maxTreeDepth = 1024
	// Rendering a valid tree can add grouping parentheses on top of AST depth.
	maxParseDepth = 4 * maxTreeDepth
)

// checkTreeDepth also covers left-associative operators and postfix chains,
// which the parser constructs iteratively but later stages traverse recursively.
func checkTreeDepth(expression ast.Expression, depth int) error {
	if depth > maxTreeDepth {
		return &Error{
			Position: expression.Position(),
			Message:  fmt.Sprintf("maximum AST depth (%d) exceeded", maxTreeDepth),
		}
	}
	childDepth := depth + 1
	switch node := expression.(type) {
	case *ast.UnaryExpression:
		return checkTreeDepth(node.Right, childDepth)
	case *ast.BinaryExpression:
		if err := checkTreeDepth(node.Left, childDepth); err != nil {
			return err
		}
		return checkTreeDepth(node.Right, childDepth)
	case *ast.LetExpression:
		if err := checkTreeDepth(node.Value, childDepth); err != nil {
			return err
		}
		return checkTreeDepth(node.Body, childDepth)
	case *ast.IfExpression:
		if err := checkTreeDepth(node.Condition, childDepth); err != nil {
			return err
		}
		if err := checkTreeDepth(node.Consequence, childDepth); err != nil {
			return err
		}
		return checkTreeDepth(node.Alternative, childDepth)
	case *ast.FunctionLiteral:
		return checkTreeDepth(node.Body, childDepth)
	case *ast.CallExpression:
		if err := checkTreeDepth(node.Function, childDepth); err != nil {
			return err
		}
		for _, argument := range node.Arguments {
			if err := checkTreeDepth(argument, childDepth); err != nil {
				return err
			}
		}
	case *ast.IndexExpression:
		if err := checkTreeDepth(node.Target, childDepth); err != nil {
			return err
		}
		return checkTreeDepth(node.Index, childDepth)
	case *ast.FieldExpression:
		return checkTreeDepth(node.Target, childDepth)
	case *ast.ListLiteral:
		for _, element := range node.Elements {
			if err := checkTreeDepth(element, childDepth); err != nil {
				return err
			}
		}
	case *ast.RecordLiteral:
		for _, field := range node.Fields {
			if err := checkTreeDepth(field.Value, childDepth); err != nil {
				return err
			}
		}
	}
	return nil
}
