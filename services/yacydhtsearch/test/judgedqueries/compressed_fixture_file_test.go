package judgedqueries_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func writeGzippedFixtureFile(t *testing.T, path string, content []byte) {
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

func contentOfTheGzippedFixtureFile(t *testing.T, path string) []byte {
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

func writeZstandardFixtureFile(t *testing.T, path string, content []byte) {
	t.Helper()

	var compressed bytes.Buffer
	compressing, err := zstd.NewWriter(
		&compressed, zstd.WithEncoderLevel(zstd.SpeedBestCompression),
	)
	if err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
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

func contentOfTheZstandardFixtureFile(t *testing.T, path string) []byte {
	t.Helper()

	compressed, err := os.ReadFile(path) //nolint:gosec // a fixture path of this test directory
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	decompressing, err := zstd.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	defer decompressing.Close()

	content, err := io.ReadAll(decompressing)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return content
}
