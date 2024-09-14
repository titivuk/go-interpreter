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

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalProgram(node.Statements, env)
	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.StringLiteral:
		return &object.String{Value: node.Value}
	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)
	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}

		return evalPrefixExpression(node.Operator, right)
	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}

		return evalInfixExpression(left, node.Operator, right)
	case *ast.IfExpression:
		return evalIfExpression(node, env)
	case *ast.BlockStatement:
		return evalBlockStatement(node.Statements, env)
	case *ast.ReturnStatement:
		return evalReturnStatement(node, env)
	case *ast.LetStatement:
		// if we encounter let statement we need to track expression
		// for this purpose we use "env"
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}

		env.Set(node.Name.Value, val)
	case *ast.Identifier:
		// if we encounter identifier there should be associated value
		// we need to replace identifier with that value
		if val, ok := env.Get(node.Value); ok {
			return val
		}

		if builtin, ok := builtins[node.Value]; ok {
			return builtin
		}

		return newError("identifier not found: " + node.Value)
	case *ast.FunctionLiteral:
		return &object.Function{Parameters: node.Parameters, Body: node.Body, Env: env}
	case *ast.CallExpression:
		// eval always returns *object.Function
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return applyFunction(function, args)
	case *ast.ArrayLiteral:
		array := object.Array{}
		elements := evalExpressions(node.Elements, env)

		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}

		array.Elements = elements

		return &array
	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}

		return evalIndexExpression(left, index)
	case *ast.HashLiteral:
		hash := &object.Hash{
			Pairs: make(map[object.HashKey]object.HashPair),
		}

		for k, v := range node.Pairs {
			key := Eval(k, env)
			if isError(key) {
				return key
			}

			// cheks if key object implements Hashable interface
			hashKey, ok := key.(object.Hashable)
			if !ok {
				return newError("unusable as hash key: %s", key.Type())
			}

			value := Eval(v, env)
			if isError(value) {
				return value
			}

			hash.Pairs[hashKey.HashKey()] = object.HashPair{
				Key:   key,
				Value: value,
			}
		}

		return hash
	}

	return nil
}

func evalProgram(statements []ast.Statement, env *object.Environment) object.Object {
	var result object.Object

	for _, st := range statements {
		result = Eval(st, env)

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
	case right.Type() == object.STRING_OBJ && left.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(left, operator, right)
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

func evalStringInfixExpression(left object.Object, operator string, right object.Object) object.Object {
	leftValue := left.(*object.String).Value
	rightValue := right.(*object.String).Value

	switch operator {
	case token.PLUS:
		return &object.String{Value: leftValue + rightValue}
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

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	}

	if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	}

	return NULL
}

func evalReturnStatement(rs *ast.ReturnStatement, env *object.Environment) object.Object {
	value := Eval(rs.ReturnValue, env)
	if isError(value) {
		return value
	}

	return &object.ReturnValue{Value: value}
}

func evalBlockStatement(statements []ast.Statement, env *object.Environment) object.Object {
	var result object.Object

	for _, st := range statements {
		result = Eval(st, env)

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

func evalExpressions(
	exps []ast.Expression,
	env *object.Environment,
) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func applyFunction(fn object.Object, args []object.Object) object.Object {

	switch function := fn.(type) {
	case *object.Function:
		extendedEnv := extendFunctionEnv(function, args)
		evaluated := Eval(function.Body, extendedEnv)
		// we only want to stop the evaluation of the last called function’s body.
		// That's why we need unwrap it,
		// so that evalBlockStatement won’t stop evaluating statements in "outer" functions
		return unwrapReturnValue(evaluated)
	case *object.Builtin:
		return function.Fn(args...)
	default:
		return newError("not a function: %s", fn.Type())
	}
}

func evalIndexExpression(left object.Object, index object.Object) object.Object {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		arrayObj := left.(*object.Array)
		indexObj := index.(*object.Integer)

		if indexObj.Value < 0 || indexObj.Value >= int64(len(arrayObj.Elements)) {
			return NULL
		}

		return arrayObj.Elements[indexObj.Value]
	case left.Type() == object.HASH_OBJ:
		hashObj := left.(*object.Hash)

		key, ok := index.(object.Hashable)
		if !ok {
			return newError("unusable as hash key: %s", index.Type())
		}

		val, ok := hashObj.Pairs[key.HashKey()]
		if !ok {
			return NULL
		}

		return val.Value
	default:
		return newError("index operator not supported: %s", left.Type())
	}
}

func extendFunctionEnv(fn *object.Function, args []object.Object) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)
	for paramIdx, param := range fn.Parameters {
		env.Set(param.Value, args[paramIdx])
	}

	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	return obj
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
