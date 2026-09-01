package eviction_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/eviction"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

var seedKeyLayout = vault.SingleKey(vault.TextKeyPart).KeyLayout()

type seedValueCodec struct{}

func (seedValueCodec) Encode(value []byte) ([]byte, error) { return value, nil }
func (seedValueCodec) Decode(raw []byte) ([]byte, error)   { return raw, nil }

func openVault(t *testing.T, quotaBytes int64) *vault.Vault {
	t.Helper()

	v, err := memoryvault.Open(quotaBytes, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	seedUsage(t, v)

	return v
}

func seedUsage(t *testing.T, v *vault.Vault) {
	t.Helper()

	collection, err := v.RegisterCollection(
		vault.Name("seed"),
		seedKeyLayout,
		seedValueCodec{},
	)
	if err != nil {
		t.Fatalf("Register seed: %v", err)
	}
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		if err := collection.Put(tx, "seed", make([]byte, 64)); err != nil {
			return fmt.Errorf("put seed: %w", err)
		}

		return nil
	}); err != nil {
		t.Fatalf("seed usage: %v", err)
	}
}

type fakeReferences struct {
	word yacymodel.Hash
}

func (f fakeReferences) WordsReferencing(
	_ *vault.Txn,
	_ yacymodel.URLHash,
) ([]yacymodel.Hash, error) {
	return []yacymodel.Hash{f.word}, nil
}

func (f fakeReferences) ReferencedURLCount(context.Context) (int, error) {
	return 0, nil
}

type fakePostings struct {
	purged []yacymodel.URLHash
}

func (f *fakePostings) PurgePosting(
	_ *vault.Txn,
	_ yacymodel.Hash,
	url yacymodel.URLHash,
) (bool, error) {
	f.purged = append(f.purged, url)

	return true, nil
}

type fakeURLs struct {
	remaining []yacymodel.URLHash
	selected  [][]yacymodel.URLHash
	noDelete  bool
	purgeErr  error
}

func (f *fakeURLs) StalestURLs(_ context.Context, limit int) ([]yacymodel.URLHash, error) {
	if limit > len(f.remaining) {
		limit = len(f.remaining)
	}
	batch := f.remaining[:limit]
	f.selected = append(f.selected, batch)

	return batch, nil
}

func (f *fakeURLs) Purge(
	_ context.Context,
	_ *vault.Txn,
	urls []yacymodel.URLHash,
) (urlmeta.PurgeResult, error) {
	if f.purgeErr != nil {
		return urlmeta.PurgeResult{}, f.purgeErr
	}
	if f.noDelete {
		return urlmeta.PurgeResult{}, nil
	}
	f.remaining = f.remaining[len(urls):]

	return urlmeta.PurgeResult{URLsDeleted: len(urls)}, nil
}

func hashes(n int) []yacymodel.URLHash {
	out := make([]yacymodel.URLHash, n)
	for i := range out {
		out[i] = urlHash(string(rune('a' + i)))
	}

	return out
}

func newSweeper(
	vault *vault.Vault,
	postings *fakePostings,
	urls *fakeURLs,
	config eviction.Config,
) eviction.Sweeper {
	return eviction.NewSweeper(
		vault,
		postings,
		fakeReferences{word: yacymodel.WordHash("w")},
		urls,
		urls,
		config,
	)
}

func sweepConfig(target float64, urlsPerBatch, batchesPerSweep int) eviction.Config {
	return eviction.Config{
		TargetFraction:  target,
		URLsPerBatch:    urlsPerBatch,
		BatchesPerSweep: batchesPerSweep,
	}
}

func TestSweepDrainsCandidatesAboveTarget(t *testing.T) {
	vault := openVault(t, 1)
	postings := &fakePostings{}
	urls := &fakeURLs{remaining: hashes(5)}

	result, err := newSweeper(
		vault,
		postings,
		urls,
		sweepConfig(1, 2, 8),
	).Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if result.URLsDeleted != 5 || result.PostingsDeleted != 5 {
		t.Fatalf("result = %+v, want 5/5", result)
	}
	if len(urls.remaining) != 0 {
		t.Fatalf("remaining = %d, want fully drained", len(urls.remaining))
	}
	if len(urls.selected) != 4 {
		t.Fatalf("select calls = %d, want 4 (2+2+1+empty)", len(urls.selected))
	}
}

func TestSweepStopsOnNoProgress(t *testing.T) {
	vault := openVault(t, 1)
	urls := &fakeURLs{remaining: hashes(4), noDelete: true}

	result, err := newSweeper(
		vault,
		&fakePostings{},
		urls,
		sweepConfig(1, 2, 8),
	).Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if result.URLsDeleted != 0 {
		t.Fatalf("URLsDeleted = %d, want 0", result.URLsDeleted)
	}
	if len(urls.selected) != 1 {
		t.Fatalf("select calls = %d, want 1 before bailing", len(urls.selected))
	}
}

func TestSweepStopsAtItsBatchBound(t *testing.T) {
	vault := openVault(t, 1)
	urls := &fakeURLs{remaining: hashes(10)}

	result, err := newSweeper(
		vault,
		&fakePostings{},
		urls,
		sweepConfig(1, 2, 3),
	).Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if result.URLsDeleted != 6 {
		t.Fatalf("URLsDeleted = %d, want 6 (2 per batch, 3 batches)", result.URLsDeleted)
	}
	if len(urls.selected) != 3 {
		t.Fatalf("select calls = %d, want 3", len(urls.selected))
	}
	if len(urls.remaining) != 4 {
		t.Fatalf("remaining = %d, want 4 left for the next sweep", len(urls.remaining))
	}
}

func TestSweepNoopUnderTarget(t *testing.T) {
	vault := openVault(t, 1<<30)
	urls := &fakeURLs{remaining: hashes(4)}

	result, err := newSweeper(
		vault,
		&fakePostings{},
		urls,
		sweepConfig(0.9, 2, 8),
	).Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if result != (eviction.Result{}) {
		t.Fatalf("result = %+v, want empty", result)
	}
	if len(urls.selected) != 0 {
		t.Fatalf("select calls = %d, want 0", len(urls.selected))
	}
}

func TestSweepNoopWithoutQuota(t *testing.T) {
	result, err := newSweeper(
		openVault(t, 0),
		&fakePostings{},
		&fakeURLs{remaining: hashes(4)},
		sweepConfig(0.5, 2, 8),
	).
		Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if result != (eviction.Result{}) {
		t.Fatalf("result = %+v, want empty", result)
	}
}

func TestSweepReportsPurgeError(t *testing.T) {
	wantErr := errors.New("boom")
	urls := &fakeURLs{remaining: hashes(4), purgeErr: wantErr}

	_, err := newSweeper(
		openVault(t, 1),
		&fakePostings{},
		urls,
		sweepConfig(1, 1, 8),
	).Sweep(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}
