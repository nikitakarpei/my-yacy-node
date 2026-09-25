package urlmetadataaskceilings_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	peercallobserversurlmetadataaskceilings "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallobservers/urlmetadataaskceilings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/urlmetadataaskceilings"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	mostDocuments  = 1000
	leastDocuments = 25
	targetTime     = time.Second
	peerAddress    = "http://peer.example"
)

var errCall = errors.New("the call failed")

type noCeilingObserver struct{}

func (noCeilingObserver) AskCeilingSet(context.Context, string, time.Duration, int) {}

type peerPacesInAMap map[string]time.Duration

func (paces peerPacesInAMap) Read(
	_ context.Context,
	address string,
) yacymodel.Optional[time.Duration] {
	pace, remembered := paces[address]
	if !remembered {
		return yacymodel.None[time.Duration]()
	}

	return yacymodel.Some(pace)
}

func (paces peerPacesInAMap) Update(
	ctx context.Context,
	address string,
	updated func(pace yacymodel.Optional[time.Duration]) time.Duration,
) time.Duration {
	paces[address] = updated(paces.Read(ctx, address))

	return paces[address]
}

type peerCallOutcome func(ctx context.Context, peerCalls peercallobserversurlmetadataaskceilings.PeerCallCeilings)

func ceilingAfter(t *testing.T, outcome peerCallOutcome) int {
	t.Helper()

	ceilings := urlmetadataaskceilings.New(
		mostDocuments,
		leastDocuments,
		targetTime,
		peerPacesInAMap{},
		noCeilingObserver{},
	)
	outcome(t.Context(), peercallobserversurlmetadataaskceilings.New(ceilings))

	return ceilings.CeilingOf(t.Context(), peerAddress)
}

func TestEachURLMetadataOutcomeFeedsTheCeilings(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		outcome peerCallOutcome
		ceiling int
	}{
		"answered": {func(ctx context.Context, peerCalls peercallobserversurlmetadataaskceilings.PeerCallCeilings) {
			peerCalls.PeerAnsweredURLMetadata(
				ctx, peerAddress, mostDocuments, 3, 2500*time.Millisecond,
			)
		}, 400},
		"cancelled": {func(ctx context.Context, peerCalls peercallobserversurlmetadataaskceilings.PeerCallCeilings) {
			peerCalls.PeerCallCancelled(
				ctx, peerAddress, peerasks.URLMetadata, mostDocuments, 1250*time.Millisecond,
			)
		}, 800},
		"refused": {func(ctx context.Context, peerCalls peercallobserversurlmetadataaskceilings.PeerCallCeilings) {
			peerCalls.PeerRefused(
				ctx, peerAddress, peerasks.URLMetadata, mostDocuments, http.StatusForbidden, time.Second,
			)
		}, leastDocuments},
		"unreachable": {func(ctx context.Context, peerCalls peercallobserversurlmetadataaskceilings.PeerCallCeilings) {
			peerCalls.PeerUnreachable(
				ctx, peerAddress, peerasks.URLMetadata, mostDocuments, errCall, time.Second,
			)
		}, leastDocuments},
		"unreadable": {func(ctx context.Context, peerCalls peercallobserversurlmetadataaskceilings.PeerCallCeilings) {
			peerCalls.PeerAnswerUnreadable(
				ctx, peerAddress, peerasks.URLMetadata, mostDocuments, errCall, time.Second,
			)
		}, leastDocuments},
	} {
		if ceiling := ceilingAfter(t, test.outcome); ceiling != test.ceiling {
			t.Fatalf("ceiling after a %s call = %d, want %d", name, ceiling, test.ceiling)
		}
	}
}

func TestASearchOutcomeChangesNothing(t *testing.T) {
	t.Parallel()

	for name, outcome := range map[string]peerCallOutcome{
		"searched": func(ctx context.Context, peerCalls peercallobserversurlmetadataaskceilings.PeerCallCeilings) {
			peerCalls.PeerSearchedDocuments(ctx, peerAddress, 3, 2, 10*time.Second)
		},
		"cancelled": func(ctx context.Context, peerCalls peercallobserversurlmetadataaskceilings.PeerCallCeilings) {
			peerCalls.PeerCallCancelled(ctx, peerAddress, peerasks.SearchDocuments, 3, time.Second)
		},
		"refused": func(ctx context.Context, peerCalls peercallobserversurlmetadataaskceilings.PeerCallCeilings) {
			peerCalls.PeerRefused(
				ctx, peerAddress, peerasks.SearchDocuments, 3, http.StatusForbidden, time.Second,
			)
		},
		"unreachable": func(ctx context.Context, peerCalls peercallobserversurlmetadataaskceilings.PeerCallCeilings) {
			peerCalls.PeerUnreachable(ctx, peerAddress, peerasks.SearchDocuments, 3, errCall, time.Second)
		},
		"unreadable": func(ctx context.Context, peerCalls peercallobserversurlmetadataaskceilings.PeerCallCeilings) {
			peerCalls.PeerAnswerUnreadable(
				ctx, peerAddress, peerasks.SearchDocuments, 3, errCall, time.Second,
			)
		},
		"headers late": func(ctx context.Context, peerCalls peercallobserversurlmetadataaskceilings.PeerCallCeilings) {
			peerCalls.PeerHeadersLate(ctx, peerAddress, peerasks.SearchDocuments, 3, time.Second)
		},
	} {
		if ceiling := ceilingAfter(t, outcome); ceiling != mostDocuments {
			t.Fatalf("ceiling after a %s search call = %d, want %d", name, ceiling, mostDocuments)
		}
	}
}
