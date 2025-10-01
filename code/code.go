package code

import (
	"bytes"
	"fmt"
	"strconv"
)

const (
	OpConstant Opcode = iota
	OpAdd
	OpPop
	OpSub
	OpMul
	OpDiv
	// load them as a constant everytime they are used is a waste
	// so we introduce opcodes for them
	OpTrue
	OpFalse
	OpEqual
	OpNotEqual
	OpGreaterThan
	OpMinus
	OpBang
	OpJump
	OpJumpNotTruthy
	OpNull
	OpSetGlobal
	OpGetGlobal
	OpArray
	OpHash
	OpIndex
	OpCall
	OpReturnValue // return from the current function value sitting on top of the stack
	OpReturn      // return from the current function, but there is nothing to return
)

type Opcode byte

type Instructions []byte

func (ins Instructions) String() string {
	var out bytes.Buffer

	i := 0
	for i < len(ins) {
		opcode := ins[i]
		def, err := Lookup(opcode)
		if err != nil {
			fmt.Fprintf(&out, "ERROR: %s\n", err)
			continue
		}

		operands, offset := ReadOperands(def, ins[i+1:])

		fmt.Fprintf(&out, "%04d %s\n", i, ins.fmtInstruction(def, operands))

		i += offset + 1
	}

	return out.String()
}

func (ins Instructions) fmtInstruction(def *Definition, operands []int) string {
	operandCount := len(def.OperandWidths)
	if len(operands) != operandCount {
		return fmt.Sprintf("ERROR: operand len %d does not match defined %d\n",
			len(operands), operandCount)
	}

	var out bytes.Buffer

	out.WriteString(def.Name)
	for _, o := range operands {
		out.WriteByte(' ')
		out.WriteString(strconv.FormatInt(int64(o), 10))
	}

	return out.String()
}

type Definition struct {
	Name string
	// contains number of bytes each operand takes
	OperandWidths []int
}

var definitions = map[Opcode]*Definition{
	OpConstant:      {"OpConstant", []int{2}}, // single operand 2 bytes width => max value uint16
	OpAdd:           {"OpAdd", []int{}},
	OpSub:           {"OpSub", []int{}},
	OpMul:           {"OpMul", []int{}},
	OpDiv:           {"OpDiv", []int{}},
	OpPop:           {"OpPop", []int{}},
	OpTrue:          {"OpTrue", []int{}},
	OpFalse:         {"OpFalse", []int{}},
	OpEqual:         {"OpEqual", []int{}},
	OpNotEqual:      {"OpNotEqual", []int{}},
	OpGreaterThan:   {"OpGreaterThan", []int{}},
	OpMinus:         {"OpMinus", []int{}},
	OpBang:          {"OpBang", []int{}},
	OpJump:          {"OpJump", []int{2}},
	OpJumpNotTruthy: {"OpJumpNotTruthy", []int{2}},
	OpNull:          {"OpNull", []int{}},
	OpSetGlobal:     {"OpSetGlobal", []int{2}},
	OpGetGlobal:     {"OpGetGlobal", []int{2}},
	OpArray:         {"OpArray", []int{2}}, // array with 65k elements
	OpHash:          {"OpHash", []int{2}},  // hash with (65k/2) elements
	OpIndex:         {"OpIndex", []int{}},
	OpCall:          {"OpCall", []int{}},
	OpReturnValue:   {"OpReturnValue", []int{}},
	OpReturn:        {"OpReturn", []int{}},
}

func Lookup(op byte) (*Definition, error) {
	def, ok := definitions[Opcode(op)]
	if !ok {
		return nil, fmt.Errorf("opcode %d undefined", op)
	}

	return def, nil
}

func Make(op Opcode, operands ...int) []byte {
	def, ok := definitions[op]
	if !ok {
		return []byte{}
	}

	// 1 for opCode
	instructionLen := 1
	// the rest for operands
	for _, w := range def.OperandWidths {
		instructionLen += w
	}

	instruction := make([]byte, instructionLen)
	instruction[0] = byte(op)

	offset := 1
	for i, o := range operands {
		width := def.OperandWidths[i]
		// binary.BigEndian.PutUint16(...) does the conversion
		// but I want to get better at bit manipulation so doing it manually
		switch width {
		// 2 bytes operand encoding
		case 2:
			instruction[offset] = byte(uint16(o) >> 8)
			offset += 1
			instruction[offset] = byte(uint16(o))
			offset += 1
		}
	}

	return instruction
}

func ReadOperands(def *Definition, ins Instructions) ([]int, int) {
	offset := 0
	operands := make([]int, len(def.OperandWidths))
	for i, width := range def.OperandWidths {
		switch width {
		case 2:
			operands[i] = int(ReadUint16(ins[offset:]))
		}
		offset += width
	}

	return operands, offset
}

func ReadUint16(ins Instructions) uint16 {
	// binary.BigEndian.Uint16(...) does the conversion
	// but I want to get better at bit manipulation so doing it manually
	return uint16(ins[1]) | uint16((ins[0]))<<8
}
