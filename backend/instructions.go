package backend

type Instruction interface {
	Generate() []byte
}

// Halt
//  - takes no arguments, unconditionally stops program execution
//  - typically appended to the end of the top-level main function
type Halt struct{}

// Generate converts this instruction to raw bytes
func (inst Halt) Generate() (blob []byte) {
	blob = append(blob, OpcodeHalt)
	return blob
}

// IntConst <32 bit integer value> <destination register>
type IntConst struct {
	Value int32
	Dest  RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst IntConst) Generate() (blob []byte) {
	blob = append(blob, OpcodeIntConst)
	blob = append(blob, int32ToBytes(inst.Value)...)
	blob = append(blob, registerToBytes(inst.Dest)...)
	return blob
}

// DecConst <32 bit floating point value> <destination register>
type DecConst struct {
	Value float32
	Dest  RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst DecConst) Generate() (blob []byte) {
	blob = append(blob, OpcodeDecConst)
	blob = append(blob, float32ToBytes(inst.Value)...)
	blob = append(blob, registerToBytes(inst.Dest)...)
	return blob
}

// FuncConst <function pool index> <destination register>
type FuncConst struct {
	ConstantIndex uint32
	Dest          RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst FuncConst) Generate() (blob []byte) {
	blob = append(blob, OpcodeFuncConst)
	blob = append(blob, uint32ToBytes(inst.ConstantIndex)...)
	blob = append(blob, registerToBytes(inst.Dest)...)
	return blob
}

// Move <source register> <destination register>
//  - copies the value in the source register into the destination register
type Move struct {
	Source RegisterAddress
	Dest   RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst Move) Generate() (blob []byte) {
	blob = append(blob, OpcodeMove)
	blob = append(blob, registerToBytes(inst.Source)...)
	blob = append(blob, registerToBytes(inst.Dest)...)
	return blob
}

// LoadUpVal <enclosing closure lookup index> <destination register>
//  - value is coped from enclosing closure's upvalue into destination register
type LoadUpVal struct {
	Index int32
	Dest  RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst LoadUpVal) Generate() (blob []byte) {
	blob = append(blob, OpcodeLoadUpVal)
	blob = append(blob, int32ToBytes(inst.Index)...)
	blob = append(blob, registerToBytes(inst.Dest)...)
	return blob
}

// LoadUpVal <source register> <enclosing closure lookup index>
//  - value is copied from source register and used to update the upvalue in the
//    enclosing closure
type SetUpVal struct {
	Source RegisterAddress
	Index  int32
}

// Generate converts this instruction to raw bytes
func (inst SetUpVal) Generate() (blob []byte) {
	blob = append(blob, OpcodeSetUpVal)
	blob = append(blob, registerToBytes(inst.Source)...)
	blob = append(blob, int32ToBytes(inst.Index)...)
	return blob
}

// Br <bytecode address to jump to>
//  - will unconditionally jump to a given address
type Br struct {
	Addr BytecodeAddress
}

// Generate converts this instruction to raw bytes
func (inst Br) Generate() (blob []byte) {
	blob = append(blob, OpcodeBr)
	blob = append(blob, addressToBytes(inst.Addr)...)
	return blob
}

// BrTrue <decision register> <bytecode address>
//  - will jump to the given address if the value in the decision register is 1
type BrTrue struct {
	Source RegisterAddress
	Addr   BytecodeAddress
}

// Generate converts this instruction to raw bytes
func (inst BrTrue) Generate() (blob []byte) {
	blob = append(blob, OpcodeBrTrue)
	blob = append(blob, registerToBytes(inst.Source)...)
	blob = append(blob, addressToBytes(inst.Addr)...)
	return blob
}

// BrFalse <decision register> <bytecode address>
//  - will jump to the given address if the value in the decision register is 0
type BrFalse struct {
	Source RegisterAddress
	Addr   BytecodeAddress
}

// Generate converts this instruction to raw bytes
func (inst BrFalse) Generate() (blob []byte) {
	blob = append(blob, OpcodeBrFalse)
	blob = append(blob, registerToBytes(inst.Source)...)
	blob = append(blob, addressToBytes(inst.Addr)...)
	return blob
}

// IntLT <left operand register> <right operand register> <destination register>
//  - if left < right, load 1 into the destination register, else load 0
type IntLT struct {
	Left  RegisterAddress
	Right RegisterAddress
	Dest  RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst IntLT) Generate() (blob []byte) {
	blob = append(blob, OpcodeIntLT)
	blob = append(blob, registerToBytes(inst.Left)...)
	blob = append(blob, registerToBytes(inst.Right)...)
	blob = append(blob, registerToBytes(inst.Dest)...)
	return blob
}

// IntLTEq <left operand register> <right operand register> <destination register>
//  - if left <= right, load 1 into the destination register, else load 0
type IntLTEq struct {
	Left  RegisterAddress
	Right RegisterAddress
	Dest  RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst IntLTEq) Generate() (blob []byte) {
	blob = append(blob, OpcodeIntLTEq)
	blob = append(blob, registerToBytes(inst.Left)...)
	blob = append(blob, registerToBytes(inst.Right)...)
	blob = append(blob, registerToBytes(inst.Dest)...)
	return blob
}

// IntGT <left operand register> <right operand register> <destination register>
//  - if left > right, load 1 into the destination register, else load 0
type IntGT struct {
	Left  RegisterAddress
	Right RegisterAddress
	Dest  RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst IntGT) Generate() (blob []byte) {
	blob = append(blob, OpcodeIntGT)
	blob = append(blob, registerToBytes(inst.Left)...)
	blob = append(blob, registerToBytes(inst.Right)...)
	blob = append(blob, registerToBytes(inst.Dest)...)
	return blob
}

// IntGTEq <left operand register> <right operand register> <destination register>
//  - if left >= right, load 1 into the destination register, else load 0
type IntGTEq struct {
	Left  RegisterAddress
	Right RegisterAddress
	Dest  RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst IntGTEq) Generate() (blob []byte) {
	blob = append(blob, OpcodeIntGTEq)
	blob = append(blob, registerToBytes(inst.Left)...)
	blob = append(blob, registerToBytes(inst.Right)...)
	blob = append(blob, registerToBytes(inst.Dest)...)
	return blob
}

// IntEq <left operand register> <right operand register> <destination register>
//  - if left == right, load 1 into the destination register, else load 0
type IntEq struct {
	Left  RegisterAddress
	Right RegisterAddress
	Dest  RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst IntEq) Generate() (blob []byte) {
	blob = append(blob, OpcodeIntEq)
	blob = append(blob, registerToBytes(inst.Left)...)
	blob = append(blob, registerToBytes(inst.Right)...)
	blob = append(blob, registerToBytes(inst.Dest)...)
	return blob
}

// Dispatch <source register storing closure> <register with first argument>
//  - after the first argument register, any other arguments are assumed to be
//    sequential in the register array
type Dispatch struct {
	Source           RegisterAddress
	FirstArgRegister RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst Dispatch) Generate() (blob []byte) {
	blob = append(blob, OpcodeDispatch)
	blob = append(blob, registerToBytes(inst.Source)...)
	blob = append(blob, registerToBytes(inst.FirstArgRegister)...)
	return blob
}

// Return <source register holding value to return>
type Return struct {
	Source RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst Return) Generate() (blob []byte) {
	blob = append(blob, OpcodeReturn)
	blob = append(blob, registerToBytes(inst.Source)...)
	return blob
}

// IntAdd <left operand> <right operand> <destination register>
type IntAdd struct {
	Left  RegisterAddress
	Right RegisterAddress
	Dest  RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst IntAdd) Generate() (blob []byte) {
	blob = append(blob, OpcodeIntAdd)
	blob = append(blob, registerToBytes(inst.Left)...)
	blob = append(blob, registerToBytes(inst.Right)...)
	blob = append(blob, registerToBytes(inst.Dest)...)
	return blob
}

// IntSub <left operand> <right operand> <destination register>
type IntSub struct {
	Left  RegisterAddress
	Right RegisterAddress
	Dest  RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst IntSub) Generate() (blob []byte) {
	blob = append(blob, OpcodeIntSub)
	blob = append(blob, registerToBytes(inst.Left)...)
	blob = append(blob, registerToBytes(inst.Right)...)
	blob = append(blob, registerToBytes(inst.Dest)...)
	return blob
}

// IntMul <left operand> <right operand> <destination register>
type IntMul struct {
	Left  RegisterAddress
	Right RegisterAddress
	Dest  RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst IntMul) Generate() (blob []byte) {
	blob = append(blob, OpcodeIntMul)
	blob = append(blob, registerToBytes(inst.Left)...)
	blob = append(blob, registerToBytes(inst.Right)...)
	blob = append(blob, registerToBytes(inst.Dest)...)
	return blob
}

// Print <register holding value to output to `stdin`>
type Print struct {
	Source RegisterAddress
}

// Generate converts this instruction to raw bytes
func (inst Print) Generate() (blob []byte) {
	blob = append(blob, OpcodePrint)
	blob = append(blob, registerToBytes(inst.Source)...)
	return blob
}
