//go:build e2e

// Package containerlog saves the log of a container that took part in a failed
// test. The container is removed when the test ends, so this log is the only
// record of what the container did. Set E2E_CONTAINER_LOG_DIR to collect the
// logs somewhere a build can archive them; without it they go to the temporary
// directory of the operating system.
package containerlog

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

const containerLogDirectoryVariable = "E2E_CONTAINER_LOG_DIR"

func DumpOnFailure(t *testing.T, label string, container testcontainers.Container) {
	t.Helper()
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		path, err := writeContainerLog(
			container,
			containerLogDirectory(),
			logFilePatternFrom(t.Name(), label),
		)
		if err != nil {
			t.Logf("%s logs: %v", label, err)
			return
		}
		t.Logf("%s logs written to %s", label, path)
	})
}

func containerLogDirectory() string {
	if directory := os.Getenv(containerLogDirectoryVariable); directory != "" {
		return directory
	}

	return os.TempDir()
}

func logFilePatternFrom(testName, label string) string {
	unsafeCharacters := strings.NewReplacer("/", "-", " ", "_", string(os.PathSeparator), "-")

	return unsafeCharacters.Replace(testName+"-"+label) + "-*.log"
}

func writeContainerLog(
	container testcontainers.Container,
	directory, filePattern string,
) (string, error) {
	reader, err := container.Logs(context.Background())
	if err != nil {
		return "", fmt.Errorf("read container log: %w", err)
	}
	defer func() { _ = reader.Close() }()

	file, err := os.CreateTemp(directory, filePattern)
	if err != nil {
		return "", fmt.Errorf("create log file in %s: %w", directory, err)
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, reader); err != nil {
		return "", fmt.Errorf("write log file %s: %w", file.Name(), err)
	}

	return file.Name(), nil
}
