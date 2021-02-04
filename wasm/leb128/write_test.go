// Copyright 2018 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package leb128

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestWriteVarUint32(t *testing.T) {
	for _, c := range casesUint {
		t.Run(fmt.Sprint(c.v), func(t *testing.T) {
			buf := new(bytes.Buffer)
			_, err := WriteVarUint32(buf, c.v)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(buf.Bytes(), c.b) {
				t.Fatalf("unexpected output: %x", buf.Bytes())
			}
		})
	}
}

func TestWriteVarint64(t *testing.T) {
	for _, c := range casesInt {
		t.Run(fmt.Sprint(c.v), func(t *testing.T) {
			buf := new(bytes.Buffer)
			_, err := WriteVarint64(buf, c.v)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(buf.Bytes(), c.b) {
				t.Fatalf("unexpected output: %x", buf.Bytes())
			}
		})
	}
}

func TestWriteReadInt64(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	var buf bytes.Buffer
	for i := 0; i < 5000000; i++ {
		n := r.Int63()

		buf.Reset()
		_, err := WriteVarint64(&buf, n)
		if err != nil {
			t.Fatalf("WriteVarint64: %v", err)
		}

		v, err := ReadVarint64(&buf)
		if err != nil {
			t.Fatalf("ReadVarint64: %v", err)
		}

		if v != n {
			t.Fatalf("wrote %v; read %v", n, v)
		}
	}
}

func TestWriteReadInt32(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	var buf bytes.Buffer
	for i := 0; i < 5000000; i++ {
		n := r.Int31()

		buf.Reset()
		_, err := WriteVarint64(&buf, int64(n))
		if err != nil {
			t.Fatalf("WriteVarint64: %v", err)
		}

		v, err := ReadVarint32(&buf)
		if err != nil {
			t.Fatalf("ReadVarint32: %v", err)
		}

		if v != n {
			t.Fatalf("wrote %v; read %v", n, v)
		}
	}
}

func TestWriteReadUint32(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	var buf bytes.Buffer
	for i := 0; i < 5000000; i++ {
		n := r.Uint32()

		buf.Reset()
		_, err := WriteVarUint32(&buf, n)
		if err != nil {
			t.Fatalf("WriteVarint64: %v", err)
		}

		v, err := ReadVarUint32(&buf)
		if err != nil {
			t.Fatalf("ReadVarint64: %v", err)
		}

		if v != n {
			t.Fatalf("wrote %v; read %v", n, v)
		}
	}
}
