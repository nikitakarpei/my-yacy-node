package pagerwi_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerwi"
)

func TestDerivationDerivesFromRenderedText(t *testing.T) {
	page := samplePage()
	representation, err := pagerwi.NewDerivation().Derive(page, []byte(sampleText))
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(representation.Postings) == 0 {
		t.Fatal("no postings")
	}
}
