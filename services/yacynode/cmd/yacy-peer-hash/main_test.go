package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestRunPrintsValidPeerHash(t *testing.T) {
	var out bytes.Buffer
	if err := run(&out, bytes.NewReader(bytes.Repeat([]byte{0x42}, 32))); err != nil {
		t.Fatalf("run: %v", err)
	}

	if _, err := yacymodel.ParseHash(strings.TrimSpace(out.String())); err != nil {
		t.Fatalf("parse printed hash: %v", err)
	}
}

func TestRunReturnsErrorOnShortEntropy(t *testing.T) {
	if err := run(&bytes.Buffer{}, bytes.NewReader(nil)); err == nil {
		t.Fatal("expected error for insufficient entropy")
	}
}
