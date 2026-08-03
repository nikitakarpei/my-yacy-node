package metrics

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type stubVaultCollections struct {
	entries map[vault.Name]int
	err     error
}

func (s stubVaultCollections) EntriesByCollection(
	context.Context,
) (map[vault.Name]int, error) {
	return s.entries, s.err
}

func TestVaultCollectionReportsEntries(t *testing.T) {
	registry := prometheus.NewRegistry()
	NewVaultCollectionMetrics(registry, stubVaultCollections{
		entries: map[vault.Name]int{"rwi": 7, "urlmeta": 0},
	})

	expected := `
# HELP vault_collection_entries Entries currently stored in each vault collection.
# TYPE vault_collection_entries gauge
vault_collection_entries{collection="rwi"} 7
vault_collection_entries{collection="urlmeta"} 0
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestVaultCollectionOmitsEntriesOnError(t *testing.T) {
	registry := prometheus.NewRegistry()
	NewVaultCollectionMetrics(registry, stubVaultCollections{err: errors.New("unavailable")})

	if got := testutil.CollectAndCount(registry, "vault_collection_entries"); got != 0 {
		t.Errorf("vault_collection_entries samples = %d, want none on error", got)
	}
}
