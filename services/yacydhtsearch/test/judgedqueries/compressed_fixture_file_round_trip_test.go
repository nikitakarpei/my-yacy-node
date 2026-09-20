package judgedqueries_test

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestAZstandardFixtureFileReadsBackWhatWasWrittenIntoIt(t *testing.T) {
	t.Parallel()

	written := bytes.Repeat([]byte("Rain in Berlin. \x00\xff"), 4096)
	path := filepath.Join(t.TempDir(), "fixture.zst")

	writeZstandardFixtureFile(t, path, written)

	if read := contentOfTheZstandardFixtureFile(t, path); !bytes.Equal(read, written) {
		t.Fatalf("the fixture file holds %d bytes, want the %d written", len(read), len(written))
	}
}

func TestAGzippedFixtureFileReadsBackWhatWasWrittenIntoIt(t *testing.T) {
	t.Parallel()

	written := []byte("Rain in Berlin.")
	path := filepath.Join(t.TempDir(), "fixture.gz")

	writeGzippedFixtureFile(t, path, written)

	if read := contentOfTheGzippedFixtureFile(t, path); !bytes.Equal(read, written) {
		t.Fatalf("the fixture file holds %q, want %q", read, written)
	}
}
