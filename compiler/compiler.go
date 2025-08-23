package compiler

import (
	"fmt"

	"github.com/titivuk/go-interpreter/ast"
	"github.com/titivuk/go-interpreter/code"
	"github.com/titivuk/go-interpreter/object"
)

type Compiler struct {
	instructions        code.Instructions
	constants           []object.Object
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
}

func New() *Compiler {
	return &Compiler{
		instructions:        code.Instructions{},
		constants:           []object.Object{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}
}

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}
	case *ast.ExpressionStatement:
		err := c.Compile(node.Expression)
		if err != nil {
			return err
		}

		c.emit(code.OpPop)
	case *ast.InfixExpression:
		// example how we can use 'OpGreaterThan' for both '>' and '<'
		// buy just changing left and right in case of opposite operator
		//
		// looks uggly as fuck sine we have to add if and return early
		// but for the sake of example how we can change code during compilation
		if node.Operator == "<" {
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}

			err = c.Compile(node.Left)
			if err != nil {
				return err
			}

			c.emit(code.OpGreaterThan)

			return nil
		}

		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		case "-":
			c.emit(code.OpSub)
		case "*":
			c.emit(code.OpMul)
		case "/":
			c.emit(code.OpDiv)
		case "!=":
			c.emit(code.OpNotEqual)
		case "==":
			c.emit(code.OpEqual)
		case ">":
			c.emit(code.OpGreaterThan)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}
	case *ast.PrefixExpression:
		err := c.Compile(node.Right)
		if err != nil {
			return nil
		}

		switch node.Operator {
		case "-":
			c.emit(code.OpMinus)
		case "!":
			c.emit(code.OpBang)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}
	case *ast.IfExpression:
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		// our compiler traverses the AST only once and called 'single-pass' compiler
		// we apply 'back-patching' - we set placeholder jump opcode with garbage value
		// and, once consequence is compiled and we know at what position we need to jump
		// we do 'back-patcing' and update instruction operand value
		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}

		// we do want the consequence and the alternative of a conditional to leave a value on the stack
		// to make the code below work
		// let result = if (5 > 3) { 5 } else { 3 };
		// The value produced by the consequence would be popped off the stack,
		// the expression wouldn’t evaluate to anything, and the let statement would end up
		// without a value on the right side of its =
		if c.lastInstruction.Opcode == code.OpPop {
			c.removeLastInstruction()
		}

		// update 'code.OpJumpNotTruthy' instructions with correct jump offset
		jumpPos := c.emit(code.OpJump, 9999)

		afterConsequencePos := len(c.instructions)
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

		// alternative
		// the steps are exactly the same as for consequence flattening
		if node.Alternative == nil {
			// expression that produces nothing returns null
			// so if there is not alternative we emit null
			c.emit(code.OpNull)
		} else {
			// compile alternative block
			err = c.Compile(node.Alternative)
			if err != nil {
				return err
			}

			// pop lastInstruction if its OpPop
			if c.lastInstruction.Opcode == code.OpPop {
				c.removeLastInstruction()
			}
		}

		// change 'code.OpJump' position with correct jump offset
		afterAlternativePos := len(c.instructions)
		c.changeOperand(jumpPos, afterAlternativePos)

	case *ast.BlockStatement:
		for _, exp := range node.Statements {
			err := c.Compile(exp)
			if err != nil {
				return err
			}
		}
	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer))
	case *ast.Boolean:
		boolean := &object.Boolean{Value: node.Value}

		if boolean.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}
	}

	return nil
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	ins := code.Make(op, operands...)
	pos := c.addInstruction(ins)

	c.setLastInstruction(op, pos)

	return pos
}

func (c *Compiler) addInstruction(ins code.Instructions) int {
	// instruction can be several bytes length so we remember its starting point
	// i.e. insPos != len(c.instructions) - 1  after instruction added
	insPos := len(c.instructions)
	c.instructions = append(c.instructions, ins...)
	return insPos
}

func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	c.previousInstruction = c.lastInstruction
	c.lastInstruction = EmittedInstruction{
		Opcode:   op,
		Position: pos,
	}
}

func (c *Compiler) removeLastInstruction() {
	c.instructions = c.instructions[:c.lastInstruction.Position]
	c.lastInstruction = c.previousInstruction
}

func (c *Compiler) changeOperand(opPos int, operand int) {
	op := code.Make(code.Opcode(c.instructions[opPos]), operand)
	c.replaceInstruction(opPos, op)
}

func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	for i := 0; i < len(newInstruction); i++ {
		c.instructions[pos+i] = newInstruction[i]
	}
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
	}
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}

type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}
