package data

import (
	_ "embed"
)

//go:embed enwik8
var enwik8 []byte

// Enwik8 is first 100M of the English Wikipedia dump on Mar. 3 2006 as used for
// the Hutter prize.
var Enwik8 = enwik8
