//go:build e2e

// Package warcarchive writes archived HTTP captures for end-to-end tests.
package warcarchive

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type Capture struct {
	URL         string
	CapturedAt  time.Time
	StatusCode  int
	ContentType string
	Body        string
}

type Archive struct {
	content []byte
}

func Write(t *testing.T, captures []Capture) Archive {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "archive.warc")
	var archive bytes.Buffer
	for captureNumber, capture := range captures {
		response := httpResponseOf(capture)
		_, _ = fmt.Fprintf(
			&archive,
			"WARC/1.0\r\nWARC-Type: response\r\nWARC-Record-ID: <urn:uuid:00000000-0000-0000-0000-%012d>\r\nWARC-Target-URI: %s\r\nWARC-Date: %s\r\nContent-Type: application/http; msgtype=response\r\nContent-Length: %d\r\n\r\n",
			captureNumber+1,
			capture.URL,
			capture.CapturedAt.UTC().Format(time.RFC3339),
			len(response),
		)
		archive.Write(response)
		archive.WriteString("\r\n\r\n")
	}
	if err := os.WriteFile(archivePath, archive.Bytes(), 0o600); err != nil {
		t.Fatalf("write WARC archive: %v", err)
	}
	return Archive{content: archive.Bytes()}
}

func (a Archive) Content() []byte { return bytes.Clone(a.content) }

func httpResponseOf(capture Capture) []byte {
	statusCode := capture.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	contentType := capture.ContentType
	if contentType == "" {
		contentType = "text/html; charset=utf-8"
	}
	return []byte(
		fmt.Sprintf(
			"HTTP/1.1 %d %s\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n%s",
			statusCode,
			http.StatusText(statusCode),
			contentType,
			len(capture.Body),
			capture.Body,
		),
	)
}
