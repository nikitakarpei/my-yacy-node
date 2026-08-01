package yacyproto

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func gzipFramed(t *testing.T, plain string) string {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write([]byte(plain)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return tagged(wireFrameGzip, yacymodel.Encode(buf.Bytes()))
}

func TestDecodeWireFormRejectsOversizedGzipPayload(t *testing.T) {
	framed := gzipFramed(t, strings.Repeat("x", wireFormMaxPlainBytes+1))

	_, err := decodeWireForm(context.Background(), framed)
	if !errors.Is(err, errBadWireFrame) {
		t.Fatalf("decodeWireForm error = %v, want errBadWireFrame", err)
	}
}
