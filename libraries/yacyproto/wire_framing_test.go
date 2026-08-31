package yacyproto_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const wireFormMaxPlainBytes = 1 << 20

func framed(tag byte, body string) string {
	return string([]byte{tag, '|'}) + body
}

func gzipFramed(t *testing.T, plain string) string {
	t.Helper()

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte(plain)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	return framed('z', yacymodel.Encode(compressed.Bytes()))
}

func seedRow(t *testing.T, seed yacymodel.Seed) string {
	t.Helper()

	encoded := seedWireForm(seed)
	if !strings.HasPrefix(encoded, "b|") {
		t.Fatalf("seed wire form does not use a base64 frame: %q", encoded)
	}
	plain, err := yacymodel.Decode(encoded[2:])
	if err != nil {
		t.Fatalf("decode seed frame: %v", err)
	}

	return string(plain)
}

func TestHelloRequestReadsEverySeedFrameTag(t *testing.T) {
	t.Parallel()

	want := sampleSeed(t, "alpha", "example-peer")
	row := seedRow(t, want)

	frames := map[string]string{
		"plain":  framed('p', row),
		"base64": framed('b', yacymodel.Encode([]byte(row))),
		"gzip":   gzipFramed(t, row),
	}
	for tag, frame := range frames {
		got, err := seedFromWire(t, frame)
		if err != nil {
			t.Errorf("parse %s seed frame: %v", tag, err)

			continue
		}
		if got.Hash != want.Hash || got.Name != want.Name {
			t.Errorf("%s frame = %+v, want %+v", tag, got, want)
		}
	}
}

func TestHelloRequestRejectsAnUnknownSeedFrameTag(t *testing.T) {
	t.Parallel()

	frame := framed('q', yacymodel.Encode([]byte(seedRow(t, sampleSeed(t, "alpha", "peer")))))

	_, err := seedFromWire(t, frame)
	if !errors.Is(err, yacymodel.ErrBadSeed) {
		t.Fatalf("seed wire form error = %v, want ErrBadSeed", err)
	}
}

func TestHelloRequestRejectsASeedFrameThatInflatesPastTheBound(t *testing.T) {
	t.Parallel()

	frame := gzipFramed(t, strings.Repeat("x", wireFormMaxPlainBytes+1))

	_, err := seedFromWire(t, frame)
	if !errors.Is(err, yacymodel.ErrBadSeed) {
		t.Fatalf("seed wire form error = %v, want ErrBadSeed", err)
	}
}
