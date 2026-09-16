package peerreliability_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerreliability"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	maturationDuration = 7 * 24 * time.Hour
	stalenessHorizon   = time.Hour
)

var theInstant = time.Date(2026, time.September, 15, 12, 0, 0, 0, time.UTC)

type observedPeer struct {
	earnedPresence time.Duration
	latestAnswer   time.Time
}

func (observed observedPeer) EarnedPresenceOf(
	_ context.Context,
	_ peeranswerhistory.PeerAtAddress,
) time.Duration {
	return observed.earnedPresence
}

func (observed observedPeer) LatestAnswerOf(
	_ context.Context,
	_ peeranswerhistory.PeerAtAddress,
) time.Time {
	return observed.latestAnswer
}

func reliabilityOf(
	t *testing.T,
	earnedPresence time.Duration,
	latestAnswer time.Time,
	now time.Time,
) float64 {
	t.Helper()

	return peerreliability.New(
		observedPeer{earnedPresence: earnedPresence, latestAnswer: latestAnswer},
		peerreliability.ReliabilityWeights{
			MaturationDuration: maturationDuration,
			StalenessHorizon:   stalenessHorizon,
		},
		func() time.Time { return now },
	).ReliabilityOf(t.Context(), peeranswerhistory.PeerAtAddress{
		Hash:    yacymodel.WordHash("peer"),
		Address: "http://10.0.0.1:8090",
	})
}

func TestPresenceEarnsNoMoreReliabilityOnceItHasMatured(t *testing.T) {
	t.Parallel()

	matured := reliabilityOf(t, maturationDuration, theInstant, theInstant)
	overdue := reliabilityOf(t, 10*maturationDuration, theInstant, theInstant)

	if matured != overdue {
		t.Fatalf(
			"matured = %v, ten times matured = %v, want the same reliability",
			matured,
			overdue,
		)
	}
	if halfMatured := reliabilityOf(
		t, maturationDuration/2, theInstant, theInstant,
	); !(halfMatured < matured) {
		t.Fatalf("half matured = %v, matured = %v, want less reliability", halfMatured, matured)
	}
}

func TestAPeerWithNoPresenceIsNotReliable(t *testing.T) {
	t.Parallel()

	reliability := reliabilityOf(t, 0, theInstant, theInstant)

	if reliability != 0 {
		t.Fatalf("ReliabilityOf = %v, want no reliability for no presence", reliability)
	}
}

func TestAMaturedPeerIsWhollyReliableWhileItsLatestAnswerIsFresh(t *testing.T) {
	t.Parallel()

	reliability := reliabilityOf(t, maturationDuration, theInstant, theInstant)

	if reliability != 1 {
		t.Fatalf("ReliabilityOf = %v, want the whole reliability of 1", reliability)
	}
}

func TestReliabilityFallsAsTheLatestAnswerGoesStale(t *testing.T) {
	t.Parallel()

	fresh := reliabilityOf(t, maturationDuration, theInstant, theInstant)
	halfStale := reliabilityOf(
		t, maturationDuration, theInstant, theInstant.Add(stalenessHorizon/2),
	)
	stale := reliabilityOf(
		t, maturationDuration, theInstant, theInstant.Add(2*stalenessHorizon),
	)

	if !(fresh > halfStale && halfStale > stale) {
		t.Fatalf(
			"fresh = %v, half stale = %v, stale = %v, want a falling reliability",
			fresh,
			halfStale,
			stale,
		)
	}
	if stale != 0 {
		t.Fatalf("stale = %v, want no reliability past the staleness horizon", stale)
	}
}
