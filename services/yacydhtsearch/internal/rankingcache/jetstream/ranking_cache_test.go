package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	rankingcachejetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcache/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	bucketName   = "rankings"
	valueCeiling = 1024
)

type recordedFailure struct {
	lookups int
	stores  int
}

func (r *recordedFailure) RankingLookupFailed(context.Context, searchquery.Query, error) {
	r.lookups++
}

func (r *recordedFailure) RankingStoreFailed(context.Context, searchquery.Query, error) {
	r.stores++
}

func rankingOver(t *testing.T, address string) searchresult.Ranking {
	t.Helper()

	item, ok := searchresult.ItemFrom(yacymodel.URLMetadata{
		Address: address,
		Title:   "Weather",
		Snippet: "prose",
	})
	if !ok {
		t.Fatalf("ItemFrom(%q) refused a well-formed address", address)
	}

	return searchresult.Ranking{Items: []searchresult.Item{item}}
}

func bucketFor(t *testing.T, config natsjetstream.KeyValueConfig) natsjetstream.KeyValue {
	t.Helper()

	stream := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	config.Bucket = bucketName
	bucket, err := stream.CreateOrUpdateKeyValue(t.Context(), config)
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	return bucket
}

func TestARankingIsReadBackForTheQueryItWasHeldFor(t *testing.T) {
	t.Parallel()

	failures := &recordedFailure{}
	cache := rankingcachejetstream.New(
		bucketFor(t, natsjetstream.KeyValueConfig{}),
		rankingcachejetstream.RankingCacheObservers{failures},
	)
	query := searchquery.QueryFrom("berlin")
	cache.StoreRanking(t.Context(), query, rankingOver(t, "https://a.example/"))

	ranking, found := cache.CachedRankingFor(t.Context(), query)

	if !found || len(ranking.Items) != 1 {
		t.Fatalf("CachedRankingFor = %+v (found %t), want the ranking cache", ranking, found)
	}
	item := ranking.Items[0]
	if item.Address != "https://a.example/" || item.Title != "Weather" || item.Hash.IsZero() {
		t.Fatalf("item = %+v, want every field it was cache with", item)
	}
	if failures.lookups != 0 || failures.stores != 0 {
		t.Fatalf("failures reported %+v, want none", failures)
	}
}

func TestNoRankingIsHeldForAQueryNobodyAsked(t *testing.T) {
	t.Parallel()

	failures := &recordedFailure{}
	cache := rankingcachejetstream.New(
		bucketFor(t, natsjetstream.KeyValueConfig{}),
		rankingcachejetstream.RankingCacheObservers{failures},
	)

	_, found := cache.CachedRankingFor(t.Context(), searchquery.QueryFrom("berlin"))

	if found || failures.lookups != 0 {
		t.Fatalf(
			"CachedRankingFor found %t with %d failures, want a plain miss",
			found,
			failures.lookups,
		)
	}
}

func TestARankingIsGoneOnceItsLifetimeIsSpent(t *testing.T) {
	t.Parallel()

	cache := rankingcachejetstream.New(
		bucketFor(t, natsjetstream.KeyValueConfig{TTL: 100 * time.Millisecond}),
		rankingcachejetstream.RankingCacheObservers{&recordedFailure{}},
	)
	query := searchquery.QueryFrom("berlin")
	cache.StoreRanking(t.Context(), query, rankingOver(t, "https://a.example/"))

	time.Sleep(time.Second)

	if _, found := cache.CachedRankingFor(t.Context(), query); found {
		t.Fatal("CachedRankingFor still stores a ranking past its lifetime")
	}
}

func TestARankingTheBucketRefusesIsReported(t *testing.T) {
	t.Parallel()

	failures := &recordedFailure{}
	cache := rankingcachejetstream.New(
		bucketFor(t, natsjetstream.KeyValueConfig{MaxValueSize: valueCeiling}),
		rankingcachejetstream.RankingCacheObservers{failures},
	)
	query := searchquery.QueryFrom("berlin")

	items := make([]searchresult.Item, 0, 100)
	for range 100 {
		items = append(items, rankingOver(t, "https://a.example/").Items[0])
	}
	cache.StoreRanking(t.Context(), query, searchresult.Ranking{Items: items})

	if failures.stores != 1 {
		t.Fatalf("store failures = %d, want one", failures.stores)
	}
	if _, found := cache.CachedRankingFor(t.Context(), query); found {
		t.Fatal("CachedRankingFor stores a ranking the bucket refused")
	}
}
