package peerjudgements

import (
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Standing string

const (
	Honoring       Standing = "honoring"
	Ignoring       Standing = "ignoring"
	NeverJudged    Standing = "never judged"
	VersionChanged Standing = "version changed"
	IntervalPassed Standing = "interval passed"
)

type PeerAtVersion struct {
	Peer    yacymodel.Hash
	Version string
}

type PeerStanding struct {
	PeerAtVersion
	Standing Standing
}

type PeerStandings []PeerStanding

func (standings PeerStandings) StandingOf(peer yacymodel.Hash) Standing {
	place := slices.IndexFunc(standings, func(standing PeerStanding) bool {
		return standing.Peer == peer
	})

	return standings[place].Standing
}

func standingFrom(
	recordedJudgement yacymodel.Optional[RecordedJudgement],
	versionClaimed string,
	retrialInterval time.Duration,
	now time.Time,
) Standing {
	judgement, wasJudged := recordedJudgement.Get()
	switch {
	case !wasJudged:
		return NeverJudged
	case judgement.Version == "" || versionClaimed == "":
		return standingByTheRetrialInterval(judgement, retrialInterval, now)
	case judgement.Version != versionClaimed:
		return VersionChanged
	default:
		return standingAsJudged(judgement.Judgement)
	}
}

func standingByTheRetrialInterval(
	judgement RecordedJudgement,
	retrialInterval time.Duration,
	now time.Time,
) Standing {
	if now.Sub(judgement.JudgedAt) >= retrialInterval {
		return IntervalPassed
	}

	return standingAsJudged(judgement.Judgement)
}

func standingAsJudged(judgement Judgement) Standing {
	if judgement == Honored {
		return Honoring
	}

	return Ignoring
}
