package code

const (
	BlockTypeSpecial = 0x8000000000000000
	BlockTypeMask    = 0x80000000ffffffff
	StackHeightMask  = 0x7fffffff00000000

	BlockTypeEmpty = 0x40 | BlockTypeSpecial
	BlockTypeI32   = 0x7f | BlockTypeSpecial
	BlockTypeI64   = 0x7e | BlockTypeSpecial
	BlockTypeF32   = 0x7d | BlockTypeSpecial
	BlockTypeF64   = 0x7c | BlockTypeSpecial
)

func BlockType(typeidx uint32) uint64 {
	return uint64(typeidx)
}
