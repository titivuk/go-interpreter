package vm

import (
	"fmt"

	"github.com/titivuk/go-interpreter/code"
	"github.com/titivuk/go-interpreter/compiler"
	"github.com/titivuk/go-interpreter/object"
)

const StackSize = 2048
const GlobalsSize = 65536
const MaxFrames = 1024

var (
	True  = &object.Boolean{Value: true}
	False = &object.Boolean{Value: false}
	Null  = &object.Null{}
)

type VM struct {
	constants    []object.Object
	globals      []object.Object
	stack        []object.Object
	sp           int // always points to the next value. Top of the stack is stack[sp-1]
	frames       []*Frame
	framesIndex  int // always points to the next value. Top of the stack is stack[sp-1]
}

func New(bytecode *compiler.Bytecode) *VM {
	mainFn := &object.CompiledFunction{Instructions: bytecode.Instructions}
	frames := make([]*Frame, MaxFrames)
	frames[0] = NewFrame(mainFn)

	return &VM{
		constants:    bytecode.Constants,
		globals:      make([]object.Object, GlobalsSize),
		stack:        make([]object.Object, StackSize),
		sp:           0,
		frames:       frames,
		framesIndex:  1,
	}
}

func NewWithGlobalsStore(bytecode *compiler.Bytecode, globals []object.Object) *VM {
	vm := New(bytecode)
	vm.globals = globals
	return vm
}

func (vm *VM) Run() error {
	var ip int
	var ins code.Instructions
	var op code.Opcode

	for vm.currentFrame().ip < len(vm.currentFrame().Instructions())-1 {
		vm.currentFrame().ip++

		ip = vm.currentFrame().ip
		ins = vm.currentFrame().Instructions()
		op = code.Opcode(ins[ip])

		switch op {
		case code.OpConstant:
			constPos := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2

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
		case code.OpJump:
			// read jump position
			pos := int(code.ReadUint16(ins[ip+1:]))
			// -1 because the loop will increment 'ip' before the next iteration
			vm.currentFrame().ip = pos - 1
		case code.OpJumpNotTruthy:
			// read jump position
			pos := int(code.ReadUint16(ins[ip+1:]))
			// move 'ip' to the pos before the next opcode
			// the loop will advance 'ip' to the next opcode
			vm.currentFrame().ip += 2 // OpJumpNotTruthy has 2 bytes operands (uint16)

			condition := vm.pop()
			// if condition is not truthy, we do not need to execute code inside
			if !isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			}
		case code.OpNull:
			err := vm.push(Null)
			if err != nil {
				return err
			}
		case code.OpSetGlobal:
			idx := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2
			vm.globals[idx] = vm.pop()
		case code.OpGetGlobal:
			idx := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			err := vm.push(vm.globals[idx])
			if err != nil {
				return err
			}
		case code.OpArray:
			len := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			elements := make([]object.Object, len)
			for i := 0; i < len; i++ {
				elements[len-1-i] = vm.pop()
			}
			arrayObj := &object.Array{Elements: elements}

			err := vm.push(arrayObj)
			if err != nil {
				return err
			}
		case code.OpHash:
			len := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			pairs := make(map[object.HashKey]object.HashPair)
			for i := 0; i < len/2; i++ {
				value := vm.pop()
				key := vm.pop()

				objPair := object.HashPair{Key: key, Value: value}

				hashKey, ok := key.(object.Hashable)
				if !ok {
					return fmt.Errorf("unusable as hash key: %s", key.Type())
				}

				pairs[hashKey.HashKey()] = objPair
			}

			hashObj := &object.Hash{Pairs: pairs}

			err := vm.push(hashObj)
			if err != nil {
				return err
			}
		case code.OpIndex:
			index := vm.pop()
			left := vm.pop()

			err := vm.executeIndexExpression(left, index)
			if err != nil {
				return err
			}
		case code.OpCall:
			fn, ok := vm.stack[vm.sp-1].(*object.CompiledFunction)
			if !ok {
				return fmt.Errorf("calling non-function")
			}

			frame := NewFrame(fn)
			vm.pushFrame(frame)
		case code.OpReturnValue:
			returnValue := vm.pop()

			vm.popFrame()
			vm.pop()

			err := vm.push(returnValue)
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
		return vm.executeBinaryIntegerOperation(op, left, right)
	case right.Type() == object.BOOLEAN_OBJ && left.Type() == object.BOOLEAN_OBJ:
		return vm.executeBinaryBooleanOperation(op, left, right)
	case right.Type() == object.STRING_OBJ && left.Type() == object.STRING_OBJ:
		return vm.executeBinaryStringOperation(op, left, right)
	default:
		return fmt.Errorf("unknown operator: %s %b %s", left.Type(), code.OpAdd, right.Type())
	}
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

func (vm *VM) executeBinaryStringOperation(op code.Opcode, left, right object.Object) error {
	leftValue := left.(*object.String).Value
	rightValue := right.(*object.String).Value

	switch op {
	case code.OpAdd:
		vm.push(&object.String{Value: leftValue + rightValue})
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
	default:
		return fmt.Errorf("unknown string operator: %d", op)
	}

	return nil
}

func (vm *VM) executeIndexExpression(left, index object.Object) error {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.executeArrayIndex(left, index)
	case left.Type() == object.HASH_OBJ:
		return vm.executeHashIndex(left, index)
	default:
		return fmt.Errorf("index operator not supported: %s", left.Type())
	}
}

func (vm *VM) executeBangOperator() error {
	operand := vm.pop()
	switch operand {
	case True:
		return vm.push(False)
	case False:
		return vm.push(True)
	case Null:
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

func (vm *VM) executeArrayIndex(array, index object.Object) error {
	arrayObject := array.(*object.Array)
	i := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)
	if i < 0 || i > max {
		return vm.push(Null)
	}
	return vm.push(arrayObject.Elements[i])
}

func (vm *VM) executeHashIndex(hash, index object.Object) error {
	hashObject := hash.(*object.Hash)
	key, ok := index.(object.Hashable)
	if !ok {
		return fmt.Errorf("unusable as hash key: %s", index.Type())
	}
	pair, ok := hashObject.Pairs[key.HashKey()]
	if !ok {
		return vm.push(Null)
	}
	return vm.push(pair.Value)
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

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIndex-1]
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
}

func (vm *VM) popFrame() *Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
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

func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {
	case *object.Boolean:
		return obj.Value
	case *object.Null:
		return false
	default:
		return true
	}
}
