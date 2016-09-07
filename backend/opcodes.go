package backend

// TODO document opcodes
const (
	// Program halt
	OpcodeHalt uint8 = 0x1

	// Constant loading, move, and upvalue manipulation
	OpcodeIntConst uint8 = 0x10
	OpcodeDecConst uint8 = 0x11
	// OpcodeStrConst  uint8 = 0x12
	OpcodeFuncConst uint8 = 0x13
	OpcodeMove      uint8 = 0x14
	OpcodeLoadUpVal uint8 = 0x15
	OpcodeSetUpVal  uint8 = 0x16

	// Branching
	OpcodeBr      uint8 = 0x20
	OpcodeBrTrue  uint8 = 0x21
	OpcodeBrFalse uint8 = 0x22

	// Interger comparison
	OpcodeIntLT   uint8 = 0x30
	OpcodeIntLTEq uint8 = 0x31
	OpcodeIntGT   uint8 = 0x32
	OpcodeIntGTEq uint8 = 0x33
	OpcodeIntEq   uint8 = 0x34

	// Decimal comparison
	OpcodeDecLT   uint8 = 0x35
	OpcodeDecLTEq uint8 = 0x36
	OpcodeDecGT   uint8 = 0x37
	OpcodeDecGTEq uint8 = 0x38
	OpcodeDecEq   uint8 = 0x39

	// Function calling and returning
	OpcodeDispatch uint8 = 0x40
	OpcodeReturn   uint8 = 0x41

	// Integer arithmetic
	OpcodeIntAdd uint8 = 0x50
	OpcodeIntSub uint8 = 0x51
	OpcodeIntMul uint8 = 0x52
	// OpcodeIntDiv uint8 = 0x53

	// Decimal arithmetic
	OpcodeDecAdd uint8 = 0x54
	OpcodeDecSub uint8 = 0x55
	OpcodeDecMul uint8 = 0x56
	// OpcodeDecDiv uint8 = 0x57

	// Print statement
	OpcodePrint uint8 = 0x90
)
