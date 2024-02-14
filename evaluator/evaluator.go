package evaluator

import (
	"fmt"

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
		return evalProgram(node.Statements)
	case *ast.ExpressionStatement:
		return Eval(node.Expression)
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)
	case *ast.PrefixExpression:
		right := Eval(node.Right)
		if isError(right) {
			return right
		}

		return evalPrefixExpression(node.Operator, right)
	case *ast.InfixExpression:
		left := Eval(node.Left)
		if isError(left) {
			return left
		}

		right := Eval(node.Right)
		if isError(right) {
			return right
		}

		return evalInfixExpression(left, node.Operator, right)
	case *ast.IfExpression:
		return evalIfExpression(node)
	case *ast.BlockStatement:
		return evalBlockStatement(node.Statements)
	case *ast.ReturnStatement:
		return evalReturnStatement(node)
	default:
		return NULL
	}
}

func evalProgram(statements []ast.Statement) object.Object {
	var result object.Object

	for _, st := range statements {
		result = Eval(st)

		switch result := result.(type) {
		// if we encounter return statements or errors
		// all statements after are unreachable
		// so we stop evaluation
		// and return Value of return statement or error
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
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
			return newError("unknown operator: -%s", right.Type())
		}

		value := right.(*object.Integer).Value
		return &object.Integer{Value: -value}
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalInfixExpression(left object.Object, operator string, right object.Object) object.Object {
	switch {
	case right.Type() != left.Type():
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	case right.Type() == object.INTEGER_OBJ && left.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(left, operator, right)
	case right.Type() == object.BOOLEAN_OBJ && left.Type() == object.BOOLEAN_OBJ:
		return evalBooleanInfixExpression(left, operator, right)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
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
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
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
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIfExpression(ie *ast.IfExpression) object.Object {
	condition := Eval(ie.Condition)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence)
	}

	if ie.Alternative != nil {
		return Eval(ie.Alternative)
	}

	return NULL
}

func evalReturnStatement(rs *ast.ReturnStatement) object.Object {
	value := Eval(rs.ReturnValue)
	if isError(value) {
		return value
	}

	return &object.ReturnValue{Value: value}
}

func evalBlockStatement(statements []ast.Statement) object.Object {
	var result object.Object

	for _, st := range statements {
		result = Eval(st)

		// Here we explicitly don’t unwrap the return value and only check the Type() of each evaluation result.
		// If it’s object.RETURN_VALUE_OBJECT we simply return the *object.ReturnValue,
		// without unwrapping its .Value, so it stops execution in a possible outer block statement
		// and bubbles up to evalProgram, where it finally get’s unwrapped
		// Example:
		// if (10 > 1) {
		// 	if (10 > 1) {
		// 		return 10;
		// 	}
		// 	return 1;
		// }
		// we do not unwrap "return 10;" and pass it to outer block
		// then, evalProgram checks if it's object.ReturnValue, unwraps it
		// and stops the execution, i.e. "return 1" is not executed
		//
		// the same applies to errors
		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case NULL:
		return false
	case FALSE:
		return false
	case TRUE:
		return true
	default:
		return true
	}
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}
