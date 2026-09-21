package judgedqueries_test

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestAZstandardFixtureFileReadsBackWhatWasWrittenIntoIt(t *testing.T) {
	t.Parallel()

	writtenContent := bytes.Repeat([]byte("Rain in Berlin. \x00\xff"), 4096)
	path := filepath.Join(t.TempDir(), "fixture.zst")

	writeZstandardFixtureFile(t, path, writtenContent)

	if readContent := contentOfZstandardFixtureFile(
		t,
		path,
	); !bytes.Equal(
		readContent,
		writtenContent,
	) {
		t.Fatalf(
			"the fixture file holds %d bytes, want the %d writtenContent",
			len(readContent),
			len(writtenContent),
		)
	}
}

func TestAGzippedFixtureFileReadsBackWhatWasWrittenIntoIt(t *testing.T) {
	t.Parallel()

	writtenContent := []byte("Rain in Berlin.")
	path := filepath.Join(t.TempDir(), "fixture.gz")

	writeGzippedFixtureFile(t, path, writtenContent)

	if readContent := contentOfGzippedFixtureFile(
		t,
		path,
	); !bytes.Equal(
		readContent,
		writtenContent,
	) {
		t.Fatalf("the fixture file holds %q, want %q", readContent, writtenContent)
	}
}
