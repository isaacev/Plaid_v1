package backend

// TODO document opcodes
const (
	// Basic opcodes
	OpcodeNop       uint8 = 0x01
	OpcodeHalt      uint8 = 0x02
	OpcodeBoolConst uint8 = 0x03
	OpcodeIntConst  uint8 = 0x04
	OpcodeDecConst  uint8 = 0x05
	OpcodeStrConst  uint8 = 0x06
	OpcodeFuncConst uint8 = 0x07
	OpcodeMove      uint8 = 0x08
	OpcodeLoadUpVal uint8 = 0x09
	OpcodeSetUpVal  uint8 = 0x0A
	OpcodeBrAlways  uint8 = 0x0B
	OpcodeBrTrue    uint8 = 0x0C
	OpcodeBrFalse   uint8 = 0x0D
	OpcodeDispatch  uint8 = 0x0E
	OpcodeReturn    uint8 = 0x0F
	OpcodePrint     uint8 = 0x10

	// Interger manipulation (0x70...0x7F)
	OpcodeIntLT   uint8 = 0x70
	OpcodeIntLTEq uint8 = 0x71
	OpcodeIntGT   uint8 = 0x72
	OpcodeIntGTEq uint8 = 0x73
	OpcodeIntEq   uint8 = 0x74
	OpcodeIntAdd  uint8 = 0x75
	OpcodeIntSub  uint8 = 0x76
	OpcodeIntMul  uint8 = 0x77
	OpcodeIntDiv  uint8 = 0x78
	OpcodeIntNeg  uint8 = 0x79

	// Decimal manipulation (0x80...0x8F)
	OpcodeDecLT   uint8 = 0x80
	OpcodeDecLTEq uint8 = 0x81
	OpcodeDecGT   uint8 = 0x82
	OpcodeDecGTEq uint8 = 0x83
	OpcodeDecEq   uint8 = 0x84
	OpcodeDecAdd  uint8 = 0x85
	OpcodeDecSub  uint8 = 0x86
	OpcodeDecMul  uint8 = 0x87
	OpcodeDecDiv  uint8 = 0x88
	OpcodeDecNeg  uint8 = 0x89
)
