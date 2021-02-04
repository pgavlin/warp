package wast

import (
	"fmt"
	"math"
	"math/big"
)

func any(t TokenKind, x []TokenKind) bool {
	for _, k := range x {
		if t == k {
			return true
		}
	}
	return false
}

type parser struct {
	s   *Scanner
	tok *Token
}

func (p *parser) start() {
	p.scan()
	p.scan()
}

func (p *parser) scan() {
	p.tok = p.s.token()
	if _, err := p.s.Scan(); err != nil {
		panic(err)
	}
}

func (p *parser) peek() TokenKind {
	return p.s.tok
}

func (p *parser) peekSExpr(word TokenKind) bool {
	return p.tok.Kind == '(' && p.peek() == word
}

func (p *parser) scanSExpr(word TokenKind) bool {
	if p.peekSExpr(word) {
		p.scan()
		p.scan()
		return true
	}
	return false
}

func (p *parser) expectSExpr(word TokenKind) {
	p.expect('(', word)
}

func (p *parser) closeSExpr() {
	p.expect(')')
}

func (p *parser) expectI(kinds ...TokenKind) int64 {
	b := p.expect(kinds...).(*BigInt)
	v, err := b.I()
	if err != nil {
		panic(err)
	}
	return v
}

func (p *parser) I32() (int32, bool) {
	v, ok := p.tok.Value.(*BigInt)
	if !ok {
		return 0, false
	}
	i, err := v.I()
	if err != nil {
		panic(err)
	}

	// TODO: range checks
	return int32(i), true
}

func (p *parser) I64() (int64, bool) {
	v, ok := p.tok.Value.(*BigInt)
	if !ok {
		return 0, false
	}
	i, err := v.I()
	if err != nil {
		panic(err)
	}
	return i, true
}

func (p *parser) F32() (float32, bool) {
	switch v := p.tok.Value.(type) {
	case *BigInt:
		bf, err := v.F()
		if err != nil {
			panic(err)
		}
		f, _ := bf.Float32()
		return f, true
	case *big.Float:
		// TODO: range checks
		f, _ := v.Float32()
		return f, true
	case float64:
		if !math.IsNaN(v) {
			return float32(v), true
		}

		bits := math.Float64bits(v)
		sign := uint32(bits >> 63)
		payload := uint32(bits&0x7fffff) | uint32(bits>>29)&0x00400000
		return math.Float32frombits(sign<<31 | 0x7f800000 | payload), true
	default:
		return 0, false
	}
}

func (p *parser) F64() (float64, bool) {
	switch v := p.tok.Value.(type) {
	case *BigInt:
		bf, err := v.F()
		if err != nil {
			panic(err)
		}
		f, _ := bf.Float64()
		return f, true
	case *big.Float:
		// TODO: range checks
		f, _ := v.Float64()
		return f, true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func (p *parser) errorf(s string, args ...interface{}) error {
	return fmt.Errorf("%v,%v: %s", p.tok.Pos.Line, p.tok.Pos.Column, fmt.Sprintf(s, args...))
}

func (p *parser) expect(kinds ...TokenKind) interface{} {
	var v interface{}
	for _, k := range kinds {
		if p.tok.Kind != k {
			panic(p.errorf("expected %v", k))
		}
		v = p.tok.Value
		p.scan()
	}
	return v
}

func (p *parser) maybe(kinds ...TokenKind) interface{} {
	var v interface{}
	for _, k := range kinds {
		if p.tok.Kind != k {
			break
		}
		v = p.tok.Value
		p.scan()
	}
	return v
}
