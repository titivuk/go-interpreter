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
	case *ast.InfixExpression:
		left := Eval(node.Left)
		right := Eval(node.Right)
		return evalInfixExpression(left, node.Operator, right)
	default:
		return NULL
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

func evalInfixExpression(left object.Object, operator string, right object.Object) object.Object {
	switch {
	case right.Type() == object.INTEGER_OBJ && left.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(left, operator, right)
	case right.Type() == object.BOOLEAN_OBJ && left.Type() == object.BOOLEAN_OBJ:
		return evalBooleanInfixExpression(left, operator, right)
	default:
		return NULL
	}
}

func evalIntegerInfixExpression(left object.Object, operator string, right object.Object) object.Object {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	switch operator {
	case token.PLUS:
		return &object.Integer{Value: leftValue + rightValue}
	case token.MINUS:
		return &object.Integer{Value: leftValue - rightValue}
	case token.ASTERISK:
		return &object.Integer{Value: leftValue * rightValue}
	case token.SLASH:
		return &object.Integer{Value: leftValue / rightValue}
	case token.LT:
		return nativeBoolToBooleanObject(leftValue < rightValue)
	case token.GT:
		return nativeBoolToBooleanObject(leftValue > rightValue)
	case token.EQ:
		return nativeBoolToBooleanObject(leftValue == rightValue)
	case token.NOT_EQ:
		return nativeBoolToBooleanObject(leftValue != rightValue)
	default:
		return NULL
	}
}

func evalBooleanInfixExpression(left object.Object, operator string, right object.Object) object.Object {
	// we can use pointer comparison because we always use TRUE and FALSE objects
	switch operator {
	case token.EQ:
		return nativeBoolToBooleanObject(left == right)
	case token.NOT_EQ:
		return nativeBoolToBooleanObject(left != right)
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
