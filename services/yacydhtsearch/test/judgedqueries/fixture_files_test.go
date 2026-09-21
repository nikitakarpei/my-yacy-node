package judgedqueries_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
)

const (
	fixtureFilePermissions = 0o644
	fixtureDirPermissions  = 0o755
)

func queryInFileNames(query string) string {
	return strings.Join(strings.Fields(strings.ToLower(query)), "-")
}

func writeFixtureFile(t *testing.T, path string, fixture any) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), fixtureDirPermissions); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	content := append(indentedJSONOf(t, fixture), '\n')
	if err := os.WriteFile(path, content, fixtureFilePermissions); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func indentedJSONOf(t *testing.T, fixture any) []byte {
	t.Helper()

	content, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatalf("write a fixture as JSON: %v", err)
	}

	return content
}

func writeGzippedFixtureFile(t *testing.T, path string, content []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), fixtureDirPermissions); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
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

func contentOfGzippedFixtureFile(t *testing.T, path string) []byte {
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

	if err := os.MkdirAll(filepath.Dir(path), fixtureDirPermissions); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
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

func contentOfZstandardFixtureFile(t *testing.T, path string) []byte {
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
