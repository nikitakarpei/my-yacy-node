package rwipostings_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type harness struct {
	vault  *vault.Vault
	index  rwipostings.PostingIndex
	writer rwipostings.PostingWriter
	purger rwipostings.PostingPurger
}

func (h harness) postingOf(
	t *testing.T,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool) {
	t.Helper()

	var (
		entry yacymodel.RWIPosting
		found bool
	)
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		stored, ok, err := h.index.PostingOf(tx, word, url)
		entry, found = stored, ok

		return err
	}); err != nil {
		t.Fatalf("PostingOf: %v", err)
	}

	return entry, found
}

func (h harness) postingsOfWordIn(
	t *testing.T,
	word yacymodel.Hash,
	partition uint,
	wantedPostings int,
) []yacymodel.RWIPosting {
	t.Helper()

	var visitedPostings []yacymodel.RWIPosting
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		return h.index.PostingsOfWordIn(
			tx,
			word,
			partition,
			func(posting yacymodel.RWIPosting) (bool, error) {
				visitedPostings = append(visitedPostings, posting)

				return len(visitedPostings) < wantedPostings, nil
			},
		)
	}); err != nil {
		t.Fatalf("PostingsOfWordIn: %v", err)
	}

	return visitedPostings
}

func openHarness(t *testing.T) harness {
	t.Helper()

	return openHarnessRanking(t, nil)
}

func openHarnessRanking(t *testing.T, rankedWords []yacymodel.Hash) harness {
	t.Helper()

	v := openVault(t)
	index, writer, purger, err := rwipostings.Open(
		context.Background(),
		v,
		ringPartitions,
		rankedWords,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}

	return harness{
		vault:  v,
		index:  index,
		writer: writer,
		purger: purger,
	}
}

func openVault(t *testing.T) *vault.Vault {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	return v
}

var ringPartitions = yacymodel.DHTRingPartitions(1) << 4

func (h harness) writeEach(t *testing.T, postings ...yacymodel.RWIPosting) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		for _, entry := range postings {
			if err := h.writer.Write(tx, entry); err != nil {
				return fmt.Errorf("write posting: %w", err)
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func urlHash(seed string) yacymodel.URLHash {
	address, err := url.Parse("http://example.com/" + seed)
	if err != nil {
		panic(err)
	}

	return yacymodel.URLNormalformOf(address).Hash()
}

func posting(word, urlSeed string) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash:   yacymodel.WordHash(word),
		URLHash:    urlHash(urlSeed),
		Language:   yacymodel.LanguageOfUndeclaredDocument,
		LocalLinks: 1,
		Hits:       1,
	}
}
