package judgedqueries_test

import (
	"os"
	"path/filepath"
	"testing"
)

type storedPage struct {
	address     string
	contentType string
	body        []byte
}

const (
	storedPagesDirectory  = "testdata/pages"
	storedPagesFileSuffix = ".warc.zst"
)

func writeStoredPagesOf(t *testing.T, query string, pages []storedPage) {
	t.Helper()

	writeZstandardFixtureFile(t, storedPagesFileOf(query), warcOf(t, pages))
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

	path := storedPagesFileOf(query)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	return pagesOfWARC(t, contentOfZstandardFixtureFile(t, path))
}

func storedPagesFileOf(query string) string {
	return filepath.Join(storedPagesDirectory, queryInFileNames(query)+storedPagesFileSuffix)
}
