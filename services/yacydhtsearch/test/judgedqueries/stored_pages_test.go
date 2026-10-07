package judgedqueries_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type storedPage struct {
	address     string
	contentType string
	body        []byte
	capturedAt  time.Time
	capturedBy  string
	status      int
}

const (
	storedPagesDirectory    = "testdata/pages"
	storedPagesFileSuffix   = ".warc.zst"
	evidencePagesFileSuffix = ".evidence.warc.zst"
)

func writeStoredPagesOf(t *testing.T, query string, pages []storedPage) {
	t.Helper()

	writeZstandardFixtureFile(t, storedPagesFileOf(query), warcOf(t, pages))
}

func appendStoredPagesOf(t *testing.T, query string, pages []storedPage) {
	t.Helper()

	appendZstandardFrameTo(t, storedPagesFileOf(query), warcOf(t, pages))
}

func storedPagePerAddressOf(t *testing.T, query string) map[string]storedPage {
	t.Helper()

	return pagePerAddressOf(storedPagesOf(t, query))
}

func pagePerAddressOf(pages []storedPage) map[string]storedPage {
	pagePerAddress := make(map[string]storedPage, len(pages))
	for _, page := range pages {
		pagePerAddress[page.address] = page
	}

	return pagePerAddress
}

func storedPagesOf(t *testing.T, query string) []storedPage {
	t.Helper()

	return pagesAt(t, storedPagesFileOf(query))
}

func pagesAt(t *testing.T, path string) []storedPage {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	return pagesOfWARC(t, contentOfZstandardFixtureFile(t, path))
}

func storedPagesFileOf(query string) string {
	return filepath.Join(storedPagesDirectory, queryInFileNames(query)+storedPagesFileSuffix)
}

func appendEvidencePagesOf(t *testing.T, query string, pages []storedPage) {
	t.Helper()

	appendZstandardFrameTo(t, evidencePagesFileOf(query), warcOf(t, pages))
}

func evidencePagePerAddressOf(t *testing.T, query string) map[string]storedPage {
	t.Helper()

	return pagePerAddressOf(pagesAt(t, evidencePagesFileOf(query)))
}

func evidencePagesFileOf(query string) string {
	return filepath.Join(storedPagesDirectory, queryInFileNames(query)+evidencePagesFileSuffix)
}
