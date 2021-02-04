// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wast

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"math/bits"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Scanner struct {
	r io.Reader

	buf [2]rune
	err [2]error
	nb  int

	line, column int

	tok   TokenKind
	text  bytes.Buffer
	value interface{}
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: r, line: 1}
}

func (s *Scanner) Pos() Pos {
	return Pos{Line: s.line, Column: s.column}
}

func (s *Scanner) Text() string {
	return s.text.String()
}

func (s *Scanner) Value() interface{} {
	return s.value
}

func (s *Scanner) readRune() (rune, error) {
	var buf [4]byte

	if _, err := s.r.Read(buf[:1]); err != nil {
		return 0, err
	}

	if !utf8.RuneStart(buf[0]) {
		return utf8.RuneError, nil
	}

	sz := bits.LeadingZeros8(^buf[0])
	if sz == 0 {
		return rune(buf[0]), nil
	}

	if _, err := s.r.Read(buf[1:sz]); err != nil {
		return 0, err
	}

	c, _ := utf8.DecodeRune(buf[:sz])
	return c, nil
}

func (s *Scanner) peek() rune {
	if s.nb == 0 {
		s.buf[0], s.err[0] = s.readRune()
		s.nb = 1
	}
	return s.buf[0]
}

func (s *Scanner) peek2() (rune, rune) {
	for s.nb < 2 {
		s.buf[s.nb], s.err[s.nb] = s.readRune()
		s.nb++
	}
	return s.buf[0], s.buf[1]
}

func (s *Scanner) skip() {
	if s.nb == 0 {
		panic("expected a buffered rune")
	}
	s.nb--
	s.buf[0] = s.buf[1]
	s.column++
}

func (s *Scanner) chomp() rune {
	if s.nb == 0 {
		s.peek()
	}
	r := s.buf[0]
	s.skip()
	s.text.WriteRune(r)
	return r
}

func (s *Scanner) chomp2() {
	s.chomp()
	s.chomp()
}

func (s *Scanner) expect(str string) bool {
	for _, r := range str {
		if s.peek() != r {
			return false
		}
		s.chomp()
	}
	return true
}

func (s *Scanner) scanNum(b *strings.Builder) {
	for {
		m, n := s.peek2()
		if m == '_' && isDigit(n) {
			s.chomp2()
			b.WriteRune(n)
		} else if isDigit(m) {
			s.chomp()
			b.WriteRune(m)
		} else {
			break
		}
	}
}

func (s *Scanner) scanHexNum(b *strings.Builder) uint64 {
	v := uint64(0)
	for {
		m, n := s.peek2()
		if m == '_' && isHexDigit(n) {
			s.chomp2()
			b.WriteByte(byte(n))
			v = v<<4 | hexNibble(n)
		} else if isHexDigit(m) {
			s.chomp()
			b.WriteByte(byte(m))
			v = v<<4 | hexNibble(m)
		} else {
			break
		}
	}
	return v
}

func (s *Scanner) scanNat(b *strings.Builder) {
	m, n := s.peek2()
	if m == '0' && n == 'x' {
		s.chomp2()
		s.scanHexNum(b)
	} else {
		s.scanNum(b)
	}
}

func (s *Scanner) scanHexNumeric(b *strings.Builder) TokenKind {
	s.chomp2()
	s.scanHexNum(b)

	n := s.peek()
	if n != '.' && n != 'p' && n != 'P' {
		return s.bigInt(b.String(), 16)
	}

	if n == '.' {
		s.chomp()
		b.WriteRune('.')
		s.scanHexNum(b)
		n = s.peek()
	}

	if n == 'p' || n == 'P' {
		s.chomp()
		b.WriteRune(n)
		if sign := s.peek(); sign == '-' || sign == '+' {
			s.chomp()
			b.WriteRune(sign)
		}
		s.scanNum(b)
	}

	return s.bigFloat(b.String(), 16)
}

func (s *Scanner) scanInf(sign rune) TokenKind {
	if !s.expect("inf") {
		return ERROR
	}
	if sign == '-' {
		s.value = math.Inf(-1)
	} else {
		s.value = math.Inf(0)
	}
	return FLOAT
}

func (s *Scanner) scanNan(sign rune) TokenKind {
	if !s.expect("nan") {
		return ERROR
	}

	var bits uint64
	if s.peek() == ':' {
		s.chomp()
		s.chomp2()

		var b strings.Builder
		s.scanHexNum(&b)

		v, err := strconv.ParseUint(b.String(), 16, 0)
		if err != nil {
			s.value = err
			return ERROR
		}

		bits = 0x7ff0000000000000 | v&0x000fffffffffffff
	} else {
		bits = 0x7ff8000000000000
	}

	if sign == '-' {
		bits |= 0x8000000000000000
	}
	s.value = math.Float64frombits(bits)
	return FLOAT
}

func (s *Scanner) scanNumeric() TokenKind {
	var b strings.Builder

	// Already positioned at a '+', '-', or digit
	sign := s.peek()
	if sign == '+' || sign == '-' {
		s.chomp()
		b.WriteRune(sign)
	}

	m, n := s.peek2()
	if m == '0' && n == 'x' {
		return s.scanHexNumeric(&b)
	}
	if m == 'i' {
		return s.scanInf(sign)
	}
	if m == 'n' {
		return s.scanNan(sign)
	}
	if !isDigit(m) {
		return TokenKind(sign)
	}

	s.scanNum(&b)

	n = s.peek()
	if n != '.' && n != 'e' && n != 'E' {
		return s.bigInt(b.String(), 10)
	}

	if n == '.' {
		s.chomp()
		b.WriteRune('.')
		s.scanNum(&b)
		n = s.peek()
	}

	if n == 'e' || n == 'E' {
		s.chomp()
		b.WriteRune(n)
		if sign := s.peek(); sign == '-' || sign == '+' {
			s.chomp()
			b.WriteRune(sign)
		}
		s.scanNum(&b)
	}

	return s.bigFloat(b.String(), 10)
}

func (s *Scanner) scanString() TokenKind {
	// Already positioned at a '"'
	s.chomp()

	var b strings.Builder
	for {
		n := s.chomp()
		if n == '"' {
			break
		}
		if n != '\\' {
			b.WriteRune(n)
			continue
		}

		n = s.chomp()
		switch n {
		case 'n':
			b.WriteRune('\n')
		case 'r':
			b.WriteRune('\r')
		case 't':
			b.WriteRune('\t')
		case '\\', '\'', '"':
			b.WriteRune(n)
		case 'u':
			b.WriteRune(rune(s.scanHexNum(&strings.Builder{})))
		default:
			if isHexDigit(n) {
				hi, lo := n, s.chomp()
				b.WriteByte(byte(hexNibble(hi)<<4 | hexNibble(lo)))
			} else {
				b.WriteRune('\\')
				b.WriteRune(n)
			}
		}
	}

	s.value = b.String()
	return STRING
}

func (s *Scanner) scanName() TokenKind {
	// Already positioned at a '$'
	s.chomp()

	for {
		n := s.peek()
		if !isLetter(n) && !isDigit(n) && !isSymbol(n) && n != '_' {
			break
		}
		s.chomp()
	}

	s.value = s.text.String()
	return VAR
}

func (s *Scanner) scanKeyword() TokenKind {
	// Already positioned at a letter
	s.chomp()

	for {
		n := s.peek()
		if n == '=' && (bytes.Equal(s.text.Bytes(), []byte("offset")) || bytes.Equal(s.text.Bytes(), []byte("align"))) {
			break
		}
		if !isLetter(n) && !isDigit(n) && !isSymbol(n) && n != '_' {
			break
		}
		s.chomp()
	}

	kw := s.text.String()
	if tk, ok := tokenKindOf[kw]; ok {
		return tk
	}

	switch kw {
	case "inf":
		s.value = math.Inf(0)
		return FLOAT
	case "nan":
		s.value = math.Float64frombits(0x7ff8000000000000)
		return FLOAT
	}
	if strings.HasPrefix(kw, "nan:0x") {
		if v, ok := hexNum(kw[len("nan:0x"):]); ok {
			s.value = math.Float64frombits(0x7ff0000000000000 | v&0x000fffffffffffff)
			return FLOAT
		}
	}

	s.value = kw
	return ERROR
}

func (s *Scanner) scanLineComment() {
	// Already positioned at ;;
	s.skip()
	s.skip()

	for {
		n := s.peek()
		s.skip()
		if n == '\n' || n == 0 {
			s.line++
			s.column = 0
			break
		}
	}
}

func (s *Scanner) scanBlockComment() {
	// Already positioned at (;
	s.skip()
	s.skip()

	nest := 1
	for {
		m, n := s.peek2()
		if m == '(' && n == ';' {
			s.skip()
			s.skip()
			nest++
		} else if m == ';' && n == ')' {
			s.skip()
			s.skip()
			nest--
			if nest == 0 {
				break
			}
		} else if m == 0 {
			break
		} else {
			if m == '\n' {
				s.line++
				s.column = 0
			}
			s.skip()
		}
	}
}

func (s *Scanner) scan() (TokenKind, error) {
	s.text.Reset()
	s.value = nil

	for {
		m, n := s.peek2()
		switch {
		case isDigit(m) || m == '-' || m == '+':
			return s.scanNumeric(), nil
		case m == '$':
			return s.scanName(), nil
		case m >= 'a' && m <= 'z':
			return s.scanKeyword(), nil
		case m == '"':
			return s.scanString(), nil
		case m == ';' && n == ';':
			s.scanLineComment()
		case m == '(' && n == ';':
			s.scanBlockComment()
		case isSymbol(m) || m == '(' || m == ')':
			s.chomp()
			return TokenKind(m), nil
		case m == '\n':
			s.line++
			s.column = 0
			fallthrough
		case isSpace(m):
			s.skip()
		case m == 0:
			if err := s.err[0]; err != nil && err != io.EOF {
				return ERROR, err
			}
			return EOF, nil
		case m == utf8.RuneError:
			return ERROR, errors.New("malformed UTF-8 encoding")
		default:
			return ERROR, fmt.Errorf("unexpected token '%c'", n)
		}
	}
}

func (s *Scanner) token() *Token {
	return &Token{Kind: s.tok, Text: s.Text(), Pos: s.Pos(), Value: s.value}
}

func (s *Scanner) Scan() (*Token, error) {
	tok, err := s.scan()
	s.tok = tok

	if err != nil {
		return nil, err
	}
	return &Token{Kind: tok, Text: s.Text(), Pos: s.Pos(), Value: s.value}, nil
}

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n':
		return true
	}
	return false
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isHexDigit(r rune) bool {
	return isDigit(r) || r >= 'A' && r <= 'F' || r >= 'a' && r <= 'f'
}

func isLetter(r rune) bool {
	return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z'
}

func isSymbol(r rune) bool {
	switch r {
	case '+', '-', '*', '/', '\\', '^', '~', '=', '<', '>', '!', '?', '@', '#', '$', '%', '&', '|', ':', '`', '.', '\'':
		return true
	}
	return false
}

func hexNibble(r rune) uint64 {
	if r >= 'A' && r <= 'F' {
		return uint64(r - 'A' + 10)
	} else if r >= 'a' && r <= 'f' {
		return uint64(r - 'a' + 10)
	} else if isDigit(r) {
		return uint64(r - '0')
	}
	return 0
}

func hexNum(s string) (uint64, bool) {
	v := uint64(0)
	for len(s) >= 2 {
		m, n := rune(s[0]), rune(s[1])
		if m == '_' && isHexDigit(n) {
			s = s[2:]
			v = v<<4 | hexNibble(m)
		} else if isHexDigit(m) {
			s = s[1:]
			v = v<<4 | hexNibble(m)
		} else {
			return 0, false
		}
	}
	if len(s) == 1 {
		m := rune(s[0])
		if !isHexDigit(m) {
			return 0, false
		}
		v = v<<4 | hexNibble(m)
	}
	return v, true
}

type BigInt struct {
	text string
	base int
}

func (b *BigInt) I() (int64, error) {
	if b.text[0] == '-' {
		// parse as a signed integer
		return strconv.ParseInt(b.text, b.base, 64)
	}

	text := b.text
	if b.text[0] == '+' {
		text = text[1:]
	}

	// parse as an unsigned integer
	v, err := strconv.ParseUint(text, b.base, 64)
	if err != nil {
		return 0, err
	}
	return int64(v), nil
}

func (b *BigInt) F() (*big.Float, error) {
	var z big.Float
	f, _, err := z.Parse(b.text, b.base)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Scanner) bigInt(t string, base int) TokenKind {
	s.value = &BigInt{text: t, base: base}
	return INT
}

func (s *Scanner) bigFloat(t string, base int) TokenKind {
	var z big.Float
	f, _, err := z.Parse(t, base)
	if err != nil {
		s.value = err
		return ERROR
	}
	s.value = f
	return FLOAT
}
