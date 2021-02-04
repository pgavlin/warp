// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wast

import (
	"fmt"
	"unicode"
)

type Pos struct {
	Line, Column int
}

type Token struct {
	Kind  TokenKind
	Pos   Pos
	Text  string
	Value interface{}
}

func (t *Token) String() string {
	switch t.Kind {
	case EOF:
		return "<EOF>"
	default:
		return fmt.Sprintf("<%v %q>", t.Kind, t.Text)
	}
}

type TokenKind rune

const (
	INVALID = iota + unicode.MaxRune
	ALIGN
	ASSERT_EXHAUSTION
	ASSERT_INVALID
	ASSERT_MALFORMED
	ASSERT_RETURN
	ASSERT_TRAP
	ASSERT_UNLINKABLE
	BINARY
	BLOCK
	BR
	BR_IF
	BR_TABLE
	CALL
	CALL_INDIRECT
	COMPARE
	CONST
	CONVERT
	DATA
	DROP
	ELEM
	ELSE
	END
	EOF
	ERROR
	EXPORT
	F32
	F32_ABS
	F32_ADD
	F32_CEIL
	F32_CONST
	F32_CONVERT_I32_S
	F32_CONVERT_I32_U
	F32_CONVERT_I64_S
	F32_CONVERT_I64_U
	F32_COPYSIGN
	F32_DEMOTE_F64
	F32_DIV
	F32_EQ
	F32_FLOOR
	F32_GE
	F32_GT
	F32_LE
	F32_LOAD
	F32_LT
	F32_MAX
	F32_MIN
	F32_MUL
	F32_NE
	F32_NEAREST
	F32_NEG
	F32_REINTERPRET_I32
	F32_SQRT
	F32_STORE
	F32_SUB
	F32_TRUNC
	F64
	F64_ABS
	F64_ADD
	F64_CEIL
	F64_CONST
	F64_CONVERT_I32_S
	F64_CONVERT_I32_U
	F64_CONVERT_I64_S
	F64_CONVERT_I64_U
	F64_COPYSIGN
	F64_DIV
	F64_EQ
	F64_FLOOR
	F64_GE
	F64_GT
	F64_LE
	F64_LOAD
	F64_LT
	F64_MAX
	F64_MIN
	F64_MUL
	F64_NE
	F64_NEAREST
	F64_NEG
	F64_PROMOTE_F32
	F64_REINTERPRET_I64
	F64_SQRT
	F64_STORE
	F64_SUB
	F64_TRUNC
	FLOAT
	FUNC
	FUNCREF
	GET
	GLOBAL
	GLOBAL_GET
	GLOBAL_SET
	I32
	I32_ADD
	I32_AND
	I32_CLZ
	I32_CONST
	I32_CTZ
	I32_DIV_S
	I32_DIV_U
	I32_EQ
	I32_EQZ
	I32_EXTEND16_S
	I32_EXTEND8_S
	I32_GE_S
	I32_GE_U
	I32_GT_S
	I32_GT_U
	I32_LE_S
	I32_LE_U
	I32_LOAD
	I32_LOAD16_S
	I32_LOAD16_U
	I32_LOAD8_S
	I32_LOAD8_U
	I32_LT_S
	I32_LT_U
	I32_MUL
	I32_NE
	I32_OR
	I32_POPCNT
	I32_REINTERPRET_F32
	I32_REM_S
	I32_REM_U
	I32_ROTL
	I32_ROTR
	I32_SHL
	I32_SHR_S
	I32_SHR_U
	I32_STORE
	I32_STORE16
	I32_STORE8
	I32_SUB
	I32_TRUNC_F32_S
	I32_TRUNC_F32_U
	I32_TRUNC_F64_S
	I32_TRUNC_F64_U
	I32_TRUNC_SAT_F32_S
	I32_TRUNC_SAT_F32_U
	I32_TRUNC_SAT_F64_S
	I32_TRUNC_SAT_F64_U
	I32_WRAP_I64
	I32_XOR
	I64
	I64_ADD
	I64_AND
	I64_CLZ
	I64_CONST
	I64_CTZ
	I64_DIV_S
	I64_DIV_U
	I64_EQ
	I64_EQZ
	I64_EXTEND16_S
	I64_EXTEND32_S
	I64_EXTEND8_S
	I64_EXTEND_I32_S
	I64_EXTEND_I32_U
	I64_GE_S
	I64_GE_U
	I64_GT_S
	I64_GT_U
	I64_LE_S
	I64_LE_U
	I64_LOAD
	I64_LOAD16_S
	I64_LOAD16_U
	I64_LOAD32_S
	I64_LOAD32_U
	I64_LOAD8_S
	I64_LOAD8_U
	I64_LT_S
	I64_LT_U
	I64_MUL
	I64_NE
	I64_OR
	I64_POPCNT
	I64_REINTERPRET_F64
	I64_REM_S
	I64_REM_U
	I64_ROTL
	I64_ROTR
	I64_SHL
	I64_SHR_S
	I64_SHR_U
	I64_STORE
	I64_STORE16
	I64_STORE32
	I64_STORE8
	I64_SUB
	I64_TRUNC_F32_S
	I64_TRUNC_F32_U
	I64_TRUNC_F64_S
	I64_TRUNC_F64_U
	I64_TRUNC_SAT_F32_S
	I64_TRUNC_SAT_F32_U
	I64_TRUNC_SAT_F64_S
	I64_TRUNC_SAT_F64_U
	I64_XOR
	IF
	IMPORT
	INPUT
	INT
	INVOKE
	LOCAL
	LOCAL_GET
	LOCAL_SET
	LOCAL_TEE
	LOOP
	MEMORY
	MEMORY_GROW
	MEMORY_SIZE
	MODULE
	MUT
	NAN_ARITHMETIC
	NAN_CANONICAL
	NAT
	NOP
	OFFSET
	OUTPUT
	PARAM
	QUOTE
	REGISTER
	RESULT
	RETURN
	SCRIPT
	SELECT
	START
	STRING
	TABLE
	TEST
	THEN
	TYPE
	UNARY
	UNREACHABLE
	VALUE_TYPE
	VAR
)

var tokenKindOf = map[string]TokenKind{
	"align":               ALIGN,
	"assert_exhaustion":   ASSERT_EXHAUSTION,
	"assert_invalid":      ASSERT_INVALID,
	"assert_malformed":    ASSERT_MALFORMED,
	"assert_return":       ASSERT_RETURN,
	"assert_trap":         ASSERT_TRAP,
	"assert_unlinkable":   ASSERT_UNLINKABLE,
	"binary":              BINARY,
	"block":               BLOCK,
	"br":                  BR,
	"br_if":               BR_IF,
	"br_table":            BR_TABLE,
	"call":                CALL,
	"call_indirect":       CALL_INDIRECT,
	"data":                DATA,
	"drop":                DROP,
	"elem":                ELEM,
	"else":                ELSE,
	"end":                 END,
	"export":              EXPORT,
	"f32":                 F32,
	"f32.abs":             F32_ABS,
	"f32.add":             F32_ADD,
	"f32.ceil":            F32_CEIL,
	"f32.const":           F32_CONST,
	"f32.convert_i32_s":   F32_CONVERT_I32_S,
	"f32.convert_i32_u":   F32_CONVERT_I32_U,
	"f32.convert_i64_s":   F32_CONVERT_I64_S,
	"f32.convert_i64_u":   F32_CONVERT_I64_U,
	"f32.copysign":        F32_COPYSIGN,
	"f32.demote_f64":      F32_DEMOTE_F64,
	"f32.div":             F32_DIV,
	"f32.eq":              F32_EQ,
	"f32.floor":           F32_FLOOR,
	"f32.ge":              F32_GE,
	"f32.gt":              F32_GT,
	"f32.le":              F32_LE,
	"f32.load":            F32_LOAD,
	"f32.lt":              F32_LT,
	"f32.max":             F32_MAX,
	"f32.min":             F32_MIN,
	"f32.mul":             F32_MUL,
	"f32.ne":              F32_NE,
	"f32.nearest":         F32_NEAREST,
	"f32.neg":             F32_NEG,
	"f32.reinterpret_i32": F32_REINTERPRET_I32,
	"f32.sqrt":            F32_SQRT,
	"f32.store":           F32_STORE,
	"f32.sub":             F32_SUB,
	"f32.trunc":           F32_TRUNC,
	"f64":                 F64,
	"f64.abs":             F64_ABS,
	"f64.add":             F64_ADD,
	"f64.ceil":            F64_CEIL,
	"f64.const":           F64_CONST,
	"f64.convert_i32_s":   F64_CONVERT_I32_S,
	"f64.convert_i32_u":   F64_CONVERT_I32_U,
	"f64.convert_i64_s":   F64_CONVERT_I64_S,
	"f64.convert_i64_u":   F64_CONVERT_I64_U,
	"f64.copysign":        F64_COPYSIGN,
	"f64.div":             F64_DIV,
	"f64.eq":              F64_EQ,
	"f64.floor":           F64_FLOOR,
	"f64.ge":              F64_GE,
	"f64.gt":              F64_GT,
	"f64.le":              F64_LE,
	"f64.load":            F64_LOAD,
	"f64.lt":              F64_LT,
	"f64.max":             F64_MAX,
	"f64.min":             F64_MIN,
	"f64.mul":             F64_MUL,
	"f64.ne":              F64_NE,
	"f64.nearest":         F64_NEAREST,
	"f64.neg":             F64_NEG,
	"f64.promote_f32":     F64_PROMOTE_F32,
	"f64.reinterpret_i64": F64_REINTERPRET_I64,
	"f64.sqrt":            F64_SQRT,
	"f64.store":           F64_STORE,
	"f64.sub":             F64_SUB,
	"f64.trunc":           F64_TRUNC,
	"func":                FUNC,
	"funcref":             FUNCREF,
	"get":                 GET,
	"global":              GLOBAL,
	"global.get":          GLOBAL_GET,
	"global.set":          GLOBAL_SET,
	"i32":                 I32,
	"i32.add":             I32_ADD,
	"i32.and":             I32_AND,
	"i32.clz":             I32_CLZ,
	"i32.const":           I32_CONST,
	"i32.ctz":             I32_CTZ,
	"i32.div_s":           I32_DIV_S,
	"i32.div_u":           I32_DIV_U,
	"i32.eq":              I32_EQ,
	"i32.eqz":             I32_EQZ,
	"i32.extend16_s":      I32_EXTEND16_S,
	"i32.extend8_s":       I32_EXTEND8_S,
	"i32.ge_s":            I32_GE_S,
	"i32.ge_u":            I32_GE_U,
	"i32.gt_s":            I32_GT_S,
	"i32.gt_u":            I32_GT_U,
	"i32.le_s":            I32_LE_S,
	"i32.le_u":            I32_LE_U,
	"i32.load":            I32_LOAD,
	"i32.load16_s":        I32_LOAD16_S,
	"i32.load16_u":        I32_LOAD16_U,
	"i32.load8_s":         I32_LOAD8_S,
	"i32.load8_u":         I32_LOAD8_U,
	"i32.lt_s":            I32_LT_S,
	"i32.lt_u":            I32_LT_U,
	"i32.mul":             I32_MUL,
	"i32.ne":              I32_NE,
	"i32.or":              I32_OR,
	"i32.popcnt":          I32_POPCNT,
	"i32.reinterpret_f32": I32_REINTERPRET_F32,
	"i32.rem_s":           I32_REM_S,
	"i32.rem_u":           I32_REM_U,
	"i32.rotl":            I32_ROTL,
	"i32.rotr":            I32_ROTR,
	"i32.shl":             I32_SHL,
	"i32.shr_s":           I32_SHR_S,
	"i32.shr_u":           I32_SHR_U,
	"i32.store":           I32_STORE,
	"i32.store16":         I32_STORE16,
	"i32.store8":          I32_STORE8,
	"i32.sub":             I32_SUB,
	"i32.trunc_f32_s":     I32_TRUNC_F32_S,
	"i32.trunc_f32_u":     I32_TRUNC_F32_U,
	"i32.trunc_f64_s":     I32_TRUNC_F64_S,
	"i32.trunc_f64_u":     I32_TRUNC_F64_U,
	"i32.trunc_sat_f32_s": I32_TRUNC_SAT_F32_S,
	"i32.trunc_sat_f32_u": I32_TRUNC_SAT_F32_U,
	"i32.trunc_sat_f64_s": I32_TRUNC_SAT_F64_S,
	"i32.trunc_sat_f64_u": I32_TRUNC_SAT_F64_U,
	"i32.wrap_i64":        I32_WRAP_I64,
	"i32.xor":             I32_XOR,
	"i64":                 I64,
	"i64.add":             I64_ADD,
	"i64.and":             I64_AND,
	"i64.clz":             I64_CLZ,
	"i64.const":           I64_CONST,
	"i64.ctz":             I64_CTZ,
	"i64.div_s":           I64_DIV_S,
	"i64.div_u":           I64_DIV_U,
	"i64.eq":              I64_EQ,
	"i64.eqz":             I64_EQZ,
	"i64.extend16_s":      I64_EXTEND16_S,
	"i64.extend32_s":      I64_EXTEND32_S,
	"i64.extend8_s":       I64_EXTEND8_S,
	"i64.extend_i32_s":    I64_EXTEND_I32_S,
	"i64.extend_i32_u":    I64_EXTEND_I32_U,
	"i64.ge_s":            I64_GE_S,
	"i64.ge_u":            I64_GE_U,
	"i64.gt_s":            I64_GT_S,
	"i64.gt_u":            I64_GT_U,
	"i64.le_s":            I64_LE_S,
	"i64.le_u":            I64_LE_U,
	"i64.load":            I64_LOAD,
	"i64.load16_s":        I64_LOAD16_S,
	"i64.load16_u":        I64_LOAD16_U,
	"i64.load32_s":        I64_LOAD32_S,
	"i64.load32_u":        I64_LOAD32_U,
	"i64.load8_s":         I64_LOAD8_S,
	"i64.load8_u":         I64_LOAD8_U,
	"i64.lt_s":            I64_LT_S,
	"i64.lt_u":            I64_LT_U,
	"i64.mul":             I64_MUL,
	"i64.ne":              I64_NE,
	"i64.or":              I64_OR,
	"i64.popcnt":          I64_POPCNT,
	"i64.reinterpret_f64": I64_REINTERPRET_F64,
	"i64.rem_s":           I64_REM_S,
	"i64.rem_u":           I64_REM_U,
	"i64.rotl":            I64_ROTL,
	"i64.rotr":            I64_ROTR,
	"i64.shl":             I64_SHL,
	"i64.shr_s":           I64_SHR_S,
	"i64.shr_u":           I64_SHR_U,
	"i64.store":           I64_STORE,
	"i64.store16":         I64_STORE16,
	"i64.store32":         I64_STORE32,
	"i64.store8":          I64_STORE8,
	"i64.sub":             I64_SUB,
	"i64.trunc_f32_s":     I64_TRUNC_F32_S,
	"i64.trunc_f32_u":     I64_TRUNC_F32_U,
	"i64.trunc_f64_s":     I64_TRUNC_F64_S,
	"i64.trunc_f64_u":     I64_TRUNC_F64_U,
	"i64.trunc_sat_f32_s": I64_TRUNC_SAT_F32_S,
	"i64.trunc_sat_f32_u": I64_TRUNC_SAT_F32_U,
	"i64.trunc_sat_f64_s": I64_TRUNC_SAT_F64_S,
	"i64.trunc_sat_f64_u": I64_TRUNC_SAT_F64_U,
	"i64.xor":             I64_XOR,
	"if":                  IF,
	"import":              IMPORT,
	"input":               INPUT,
	"invoke":              INVOKE,
	"local":               LOCAL,
	"local.get":           LOCAL_GET,
	"local.set":           LOCAL_SET,
	"local.tee":           LOCAL_TEE,
	"loop":                LOOP,
	"memory":              MEMORY,
	"memory.grow":         MEMORY_GROW,
	"memory.size":         MEMORY_SIZE,
	"module":              MODULE,
	"mut":                 MUT,
	"nan:arithmetic":      NAN_ARITHMETIC,
	"nan:canonical":       NAN_CANONICAL,
	"nop":                 NOP,
	"offset":              OFFSET,
	"output":              OUTPUT,
	"param":               PARAM,
	"quote":               QUOTE,
	"register":            REGISTER,
	"result":              RESULT,
	"return":              RETURN,
	"script":              SCRIPT,
	"select":              SELECT,
	"start":               START,
	"table":               TABLE,
	"then":                THEN,
	"type":                TYPE,
	"unreachable":         UNREACHABLE,
}

func (t TokenKind) String() string {
	switch t {
	case ALIGN:
		return "ALIGN"
	case ASSERT_EXHAUSTION:
		return "ASSERT_EXHAUSTION"
	case ASSERT_INVALID:
		return "ASSERT_INVALID"
	case ASSERT_MALFORMED:
		return "ASSERT_MALFORMED"
	case ASSERT_RETURN:
		return "ASSERT_RETURN"
	case ASSERT_TRAP:
		return "ASSERT_TRAP"
	case ASSERT_UNLINKABLE:
		return "ASSERT_UNLINKABLE"
	case BINARY:
		return "BINARY"
	case BLOCK:
		return "BLOCK"
	case BR:
		return "BR"
	case BR_IF:
		return "BR_IF"
	case BR_TABLE:
		return "BR_TABLE"
	case CALL:
		return "CALL"
	case CALL_INDIRECT:
		return "CALL_INDIRECT"
	case COMPARE:
		return "COMPARE"
	case CONST:
		return "CONST"
	case CONVERT:
		return "CONVERT"
	case DATA:
		return "DATA"
	case DROP:
		return "DROP"
	case ELEM:
		return "ELEM"
	case ELSE:
		return "ELSE"
	case END:
		return "END"
	case EOF:
		return "EOF"
	case ERROR:
		return "ERROR"
	case EXPORT:
		return "EXPORT"
	case F32:
		return "F32"
	case F32_ABS:
		return "F32_ABS"
	case F32_ADD:
		return "F32_ADD"
	case F32_CEIL:
		return "F32_CEIL"
	case F32_CONST:
		return "F32_CONST"
	case F32_CONVERT_I32_S:
		return "F32_CONVERT_I32_S"
	case F32_CONVERT_I32_U:
		return "F32_CONVERT_I32_U"
	case F32_CONVERT_I64_S:
		return "F32_CONVERT_I64_S"
	case F32_CONVERT_I64_U:
		return "F32_CONVERT_I64_U"
	case F32_COPYSIGN:
		return "F32_COPYSIGN"
	case F32_DEMOTE_F64:
		return "F32_DEMOTE_F64"
	case F32_DIV:
		return "F32_DIV"
	case F32_EQ:
		return "F32_EQ"
	case F32_FLOOR:
		return "F32_FLOOR"
	case F32_GE:
		return "F32_GE"
	case F32_GT:
		return "F32_GT"
	case F32_LE:
		return "F32_LE"
	case F32_LOAD:
		return "F32_LOAD"
	case F32_LT:
		return "F32_LT"
	case F32_MAX:
		return "F32_MAX"
	case F32_MIN:
		return "F32_MIN"
	case F32_MUL:
		return "F32_MUL"
	case F32_NE:
		return "F32_NE"
	case F32_NEAREST:
		return "F32_NEAREST"
	case F32_NEG:
		return "F32_NEG"
	case F32_REINTERPRET_I32:
		return "F32_REINTERPRET_I32"
	case F32_SQRT:
		return "F32_SQRT"
	case F32_STORE:
		return "F32_STORE"
	case F32_SUB:
		return "F32_SUB"
	case F32_TRUNC:
		return "F32_TRUNC"
	case F64:
		return "F64"
	case F64_ABS:
		return "F64_ABS"
	case F64_ADD:
		return "F64_ADD"
	case F64_CEIL:
		return "F64_CEIL"
	case F64_CONST:
		return "F64_CONST"
	case F64_CONVERT_I32_S:
		return "F64_CONVERT_I32_S"
	case F64_CONVERT_I32_U:
		return "F64_CONVERT_I32_U"
	case F64_CONVERT_I64_S:
		return "F64_CONVERT_I64_S"
	case F64_CONVERT_I64_U:
		return "F64_CONVERT_I64_U"
	case F64_COPYSIGN:
		return "F64_COPYSIGN"
	case F64_DIV:
		return "F64_DIV"
	case F64_EQ:
		return "F64_EQ"
	case F64_FLOOR:
		return "F64_FLOOR"
	case F64_GE:
		return "F64_GE"
	case F64_GT:
		return "F64_GT"
	case F64_LE:
		return "F64_LE"
	case F64_LOAD:
		return "F64_LOAD"
	case F64_LT:
		return "F64_LT"
	case F64_MAX:
		return "F64_MAX"
	case F64_MIN:
		return "F64_MIN"
	case F64_MUL:
		return "F64_MUL"
	case F64_NE:
		return "F64_NE"
	case F64_NEAREST:
		return "F64_NEAREST"
	case F64_NEG:
		return "F64_NEG"
	case F64_PROMOTE_F32:
		return "F64_PROMOTE_F32"
	case F64_REINTERPRET_I64:
		return "F64_REINTERPRET_I64"
	case F64_SQRT:
		return "F64_SQRT"
	case F64_STORE:
		return "F64_STORE"
	case F64_SUB:
		return "F64_SUB"
	case F64_TRUNC:
		return "F64_TRUNC"
	case FLOAT:
		return "FLOAT"
	case FUNC:
		return "FUNC"
	case FUNCREF:
		return "FUNCREF"
	case GET:
		return "GET"
	case GLOBAL:
		return "GLOBAL"
	case GLOBAL_GET:
		return "GLOBAL_GET"
	case GLOBAL_SET:
		return "GLOBAL_SET"
	case I32:
		return "I32"
	case I32_ADD:
		return "I32_ADD"
	case I32_AND:
		return "I32_AND"
	case I32_CLZ:
		return "I32_CLZ"
	case I32_CONST:
		return "I32_CONST"
	case I32_CTZ:
		return "I32_CTZ"
	case I32_DIV_S:
		return "I32_DIV_S"
	case I32_DIV_U:
		return "I32_DIV_U"
	case I32_EQ:
		return "I32_EQ"
	case I32_EQZ:
		return "I32_EQZ"
	case I32_EXTEND16_S:
		return "I32_EXTEND16_S"
	case I32_EXTEND8_S:
		return "I32_EXTEND8_S"
	case I32_GE_S:
		return "I32_GE_S"
	case I32_GE_U:
		return "I32_GE_U"
	case I32_GT_S:
		return "I32_GT_S"
	case I32_GT_U:
		return "I32_GT_U"
	case I32_LE_S:
		return "I32_LE_S"
	case I32_LE_U:
		return "I32_LE_U"
	case I32_LOAD:
		return "I32_LOAD"
	case I32_LOAD16_S:
		return "I32_LOAD16_S"
	case I32_LOAD16_U:
		return "I32_LOAD16_U"
	case I32_LOAD8_S:
		return "I32_LOAD8_S"
	case I32_LOAD8_U:
		return "I32_LOAD8_U"
	case I32_LT_S:
		return "I32_LT_S"
	case I32_LT_U:
		return "I32_LT_U"
	case I32_MUL:
		return "I32_MUL"
	case I32_NE:
		return "I32_NE"
	case I32_OR:
		return "I32_OR"
	case I32_POPCNT:
		return "I32_POPCNT"
	case I32_REINTERPRET_F32:
		return "I32_REINTERPRET_F32"
	case I32_REM_S:
		return "I32_REM_S"
	case I32_REM_U:
		return "I32_REM_U"
	case I32_ROTL:
		return "I32_ROTL"
	case I32_ROTR:
		return "I32_ROTR"
	case I32_SHL:
		return "I32_SHL"
	case I32_SHR_S:
		return "I32_SHR_S"
	case I32_SHR_U:
		return "I32_SHR_U"
	case I32_STORE:
		return "I32_STORE"
	case I32_STORE16:
		return "I32_STORE16"
	case I32_STORE8:
		return "I32_STORE8"
	case I32_SUB:
		return "I32_SUB"
	case I32_TRUNC_F32_S:
		return "I32_TRUNC_F32_S"
	case I32_TRUNC_F32_U:
		return "I32_TRUNC_F32_U"
	case I32_TRUNC_F64_S:
		return "I32_TRUNC_F64_S"
	case I32_TRUNC_F64_U:
		return "I32_TRUNC_F64_U"
	case I32_TRUNC_SAT_F32_S:
		return "I32_TRUNC_SAT_F32_S"
	case I32_TRUNC_SAT_F32_U:
		return "I32_TRUNC_SAT_F32_U"
	case I32_TRUNC_SAT_F64_S:
		return "I32_TRUNC_SAT_F64_S"
	case I32_TRUNC_SAT_F64_U:
		return "I32_TRUNC_SAT_F64_U"
	case I32_WRAP_I64:
		return "I32_WRAP_I64"
	case I32_XOR:
		return "I32_XOR"
	case I64:
		return "I64"
	case I64_ADD:
		return "I64_ADD"
	case I64_AND:
		return "I64_AND"
	case I64_CLZ:
		return "I64_CLZ"
	case I64_CONST:
		return "I64_CONST"
	case I64_CTZ:
		return "I64_CTZ"
	case I64_DIV_S:
		return "I64_DIV_S"
	case I64_DIV_U:
		return "I64_DIV_U"
	case I64_EQ:
		return "I64_EQ"
	case I64_EQZ:
		return "I64_EQZ"
	case I64_EXTEND16_S:
		return "I64_EXTEND16_S"
	case I64_EXTEND32_S:
		return "I64_EXTEND32_S"
	case I64_EXTEND8_S:
		return "I64_EXTEND8_S"
	case I64_EXTEND_I32_S:
		return "I64_EXTEND_I32_S"
	case I64_EXTEND_I32_U:
		return "I64_EXTEND_I32_U"
	case I64_GE_S:
		return "I64_GE_S"
	case I64_GE_U:
		return "I64_GE_U"
	case I64_GT_S:
		return "I64_GT_S"
	case I64_GT_U:
		return "I64_GT_U"
	case I64_LE_S:
		return "I64_LE_S"
	case I64_LE_U:
		return "I64_LE_U"
	case I64_LOAD:
		return "I64_LOAD"
	case I64_LOAD16_S:
		return "I64_LOAD16_S"
	case I64_LOAD16_U:
		return "I64_LOAD16_U"
	case I64_LOAD32_S:
		return "I64_LOAD32_S"
	case I64_LOAD32_U:
		return "I64_LOAD32_U"
	case I64_LOAD8_S:
		return "I64_LOAD8_S"
	case I64_LOAD8_U:
		return "I64_LOAD8_U"
	case I64_LT_S:
		return "I64_LT_S"
	case I64_LT_U:
		return "I64_LT_U"
	case I64_MUL:
		return "I64_MUL"
	case I64_NE:
		return "I64_NE"
	case I64_OR:
		return "I64_OR"
	case I64_POPCNT:
		return "I64_POPCNT"
	case I64_REINTERPRET_F64:
		return "I64_REINTERPRET_F64"
	case I64_REM_S:
		return "I64_REM_S"
	case I64_REM_U:
		return "I64_REM_U"
	case I64_ROTL:
		return "I64_ROTL"
	case I64_ROTR:
		return "I64_ROTR"
	case I64_SHL:
		return "I64_SHL"
	case I64_SHR_S:
		return "I64_SHR_S"
	case I64_SHR_U:
		return "I64_SHR_U"
	case I64_STORE:
		return "I64_STORE"
	case I64_STORE16:
		return "I64_STORE16"
	case I64_STORE32:
		return "I64_STORE32"
	case I64_STORE8:
		return "I64_STORE8"
	case I64_SUB:
		return "I64_SUB"
	case I64_TRUNC_F32_S:
		return "I64_TRUNC_F32_S"
	case I64_TRUNC_F32_U:
		return "I64_TRUNC_F32_U"
	case I64_TRUNC_F64_S:
		return "I64_TRUNC_F64_S"
	case I64_TRUNC_F64_U:
		return "I64_TRUNC_F64_U"
	case I64_TRUNC_SAT_F32_S:
		return "I64_TRUNC_SAT_F32_S"
	case I64_TRUNC_SAT_F32_U:
		return "I64_TRUNC_SAT_F32_U"
	case I64_TRUNC_SAT_F64_S:
		return "I64_TRUNC_SAT_F64_S"
	case I64_TRUNC_SAT_F64_U:
		return "I64_TRUNC_SAT_F64_U"
	case I64_XOR:
		return "I64_XOR"
	case IF:
		return "IF"
	case IMPORT:
		return "IMPORT"
	case INPUT:
		return "INPUT"
	case INT:
		return "INT"
	case INVALID:
		return "INVALID"
	case INVOKE:
		return "INVOKE"
	case LOCAL:
		return "LOCAL"
	case LOCAL_GET:
		return "LOCAL_GET"
	case LOCAL_SET:
		return "LOCAL_SET"
	case LOCAL_TEE:
		return "LOCAL_TEE"
	case LOOP:
		return "LOOP"
	case MEMORY:
		return "MEMORY"
	case MEMORY_GROW:
		return "MEMORY_GROW"
	case MEMORY_SIZE:
		return "MEMORY_SIZE"
	case MODULE:
		return "MODULE"
	case MUT:
		return "MUT"
	case NAN_ARITHMETIC:
		return "NAN_ARITHMETIC"
	case NAN_CANONICAL:
		return "NAN_CANONICAL"
	case NAT:
		return "NAT"
	case NOP:
		return "NOP"
	case OFFSET:
		return "OFFSET"
	case OUTPUT:
		return "OUTPUT"
	case PARAM:
		return "PARAM"
	case QUOTE:
		return "QUOTE"
	case REGISTER:
		return "REGISTER"
	case RESULT:
		return "RESULT"
	case RETURN:
		return "RETURN"
	case SCRIPT:
		return "SCRIPT"
	case SELECT:
		return "SELECT"
	case START:
		return "START"
	case STRING:
		return "STRING"
	case TABLE:
		return "TABLE"
	case TEST:
		return "TEST"
	case THEN:
		return "THEN"
	case TYPE:
		return "TYPE"
	case UNARY:
		return "UNARY"
	case UNREACHABLE:
		return "UNREACHABLE"
	case VALUE_TYPE:
		return "VALUE_TYPE"
	case VAR:
		return "VAR"
	default:
		return string([]rune{rune(t)})
	}
}
