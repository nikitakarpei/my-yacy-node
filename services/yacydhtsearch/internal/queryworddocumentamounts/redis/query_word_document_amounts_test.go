package redis_test

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	queryworddocumentamountsredis "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryworddocumentamounts/redis"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const lifetime = time.Hour

type recordedFailures struct {
	lookups []yacymodel.Hash
	stores  []yacymodel.Hash
}

func (r *recordedFailures) DocumentAmountLookupFailed(
	_ context.Context,
	word yacymodel.Hash,
	_ error,
) {
	r.lookups = append(r.lookups, word)
}

func (r *recordedFailures) DocumentAmountStoreFailed(
	_ context.Context,
	word yacymodel.Hash,
	_ error,
) {
	r.stores = append(r.stores, word)
}

func amountsOn(
	server *miniredis.Miniredis,
	failures *recordedFailures,
) *queryworddocumentamountsredis.QueryWordDocumentAmounts {
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})

	return queryworddocumentamountsredis.New(
		client, lifetime, queryworddocumentamountsredis.QueryWordDocumentAmountsObservers{failures},
	)
}

func TestAnAmountOneInstanceRemembersIsReadBackByAnother(t *testing.T) {
	t.Parallel()

	server := miniredis.RunT(t)
	failures := &recordedFailures{}
	berlin, weather := yacymodel.WordHash("berlin"), yacymodel.WordHash("weather")
	amountsOn(server, failures).Remember(t.Context(), map[yacymodel.Hash]int{berlin: 12})

	got := amountsOn(
		server,
		failures,
	).DocumentAmountsOf(t.Context(), []yacymodel.Hash{berlin, weather})

	if want := map[yacymodel.Hash]int{berlin: 12}; !maps.Equal(got, want) {
		t.Fatalf("DocumentAmountsOf = %v, want %v", got, want)
	}
	if len(failures.lookups) != 0 || len(failures.stores) != 0 {
		t.Fatalf("failures reported %+v, want none", failures)
	}
}

func TestAnAmountIsForgottenOnceItsLifetimeIsSpent(t *testing.T) {
	t.Parallel()

	server := miniredis.RunT(t)
	amounts := amountsOn(server, &recordedFailures{})
	berlin := yacymodel.WordHash("berlin")
	amounts.Remember(t.Context(), map[yacymodel.Hash]int{berlin: 12})

	server.FastForward(lifetime)

	if got := amounts.DocumentAmountsOf(t.Context(), []yacymodel.Hash{berlin}); len(got) != 0 {
		t.Fatalf("DocumentAmountsOf = %v past the lifetime, want none", got)
	}
}

func refusingServer(t *testing.T) *miniredis.Miniredis {
	t.Helper()

	server := miniredis.RunT(t)
	server.SetError("server down")

	return server
}

func TestALookupTheServerRefusesIsReportedAndIsAMiss(t *testing.T) {
	t.Parallel()

	failures := &recordedFailures{}
	berlin := yacymodel.WordHash("berlin")

	got := amountsOn(
		refusingServer(t),
		failures,
	).DocumentAmountsOf(t.Context(), []yacymodel.Hash{berlin})

	if len(got) != 0 || len(failures.lookups) != 1 || failures.lookups[0] != berlin {
		t.Fatalf("DocumentAmountsOf = %v with lookup failures %v, want a miss reported for berlin",
			got, failures.lookups)
	}
}

func TestAnAmountTheServerRefusesToStoreIsReported(t *testing.T) {
	t.Parallel()

	failures := &recordedFailures{}
	berlin := yacymodel.WordHash("berlin")

	amountsOn(refusingServer(t), failures).Remember(t.Context(), map[yacymodel.Hash]int{berlin: 12})

	if len(failures.stores) != 1 || failures.stores[0] != berlin {
		t.Fatalf("store failures = %v, want one for berlin", failures.stores)
	}
}

func TestAValueThatIsNotAnAmountIsReportedAndIsAMiss(t *testing.T) {
	t.Parallel()

	server := miniredis.RunT(t)
	failures := &recordedFailures{}
	berlin := yacymodel.WordHash("berlin")
	if err := server.Set(
		"query-word-document-amounts:"+berlin.String(),
		"many",
	); err != nil {
		t.Fatalf("set: %v", err)
	}

	got := amountsOn(server, failures).DocumentAmountsOf(t.Context(), []yacymodel.Hash{berlin})

	if len(got) != 0 || len(failures.lookups) != 1 {
		t.Fatalf("DocumentAmountsOf = %v with lookup failures %v, want a reported miss",
			got, failures.lookups)
	}
}
