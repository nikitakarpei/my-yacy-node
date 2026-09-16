package peerreliability_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerreliability"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	maturationDuration = 7 * 24 * time.Hour
	stalenessHorizon   = time.Hour
	answeringAddress   = "http://peer.example:8090"
)

func weights() peerreliability.ReliabilityWeights {
	return peerreliability.ReliabilityWeights{
		MaturationDuration: maturationDuration,
		StalenessHorizon:   stalenessHorizon,
	}
}

func answeredAt() time.Time {
	return time.Date(2026, time.September, 15, 12, 0, 0, 0, time.UTC)
}

func peerObservedFor(t *testing.T, presence time.Duration) presenceaccrual.ObservedPeer {
	t.Helper()

	hash, err := yacymodel.ParseHash("aaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("ParseHash: %v", err)
	}

	return presenceaccrual.ObservedPeer{
		PeerAtAddress: presenceaccrual.PeerAtAddress{
			Hash:    hash,
			Address: answeringAddress,
		},
		FirstAnsweredAt:  answeredAt().Add(-presence),
		LatestAnsweredAt: answeredAt(),
		Presence:         presence,
	}
}

func TestPresenceEarnsNoMoreReliabilityOnceItHasMatured(t *testing.T) {
	t.Parallel()

	matured := weights().ReliabilityOf(peerObservedFor(t, maturationDuration), answeredAt())
	overdue := weights().ReliabilityOf(peerObservedFor(t, 10*maturationDuration), answeredAt())

	if matured != overdue {
		t.Fatalf(
			"matured = %v, ten times matured = %v, want the same reliability",
			matured,
			overdue,
		)
	}
	if halfMatured := weights().ReliabilityOf(
		peerObservedFor(t, maturationDuration/2), answeredAt(),
	); !(halfMatured < matured) {
		t.Fatalf("half matured = %v, matured = %v, want less reliability", halfMatured, matured)
	}
}

func TestAPeerWithNoPresenceIsNotReliable(t *testing.T) {
	t.Parallel()

	reliability := weights().ReliabilityOf(peerObservedFor(t, 0), answeredAt())

	if reliability != 0 {
		t.Fatalf("ReliabilityOf = %v, want no reliability for no presence", reliability)
	}
}

func TestAMaturedPeerIsWhollyReliableWhileItsLatestAnswerIsFresh(t *testing.T) {
	t.Parallel()

	reliability := weights().ReliabilityOf(peerObservedFor(t, maturationDuration), answeredAt())

	if reliability != 1 {
		t.Fatalf("ReliabilityOf = %v, want the whole reliability of 1", reliability)
	}
}

func TestReliabilityFallsAsTheLatestAnswerGoesStale(t *testing.T) {
	t.Parallel()

	observedPeer := peerObservedFor(t, maturationDuration)

	fresh := weights().ReliabilityOf(observedPeer, answeredAt())
	halfStale := weights().ReliabilityOf(observedPeer, answeredAt().Add(stalenessHorizon/2))
	stale := weights().ReliabilityOf(observedPeer, answeredAt().Add(2*stalenessHorizon))

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
