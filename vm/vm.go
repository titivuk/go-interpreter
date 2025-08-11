package vm

import (
	"fmt"

	"github.com/titivuk/go-interpreter/code"
	"github.com/titivuk/go-interpreter/compiler"
	"github.com/titivuk/go-interpreter/object"
)

const StackSize = 2048

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
		case code.OpAdd:
			right := vm.pop()
			left := vm.pop()

			switch {
			case right.Type() != left.Type():
				return fmt.Errorf("type mismatch: %s %s", left.Type(), right.Type())
			case right.Type() == object.INTEGER_OBJ && left.Type() == object.INTEGER_OBJ:
				leftValue := left.(*object.Integer).Value
				rightValue := right.(*object.Integer).Value
				vm.push(&object.Integer{Value: leftValue + rightValue})
			default:
				return fmt.Errorf("unknown operator: %s %b %s", left.Type(), code.OpAdd, right.Type())
			}

		}
	}

	return nil
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

func (vm *VM) StackTop() object.Object {
	if vm.sp == 0 {
		return nil
	}
	return vm.stack[vm.sp-1]
}
