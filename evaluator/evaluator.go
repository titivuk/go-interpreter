package evaluator

import (
	"github.com/titivuk/go-interpreter/ast"
	"github.com/titivuk/go-interpreter/object"
	"github.com/titivuk/go-interpreter/token"
)

// reuse some objects (similar to oddbals in v8 engine)
var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(node ast.Node) object.Object {
	switch node := node.(type) {
	case *ast.Program:
		return parseStatements(node.Statements)
	case *ast.ExpressionStatement:
		return Eval(node.Expression)
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)
	case *ast.PrefixExpression:
		right := Eval(node.Right)
		return evalPrefixExpression(node.Operator, right)
	default:
		return nil
	}
}

func parseStatements(statements []ast.Statement) object.Object {
	var result object.Object

	for _, st := range statements {
		result = Eval(st)
	}

	return result
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case token.BANG:
		switch right {
		case TRUE:
			return FALSE
		case FALSE:
			return TRUE
		case NULL:
			return FALSE
		default:
			return FALSE
		}
	case token.MINUS:
		if right.Type() != object.INTEGER_OBJ {
			return NULL
		}

		value := right.(*object.Integer).Value
		return &object.Integer{Value: -value}
	default:
		return NULL
	}
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}
