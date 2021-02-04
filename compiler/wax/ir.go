package wax

import (
	"fmt"

	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
	"github.com/willf/bitset"
)

const ValueTypeBool = 1

type Flags int32

const (
	FlagsLoadLocal = 1 << iota
	FlagsLoadGlobal
	FlagsLoadMem
	FlagsStoreLocal
	FlagsStoreGlobal
	FlagsStoreMem
	FlagsMayTrap
	FlagsPseudo

	FlagsBackend = 1 << 16

	FlagsLoadMask  = (FlagsLoadMem << 1) - 1
	FlagsStoreMask = (FlagsStoreMem << 1) - 1
)

func (f Flags) CanMoveAfter(g Flags) bool {
	return (((f&FlagsStoreMask)>>3)&g) == 0 && (((g&FlagsStoreMask)>>3)&f) == 0
}

type Expression struct {
	Function *Function

	IP    int
	Instr code.Instruction
	Uses  Uses
	Flags Flags
}

const (
	PseudoBoolConst = 0 + iota
	PseudoI32ConvertBool

	PseudoBackend = 128
)

func Pseudo(f *Function, opcode byte, immediate uint64, flags Flags, uses ...*Use) *Expression {
	return &Expression{
		Function: f,
		Instr: code.Instruction{
			Opcode:    opcode,
			Immediate: immediate,
		},
		Uses:  uses,
		Flags: flags | FlagsPseudo,
	}
}

func BoolConst(x *Expression) bool {
	return x.Instr.Immediate != 0
}

func (x *Expression) IsPseudo() bool {
	return x.Flags&FlagsPseudo != 0
}

func (x *Expression) Format(f fmt.State, verb rune) {
	x.Function.Formatter.FormatExpression(f, verb, x)
}

type Use struct {
	Function *Function

	Type     wasm.ValueType
	AllFlags Flags
	Locals   bitset.BitSet

	Temp int
	X    *Expression
}

func UseExpression(type_ wasm.ValueType, x *Expression) *Use {
	flags := x.Flags
	for _, u := range x.Uses {
		flags |= u.AllFlags
	}
	return &Use{
		Function: x.Function,
		Type:     type_,
		AllFlags: flags,
		X:        x,
	}
}

func (u *Use) Format(f fmt.State, verb rune) {
	u.Function.Formatter.FormatUse(f, verb, u)
}

func (u *Use) IsTemp() bool {
	return u.X == nil
}

func (u *Use) CanMoveAfter(flags Flags, localStores bitset.BitSet) bool {
	if u.IsTemp() {
		return true
	}

	// If there is global or memory interference, no move is possible.
	if !(u.AllFlags &^ FlagsLoadLocal).CanMoveAfter(flags &^ (FlagsLoadLocal | FlagsStoreLocal)) {
		return false
	}

	// Otherwise, the use can be moved if the local sets do not intersect.
	return u.Locals.IntersectionCardinality(&localStores) == 0
}

func (u *Use) IsConst() bool {
	if !u.IsTemp() {
		switch u.X.Instr.Opcode {
		case code.OpI32Const, code.OpI64Const, code.OpF32Const, code.OpF64Const:
			return true
		}
	}
	return false
}

func (u *Use) IsZeroIConst() bool {
	if !u.IsTemp() {
		switch u.X.Instr.Opcode {
		case code.OpI32Const:
			return u.X.Instr.I32() == 0
		case code.OpI64Const:
			return u.X.Instr.I64() == 0
		}
	}
	return false
}

type Uses []*Use

func (us Uses) Format(f fmt.State, verb rune) {
	if len(us) != 0 {
		us[0].Function.Formatter.FormatUses(f, verb, us)
	}
}

type Block struct {
	Entry *Expression
	Else  *Expression
	End   *Expression

	Label          int
	StackHeight    int
	BranchTarget   bool
	Unreachable    bool
	NeverReachable bool

	Ins  []wasm.ValueType
	Outs []wasm.ValueType

	InTemp  int
	OutTemp int
}

type Def struct {
	*Expression

	Block *Block

	BranchTargets []*Block
	Types         []wasm.ValueType
	Temp          int
}

func (d *Def) Format(f fmt.State, verb rune) {
	d.Function.Formatter.FormatDef(f, verb, d)
}

type Formatter interface {
	FormatExpression(f fmt.State, verb rune, x *Expression)
	FormatUse(f fmt.State, verb rune, u *Use)
	FormatUses(f fmt.State, verb rune, us Uses)
	FormatDef(f fmt.State, verb rune, d *Def)
}
