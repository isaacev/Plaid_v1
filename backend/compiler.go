package backend

import (
	"fmt"

	"github.com/isaacev/Plaid/frontend"
)

// Compile takes an abstract-syntax-tree in the form of a `frontend.Program`
// node (which is assumed to be semantically correct) and returns both a
// `FuncPrototype` for the top-level main function and a slice of all other
// function prototypes defined within the AST at any level
func Compile(prog *frontend.Program) (mainFunc *FuncPrototype, funcs []*FuncPrototype) {
	state := assembly{
		currFunc:     &FuncPrototype{Bytecode: &Bytecode{}},
		childFuncs:   make([]*FuncPrototype, 0),
		stackPtr:     RegisterAddress(0),
		returnRegs:   0,
		localRegs:    0,
		reservedRegs: 0,
	}

	// The 0th register is always reserved for closures to use when passing
	// return values back to the calling stack-frame
	state.returnRegs = 1

	// Reserve register for all parameters and local variables within this scope
	state.localRegs = len(prog.Locals)

	// The "reservedRegs" field is used for determining quickly if a given
	// register is part of the 'register stack' or is reserved for a particular
	// variable or upvalue. Registers reserved for return values are always at
	// the lowest addresses, followed by regsiters for any upvalues, followed by
	// registers for local variables
	state.reservedRegs = state.returnRegs + state.localRegs

	// The `stackPtr` field points to the next unused register which can be used
	// by an instruction for storing intermediate values. The `stackPtr` should
	// not grow between statements meaning if in a statement's compilation it
	// pushes N temporary variables into free registers, then the `stackPtr`
	// should also be decremented N times during the compilation of the statement
	state.stackPtr = RegisterAddress(state.reservedRegs)

	// During static analysis of the AST, records for all main body local
	// variables are recorded in the `Program` struct. These records
	// are transferred to `FunctionPrototype` structs during compilation. Each
	// record is also assigned a specific register in the stack-frame
	for _, recordPtr := range prog.Locals {
		// TODO consider making the LocalRecord structs in `Program.Locals` not pointers
		record := *recordPtr

		// Pass the record to the main body FunctionPrototype
		state.currFunc.Locals = append(state.currFunc.Locals, record)
	}

	// Compile statements in the main body
	for _, stmt := range prog.Statements {
		state.compile(stmt, state.stackPtr)
	}

	// Always add a Halt instruction at the end of the main body so that the
	// program will terminate before overflowing the Bytecode
	state.currFunc.Bytecode.Write(Halt{}.Generate())

	return state.currFunc, state.childFuncs
}

// assembly is used to keep track of the compiler's state between the
// compilation of seperate AST nodes. The struct keeps track of how many things:
//  - how many registers are reserved for return values and local variables at
//    each scope level (in `returnRegs`, `localRegs`, `reservedRegs`)
//  - the lowest available register that can be used for temporary values
//    (in `stackPtr`)
type assembly struct {
	parent       *assembly        // pointer to state of compiler in enclosing function
	currFunc     *FuncPrototype   // prototype being written to
	childFuncs   []*FuncPrototype // only used by top-level assembly struct
	stackPtr     RegisterAddress  // initialized to `reservedRegs`
	returnRegs   int              // total # of reg's needed for return values (always 1)
	localRegs    int              // total # of reg's needed for local values
	reservedRegs int              // total # of reserved reg's (`returnRegs` + `localRegs`)
}

// storeFunc appends a given `FuncPrototype` to the top-level state manager's
// list of all child function prototypes. It returns the index at which the
// prototype was added to the list
func (state *assembly) storeFunc(fn *FuncPrototype) (constantIndex uint32) {
	if state.parent == nil {
		constantIndex = uint32(len(state.childFuncs))
		state.childFuncs = append(state.childFuncs, fn)
		return constantIndex
	}

	return state.parent.storeFunc(fn)
}

// isRegisterOnStack returns true if a given register is NOT reserved for use
// by return values or as a local variable
func (state *assembly) isRegisterOnStack(reg RegisterAddress) bool {
	return int(reg) >= state.reservedRegs
}

// lookupLocalRegister takes a variable name and returns the address of the
// register holding that variable's value. If a local variable with the given
// name cannot be found, the function panics
func (state *assembly) lookupLocalRegister(name string) RegisterAddress {
	for _, localRecord := range state.currFunc.Locals {
		if localRecord.Name == name {
			return RegisterAddress(state.returnRegs + localRecord.LookupIndex)
		}
	}

	panic(fmt.Sprintf("unknown local variable %s", name))
}

// getUpvalueRecord takes a variable name and returns an `exists` flag set to
// true if the variable name corresponds to an upvalue recognized by the current
// closure. If `exists` is `true`, the function also returns the appropriate
// `frontend.UpvalueRecord` corresponding to the upvalue in the current closure.
// If `exists` is false, the second return value is meaningless
func (state *assembly) getUpvalueRecord(name string) (exists bool, record frontend.UpvalueRecord) {
	for _, upvalueRecord := range state.currFunc.Upvalues {
		if upvalueRecord.Name == name {
			return true, upvalueRecord
		}
	}

	return false, frontend.UpvalueRecord{}
}

// compileFunction handles the compilation of a `frontend.FuncExpr` node when
// encountered in the AST. The function compiles the function to a new
// `FuncPrototype`, appends that prototype to the global list of all function
// prototypes, and returns the index of the new prototype in the global
// prototype list
func (state *assembly) compileFunction(n *frontend.FuncExpr) (prototypeIndex uint32) {
	subState := &assembly{
		parent:       state,
		currFunc:     &FuncPrototype{Bytecode: &Bytecode{}},
		stackPtr:     RegisterAddress(0),
		returnRegs:   0,
		localRegs:    0,
		reservedRegs: 0,
	}

	subState.returnRegs = 1
	subState.localRegs = len(n.Locals)
	subState.reservedRegs = subState.returnRegs + subState.localRegs
	subState.stackPtr = RegisterAddress(subState.reservedRegs)

	for _, recordPtr := range n.Upvalues {
		record := *recordPtr
		subState.currFunc.Upvalues = append(subState.currFunc.Upvalues, record)
	}

	for _, recordPtr := range n.Locals {
		record := *recordPtr
		subState.currFunc.Locals = append(subState.currFunc.Locals, record)
	}

	prototypeIndex = state.storeFunc(subState.currFunc)

	for _, stmt := range n.Body.Statements {
		subState.compile(stmt, subState.stackPtr)
	}

	subState.currFunc.Bytecode.Write(Return{}.Generate())

	return prototypeIndex
}

// compile takes any AST node and represent's the node's semantic meaning in a
// series of bytecode instructions written to the current `FuncPrototype`'s
// `Bytecode` field
func (state *assembly) compile(node frontend.Node, destReg RegisterAddress) RegisterAddress {
	switch n := node.(type) {
	case *frontend.IntegerExpr:
		state.currFunc.Bytecode.Write(IntConst{Value: n.Value, Dest: destReg}.Generate())

		// increment the stackPtr if the integer constant wan't stored in a reserved regsiter
		if state.isRegisterOnStack(destReg) {
			state.stackPtr++
		}
	case *frontend.FuncExpr:
		constantIndex := state.compileFunction(n)
		state.currFunc.Bytecode.Write(FuncConst{ConstantIndex: constantIndex, Dest: destReg}.Generate())

		// increment the stackPtr if the function closure wan't stored in a reserved regsiter
		if state.isRegisterOnStack(destReg) {
			state.stackPtr++
		}
	case *frontend.IdentExpr:
		if exists, record := state.getUpvalueRecord(n.Name); exists {
			state.currFunc.Bytecode.Write(LoadUpVal{Index: int32(record.LookupIndex), Dest: destReg}.Generate())

			if state.isRegisterOnStack(destReg) {
				state.stackPtr++
			}
		} else {
			return state.lookupLocalRegister(n.Name)
		}
	case *frontend.TypeAnnotationStmt:
		// TODO
	case *frontend.BinaryExpr:
		leftReg := state.compile(n.Left, state.stackPtr)
		rightReg := state.compile(n.Right, state.stackPtr)

		switch n.Operator {
		case "<":
			state.currFunc.Bytecode.Write(IntLT{Left: leftReg, Right: rightReg, Dest: destReg}.Generate())
		case "<=":
			state.currFunc.Bytecode.Write(IntLTEq{Left: leftReg, Right: rightReg, Dest: destReg}.Generate())
		case ">":
			state.currFunc.Bytecode.Write(IntGT{Left: leftReg, Right: rightReg, Dest: destReg}.Generate())
		case ">=":
			state.currFunc.Bytecode.Write(IntGTEq{Left: leftReg, Right: rightReg, Dest: destReg}.Generate())
		case "==":
			state.currFunc.Bytecode.Write(IntEq{Left: leftReg, Right: rightReg, Dest: destReg}.Generate())
		case "+":
			state.currFunc.Bytecode.Write(IntAdd{Left: leftReg, Right: rightReg, Dest: destReg}.Generate())
		case "-":
			state.currFunc.Bytecode.Write(IntSub{Left: leftReg, Right: rightReg, Dest: destReg}.Generate())
		case "*":
			state.currFunc.Bytecode.Write(IntMul{Left: leftReg, Right: rightReg, Dest: destReg}.Generate())
		default:
			panic(fmt.Sprintf("unknown operator: '%s'", string(n.Operator)))
		}

		if state.isRegisterOnStack(leftReg) {
			state.stackPtr--
		}

		if state.isRegisterOnStack(rightReg) {
			state.stackPtr--
		}

		if state.isRegisterOnStack(destReg) {
			state.stackPtr++
		}
	case *frontend.DeclarationStmt:
		destReg := state.lookupLocalRegister(n.Assignee.Name)
		assignmentReg := state.compile(n.Assignment, destReg)

		if assignmentReg != destReg && state.isRegisterOnStack(assignmentReg) == false {
			state.currFunc.Bytecode.Write(Move{Source: assignmentReg, Dest: destReg}.Generate())
		}
	case *frontend.AssignmentStmt:
		if exists, record := state.getUpvalueRecord(n.Assignee.Name); exists {
			// assignee is an upvalue so the assignment is to be loaded onto the
			// register-stack and then saved to the upvalue via "SetUpVal"
			assignmentReg := state.compile(n.Assignment, state.stackPtr)
			state.currFunc.Bytecode.Write(SetUpVal{Source: assignmentReg, Index: int32(record.LookupIndex)}.Generate())

			if state.isRegisterOnStack(assignmentReg) {
				state.stackPtr--
			}
		} else {
			destReg = state.compile(n.Assignee, state.stackPtr)
			assignmentReg := state.compile(n.Assignment, destReg)

			// The additional Move instruction handles the case when a variable is
			// assigned to another variable. In the cases where a constant or some
			// more complex expression is being loaded into a register, the
			// expression handles loading the register
			if assignmentReg != destReg && state.isRegisterOnStack(assignmentReg) == false {
				state.currFunc.Bytecode.Write(Move{Source: assignmentReg, Dest: destReg}.Generate())
			}
		}
	case *frontend.DispatchExpr:
		// First parameter register defaults to r0 if no arguments are given
		firstArgRegister := RegisterAddress(0)

		// Compile any argument expressions and store their output on the
		// register stack. Record the register storing the output of the first
		// argument since that register will become part of the Dispatch
		// instruction. If the `firstArgRegister` variable points to r0, that means
		// that no arguments are being passed. Each closure knows how many
		// arguments it accepts
		for i, arg := range n.Arguments {
			paramReg := state.compile(arg, state.stackPtr)

			if i == 0 {
				firstArgRegister = paramReg
			}
		}

		// reset stack pointer so that any temporary parameters written as
		// function arguments will can be overwritten
		state.stackPtr = firstArgRegister

		// determine which register is holding the closure to call
		// FIXME handle call to upvalue
		sourceReg := state.lookupLocalRegister(n.Root.Name)

		// Any values returned from closures are stored at the calling stack
		// frame's 0th register. The Dispatch/Move instruction pair calls a
		// given closure and transfers any return values from the 0th register
		// to either a variable's register or to the next available register on
		// the stack
		state.currFunc.Bytecode.Write(Dispatch{Source: sourceReg, FirstArgRegister: firstArgRegister}.Generate())

		// Only move the return value from r0 if the function call specifies
		// some other destination for the result
		if destReg != 0 {
			state.currFunc.Bytecode.Write(Move{Source: 0, Dest: destReg}.Generate())
		}
	case *frontend.ReturnStmt:
		sourceReg := RegisterAddress(0)

		if len(n.Arguments) > 0 {
			sourceReg = state.compile(n.Arguments[0], state.stackPtr)
		}

		state.currFunc.Bytecode.Write(Return{Source: sourceReg}.Generate())
	case *frontend.PrintStmt:
		sourceReg := state.compile(n.Arguments[0], state.stackPtr)
		state.currFunc.Bytecode.Write(Print{Source: sourceReg}.Generate())
	default:
		panic(fmt.Sprintf("unknown node of type %T", n))
	}

	return destReg
}