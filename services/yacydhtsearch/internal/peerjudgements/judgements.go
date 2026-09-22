// Package peerjudgements records, per peer and per form of an ask, whether the
// peer honors the form, and turns a recorded judgement and the version the peer
// claims now into the standing of the peer. One Judgements is built per form
// over one shared ledger.
package peerjudgements

import (
	"context"
	"time"
)

type Form string

const NamedDocuments Form = "named documents"

type Judgements struct {
	form            Form
	ledger          JudgementLedger
	retrialInterval time.Duration
	now             func() time.Time
	observer        JudgementsObserver
}

func New(
	form Form,
	ledger JudgementLedger,
	retrialInterval time.Duration,
	now func() time.Time,
	observer JudgementsObserver,
) Judgements {
	return Judgements{
		form:            form,
		ledger:          ledger,
		retrialInterval: retrialInterval,
		now:             now,
		observer:        observer,
	}
}

func (j Judgements) StandingsOf(ctx context.Context, peers []PeerAtVersion) []PeerStanding {
	standings := make([]PeerStanding, 0, len(peers))
	for _, peer := range peers {
		standing := j.standingOf(ctx, peer)
		j.observer.PeerStood(ctx, j.form, standing)
		standings = append(standings, standing)
	}

	return standings
}

func (j Judgements) standingOf(ctx context.Context, peer PeerAtVersion) PeerStanding {
	return PeerStanding{
		PeerAtVersion: peer,
		Standing: standingFrom(
			j.ledger.JudgementOf(ctx, j.form, peer.Peer),
			peer.Version,
			j.retrialInterval,
			j.now(),
		),
	}
}

func (j Judgements) Add(ctx context.Context, judgedPeers []JudgedPeer) {
	for _, judgedPeer := range judgedPeers {
		j.observer.PeerJudged(ctx, j.form, judgedPeer)
		if judgedPeer.Judgement == NoEvidence {
			continue
		}
		j.ledger.HoldJudgement(ctx, RecordedJudgement{
			Form:       j.form,
			JudgedPeer: judgedPeer,
			JudgedAt:   j.now(),
		})
	}
}
