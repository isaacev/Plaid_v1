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
		var constType string
		var constRep string

		switch v := constant.(type) {
		case string:
			constType = "<string>"
			constRep = fmt.Sprintf("\"%s\"", v)
		default:
			constType = "<?>"
			constRep = fmt.Sprintf("%v", v)
		}

		fmt.Printf("   #%d %-8s %s\n", i, constType, constRep)
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
		case OpcodeBoolConst:
			value := bytesToInt32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			dest := b.Bytes[i+5]

			strValue := "<false>"

			if value == 1 {
				strValue = "<true>"
			}

			fmt.Printf("   %4d %-9s %s, r%d\n", i, "BoolConst", strValue, dest)
			i += 6
		case OpcodeIntConst:
			value := bytesToInt32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			dest := b.Bytes[i+5]
			fmt.Printf("   %4d %-9s $%d, r%d\n", i, "IntConst", value, dest)
			i += 6
		case OpcodeDecConst:
			value := bytesToFloat32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			dest := b.Bytes[i+5]
			fmt.Printf("   %4d %-9s $%.2f, r%d\n", i, "DecConst", value, dest)
			i += 6
		case OpcodeStrConst:
			index := bytesToUint32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			dest := b.Bytes[i+5]
			fmt.Printf("   %4d %-9s #%d, r%d\n", i, "StrConst", index, dest)
			i += 6
		case OpcodeFuncConst:
			index := bytesToUint32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			dest := b.Bytes[i+5]
			fmt.Printf("   %4d %-9s #%d, r%d\n", i, "FuncConst", index, dest)
			i += 6
		case OpcodeMove:
			source := b.Bytes[i+1]
			dest := b.Bytes[i+2]
			fmt.Printf("   %4d %-9s r%d, r%d\n", i, "Move", source, dest)
			i += 3
		case OpcodeLoadUpVal:
			index := bytesToInt32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			dest := b.Bytes[i+5]
			fmt.Printf("   %4d %-9s #%d, r%d\n", i, "LoadUpVal", index, dest)
			i += 6
		case OpcodeSetUpVal:
			source := b.Bytes[i+1]
			index := bytesToUint32(b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4], b.Bytes[i+5])
			fmt.Printf("   %4d %-9s r%d, #%d\n", i, "SetUpVal", source, index)
			i += 6
		case OpcodeBrAlways:
			addr := bytesToUint32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			fmt.Printf("   %4d %-9s @%d\n", i, "BrAlways", addr)
			i += 5
		case OpcodeBrTrue:
			test := b.Bytes[i+1]
			addr := bytesToUint32(b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4], b.Bytes[i+5])
			fmt.Printf("   %4d %-9s r%d, @%d\n", i, "BrTrue", test, addr)
			i += 6
		case OpcodeIntLT:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "IntLT", left, right, dest)
			i += 4
		case OpcodeIntLTEq:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "IntLT", left, right, dest)
			i += 4
		case OpcodeIntGT:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "IntLT", left, right, dest)
			i += 4
		case OpcodeIntGTEq:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "IntLT", left, right, dest)
			i += 4
		case OpcodeIntEq:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "IntEq", left, right, dest)
			i += 4
		case OpcodeDispatch:
			source := b.Bytes[i+1]
			firstParamRegister := b.Bytes[i+2]
			fmt.Printf("   %4d %-9s r%d, (r%d...)\n", i, "Dispatch", source, firstParamRegister)
			i += 3
		case OpcodeReturn:
			source := b.Bytes[i+1]
			fmt.Printf("   %4d %-9s r%d\n", i, "Return", source)
			i += 2
		case OpcodeIntAdd:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "IntAdd", left, right, dest)
			i += 4
		case OpcodeIntSub:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "IntSub", left, right, dest)
			i += 4
		case OpcodeIntMul:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "IntMul", left, right, dest)
			i += 4
		case OpcodeIntDiv:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "IntDiv", left, right, dest)
			i += 4
		case OpcodeIntNeg:
			operand := b.Bytes[i+1]
			dest := b.Bytes[i+2]
			fmt.Printf("   %4d %-9s r%d, r%d\n", i, "IntNeg", operand, dest)
			i += 3
		case OpcodeDecAdd:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "DecAdd", left, right, dest)
			i += 4
		case OpcodeDecSub:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "DecSub", left, right, dest)
			i += 4
		case OpcodeDecMul:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "DecMul", left, right, dest)
			i += 4
		case OpcodeDecDiv:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d\n", i, "DecDiv", left, right, dest)
			i += 4
		case OpcodeDecNeg:
			operand := b.Bytes[i+1]
			dest := b.Bytes[i+2]
			fmt.Printf("   %4d %-9s r%d, r%d\n", i, "DecNeg", operand, dest)
			i += 3
		case OpcodeDecLT:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "DecLT", left, right, dest)
			i += 4
		case OpcodeDecLTEq:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "DecLT", left, right, dest)
			i += 4
		case OpcodeDecGT:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "DecLT", left, right, dest)
			i += 4
		case OpcodeDecGTEq:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "DecLT", left, right, dest)
			i += 4
		case OpcodeDecEq:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "DecEq", left, right, dest)
			i += 4
		case OpcodeStrConcat:
			left := b.Bytes[i+1]
			right := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d, r%d, r%d\n", i, "StrConcat", left, right, dest)
			i += 4
		case OpcodePrint:
			source := b.Bytes[i+1]
			fmt.Printf("   %4d %-9s r%d\n", i, "Print", source)
			i += 2
		case OpcodeCastToStr:
			source := b.Bytes[i+1]
			dest := b.Bytes[i+2]
			fmt.Printf("   %4d %-9s r%d, r%d\n", i, "CastToStr", source, dest)
			i += 3
		case OpcodeListBuild:
			length := bytesToUint32(b.Bytes[i+1], b.Bytes[i+2], b.Bytes[i+3], b.Bytes[i+4])
			first := b.Bytes[i+5]
			dest := b.Bytes[i+6]
			fmt.Printf("   %4d %-9s %d r%d r%d\n", i, "ListBuild", length, first, dest)
			i += 7
		case OpcodeListAccess:
			index := b.Bytes[i+1]
			source := b.Bytes[i+2]
			dest := b.Bytes[i+3]
			fmt.Printf("   %4d %-9s r%d r%d r%d\n", i, "ListAccess", index, source, dest)
			i += 4
		case OpcodeListLen:
			source := b.Bytes[i+1]
			dest := b.Bytes[i+2]
			fmt.Printf("   %4d %-9s r%d r%d\n", i, "ListLen", source, dest)
			i += 3
		default:
			panic(fmt.Sprintf("unknown opcode 0x%x", uint8(b.Bytes[i])))
		}
	}
}
