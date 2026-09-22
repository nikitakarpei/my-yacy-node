// Package peerjudgements records, per peer and per question about the peer, the
// judgement its answers gave, and turns a recorded judgement and the version
// the peer claims now into the standing of the peer. One Judgements is built
// per question over one shared ledger.
package peerjudgements

import (
	"context"
	"time"
)

type Question string

type Judgements struct {
	question        Question
	ledger          JudgementLedger
	retrialInterval time.Duration
	now             func() time.Time
}

func New(
	question Question,
	ledger JudgementLedger,
	retrialInterval time.Duration,
	now func() time.Time,
) Judgements {
	return Judgements{
		question:        question,
		ledger:          ledger,
		retrialInterval: retrialInterval,
		now:             now,
	}
}

func (j Judgements) StandingsOf(ctx context.Context, peers []PeerAtVersion) PeerStandings {
	standings := make(PeerStandings, 0, len(peers))
	for _, peer := range peers {
		standings = append(standings, j.standingOf(ctx, peer))
	}

	return standings
}

func (j Judgements) standingOf(ctx context.Context, peer PeerAtVersion) PeerStanding {
	return PeerStanding{
		PeerAtVersion: peer,
		Standing: standingFrom(
			j.ledger.JudgementOf(ctx, peer.Peer, j.question),
			peer.Version,
			j.retrialInterval,
			j.now(),
		),
	}
}

func (j Judgements) Add(ctx context.Context, judgedPeers []JudgedPeer) {
	for _, judgedPeer := range judgedPeers {
		if judgedPeer.Judgement == NoEvidence {
			continue
		}
		j.ledger.HoldJudgement(ctx, RecordedJudgement{
			Question:   j.question,
			JudgedPeer: judgedPeer,
			JudgedAt:   j.now(),
		})
	}
}
