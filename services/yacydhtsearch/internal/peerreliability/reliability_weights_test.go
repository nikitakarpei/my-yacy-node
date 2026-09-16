package peerreliability_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerreliability"
)

const (
	maturationDuration = 7 * 24 * time.Hour
	stalenessHorizon   = time.Hour
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

func TestPresenceEarnsNoMoreReliabilityOnceItHasMatured(t *testing.T) {
	t.Parallel()

	matured := weights().ReliabilityOf(maturationDuration, answeredAt(), answeredAt())
	overdue := weights().ReliabilityOf(10*maturationDuration, answeredAt(), answeredAt())

	if matured != overdue {
		t.Fatalf(
			"matured = %v, ten times matured = %v, want the same reliability",
			matured,
			overdue,
		)
	}
	if halfMatured := weights().ReliabilityOf(
		maturationDuration/2, answeredAt(), answeredAt(),
	); !(halfMatured < matured) {
		t.Fatalf("half matured = %v, matured = %v, want less reliability", halfMatured, matured)
	}
}

func TestAPeerWithNoPresenceIsNotReliable(t *testing.T) {
	t.Parallel()

	reliability := weights().ReliabilityOf(0, answeredAt(), answeredAt())

	if reliability != 0 {
		t.Fatalf("ReliabilityOf = %v, want no reliability for no presence", reliability)
	}
}

func TestAMaturedPeerIsWhollyReliableWhileItsLatestAnswerIsFresh(t *testing.T) {
	t.Parallel()

	reliability := weights().ReliabilityOf(maturationDuration, answeredAt(), answeredAt())

	if reliability != 1 {
		t.Fatalf("ReliabilityOf = %v, want the whole reliability of 1", reliability)
	}
}

func TestReliabilityFallsAsTheLatestAnswerGoesStale(t *testing.T) {
	t.Parallel()

	fresh := weights().ReliabilityOf(maturationDuration, answeredAt(), answeredAt())
	halfStale := weights().ReliabilityOf(
		maturationDuration, answeredAt(), answeredAt().Add(stalenessHorizon/2),
	)
	stale := weights().ReliabilityOf(
		maturationDuration, answeredAt(), answeredAt().Add(2*stalenessHorizon),
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
