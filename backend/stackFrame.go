package backend

type BytecodeAddress uint32
type RegisterAddress uint8

type Register struct {
	Value interface{}
}

type StackFrame struct {
	Closure         *Closure
	ReturnToAddress BytecodeAddress
	Registers       []*Register
}
