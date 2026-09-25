package askceilings_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallobservers/askceilings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/urlmetadataaskceilings"
)

const (
	mostDocuments  = 1000
	leastDocuments = 25
	targetTime     = time.Second
	peerAddress    = "http://peer.example"
)

var errCall = errors.New("the call failed")

type peerCallOutcome func(t *testing.T, peerCalls askceilings.PeerCallCeilings)

func ceilingAfter(t *testing.T, askedDocuments int, outcome peerCallOutcome) int {
	t.Helper()

	ceilings := urlmetadataaskceilings.New(
		mostDocuments, leastDocuments, targetTime, urlmetadataaskceilings.AskCeilingObservers{},
	)
	if askedDocuments > 0 {
		ceilings.Asked(peerAddress, askedDocuments)
	}
	outcome(t, askceilings.New(ceilings))
	ceiling, _ := ceilings.CeilingOf(t.Context(), peerAddress)

	return ceiling
}

func TestEachURLMetadataOutcomeFeedsItsSample(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		outcome peerCallOutcome
		ceiling int
	}{
		"answered": {func(t *testing.T, peerCalls askceilings.PeerCallCeilings) {
			t.Helper()
			peerCalls.PeerAnsweredURLMetadata(t.Context(), peerAddress, 3, 2500*time.Millisecond)
		}, 400},
		"cancelled": {func(t *testing.T, peerCalls askceilings.PeerCallCeilings) {
			t.Helper()
			peerCalls.PeerCallCancelled(
				t.Context(), peerAddress, peerasks.URLMetadata, 1250*time.Millisecond,
			)
		}, 800},
		"refused": {func(t *testing.T, peerCalls askceilings.PeerCallCeilings) {
			t.Helper()
			peerCalls.PeerRefused(
				t.Context(), peerAddress, peerasks.URLMetadata, http.StatusForbidden, time.Second,
			)
		}, leastDocuments},
		"unreachable": {func(t *testing.T, peerCalls askceilings.PeerCallCeilings) {
			t.Helper()
			peerCalls.PeerUnreachable(
				t.Context(), peerAddress, peerasks.URLMetadata, errCall, time.Second,
			)
		}, leastDocuments},
		"unreadable": {func(t *testing.T, peerCalls askceilings.PeerCallCeilings) {
			t.Helper()
			peerCalls.PeerAnswerUnreadable(
				t.Context(), peerAddress, peerasks.URLMetadata, errCall, time.Second,
			)
		}, leastDocuments},
	} {
		if ceiling := ceilingAfter(t, 1000, test.outcome); ceiling != test.ceiling {
			t.Fatalf("ceiling after a %s call = %d, want %d", name, ceiling, test.ceiling)
		}
	}
}

func TestASearchOutcomeOrAnOutcomeWithoutAnAskChangesNothing(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		askedDocuments int
		outcome        peerCallOutcome
	}{
		"cancelled search": {1000, func(t *testing.T, peerCalls askceilings.PeerCallCeilings) {
			t.Helper()
			peerCalls.PeerCallCancelled(t.Context(), peerAddress, peerasks.SearchDocuments, time.Second)
		}},
		"refused search": {1000, func(t *testing.T, peerCalls askceilings.PeerCallCeilings) {
			t.Helper()
			peerCalls.PeerRefused(
				t.Context(), peerAddress, peerasks.SearchDocuments, http.StatusForbidden, time.Second,
			)
		}},
		"unreachable search": {1000, func(t *testing.T, peerCalls askceilings.PeerCallCeilings) {
			t.Helper()
			peerCalls.PeerUnreachable(
				t.Context(), peerAddress, peerasks.SearchDocuments, errCall, time.Second,
			)
		}},
		"unreadable search": {1000, func(t *testing.T, peerCalls askceilings.PeerCallCeilings) {
			t.Helper()
			peerCalls.PeerAnswerUnreadable(
				t.Context(), peerAddress, peerasks.SearchDocuments, errCall, time.Second,
			)
		}},
		"URL metadata failure without an ask": {0, func(t *testing.T, peerCalls askceilings.PeerCallCeilings) {
			t.Helper()
			peerCalls.PeerUnreachable(
				t.Context(), peerAddress, peerasks.URLMetadata, errCall, time.Second,
			)
		}},
		"URL metadata answer without an ask": {0, func(t *testing.T, peerCalls askceilings.PeerCallCeilings) {
			t.Helper()
			peerCalls.PeerAnsweredURLMetadata(t.Context(), peerAddress, 3, 10*time.Second)
		}},
	} {
		if ceiling := ceilingAfter(t, test.askedDocuments, test.outcome); ceiling != mostDocuments {
			t.Fatalf("ceiling after a %s = %d, want %d", name, ceiling, mostDocuments)
		}
	}
}
