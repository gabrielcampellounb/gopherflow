package evaluator

import (
	"math"
	"slices"
	"strconv"

	"github.com/gabrielcampellounb/gopherflow/internal/ast"
	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
	"github.com/gabrielcampellounb/gopherflow/internal/token"
)

// Optimize rewrites a program to a fixed point without modifying the input AST.
// Unchanged nodes are shared with the input. Each rewrite removes syntax nodes
// or copies an ancestor of such a rewrite.
func Optimize(program *ast.Program) *ast.Program {
	for {
		next := *program
		var expressionsChanged bool
		next.Expressions, expressionsChanged = optimizeExpressions(program.Expressions)
		definitionsChanged := false
		for i, definition := range program.Definitions {
			body := definition.OptimizedBody
			if body == nil {
				body = definition.Body
			}
			if optimized := optimizeExpression(body); optimized != body {
				if !definitionsChanged {
					next.Definitions = slices.Clone(program.Definitions)
					definitionsChanged = true
				}
				copy := *definition
				copy.OptimizedBody = optimized
				next.Definitions[i] = &copy
			}
		}
		if !definitionsChanged && !expressionsChanged {
			return program
		}
		program = &next
	}
}

func optimizeExpressions(expressions []ast.Expression) ([]ast.Expression, bool) {
	result := expressions
	changed := false
	for i, expression := range expressions {
		if next := optimizeExpression(expression); next != expression {
			if !changed {
				result = slices.Clone(expressions)
				changed = true
			}
			result[i] = next
		}
	}
	return result, changed
}

func optimizeExpression(expression ast.Expression) ast.Expression {
	switch node := expression.(type) {
	case *ast.UnaryExpression:
		next := *node
		next.Right = optimizeExpression(node.Right)
		if _, ok := next.Right.(*ast.Literal); ok {
			if folded := foldLiteral(&next); folded != nil {
				return folded
			}
		}
		if next.Right != node.Right {
			return &next
		}
	case *ast.BinaryExpression:
		if node.Operator.Type == token.Pipe {
			return optimizeExpression(pipelineCall(node))
		}
		next := *node
		next.Left = optimizeExpression(node.Left)
		left, leftIsLiteral := next.Left.(*ast.Literal)
		if leftIsLiteral && ((left.Kind == token.False && node.Operator.Type == token.And) ||
			(left.Kind == token.True && node.Operator.Type == token.Or)) {
			return foldLiteral(&next)
		}
		next.Right = optimizeExpression(node.Right)
		if _, ok := next.Right.(*ast.Literal); leftIsLiteral && ok {
			if folded := foldLiteral(&next); folded != nil {
				return folded
			}
		}
		if next.Left != node.Left || next.Right != node.Right {
			return &next
		}
	case *ast.IfExpression:
		next := *node
		next.Condition = optimizeExpression(node.Condition)
		if condition, ok := next.Condition.(*ast.Literal); ok {
			switch condition.Kind {
			case token.True:
				return optimizeExpression(node.Consequence)
			case token.False:
				return optimizeExpression(node.Alternative)
			}
		}
		next.Consequence = optimizeExpression(node.Consequence)
		next.Alternative = optimizeExpression(node.Alternative)
		if next.Condition != node.Condition || next.Consequence != node.Consequence || next.Alternative != node.Alternative {
			return &next
		}
	case *ast.LetExpression:
		next := *node
		next.Value = optimizeExpression(node.Value)
		next.Body = optimizeExpression(node.Body)
		if value, ok := next.Value.(*ast.Literal); ok {
			// A malformed literal in a hand-built AST still raises its error.
			if _, err := evaluateLiteral(value); err == nil {
				switch body := next.Body.(type) {
				case *ast.Literal:
					return body
				case *ast.Identifier:
					if body.Name == node.Name.Name {
						return value
					}
				}
			}
		}
		if next.Value != node.Value || next.Body != node.Body {
			return &next
		}
	case *ast.FunctionLiteral:
		body := node.OptimizedBody
		if body == nil {
			body = node.Body
		}
		if optimized := optimizeExpression(body); optimized != body {
			next := *node
			next.OptimizedBody = optimized
			return &next
		}
	case *ast.CallExpression:
		// Calls may have effects. Only rewrite their operands and preserve their order.
		next := *node
		next.Function = optimizeExpression(node.Function)
		var changed bool
		next.Arguments, changed = optimizeExpressions(node.Arguments)
		if changed || next.Function != node.Function {
			return &next
		}
	case *ast.ListLiteral:
		if elements, changed := optimizeExpressions(node.Elements); changed {
			next := *node
			next.Elements = elements
			return &next
		}
	case *ast.RecordLiteral:
		next := *node
		changed := false
		for i, field := range node.Fields {
			if value := optimizeExpression(field.Value); value != field.Value {
				if !changed {
					next.Fields = slices.Clone(node.Fields)
					changed = true
				}
				next.Fields[i].Value = value
			}
		}
		if changed {
			return &next
		}
	case *ast.FieldExpression:
		if target := optimizeExpression(node.Target); target != node.Target {
			next := *node
			next.Target = target
			return &next
		}
	case *ast.IndexExpression:
		next := *node
		next.Target = optimizeExpression(node.Target)
		next.Index = optimizeExpression(node.Index)
		if list, ok := next.Target.(*ast.ListLiteral); ok {
			if _, ok := next.Index.(*ast.Literal); ok && hasOnlyLiteralElements(list) {
				if folded := foldLiteral(&next); folded != nil {
					return folded
				}
			}
		}
		if next.Target != node.Target || next.Index != node.Index {
			return &next
		}
	}
	return expression
}

func hasOnlyLiteralElements(list *ast.ListLiteral) bool {
	for _, element := range list.Elements {
		if _, ok := element.(*ast.Literal); !ok {
			return false
		}
	}
	return true
}

// The caller proves evaluation only touches literals, including cases where
// the evaluator short-circuits before an arbitrary right operand.
func foldLiteral(expression ast.Expression) *ast.Literal {
	value, err := evaluateExpression(expression, nil)
	if err != nil {
		return nil
	}
	literal := &ast.Literal{Pos: expression.Position()}
	switch value := value.(type) {
	case runtime.Number:
		number := float64(value)
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return nil
		}
		literal.Kind = token.Number
		literal.Value = strconv.FormatFloat(number, 'f', -1, 64)
	case runtime.Bool:
		literal.Kind = token.False
		if value {
			literal.Kind = token.True
		}
		literal.Value = value.Inspect()
	case runtime.String:
		literal.Kind = token.String
		literal.Value = string(value)
	case runtime.Null:
		literal.Kind = token.Null
		literal.Value = "null"
	default:
		return nil
	}
	return literal
}
