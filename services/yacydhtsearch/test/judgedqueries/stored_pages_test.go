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

func storePagesOfTheQuery(t *testing.T, query string, pages []storedPage) {
	t.Helper()

	path := storedPagesFileOf(query)
	if err := os.MkdirAll(filepath.Dir(path), fixtureDirPermissions); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	writeZstandardFixtureFile(t, path, warcOfThePages(t, pages))
}

func storedPagePerAddress(t *testing.T, query string) map[string]storedPage {
	t.Helper()

	return pagePerAddressOf(storedPagesOfTheQuery(t, query))
}

func pagePerAddressOf(pages []storedPage) map[string]storedPage {
	pagePerAddress := make(map[string]storedPage, len(pages))
	for _, page := range pages {
		pagePerAddress[page.address] = page
	}

	return pagePerAddress
}

func storedPagesOfTheQuery(t *testing.T, query string) []storedPage {
	t.Helper()

	path := storedPagesFileOf(query)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	return pagesOfTheWARC(t, contentOfTheZstandardFixtureFile(t, path))
}

func storedPagesFileOf(query string) string {
	return filepath.Join(storedPagesDirectory, queryInFileNames(query)+storedPagesFileSuffix)
}
