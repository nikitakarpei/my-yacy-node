package metrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

const urlWordsBucket vault.Name = "rwi_url_words"

func TestURLWordsCountTheAddedWordsByForm(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewURLWordsMetrics(registry)

	observer.ObserveMemberAdded(urlWordsBucket, vault.MemberFormID)
	observer.ObserveMemberAdded(urlWordsBucket, vault.MemberFormID)
	observer.ObserveMemberAdded(urlWordsBucket, vault.MemberFormRaw)

	if err := testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP yacynode_url_words_added_total Words added to the word sets of urls, by form: id for a word the ranking lists, raw for a full word hash.
# TYPE yacynode_url_words_added_total counter
yacynode_url_words_added_total{form="id"} 2
yacynode_url_words_added_total{form="raw"} 1
`), "yacynode_url_words_added_total"); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestURLWordsCountTheWordsTheNodeCouldNotDecode(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewURLWordsMetrics(registry)

	observer.ObserveUndecodableMembers(urlWordsBucket, 3)
	observer.ObserveUndecodableMembers(urlWordsBucket, 2)

	if err := testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP yacynode_url_words_undecodable_total Words read from the word sets of urls that the node could not decode.
# TYPE yacynode_url_words_undecodable_total counter
yacynode_url_words_undecodable_total 5
`), "yacynode_url_words_undecodable_total"); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
