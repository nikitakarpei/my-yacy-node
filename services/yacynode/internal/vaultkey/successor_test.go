package vaultkey

import (
	"bytes"
	"testing"
)

func TestSuccessorTruncatesAtTheLastByteBelow0xFF(t *testing.T) {
	successor := successorOf([]byte{0x01, 0xFF, 0xFF})

	if want := []byte{0x02}; !bytes.Equal(successor, want) {
		t.Fatalf("successorOf(01ffff) = %x, want %x", successor, want)
	}
}

func TestSuccessorOfAPrefixOfOnly0xFFIsAbsent(t *testing.T) {
	for _, prefix := range [][]byte{nil, {}, {0xFF}, {0xFF, 0xFF}} {
		if successor := successorOf(prefix); successor != nil {
			t.Fatalf("successorOf(%x) = %x, want nil", prefix, successor)
		}
	}
}
