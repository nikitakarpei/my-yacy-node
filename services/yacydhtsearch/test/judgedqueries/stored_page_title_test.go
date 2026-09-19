package judgedqueries_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const storedPageTitlesFileName = "titles.json"

func storePageTitlesOfTheQuery(
	t *testing.T,
	query string,
	pageTitlePerDocument map[yacymodel.URLHash]string,
) {
	t.Helper()

	writeFixtureFile(t, storedPageTitlesFileOf(query), pageTitlePerDocument)
}

func storedPageTitlePerDocument(t *testing.T, query string) map[yacymodel.URLHash]string {
	t.Helper()

	path := storedPageTitlesFileOf(query)
	content, err := os.ReadFile(path) //nolint:gosec // a fixture path of this test directory
	if errors.Is(err, fs.ErrNotExist) {
		return map[yacymodel.URLHash]string{}
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var pageTitlePerDocument map[yacymodel.URLHash]string
	if err := json.Unmarshal(content, &pageTitlePerDocument); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return pageTitlePerDocument
}

func storedPageTitlesFileOf(query string) string {
	return filepath.Join(storedPageTextDirectoryOf(query), storedPageTitlesFileName)
}
