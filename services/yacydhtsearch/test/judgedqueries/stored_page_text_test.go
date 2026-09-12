package judgedqueries_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	storedPageTextDirectory  = "testdata/pagetext"
	storedPageTextFileSuffix = ".txt.gz"
)

func storePageTextOfTheQuery(
	t *testing.T,
	query string,
	pageTextPerDocument map[yacymodel.URLHash]string,
) {
	t.Helper()

	queryDirectory := storedPageTextDirectoryOf(query)
	if err := os.RemoveAll(queryDirectory); err != nil {
		t.Fatalf("write %s: %v", queryDirectory, err)
	}
	if err := os.MkdirAll(queryDirectory, fixtureDirPermissions); err != nil {
		t.Fatalf("write %s: %v", queryDirectory, err)
	}
	for document, pageText := range pageTextPerDocument {
		writeStoredPageTextFile(t, storedPageTextFileOf(query, document), pageText)
	}
}

func writeStoredPageTextFile(t *testing.T, path string, pageText string) {
	t.Helper()

	writeCompressedFixtureFile(t, path, []byte(pageText))
}

func storedPageTextPerDocument(
	t *testing.T,
	query string,
) map[yacymodel.URLHash]string {
	t.Helper()

	storedFiles, err := filepath.Glob(
		filepath.Join(storedPageTextDirectoryOf(query), "*"+storedPageTextFileSuffix),
	)
	if err != nil {
		t.Fatalf("read %s: %v", storedPageTextDirectoryOf(query), err)
	}

	pageTextPerDocument := make(map[yacymodel.URLHash]string, len(storedFiles))
	for _, storedFile := range storedFiles {
		pageTextPerDocument[documentOfTheStoredPageTextFile(t, storedFile)] = storedPageTextInTheFile(
			t,
			storedFile,
		)
	}

	return pageTextPerDocument
}

func documentOfTheStoredPageTextFile(t *testing.T, path string) yacymodel.URLHash {
	t.Helper()

	document, err := yacymodel.ParseURLHash(
		strings.TrimSuffix(filepath.Base(path), storedPageTextFileSuffix),
	)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return document
}

func storedPageTextInTheFile(t *testing.T, path string) string {
	t.Helper()

	return string(contentOfTheCompressedFixtureFile(t, path))
}

func storedPageTextFileOf(query string, document yacymodel.URLHash) string {
	return filepath.Join(
		storedPageTextDirectoryOf(query), document.String()+storedPageTextFileSuffix,
	)
}

func storedPageTextDirectoryOf(query string) string {
	return filepath.Join(storedPageTextDirectory, queryInFileNames(query))
}
