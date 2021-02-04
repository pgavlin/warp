package code

const (
	OpUnreachable  = 0x00
	OpNop          = 0x01
	OpBlock        = 0x02
	OpLoop         = 0x03
	OpIf           = 0x04
	OpElse         = 0x05
	OpEnd          = 0x0b
	OpBr           = 0x0c
	OpBrIf         = 0x0d
	OpBrTable      = 0x0e
	OpReturn       = 0x0f
	OpCall         = 0x10
	OpCallIndirect = 0x11

	OpDrop   = 0x1a
	OpSelect = 0x1b

	OpLocalGet  = 0x20
	OpLocalSet  = 0x21
	OpLocalTee  = 0x22
	OpGlobalGet = 0x23
	OpGlobalSet = 0x24

	OpI32Load    = 0x28
	OpI64Load    = 0x29
	OpF32Load    = 0x2a
	OpF64Load    = 0x2b
	OpI32Load8S  = 0x2c
	OpI32Load8U  = 0x2d
	OpI32Load16S = 0x2e
	OpI32Load16U = 0x2f
	OpI64Load8S  = 0x30
	OpI64Load8U  = 0x31
	OpI64Load16S = 0x32
	OpI64Load16U = 0x33
	OpI64Load32S = 0x34
	OpI64Load32U = 0x35
	OpI32Store   = 0x36
	OpI64Store   = 0x37
	OpF32Store   = 0x38
	OpF64Store   = 0x39
	OpI32Store8  = 0x3a
	OpI32Store16 = 0x3b
	OpI64Store8  = 0x3c
	OpI64Store16 = 0x3d
	OpI64Store32 = 0x3e
	OpMemorySize = 0x3f
	OpMemoryGrow = 0x40

	OpI32Const = 0x41
	OpI64Const = 0x42
	OpF32Const = 0x43
	OpF64Const = 0x44

	OpI32Eqz = 0x45
	OpI32Eq  = 0x46
	OpI32Ne  = 0x47
	OpI32LtS = 0x48
	OpI32LtU = 0x49
	OpI32GtS = 0x4a
	OpI32GtU = 0x4b
	OpI32LeS = 0x4c
	OpI32LeU = 0x4d
	OpI32GeS = 0x4e
	OpI32GeU = 0x4f

	OpI64Eqz = 0x50
	OpI64Eq  = 0x51
	OpI64Ne  = 0x52
	OpI64LtS = 0x53
	OpI64LtU = 0x54
	OpI64GtS = 0x55
	OpI64GtU = 0x56
	OpI64LeS = 0x57
	OpI64LeU = 0x58
	OpI64GeS = 0x59
	OpI64GeU = 0x5a

	OpF32Eq = 0x5b
	OpF32Ne = 0x5c
	OpF32Lt = 0x5d
	OpF32Gt = 0x5e
	OpF32Le = 0x5f
	OpF32Ge = 0x60

	OpF64Eq = 0x61
	OpF64Ne = 0x62
	OpF64Lt = 0x63
	OpF64Gt = 0x64
	OpF64Le = 0x65
	OpF64Ge = 0x66

	OpI32Clz    = 0x67
	OpI32Ctz    = 0x68
	OpI32Popcnt = 0x69
	OpI32Add    = 0x6a
	OpI32Sub    = 0x6b
	OpI32Mul    = 0x6c
	OpI32DivS   = 0x6d
	OpI32DivU   = 0x6e
	OpI32RemS   = 0x6f
	OpI32RemU   = 0x70
	OpI32And    = 0x71
	OpI32Or     = 0x72
	OpI32Xor    = 0x73
	OpI32Shl    = 0x74
	OpI32ShrS   = 0x75
	OpI32ShrU   = 0x76
	OpI32Rotl   = 0x77
	OpI32Rotr   = 0x78

	OpI64Clz    = 0x79
	OpI64Ctz    = 0x7a
	OpI64Popcnt = 0x7b
	OpI64Add    = 0x7c
	OpI64Sub    = 0x7d
	OpI64Mul    = 0x7e
	OpI64DivS   = 0x7f
	OpI64DivU   = 0x80
	OpI64RemS   = 0x81
	OpI64RemU   = 0x82
	OpI64And    = 0x83
	OpI64Or     = 0x84
	OpI64Xor    = 0x85
	OpI64Shl    = 0x86
	OpI64ShrS   = 0x87
	OpI64ShrU   = 0x88
	OpI64Rotl   = 0x89
	OpI64Rotr   = 0x8a

	OpF32Abs      = 0x8b
	OpF32Neg      = 0x8c
	OpF32Ceil     = 0x8d
	OpF32Floor    = 0x8e
	OpF32Trunc    = 0x8f
	OpF32Nearest  = 0x90
	OpF32Sqrt     = 0x91
	OpF32Add      = 0x92
	OpF32Sub      = 0x93
	OpF32Mul      = 0x94
	OpF32Div      = 0x95
	OpF32Min      = 0x96
	OpF32Max      = 0x97
	OpF32Copysign = 0x98

	OpF64Abs      = 0x99
	OpF64Neg      = 0x9a
	OpF64Ceil     = 0x9b
	OpF64Floor    = 0x9c
	OpF64Trunc    = 0x9d
	OpF64Nearest  = 0x9e
	OpF64Sqrt     = 0x9f
	OpF64Add      = 0xa0
	OpF64Sub      = 0xa1
	OpF64Mul      = 0xa2
	OpF64Div      = 0xa3
	OpF64Min      = 0xa4
	OpF64Max      = 0xa5
	OpF64Copysign = 0xa6

	OpI32WrapI64        = 0xa7
	OpI32TruncF32S      = 0xa8
	OpI32TruncF32U      = 0xa9
	OpI32TruncF64S      = 0xaa
	OpI32TruncF64U      = 0xab
	OpI64ExtendI32S     = 0xac
	OpI64ExtendI32U     = 0xad
	OpI64TruncF32S      = 0xae
	OpI64TruncF32U      = 0xaf
	OpI64TruncF64S      = 0xb0
	OpI64TruncF64U      = 0xb1
	OpF32ConvertI32S    = 0xb2
	OpF32ConvertI32U    = 0xb3
	OpF32ConvertI64S    = 0xb4
	OpF32ConvertI64U    = 0xb5
	OpF32DemoteF64      = 0xb6
	OpF64ConvertI32S    = 0xb7
	OpF64ConvertI32U    = 0xb8
	OpF64ConvertI64S    = 0xb9
	OpF64ConvertI64U    = 0xba
	OpF64PromoteF32     = 0xbb
	OpI32ReinterpretF32 = 0xbc
	OpI64ReinterpretF64 = 0xbd
	OpF32ReinterpretI32 = 0xbe
	OpF64ReinterpretI64 = 0xbf

	OpI32Extend8S  = 0xc0
	OpI32Extend16S = 0xc1
	OpI64Extend8S  = 0xc2
	OpI64Extend16S = 0xc3
	OpI64Extend32S = 0xc4

	OpPrefix = 0xfc

	OpI32TruncSatF32S = 0
	OpI32TruncSatF32U = 1
	OpI32TruncSatF64S = 2
	OpI32TruncSatF64U = 3
	OpI64TruncSatF32S = 4
	OpI64TruncSatF32U = 5
	OpI64TruncSatF64S = 6
	OpI64TruncSatF64U = 7
)
