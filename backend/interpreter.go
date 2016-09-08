package backend

import (
	"fmt"
	"math"
)

// Execute is a simple wrapper around the `Interpreter` creation and execution
func Execute(mainFunc *FuncPrototype, funcs []*FuncPrototype) {
	inter := NewInterpreter(mainFunc, funcs)
	inter.Execute()
}

// Interpreter represents the state of a "virtual machine" running a program.
// This includes special registers for the instruction pointer (`ip`) and frame
// pointer (`fp`). The Interpreter also contains a stack of all current execution
// frames with the active frame at the top of the stack. The `funcs` fields
// contains all `FuncPrototype`s used by the program. These prototypes are used
// used to generate `Closure`s when a function is called during execution
type Interpreter struct {
	ip        BytecodeAddress
	fp        *StackFrame
	callStack []*StackFrame
	funcs     []*FuncPrototype
}

// TODO document `NewInterpreter`
func NewInterpreter(mainFunc *FuncPrototype, funcs []*FuncPrototype) *Interpreter {
	inter := &Interpreter{}
	mainStackFrame := &StackFrame{
		Closure:   NewClosure(inter.callStack, mainFunc),
		Registers: make([]*Register, 256),
	}

	inter.callStack = []*StackFrame{mainStackFrame}
	inter.fp = mainStackFrame
	inter.funcs = funcs

	return inter
}

// TODO document `Interpreter.Execute`
func (inter *Interpreter) Execute() {
	for {
		switch opcode := inter.readOpcode(); opcode {
		case OpcodeHalt:
			return
		case OpcodeBoolConst:
			value := (inter.readInt32() == 1)
			dest := inter.readRegister()
			inter.fp.Registers[dest] = &Register{Value: value}
		case OpcodeIntConst:
			value := inter.readInt32()
			dest := inter.readRegister()
			inter.fp.Registers[dest] = &Register{Value: value}
		case OpcodeDecConst:
			value := inter.readFloat32()
			dest := inter.readRegister()
			inter.fp.Registers[dest] = &Register{Value: value}
		case OpcodeStrConst:
			index := inter.readUint32()
			dest := inter.readRegister()
			value := inter.fp.Closure.Prototype.Constants[index]
			inter.fp.Registers[dest] = &Register{Value: value}
		case OpcodeFuncConst:
			id := inter.readUint32()
			dest := inter.readRegister()
			closure := NewClosure(inter.callStack, inter.funcs[id])
			inter.fp.Registers[dest] = &Register{Value: closure}
		case OpcodeMove:
			source := inter.readRegister()
			dest := inter.readRegister()
			inter.fp.Registers[dest] = &Register{Value: inter.fp.Registers[source].Value}
		case OpcodeLoadUpVal:
			index := inter.readInt32()
			dest := inter.readRegister()
			inter.fp.Registers[dest] = &Register{Value: inter.fp.Closure.Upvalues[index].Cell.Value}
		case OpcodeSetUpVal:
			source := inter.readRegister()
			index := inter.readInt32()
			inter.fp.Closure.Upvalues[index].Cell.Value = inter.fp.Registers[source].Value
		case OpcodeBrAlways:
			addr := inter.readBytecodeAddress()
			inter.ip = addr
		case OpcodeBrTrue:
			fallthrough
		case OpcodeBrFalse:
			testReg := inter.readRegister()
			addr := inter.readBytecodeAddress()
			testArg := inter.fp.Registers[testReg]

			var testValue, ok bool

			if testValue, ok = testArg.Value.(bool); ok == false {
				panic(fmt.Sprintf("expected `bool`, found `%T`", testArg))
			}

			switch opcode {
			case OpcodeBrTrue:
				if testValue {
					inter.ip = addr
				}
			case OpcodeBrFalse:
				if testValue == false {
					inter.ip = addr
				}
			}
		case OpcodeDispatch:
			closure := inter.fp.Registers[inter.readRegister()].Value.(*Closure)
			firstParam := inter.readRegister()

			// Create a new stack frame using the closure stored in the first
			// instruction argument, this frame will get pushed to the callstack
			frame := &StackFrame{
				Closure:   closure,
				Registers: make([]*Register, 256),
			}

			// Save the current instruction pointer to the stack frame so
			// that when the function returns and this stack frame is
			// popped, the Interpreter can resume execution at whatever
			// instruction follows this "Dispatch" instruction
			inter.fp.ReturnToAddress = inter.ip

			// Copy dispatch arguments to the new frame's registers
			totalReturns := 1
			for argIndex := 0; argIndex < len(closure.Prototype.Locals); argIndex++ {
				argReg := inter.fp.Registers[int(firstParam)+argIndex]
				frame.Registers[totalReturns+argIndex] = argReg
			}

			// Push the new stack frame onto the call stack
			inter.callStack = append(inter.callStack, frame)

			// Update the frame pointer to the newly created stack frame
			inter.fp = frame

			// Reset the instruction pointer so it begins executing at the start
			// of the new closure's bytecode
			inter.ip = 0
		case OpcodeReturn:
			// Save pointers to both the top frame (about to be popped) and the
			// lower frame (about to resume execution control)
			topFrame := inter.fp
			lowerFrame := inter.callStack[len(inter.callStack)-2]

			// If the return statement is passed an argument, move that
			// argument's value into the return register, r0
			if sourceReg := inter.readRegister(); sourceReg > 0 {
				topFrame.Registers[0] = topFrame.Registers[sourceReg]
			}

			// Pass any return value from the top frame's return register and
			// store it in the lower frame's return register
			lowerFrame.Registers[0] = topFrame.Registers[0]

			// Pop top stack frame from call stack
			inter.callStack = inter.callStack[:len(inter.callStack)-1]

			// Set lower frame as the current stack frame
			inter.fp = lowerFrame

			// Reset the instruction poiner to what it was before the dispatch
			inter.ip = lowerFrame.ReturnToAddress
		case OpcodeIntLT:
			fallthrough
		case OpcodeIntLTEq:
			fallthrough
		case OpcodeIntGT:
			fallthrough
		case OpcodeIntGTEq:
			fallthrough
		case OpcodeIntEq:
			fallthrough
		case OpcodeIntAdd:
			fallthrough
		case OpcodeIntSub:
			fallthrough
		case OpcodeIntMul:
			fallthrough
		case OpcodeIntDiv:
			leftReg := inter.readRegister()
			rightReg := inter.readRegister()
			leftArg := inter.fp.Registers[leftReg]
			rightArg := inter.fp.Registers[rightReg]

			// Register in which to store the product
			dest := inter.readRegister()

			var leftValue, rightValue int32
			var ok bool

			if leftArg == nil {
				panic("expected `int32`, found <nil>")
			}

			if leftValue, ok = leftArg.Value.(int32); ok == false {
				panic(fmt.Sprintf("expected `int32`, found `%T`", leftArg))
			}

			if rightArg == nil {
				panic("expected `int32`, found <nil>")
			}

			if rightValue, ok = rightArg.Value.(int32); ok == false {
				panic(fmt.Sprintf("expected `int32`, found `%T`", rightArg))
			}

			if inter.fp.Registers[dest] == nil {
				inter.fp.Registers[dest] = &Register{}
			}

			// Actual math done here, once arguments have been cast
			var result interface{}

			switch opcode {
			case OpcodeIntLT:
				// Compute the result
				result = leftValue < rightValue
			case OpcodeIntLTEq:
				// Compute the result
				result = leftValue <= rightValue
			case OpcodeIntGT:
				// Compute the result
				result = leftValue > rightValue
			case OpcodeIntGTEq:
				// Compute the result
				result = leftValue >= rightValue
			case OpcodeIntEq:
				// Compute the result
				result = leftValue == rightValue
			case OpcodeIntAdd:
				// Compute the result
				result = leftValue + rightValue
			case OpcodeIntSub:
				// Compute the result
				result = leftValue - rightValue
			case OpcodeIntMul:
				// Compute the result
				result = leftValue * rightValue
			case OpcodeIntDiv:
				// Compute the result
				leftFloat32 := float32(leftValue)
				rightFloat32 := float32(rightValue)
				result = leftFloat32 / rightFloat32
			}

			// Store the value in the appropriate register
			inter.fp.Registers[dest].Value = result
		case OpcodeIntNeg:
			operandReg := inter.readRegister()
			operand := inter.fp.Registers[operandReg]

			// Register in which to store the product
			dest := inter.readRegister()

			var operandValue int32
			var ok bool

			if operand == nil {
				panic("expected `int32`, found <nil>")
			}

			if operandValue, ok = operand.Value.(int32); ok == false {
				panic(fmt.Sprintf("expected `int32`, found `%T`", operand))
			}

			// Populate the register slot with an empty Register struct if the
			// slot is only `nil`
			if inter.fp.Registers[dest] == nil {
				inter.fp.Registers[dest] = &Register{}
			}

			// Actual math done here, once arguments have been cast
			var result interface{}

			switch opcode {
			case OpcodeIntNeg:
				// Compute the result
				result = -1 * operandValue
			}

			// Store the value in the appropriate register
			inter.fp.Registers[dest].Value = result
		case OpcodeDecLT:
			fallthrough
		case OpcodeDecLTEq:
			fallthrough
		case OpcodeDecGT:
			fallthrough
		case OpcodeDecGTEq:
			fallthrough
		case OpcodeDecEq:
			fallthrough
		case OpcodeDecAdd:
			fallthrough
		case OpcodeDecSub:
			fallthrough
		case OpcodeDecMul:
			fallthrough
		case OpcodeDecDiv:
			leftReg := inter.readRegister()
			rightReg := inter.readRegister()
			leftArg := inter.fp.Registers[leftReg]
			rightArg := inter.fp.Registers[rightReg]

			// Register in which to store the product
			dest := inter.readRegister()

			var leftValue, rightValue float32
			var ok bool

			if leftArg == nil {
				panic("expected `float32`, found <nil>")
			}

			if leftValue, ok = leftArg.Value.(float32); ok == false {
				panic(fmt.Sprintf("expected `float32`, found `%T`", leftArg))
			}

			if rightArg == nil {
				panic("expected `float32`, found <nil>")
			}

			if rightValue, ok = rightArg.Value.(float32); ok == false {
				panic(fmt.Sprintf("expected `float32`, found `%T`", rightArg))
			}

			// Actual math done here, once arguments have been cast
			var result interface{}

			switch opcode {
			case OpcodeDecLT:
				result = leftValue < rightValue
			case OpcodeDecLTEq:
				result = leftValue <= rightValue
			case OpcodeDecGT:
				result = leftValue > rightValue
			case OpcodeDecGTEq:
				result = leftValue >= rightValue
			case OpcodeDecEq:
				result = leftValue == rightValue
			case OpcodeDecAdd:
				result = leftValue + rightValue
			case OpcodeDecSub:
				result = leftValue - rightValue
			case OpcodeDecMul:
				result = leftValue * rightValue
			case OpcodeDecDiv:
				result = leftValue / rightValue
			}

			if inter.fp.Registers[dest] == nil {
				inter.fp.Registers[dest] = &Register{}
			}

			inter.fp.Registers[dest].Value = result
		case OpcodePrint:
			arg := inter.fp.Registers[inter.readRegister()]
			fmt.Println(arg.Value)
		default:
			panic(fmt.Sprintf("unknown opcode 0x%x", opcode))
		}
	}
}

func (inter *Interpreter) readOpcode() uint8 {
	b := inter.fp.Closure.Prototype.Bytecode.Bytes[inter.ip]
	inter.ip += 1
	return uint8(b)
}

func (inter *Interpreter) readUint32() (value uint32) {
	b0, b1, b2, b3 := inter.getNext4Bytes()
	return uint32(b3) | (uint32(b2) << 8) | (uint32(b1) << 16) | (uint32(b0) << 24)
}

func (inter *Interpreter) readInt32() int32 {
	b0, b1, b2, b3 := inter.getNext4Bytes()
	return int32(b3) | (int32(b2) << 8) | (int32(b1) << 16) | (int32(b0) << 24)
}

func (i *Interpreter) readFloat32() float32 {
	bits := i.readUint32()
	return math.Float32frombits(bits)
}

func (inter *Interpreter) readRegister() RegisterAddress {
	return RegisterAddress(inter.readUint32())
}

func (i *Interpreter) readBytecodeAddress() BytecodeAddress {
	return BytecodeAddress(i.readUint32())
}

func (inter *Interpreter) getNext4Bytes() (b0, b1, b2, b3 byte) {
	// Get the next 4 bytes from the Bytecode
	b := inter.fp.Closure.Prototype.Bytecode.Bytes[inter.ip : inter.ip+4]
	b0, b1, b2, b3 = b[0], b[1], b[2], b[3]

	// Increment the instruction pointer so these bytes don't get read again
	inter.ip += 4

	return b0, b1, b2, b3
}
