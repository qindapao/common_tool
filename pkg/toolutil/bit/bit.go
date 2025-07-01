package bit

import (
	"encoding/binary"

	"golang.org/x/exp/constraints"
)

// SplitUint16ToBytes 将 uint16 拆成高位和低位
func SplitUint16ToBytes(val uint16) (hi, lo byte) {
	hi = byte(val >> 8)
	lo = byte(val & 0xFF)
	return
}

func SplitUint32ToBytes(val uint32) (b3, b2, b1, b0 byte) {
	b3 = byte((val >> 24) & 0xFF)
	b2 = byte((val >> 16) & 0xFF)
	b1 = byte((val >> 8) & 0xFF)
	b0 = byte(val & 0xFF)
	return
}

// SplitUint64ToBytes 将 uint64 拆成 8 个字节（高位在前）
func SplitUint64ToBytes(val uint64) (b7, b6, b5, b4, b3, b2, b1, b0 byte) {
	b7 = byte((val >> 56) & 0xFF)
	b6 = byte((val >> 48) & 0xFF)
	b5 = byte((val >> 40) & 0xFF)
	b4 = byte((val >> 32) & 0xFF)
	b3 = byte((val >> 24) & 0xFF)
	b2 = byte((val >> 16) & 0xFF)
	b1 = byte((val >> 8) & 0xFF)
	b0 = byte(val & 0xFF)
	return
}

// SplitUint64ByEndian 根据字节序将 uint64 拆成 8 个 byte
// 例如：
// val := uint64(0x1122334455667788)
// SplitUint64ByEndian(val, binary.BigEndian)  => [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
// SplitUint64ByEndian(val, binary.LittleEndian) => [8]byte{0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11}
func SplitUint64ByEndian(val uint64, order binary.ByteOrder) [8]byte {
	var b [8]byte
	order.PutUint64(b[:], val)
	return b
}

// SplitUint64BE 将 uint64 拆成 [8]byte，使用大端序（高位在前）
// 例如：
// val := uint64(0x1122334455667788)
// SplitUint64BE(val) => [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
func SplitUint64BE(val uint64) [8]byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], val)
	return b
}

// SplitUint64LE 将 uint64 拆成 [8]byte，使用小端序（低位在前）
// 例如：
// val := uint64(0x1122334455667788)
// SplitUint64LE(val) => [8]byte{0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11}
func SplitUint64LE(val uint64) [8]byte {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], val)
	return b
}

// JoinBytesToUint64 根据字节序将 8 个 byte 合并为 uint64
// 可以动态处理，先留着
func JoinBytesToUint64(b [8]byte, order binary.ByteOrder) uint64 {
	return order.Uint64(b[:])
}

// JoinBytesBE 将 8 字节按照大端顺序拼接为 uint64（b[0] 为高位）
func JoinBytesBE(b [8]byte) uint64 {
	return binary.BigEndian.Uint64(b[:])
}

// JoinBytesLE 将 8 字节按照小端顺序拼接为 uint64（b[0] 为低位）
func JoinBytesLE(b [8]byte) uint64 {
	return binary.LittleEndian.Uint64(b[:])
}

type Uint interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

// 提取 val 中 [start:start+width) 范围的字段
// v16 := uint16(0b1011001111001010)
// fmt.Printf("%08b\n", ExtractBits(v16, 4, 4))  // 输出 1110
// v64 := uint64(0x0FAB_CDEF_0000_1234)
// fmt.Printf("%x\n", ExtractBits(v64, 16, 8))  // 提取 bits 23:16
func ExtractBits[T Uint](val T, start, width byte) T {
	var mask T = (1 << width) - 1
	return (val >> start) & mask
}

// 把字段放回 start 起始位位置（等价于还原为原始结构一部分）
// code := uint32(0x060400)
// base := ExtractBits(code, 16, 8)      // 0x06
// sub  := ExtractBits(code, 8, 8)       // 0x04
// prog := ExtractBits(code, 0, 8)       // 0x00
// 还原成原始结构（复合回 0x060400）
// restored := RestoreBits(base, 16) |
//
//	RestoreBits(sub, 8) |
//	RestoreBits(prog, 0)
func RestoreFieldToOffset[T constraints.Unsigned](val T, offset byte) T {
	return val << offset
}