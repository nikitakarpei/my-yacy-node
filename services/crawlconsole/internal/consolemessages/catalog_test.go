package consolemessages_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/consolemessages"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestEnglishNamesEveryWordOfTheCrawler(t *testing.T) {
	keys := make(
		[]string,
		0,
		len(yacycrawlcontract.CrawlStates)+2*len(yacycrawlcontract.CrawlScopes),
	)
	for _, state := range yacycrawlcontract.CrawlStates {
		keys = append(keys, "crawl-state."+string(state))
	}
	for _, scope := range yacycrawlcontract.CrawlScopes {
		keys = append(keys, "crawl-scope."+string(scope), "crawl-scope-hint."+string(scope))
	}
	for _, key := range keys {
		if _, err := consolemessages.English().TextOf(key); err != nil {
			t.Errorf("TextOf(%q): %v", key, err)
		}
	}
}

func TestTextOfFillsThePlaceholders(t *testing.T) {
	text, err := consolemessages.English().TextOf("header.figures", 2, 30)
	if err != nil {
		t.Fatalf("TextOf: %v", err)
	}
	if text != "2 running · 30 pages pending" {
		t.Fatalf("TextOf = %q", text)
	}
}

func TestTextOfRefusesAnUnknownKey(t *testing.T) {
	_, err := consolemessages.English().TextOf("no.such.key")
	if err == nil {
		t.Fatalf("TextOf of an unknown key = %v, want an error", err)
	}
}
