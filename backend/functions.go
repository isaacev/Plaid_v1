package backend

import (
	"github.com/isaacev/Plaid/frontend"
)

// Closure is the combination of a static `FuncPrototype` and the live upvalues
// the prototype needs to be executable. A `FuncPrototype` must first be
// converted into a Closure before it can used by the Interpreter. A function
// prototype may be used and reused but Closure is only valid so long as it is
// being interpreted. Once its function returns, it can be discarded. If the
// same function needs to be re-run, a new Closure will be created with the same
// `FuncPrototype`
type Closure struct {
	Prototype *FuncPrototype
	Upvalues  []*Upvalue
}

// NewClosure returns a newly created `Closure` given a `FuncPrototype` to use
// and a stack of live `StackFrame`s on which to look up upvalues. This function
// is responsible for converting static `UpvalueRecord`s into live `Upvalue`s
// which will point to a live register value higher up the `callstack`
func NewClosure(callstack []*StackFrame, fn *FuncPrototype) *Closure {
	closure := &Closure{Prototype: fn}

	if len(fn.Upvalues) > 0 {
		enclosingStackFrame := callstack[len(callstack)-1]
		totalReturns := 1

		for _, record := range fn.Upvalues {
			upvalue := &Upvalue{}

			if record.LocalToParent {
				// `upvalue` is a local variable of the enclosing function so the
				// "LookupIndex" field is register address of the local variable
				// in the enclosing function's register array
				upvalue.Cell = enclosingStackFrame.Registers[totalReturns+record.LookupIndex]
			} else {
				// `upvalue` is also an upvalue to the enclosing function so the
				// "LookupIndex" field represents the index of the upvalue in
				// the enclosing function's list of upvalues
				upvalue = enclosingStackFrame.Closure.Upvalues[record.LookupIndex]
			}

			closure.Upvalues = append(closure.Upvalues, upvalue)
		}
	}

	return closure
}

type Upvalue struct {
	// When an Upvalue is open, the `Cell` field points to a register in some
	// activation record on the call stack. After the Upvalue has been closed,
	// the value of the `Cell` register is copied to the `Value` field and the
	// `Cell` pointer is updated to point at the field `Value`
	Cell  *Register
	Value interface{}
}

// FuncPrototype stores static information about a first-class function value.
// This includes information about what upvalues the closure requires, what
// local variables need reserved registers, any constants to supply, and the
// raw bytecode instructions to execute
type FuncPrototype struct {
	Upvalues  []frontend.UpvalueRecord
	Locals    []frontend.LocalRecord
	Constants []interface{}
	Bytecode  *Bytecode
}

// Bytecode is a byte-slice of raw compiled instructions. Bytecode's can't be
// executed without the context of of `FuncPrototype` which together can be
// converted into a executable `Closure`
type Bytecode struct {
	Size  int
	Bytes []byte
}

// Write implements io.Writer for the Bytecode struct so that in the compilation
// stage instructions can more easily write their bytes to the byte buffer
func (b *Bytecode) Write(p []byte) (n int, err error) {
	b.Size += len(p)
	b.Bytes = append(b.Bytes, p...)
	return len(p), nil
}
