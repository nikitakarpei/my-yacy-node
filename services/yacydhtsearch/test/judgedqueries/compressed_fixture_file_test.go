package judgedqueries_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"testing"
)

func writeCompressedFixtureFile(t *testing.T, path string, content []byte) {
	t.Helper()

	var compressed bytes.Buffer
	compressing := gzip.NewWriter(&compressed)
	if _, err := compressing.Write(content); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := compressing.Close(); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.WriteFile(path, compressed.Bytes(), fixtureFilePermissions); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func contentOfTheCompressedFixtureFile(t *testing.T, path string) []byte {
	t.Helper()

	compressed, err := os.ReadFile(path) //nolint:gosec // a fixture path of this test directory
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	decompressing, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	content, err := io.ReadAll(decompressing)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return content
}
