package vm

import (
	"fmt"

	"github.com/titivuk/go-interpreter/code"
	"github.com/titivuk/go-interpreter/compiler"
	"github.com/titivuk/go-interpreter/object"
)

const StackSize = 2048

var (
	True  = &object.Boolean{Value: true}
	False = &object.Boolean{Value: false}
)

type VM struct {
	constants    []object.Object
	instructions code.Instructions

	stack []object.Object
	sp    int // always points to the next value. Top of the stack is stack[sp-1]
}

func New(bytecode *compiler.Bytecode) *VM {
	return &VM{
		constants:    bytecode.Constants,
		instructions: bytecode.Instructions,

		stack: make([]object.Object, StackSize),
		sp:    0,
	}
}

func (vm *VM) Run() error {
	for ip := 0; ip < len(vm.instructions); ip += 1 {
		op := code.Opcode(vm.instructions[ip])

		switch op {
		case code.OpConstant:
			constPos := code.ReadUint16(vm.instructions[ip+1:])
			ip += 2

			err := vm.push(vm.constants[constPos])
			if err != nil {
				return err
			}
		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpEqual, code.OpGreaterThan, code.OpNotEqual:
			err := vm.executeBinaryOperation(op)
			if err != nil {
				return err
			}
		case code.OpPop:
			vm.pop()
		case code.OpTrue:
			err := vm.push(True)
			if err != nil {
				return err
			}
		case code.OpFalse:
			err := vm.push(False)
			if err != nil {
				return err
			}
		case code.OpMinus:
			err := vm.executeMinusOperator()
			if err != nil {
				return err
			}
		case code.OpBang:
			err := vm.executeBangOperator()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (vm *VM) executeBinaryOperation(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	switch {
	case right.Type() != left.Type():
		return fmt.Errorf("type mismatch: %s %s", left.Type(), right.Type())
	case right.Type() == object.INTEGER_OBJ && left.Type() == object.INTEGER_OBJ:
		err := vm.executeBinaryIntegerOperation(op, left, right)
		if err != nil {
			return err
		}
	case right.Type() == object.BOOLEAN_OBJ && left.Type() == object.BOOLEAN_OBJ:
		err := vm.executeBinaryBooleanOperation(op, left, right)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown operator: %s %b %s", left.Type(), code.OpAdd, right.Type())
	}

	return nil
}

func (vm *VM) executeBinaryIntegerOperation(op code.Opcode, left, right object.Object) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	switch op {
	case code.OpAdd:
		vm.push(&object.Integer{Value: leftValue + rightValue})
	case code.OpSub:
		vm.push(&object.Integer{Value: leftValue - rightValue})
	case code.OpMul:
		vm.push(&object.Integer{Value: leftValue * rightValue})
	case code.OpDiv:
		vm.push(&object.Integer{Value: leftValue / rightValue})
	case code.OpEqual:
		if leftValue == rightValue {
			vm.push(True)
		} else {
			vm.push(False)
		}
	case code.OpNotEqual:
		if leftValue != rightValue {
			vm.push(True)
		} else {
			vm.push(False)
		}
	case code.OpGreaterThan:
		if leftValue > rightValue {
			vm.push(True)
		} else {
			vm.push(False)
		}
	default:
		return fmt.Errorf("unknown integer operator: %d", op)
	}

	return nil
}

func (vm *VM) executeBinaryBooleanOperation(op code.Opcode, left, right object.Object) error {
	leftValue := left.(*object.Boolean).Value
	rightValue := right.(*object.Boolean).Value

	switch op {
	case code.OpEqual:
		vm.push(nativeBoolToBooleanObject(leftValue == rightValue))
	case code.OpNotEqual:
		vm.push(nativeBoolToBooleanObject(leftValue != rightValue))
	default:
		return fmt.Errorf("unknown boolean operator: %d", op)
	}

	return nil
}

func (vm *VM) executeBangOperator() error {
	operand := vm.pop()
	switch operand {
	case True:
		return vm.push(False)
	case False:
		return vm.push(True)
	default:
		return vm.push(False)
	}
}

func (vm *VM) executeMinusOperator() error {
	operand := vm.pop()
	if operand.Type() != object.INTEGER_OBJ {
		return fmt.Errorf("unsupported type for negation: %s", operand.Type())
	}

	value := operand.(*object.Integer).Value
	return vm.push(&object.Integer{Value: -value})
}

func (vm *VM) push(obj object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow")
	}

	vm.stack[vm.sp] = obj
	vm.sp += 1

	return nil
}

func (vm *VM) pop() object.Object {
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp]
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return True
	}
	return False
}
