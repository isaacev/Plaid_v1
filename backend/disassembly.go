package backend

import (
	"fmt"
)

// Disassemble represents a `FuncPrototype` produced by the compiler in a more
// digestable form. The disassembly lists the each disassembled instruction,
// each constant, each disassembled upvalue record, and each disassembled local
// record
func Disassemble(fn *FuncPrototype) {
	fmt.Printf("<function at %p>\n", fn)

	fmt.Printf("  instructions for %p\n", fn)
	disassembleBytecode(fn, fn.Bytecode)

	fmt.Printf("  constants (%d) for %p\n", len(fn.Constants), fn)
	for i, constant := range fn.Constants {
		fmt.Printf("   #%d %v", i, constant)
	}

	fmt.Printf("  upvalues (%d) for %p\n", len(fn.Upvalues), fn)
	for i, record := range fn.Upvalues {
		fmt.Printf("   #%d \"%s\" localToParent=%t lookupIndex=%d \n", i, record.Name, record.LocalToParent, record.LookupIndex)
	}

	fmt.Printf("  locals (%d) for %p\n", len(fn.Locals), fn)
	for _, record := range fn.Locals {
		reg := 1 + record.LookupIndex
		fmt.Printf("   #%d r%d \"%s\" isParam=%t\n", record.LookupIndex, reg, record.Name, record.IsParameter)
	}
}

// disassembleBytecode converts a single instruction from a series of bytes into
// a printed formatted string. Each disassembled instruction includes the
// instruction's starting byte offset, the instruction's name and any arguments
// it may have. If an unknown opcode is encountered, the disassembler panics
func disassembleBytecode(fn *FuncPrototype, b *Bytecode) {
	for i, l := 0, b.Size; i < l; {
		switch uint8(b.Bytes[i]) {
		case OpcodeHalt:
			fmt.Printf("   %4d Halt\n", i)
			i += 1
		case OpcodeIntConst:
			value := bytesToInt32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			dest := bytesToUint32(b.Bytes[i+5], b.Bytes[i+6], b.Bytes[i+7], b.Bytes[i+8])
			fmt.Printf("   %4d %-9s $%d, r%d\n", i, "IntConst", value, dest)
			i += 9
		case OpcodeFuncConst:
			index := bytesToUint32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			dest := bytesToUint32(b.Bytes[i+5], b.Bytes[i+6], b.Bytes[i+7], b.Bytes[i+8])
			fmt.Printf("   %4d %-9s #%d, r%d\n", i, "FuncConst", index, dest)
			i += 9
		case OpcodeMove:
			source := bytesToUint32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			dest := bytesToUint32(b.Bytes[i+5], b.Bytes[i+6], b.Bytes[i+7], b.Bytes[i+8])
			fmt.Printf("   %4d %-9s r%d, r%d\n", i, "Move", source, dest)
			i += 9
		case OpcodeLoadUpVal:
			index := bytesToInt32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			dest := bytesToUint32(b.Bytes[i+5], b.Bytes[i+6], b.Bytes[i+7], b.Bytes[i+8])
			fmt.Printf("   %4d %-9s #%d, r%d\n", i, "LoadUpVal", index, dest)
			i += 9
		case OpcodeSetUpVal:
			source := bytesToInt32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			index := bytesToUint32(b.Bytes[i+5], b.Bytes[i+6], b.Bytes[i+7], b.Bytes[i+8])
			fmt.Printf("   %4d %-9s r%d, #%d\n", i, "SetUpVal", source, index)
			i += 9
		case OpcodeDispatch:
			source := bytesToUint32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			firstParamRegister := bytesToUint32(b.Bytes[i+5], b.Bytes[i+6], b.Bytes[i+7], b.Bytes[i+8])
			fmt.Printf("   %4d %-9s r%d, (r%d...)\n", i, "Dispatch", source, firstParamRegister)
			i += 9
		case OpcodeReturn:
			source := bytesToUint32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			fmt.Printf("   %4d %-9s r%d\n", i, "Return", source)
			i += 5
		case OpcodeIntAdd:
			left := bytesToUint32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			right := bytesToUint32(b.Bytes[i+5], b.Bytes[i+6], b.Bytes[i+7], b.Bytes[i+8])
			dest := bytesToUint32(b.Bytes[i+9], b.Bytes[i+10], b.Bytes[i+11], b.Bytes[i+12])
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "IntAdd", left, right, dest)
			i += 13
		case OpcodeIntSub:
			left := bytesToUint32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			right := bytesToUint32(b.Bytes[i+5], b.Bytes[i+6], b.Bytes[i+7], b.Bytes[i+8])
			dest := bytesToUint32(b.Bytes[i+9], b.Bytes[i+10], b.Bytes[i+11], b.Bytes[i+12])
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "IntSub", left, right, dest)
			i += 13
		case OpcodeIntMul:
			left := bytesToUint32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			right := bytesToUint32(b.Bytes[i+5], b.Bytes[i+6], b.Bytes[i+7], b.Bytes[i+8])
			dest := bytesToUint32(b.Bytes[i+9], b.Bytes[i+10], b.Bytes[i+11], b.Bytes[i+12])
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "IntMul", left, right, dest)
			i += 13
		case OpcodePrint:
			source := bytesToUint32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			fmt.Printf("   %4d %-9s r%d\n", i, "Print", source)
			i += 5
		default:
			panic(fmt.Sprintf("unknown opcode 0x%x", uint8(b.Bytes[i])))
		}
	}
}