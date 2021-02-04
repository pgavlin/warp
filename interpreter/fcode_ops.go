package interpreter

import "github.com/pgavlin/warp/wasm/code"

const (
	fopUnreachable  opcode = code.OpUnreachable
	fopNop          opcode = code.OpNop
	fopBlock        opcode = code.OpBlock
	fopLoop         opcode = code.OpLoop
	fopIf           opcode = code.OpIf
	fopElse         opcode = code.OpElse
	fopEnd          opcode = code.OpEnd
	fopBr           opcode = code.OpBr
	fopBrIf         opcode = code.OpBrIf
	fopBrTable      opcode = code.OpBrTable
	fopReturn       opcode = code.OpReturn
	fopCall         opcode = code.OpCall
	fopCallIndirect opcode = code.OpCallIndirect

	fopDrop   opcode = code.OpDrop
	fopSelect opcode = code.OpSelect

	fopLocalGet  opcode = code.OpLocalGet
	fopLocalSet  opcode = code.OpLocalSet
	fopLocalTee  opcode = code.OpLocalTee
	fopGlobalGet opcode = code.OpGlobalGet
	fopGlobalSet opcode = code.OpGlobalSet

	fopI32Load    opcode = code.OpI32Load
	fopI64Load    opcode = code.OpI64Load
	fopF32Load    opcode = code.OpF32Load
	fopF64Load    opcode = code.OpF64Load
	fopI32Load8S  opcode = code.OpI32Load8S
	fopI32Load8U  opcode = code.OpI32Load8U
	fopI32Load16S opcode = code.OpI32Load16S
	fopI32Load16U opcode = code.OpI32Load16U
	fopI64Load8S  opcode = code.OpI64Load8S
	fopI64Load8U  opcode = code.OpI64Load8U
	fopI64Load16S opcode = code.OpI64Load16S
	fopI64Load16U opcode = code.OpI64Load16U
	fopI64Load32S opcode = code.OpI64Load32S
	fopI64Load32U opcode = code.OpI64Load32U
	fopI32Store   opcode = code.OpI32Store
	fopI64Store   opcode = code.OpI64Store
	fopF32Store   opcode = code.OpF32Store
	fopF64Store   opcode = code.OpF64Store
	fopI32Store8  opcode = code.OpI32Store8
	fopI32Store16 opcode = code.OpI32Store16
	fopI64Store8  opcode = code.OpI64Store8
	fopI64Store16 opcode = code.OpI64Store16
	fopI64Store32 opcode = code.OpI64Store32
	fopMemorySize opcode = code.OpMemorySize
	fopMemoryGrow opcode = code.OpMemoryGrow

	fopI32Const opcode = code.OpI32Const
	fopI64Const opcode = code.OpI64Const
	fopF32Const opcode = code.OpF32Const
	fopF64Const opcode = code.OpF64Const

	fopI32Eqz opcode = code.OpI32Eqz
	fopI32Eq  opcode = code.OpI32Eq
	fopI32Ne  opcode = code.OpI32Ne
	fopI32LtS opcode = code.OpI32LtS
	fopI32LtU opcode = code.OpI32LtU
	fopI32GtS opcode = code.OpI32GtS
	fopI32GtU opcode = code.OpI32GtU
	fopI32LeS opcode = code.OpI32LeS
	fopI32LeU opcode = code.OpI32LeU
	fopI32GeS opcode = code.OpI32GeS
	fopI32GeU opcode = code.OpI32GeU

	fopI64Eqz opcode = code.OpI64Eqz
	fopI64Eq  opcode = code.OpI64Eq
	fopI64Ne  opcode = code.OpI64Ne
	fopI64LtS opcode = code.OpI64LtS
	fopI64LtU opcode = code.OpI64LtU
	fopI64GtS opcode = code.OpI64GtS
	fopI64GtU opcode = code.OpI64GtU
	fopI64LeS opcode = code.OpI64LeS
	fopI64LeU opcode = code.OpI64LeU
	fopI64GeS opcode = code.OpI64GeS
	fopI64GeU opcode = code.OpI64GeU

	fopF32Eq opcode = code.OpF32Eq
	fopF32Ne opcode = code.OpF32Ne
	fopF32Lt opcode = code.OpF32Lt
	fopF32Gt opcode = code.OpF32Gt
	fopF32Le opcode = code.OpF32Le
	fopF32Ge opcode = code.OpF32Ge

	fopF64Eq opcode = code.OpF64Eq
	fopF64Ne opcode = code.OpF64Ne
	fopF64Lt opcode = code.OpF64Lt
	fopF64Gt opcode = code.OpF64Gt
	fopF64Le opcode = code.OpF64Le
	fopF64Ge opcode = code.OpF64Ge

	fopI32Clz    opcode = code.OpI32Clz
	fopI32Ctz    opcode = code.OpI32Ctz
	fopI32Popcnt opcode = code.OpI32Popcnt
	fopI32Add    opcode = code.OpI32Add
	fopI32Sub    opcode = code.OpI32Sub
	fopI32Mul    opcode = code.OpI32Mul
	fopI32DivS   opcode = code.OpI32DivS
	fopI32DivU   opcode = code.OpI32DivU
	fopI32RemS   opcode = code.OpI32RemS
	fopI32RemU   opcode = code.OpI32RemU
	fopI32And    opcode = code.OpI32And
	fopI32Or     opcode = code.OpI32Or
	fopI32Xor    opcode = code.OpI32Xor
	fopI32Shl    opcode = code.OpI32Shl
	fopI32ShrS   opcode = code.OpI32ShrS
	fopI32ShrU   opcode = code.OpI32ShrU
	fopI32Rotl   opcode = code.OpI32Rotl
	fopI32Rotr   opcode = code.OpI32Rotr

	fopI64Clz    opcode = code.OpI64Clz
	fopI64Ctz    opcode = code.OpI64Ctz
	fopI64Popcnt opcode = code.OpI64Popcnt
	fopI64Add    opcode = code.OpI64Add
	fopI64Sub    opcode = code.OpI64Sub
	fopI64Mul    opcode = code.OpI64Mul
	fopI64DivS   opcode = code.OpI64DivS
	fopI64DivU   opcode = code.OpI64DivU
	fopI64RemS   opcode = code.OpI64RemS
	fopI64RemU   opcode = code.OpI64RemU
	fopI64And    opcode = code.OpI64And
	fopI64Or     opcode = code.OpI64Or
	fopI64Xor    opcode = code.OpI64Xor
	fopI64Shl    opcode = code.OpI64Shl
	fopI64ShrS   opcode = code.OpI64ShrS
	fopI64ShrU   opcode = code.OpI64ShrU
	fopI64Rotl   opcode = code.OpI64Rotl
	fopI64Rotr   opcode = code.OpI64Rotr

	fopF32Abs      opcode = code.OpF32Abs
	fopF32Neg      opcode = code.OpF32Neg
	fopF32Ceil     opcode = code.OpF32Ceil
	fopF32Floor    opcode = code.OpF32Floor
	fopF32Trunc    opcode = code.OpF32Trunc
	fopF32Nearest  opcode = code.OpF32Nearest
	fopF32Sqrt     opcode = code.OpF32Sqrt
	fopF32Add      opcode = code.OpF32Add
	fopF32Sub      opcode = code.OpF32Sub
	fopF32Mul      opcode = code.OpF32Mul
	fopF32Div      opcode = code.OpF32Div
	fopF32Min      opcode = code.OpF32Min
	fopF32Max      opcode = code.OpF32Max
	fopF32Copysign opcode = code.OpF32Copysign

	fopF64Abs      opcode = code.OpF64Abs
	fopF64Neg      opcode = code.OpF64Neg
	fopF64Ceil     opcode = code.OpF64Ceil
	fopF64Floor    opcode = code.OpF64Floor
	fopF64Trunc    opcode = code.OpF64Trunc
	fopF64Nearest  opcode = code.OpF64Nearest
	fopF64Sqrt     opcode = code.OpF64Sqrt
	fopF64Add      opcode = code.OpF64Add
	fopF64Sub      opcode = code.OpF64Sub
	fopF64Mul      opcode = code.OpF64Mul
	fopF64Div      opcode = code.OpF64Div
	fopF64Min      opcode = code.OpF64Min
	fopF64Max      opcode = code.OpF64Max
	fopF64Copysign opcode = code.OpF64Copysign

	fopI32WrapI64        opcode = code.OpI32WrapI64
	fopI32TruncF32S      opcode = code.OpI32TruncF32S
	fopI32TruncF32U      opcode = code.OpI32TruncF32U
	fopI32TruncF64S      opcode = code.OpI32TruncF64S
	fopI32TruncF64U      opcode = code.OpI32TruncF64U
	fopI64ExtendI32S     opcode = code.OpI64ExtendI32S
	fopI64ExtendI32U     opcode = code.OpI64ExtendI32U
	fopI64TruncF32S      opcode = code.OpI64TruncF32S
	fopI64TruncF32U      opcode = code.OpI64TruncF32U
	fopI64TruncF64S      opcode = code.OpI64TruncF64S
	fopI64TruncF64U      opcode = code.OpI64TruncF64U
	fopF32ConvertI32S    opcode = code.OpF32ConvertI32S
	fopF32ConvertI32U    opcode = code.OpF32ConvertI32U
	fopF32ConvertI64S    opcode = code.OpF32ConvertI64S
	fopF32ConvertI64U    opcode = code.OpF32ConvertI64U
	fopF32DemoteF64      opcode = code.OpF32DemoteF64
	fopF64ConvertI32S    opcode = code.OpF64ConvertI32S
	fopF64ConvertI32U    opcode = code.OpF64ConvertI32U
	fopF64ConvertI64S    opcode = code.OpF64ConvertI64S
	fopF64ConvertI64U    opcode = code.OpF64ConvertI64U
	fopF64PromoteF32     opcode = code.OpF64PromoteF32
	fopI32ReinterpretF32 opcode = code.OpI32ReinterpretF32
	fopI64ReinterpretF64 opcode = code.OpI64ReinterpretF64
	fopF32ReinterpretI32 opcode = code.OpF32ReinterpretI32
	fopF64ReinterpretI64 opcode = code.OpF64ReinterpretI64

	fopI32Extend8S  opcode = code.OpI32Extend8S
	fopI32Extend16S opcode = code.OpI32Extend16S
	fopI64Extend8S  opcode = code.OpI64Extend8S
	fopI64Extend16S opcode = code.OpI64Extend16S
	fopI64Extend32S opcode = code.OpI64Extend32S

	fopI32TruncSatF32S opcode = 0x0100 | code.OpI32TruncSatF32S
	fopI32TruncSatF32U opcode = 0x0100 | code.OpI32TruncSatF32U
	fopI32TruncSatF64S opcode = 0x0100 | code.OpI32TruncSatF64S
	fopI32TruncSatF64U opcode = 0x0100 | code.OpI32TruncSatF64U
	fopI64TruncSatF32S opcode = 0x0100 | code.OpI64TruncSatF32S
	fopI64TruncSatF32U opcode = 0x0100 | code.OpI64TruncSatF32U
	fopI64TruncSatF64S opcode = 0x0100 | code.OpI64TruncSatF64S
	fopI64TruncSatF64U opcode = 0x0100 | code.OpI64TruncSatF64U

	fopBrL      opcode = 0x0100 | code.OpBr
	fopBrIfL    opcode = 0x0100 | code.OpBrIf
	fopBrTableL opcode = 0x0100 | code.OpBrTable

	fopBrIfI32Eqz opcode = 0x0100 | code.OpI32Eqz
	fopBrIfI32Eq  opcode = 0x0100 | code.OpI32Eq
	fopBrIfI32Ne  opcode = 0x0100 | code.OpI32Ne
	fopBrIfI32LtS opcode = 0x0100 | code.OpI32LtS
	fopBrIfI32LtU opcode = 0x0100 | code.OpI32LtU
	fopBrIfI32GtS opcode = 0x0100 | code.OpI32GtS
	fopBrIfI32GtU opcode = 0x0100 | code.OpI32GtU
	fopBrIfI32LeS opcode = 0x0100 | code.OpI32LeS
	fopBrIfI32LeU opcode = 0x0100 | code.OpI32LeU
	fopBrIfI32GeS opcode = 0x0100 | code.OpI32GeS
	fopBrIfI32GeU opcode = 0x0100 | code.OpI32GeU

	fopBrIfI64Eqz opcode = 0x0100 | code.OpI64Eqz
	fopBrIfI64Eq  opcode = 0x0100 | code.OpI64Eq
	fopBrIfI64Ne  opcode = 0x0100 | code.OpI64Ne
	fopBrIfI64LtS opcode = 0x0100 | code.OpI64LtS
	fopBrIfI64LtU opcode = 0x0100 | code.OpI64LtU
	fopBrIfI64GtS opcode = 0x0100 | code.OpI64GtS
	fopBrIfI64GtU opcode = 0x0100 | code.OpI64GtU
	fopBrIfI64LeS opcode = 0x0100 | code.OpI64LeS
	fopBrIfI64LeU opcode = 0x0100 | code.OpI64LeU
	fopBrIfI64GeS opcode = 0x0100 | code.OpI64GeS
	fopBrIfI64GeU opcode = 0x0100 | code.OpI64GeU

	fopBrIfF32Eq opcode = 0x0100 | code.OpF32Eq
	fopBrIfF32Ne opcode = 0x0100 | code.OpF32Ne
	fopBrIfF32Lt opcode = 0x0100 | code.OpF32Lt
	fopBrIfF32Gt opcode = 0x0100 | code.OpF32Gt
	fopBrIfF32Le opcode = 0x0100 | code.OpF32Le
	fopBrIfF32Ge opcode = 0x0100 | code.OpF32Ge

	fopBrIfF64Eq opcode = 0x0100 | code.OpF64Eq
	fopBrIfF64Ne opcode = 0x0100 | code.OpF64Ne
	fopBrIfF64Lt opcode = 0x0100 | code.OpF64Lt
	fopBrIfF64Gt opcode = 0x0100 | code.OpF64Gt
	fopBrIfF64Le opcode = 0x0100 | code.OpF64Le
	fopBrIfF64Ge opcode = 0x0100 | code.OpF64Ge

	fopLocalSetI  = 0x0200 | fopLocalSet
	fopGlobalSetI = 0x0200 | fopGlobalSet

	fopI32LoadI    = 0x0200 | fopI32Load
	fopI64LoadI    = 0x0200 | fopI64Load
	fopF32LoadI    = 0x0200 | fopF32Load
	fopF64LoadI    = 0x0200 | fopF64Load
	fopI32Load8SI  = 0x0200 | fopI32Load8S
	fopI32Load8UI  = 0x0200 | fopI32Load8U
	fopI32Load16SI = 0x0200 | fopI32Load16S
	fopI32Load16UI = 0x0200 | fopI32Load16U
	fopI64Load8SI  = 0x0200 | fopI64Load8S
	fopI64Load8UI  = 0x0200 | fopI64Load8U
	fopI64Load16SI = 0x0200 | fopI64Load16S
	fopI64Load16UI = 0x0200 | fopI64Load16U
	fopI64Load32SI = 0x0200 | fopI64Load32S
	fopI64Load32UI = 0x0200 | fopI64Load32U
	fopI32StoreI   = 0x0200 | fopI32Store
	fopI64StoreI   = 0x0200 | fopI64Store
	fopF32StoreI   = 0x0200 | fopF32Store
	fopF64StoreI   = 0x0200 | fopF64Store
	fopI32Store8I  = 0x0200 | fopI32Store8
	fopI32Store16I = 0x0200 | fopI32Store16
	fopI64Store8I  = 0x0200 | fopI64Store8
	fopI64Store16I = 0x0200 | fopI64Store16
	fopI64Store32I = 0x0200 | fopI64Store32

	fopI32EqI  = 0x0200 | fopI32Eq
	fopI32NeI  = 0x0200 | fopI32Ne
	fopI32LtSI = 0x0200 | fopI32LtS
	fopI32LtUI = 0x0200 | fopI32LtU
	fopI32GtSI = 0x0200 | fopI32GtS
	fopI32GtUI = 0x0200 | fopI32GtU
	fopI32LeSI = 0x0200 | fopI32LeS
	fopI32LeUI = 0x0200 | fopI32LeU
	fopI32GeSI = 0x0200 | fopI32GeS
	fopI32GeUI = 0x0200 | fopI32GeU

	fopI64EqI  = 0x0200 | fopI64Eq
	fopI64NeI  = 0x0200 | fopI64Ne
	fopI64LtSI = 0x0200 | fopI64LtS
	fopI64LtUI = 0x0200 | fopI64LtU
	fopI64GtSI = 0x0200 | fopI64GtS
	fopI64GtUI = 0x0200 | fopI64GtU
	fopI64LeSI = 0x0200 | fopI64LeS
	fopI64LeUI = 0x0200 | fopI64LeU
	fopI64GeSI = 0x0200 | fopI64GeS
	fopI64GeUI = 0x0200 | fopI64GeU

	fopF32EqI = 0x0200 | fopF32Eq
	fopF32NeI = 0x0200 | fopF32Ne
	fopF32LtI = 0x0200 | fopF32Lt
	fopF32GtI = 0x0200 | fopF32Gt
	fopF32LeI = 0x0200 | fopF32Le
	fopF32GeI = 0x0200 | fopF32Ge

	fopF64EqI = 0x0200 | fopF64Eq
	fopF64NeI = 0x0200 | fopF64Ne
	fopF64LtI = 0x0200 | fopF64Lt
	fopF64GtI = 0x0200 | fopF64Gt
	fopF64LeI = 0x0200 | fopF64Le
	fopF64GeI = 0x0200 | fopF64Ge

	fopI32AddI  = 0x0200 | fopI32Add
	fopI32SubI  = 0x0200 | fopI32Sub
	fopI32MulI  = 0x0200 | fopI32Mul
	fopI32DivSI = 0x0200 | fopI32DivS
	fopI32DivUI = 0x0200 | fopI32DivU
	fopI32RemSI = 0x0200 | fopI32RemS
	fopI32RemUI = 0x0200 | fopI32RemU
	fopI32AndI  = 0x0200 | fopI32And
	fopI32OrI   = 0x0200 | fopI32Or
	fopI32XorI  = 0x0200 | fopI32Xor
	fopI32ShlI  = 0x0200 | fopI32Shl
	fopI32ShrSI = 0x0200 | fopI32ShrS
	fopI32ShrUI = 0x0200 | fopI32ShrU
	fopI32RotlI = 0x0200 | fopI32Rotl
	fopI32RotrI = 0x0200 | fopI32Rotr

	fopI64AddI  = 0x0200 | fopI64Add
	fopI64SubI  = 0x0200 | fopI64Sub
	fopI64MulI  = 0x0200 | fopI64Mul
	fopI64DivSI = 0x0200 | fopI64DivS
	fopI64DivUI = 0x0200 | fopI64DivU
	fopI64RemSI = 0x0200 | fopI64RemS
	fopI64RemUI = 0x0200 | fopI64RemU
	fopI64AndI  = 0x0200 | fopI64And
	fopI64OrI   = 0x0200 | fopI64Or
	fopI64XorI  = 0x0200 | fopI64Xor
	fopI64ShlI  = 0x0200 | fopI64Shl
	fopI64ShrSI = 0x0200 | fopI64ShrS
	fopI64ShrUI = 0x0200 | fopI64ShrU
	fopI64RotlI = 0x0200 | fopI64Rotl
	fopI64RotrI = 0x0200 | fopI64Rotr

	fopF32AddI      = 0x0200 | fopF32Add
	fopF32SubI      = 0x0200 | fopF32Sub
	fopF32MulI      = 0x0200 | fopF32Mul
	fopF32DivI      = 0x0200 | fopF32Div
	fopF32MinI      = 0x0200 | fopF32Min
	fopF32MaxI      = 0x0200 | fopF32Max
	fopF32CopysignI = 0x0200 | fopF32Copysign

	fopF64AddI      = 0x0200 | fopF64Add
	fopF64SubI      = 0x0200 | fopF64Sub
	fopF64MulI      = 0x0200 | fopF64Mul
	fopF64DivI      = 0x0200 | fopF64Div
	fopF64MinI      = 0x0200 | fopF64Min
	fopF64MaxI      = 0x0200 | fopF64Max
	fopF64CopysignI = 0x0200 | fopF64Copysign

	fopBrIfI32EqzI = 0x0200 | fopBrIfI32Eqz
	fopBrIfI32EqI  = 0x0200 | fopBrIfI32Eq
	fopBrIfI32NeI  = 0x0200 | fopBrIfI32Ne
	fopBrIfI32LtSI = 0x0200 | fopBrIfI32LtS
	fopBrIfI32LtUI = 0x0200 | fopBrIfI32LtU
	fopBrIfI32GtSI = 0x0200 | fopBrIfI32GtS
	fopBrIfI32GtUI = 0x0200 | fopBrIfI32GtU
	fopBrIfI32LeSI = 0x0200 | fopBrIfI32LeS
	fopBrIfI32LeUI = 0x0200 | fopBrIfI32LeU
	fopBrIfI32GeSI = 0x0200 | fopBrIfI32GeS
	fopBrIfI32GeUI = 0x0200 | fopBrIfI32GeU

	fopBrIfI64EqzI = 0x0200 | fopBrIfI64Eqz
	fopBrIfI64EqI  = 0x0200 | fopBrIfI64Eq
	fopBrIfI64NeI  = 0x0200 | fopBrIfI64Ne
	fopBrIfI64LtSI = 0x0200 | fopBrIfI64LtS
	fopBrIfI64LtUI = 0x0200 | fopBrIfI64LtU
	fopBrIfI64GtSI = 0x0200 | fopBrIfI64GtS
	fopBrIfI64GtUI = 0x0200 | fopBrIfI64GtU
	fopBrIfI64LeSI = 0x0200 | fopBrIfI64LeS
	fopBrIfI64LeUI = 0x0200 | fopBrIfI64LeU
	fopBrIfI64GeSI = 0x0200 | fopBrIfI64GeS
	fopBrIfI64GeUI = 0x0200 | fopBrIfI64GeU

	fopBrIfF32EqI = 0x0200 | fopBrIfF32Eq
	fopBrIfF32NeI = 0x0200 | fopBrIfF32Ne
	fopBrIfF32LtI = 0x0200 | fopBrIfF32Lt
	fopBrIfF32GtI = 0x0200 | fopBrIfF32Gt
	fopBrIfF32LeI = 0x0200 | fopBrIfF32Le
	fopBrIfF32GeI = 0x0200 | fopBrIfF32Ge

	fopBrIfF64EqI = 0x0200 | fopBrIfF64Eq
	fopBrIfF64NeI = 0x0200 | fopBrIfF64Ne
	fopBrIfF64LtI = 0x0200 | fopBrIfF64Lt
	fopBrIfF64GtI = 0x0200 | fopBrIfF64Gt
	fopBrIfF64LeI = 0x0200 | fopBrIfF64Le
	fopBrIfF64GeI = 0x0200 | fopBrIfF64Ge
)
