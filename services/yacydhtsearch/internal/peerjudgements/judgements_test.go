package peerjudgements_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgementledgers/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	retrialInterval       = 24 * time.Hour
	ledgerCapacity        = 16
	versionOfThePeer      = "yacy_v1.925"
	versionAfterAnUpgrade = "yacy_v1.930"

	question peerjudgements.Question = "a question about the peer"
)

type clock struct {
	reading time.Time
}

func (c *clock) Now() time.Time {
	return c.reading
}

func (c *clock) moveOn(by time.Duration) {
	c.reading = c.reading.Add(by)
}

func clockAtTheStartOfTheJudging() *clock {
	return &clock{reading: time.Date(2026, time.September, 22, 12, 0, 0, 0, time.UTC)}
}

func newJudgements(now *clock) peerjudgements.Judgements {
	return peerjudgements.New(
		question,
		memory.New(ledgerCapacity),
		retrialInterval,
		now.Now,
	)
}

func peerAtVersion(version string) peerjudgements.PeerAtVersion {
	return peerjudgements.PeerAtVersion{Peer: yacymodel.WordHash("peer"), Version: version}
}

func judgedPeer(version string, judgement peerjudgements.Judgement) peerjudgements.JudgedPeer {
	return peerjudgements.JudgedPeer{
		PeerAtVersion: peerAtVersion(version),
		Judgement:     judgement,
	}
}

func standingOf(
	t *testing.T,
	judgements peerjudgements.Judgements,
	peer peerjudgements.PeerAtVersion,
) peerjudgements.Standing {
	t.Helper()

	standings := judgements.StandingsOf(t.Context(), []peerjudgements.PeerAtVersion{peer})
	if len(standings) != 1 {
		t.Fatalf("StandingsOf gave %d standings, want one", len(standings))
	}

	return standings[0].Standing
}

func TestAPeerThatWasNeverJudgedStandsAsNeverJudged(t *testing.T) {
	t.Parallel()

	judgements := newJudgements(clockAtTheStartOfTheJudging())

	standing := standingOf(t, judgements, peerAtVersion(versionOfThePeer))

	if standing != peerjudgements.NeverJudged {
		t.Fatalf("standing = %q, want %q", standing, peerjudgements.NeverJudged)
	}
}

func TestAPeerJudgedHonoredAtTheVersionItStillClaimsStandsAsHonoring(t *testing.T) {
	t.Parallel()

	judgements := newJudgements(clockAtTheStartOfTheJudging())
	judgements.Add(t.Context(), []peerjudgements.JudgedPeer{
		judgedPeer(versionOfThePeer, peerjudgements.Honored),
	})

	standing := standingOf(t, judgements, peerAtVersion(versionOfThePeer))

	if standing != peerjudgements.Honoring {
		t.Fatalf("standing = %q, want %q", standing, peerjudgements.Honoring)
	}
}

func TestAPeerJudgedIgnoredAtTheVersionItStillClaimsStandsAsIgnoring(t *testing.T) {
	t.Parallel()

	judgements := newJudgements(clockAtTheStartOfTheJudging())
	judgements.Add(t.Context(), []peerjudgements.JudgedPeer{
		judgedPeer(versionOfThePeer, peerjudgements.Ignored),
	})

	standing := standingOf(t, judgements, peerAtVersion(versionOfThePeer))

	if standing != peerjudgements.Ignoring {
		t.Fatalf("standing = %q, want %q", standing, peerjudgements.Ignoring)
	}
}

func TestAPeerThatClaimsAnotherVersionStandsAsVersionChanged(t *testing.T) {
	t.Parallel()

	judgements := newJudgements(clockAtTheStartOfTheJudging())
	judgements.Add(t.Context(), []peerjudgements.JudgedPeer{
		judgedPeer(versionOfThePeer, peerjudgements.Ignored),
	})

	standing := standingOf(t, judgements, peerAtVersion(versionAfterAnUpgrade))

	if standing != peerjudgements.VersionChanged {
		t.Fatalf("standing = %q, want %q", standing, peerjudgements.VersionChanged)
	}
}

func TestAPeerJudgedAtAVersionItKeepsStandsAsJudgedAfterTheRetrialInterval(t *testing.T) {
	t.Parallel()

	now := clockAtTheStartOfTheJudging()
	judgements := newJudgements(now)
	judgements.Add(t.Context(), []peerjudgements.JudgedPeer{
		judgedPeer(versionOfThePeer, peerjudgements.Ignored),
	})
	now.moveOn(retrialInterval)

	standing := standingOf(t, judgements, peerAtVersion(versionOfThePeer))

	if standing != peerjudgements.Ignoring {
		t.Fatalf("standing = %q, want %q", standing, peerjudgements.Ignoring)
	}
}

func TestAPeerThatClaimsNoVersionStandsAsJudgedWithinTheRetrialInterval(t *testing.T) {
	t.Parallel()

	now := clockAtTheStartOfTheJudging()
	judgements := newJudgements(now)
	judgements.Add(t.Context(), []peerjudgements.JudgedPeer{
		judgedPeer(versionOfThePeer, peerjudgements.Honored),
	})
	now.moveOn(retrialInterval - time.Minute)

	standing := standingOf(t, judgements, peerAtVersion(""))

	if standing != peerjudgements.Honoring {
		t.Fatalf("standing = %q, want %q", standing, peerjudgements.Honoring)
	}
}

func TestAPeerJudgedWithoutAVersionStandsAsJudgedWithinTheRetrialInterval(t *testing.T) {
	t.Parallel()

	now := clockAtTheStartOfTheJudging()
	judgements := newJudgements(now)
	judgements.Add(t.Context(), []peerjudgements.JudgedPeer{
		judgedPeer("", peerjudgements.Ignored),
	})
	now.moveOn(retrialInterval - time.Minute)

	standing := standingOf(t, judgements, peerAtVersion(versionOfThePeer))

	if standing != peerjudgements.Ignoring {
		t.Fatalf("standing = %q, want %q", standing, peerjudgements.Ignoring)
	}
}

func TestAPeerThatClaimsNoVersionStandsAsIntervalPassedAfterTheRetrialInterval(t *testing.T) {
	t.Parallel()

	now := clockAtTheStartOfTheJudging()
	judgements := newJudgements(now)
	judgements.Add(t.Context(), []peerjudgements.JudgedPeer{
		judgedPeer(versionOfThePeer, peerjudgements.Honored),
	})
	now.moveOn(retrialInterval)

	standing := standingOf(t, judgements, peerAtVersion(""))

	if standing != peerjudgements.IntervalPassed {
		t.Fatalf("standing = %q, want %q", standing, peerjudgements.IntervalPassed)
	}
}

func TestAJudgementOfNoEvidenceLeavesTheJudgementAsItWas(t *testing.T) {
	t.Parallel()

	judgements := newJudgements(clockAtTheStartOfTheJudging())
	judgements.Add(t.Context(), []peerjudgements.JudgedPeer{
		judgedPeer(versionOfThePeer, peerjudgements.Honored),
	})
	judgements.Add(t.Context(), []peerjudgements.JudgedPeer{
		judgedPeer(versionOfThePeer, peerjudgements.NoEvidence),
	})

	standing := standingOf(t, judgements, peerAtVersion(versionOfThePeer))

	if standing != peerjudgements.Honoring {
		t.Fatalf("standing = %q, want %q", standing, peerjudgements.Honoring)
	}
}

func TestAJudgementOfNoEvidenceOnAPeerNeverJudgedLeavesItNeverJudged(t *testing.T) {
	t.Parallel()

	judgements := newJudgements(clockAtTheStartOfTheJudging())
	judgements.Add(t.Context(), []peerjudgements.JudgedPeer{
		judgedPeer(versionOfThePeer, peerjudgements.NoEvidence),
	})

	standing := standingOf(t, judgements, peerAtVersion(versionOfThePeer))

	if standing != peerjudgements.NeverJudged {
		t.Fatalf("standing = %q, want %q", standing, peerjudgements.NeverJudged)
	}
}
