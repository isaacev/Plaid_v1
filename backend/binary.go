package backend

import (
	"math"
)

func int32ToBytes(val int32) (blob []byte) {
	var b0, b1, b2, b3 byte
	b0 = byte((val >> 0x00) & 0xff)
	b1 = byte((val >> 0x08) & 0xff)
	b2 = byte((val >> 0x10) & 0xff)
	b3 = byte((val >> 0x18) & 0xff)

	// Arrange bytes in Big-Endian order
	return []byte{b3, b2, b1, b0}
}

func uint32ToBytes(val uint32) (blob []byte) {
	var b0, b1, b2, b3 byte
	b0 = byte((val >> 0x00) & 0xff)
	b1 = byte((val >> 0x08) & 0xff)
	b2 = byte((val >> 0x10) & 0xff)
	b3 = byte((val >> 0x18) & 0xff)

	// Arrange bytes in Big-Endian order
	return []byte{b3, b2, b1, b0}
}

func float32ToBytes(val float32) (blob []byte) {
	bits := math.Float32bits(val)
	return uint32ToBytes(bits)
}

func registerToBytes(reg RegisterAddress) (blob []byte) {
	return uint32ToBytes(uint32(reg))
}

func addressToBytes(addr BytecodeAddress) (blob []byte) {
	return uint32ToBytes(uint32(addr))
}

func bytesToInt32(b0, b1, b2, b3 byte) int32 {
	return int32(b3) | (int32(b2) << 8) | (int32(b1) << 16) | (int32(b0) << 24)
}

func bytesToUint32(b0, b1, b2, b3 byte) uint32 {
	return uint32(b3) | (uint32(b2) << 8) | (uint32(b1) << 16) | (uint32(b0) << 24)
}

func bytesToFloat32(b0, b1, b2, b3 byte) float32 {
	bits := bytesToUint32(b0, b1, b2, b3)
	float := math.Float32frombits(bits)
	return float
}
