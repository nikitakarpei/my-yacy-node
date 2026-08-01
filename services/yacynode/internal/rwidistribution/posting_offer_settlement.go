package rwidistribution

import (
	"context"
	"log/slog"
	"time"
)

type postingOfferSettlement struct {
	ledger     *replicaLedger
	schedule   *postingOfferSchedule
	cadence    postingOfferCadence
	observer   PostingOfferCycleObserver
	now        func() time.Time
	redundancy int
}

func (s *postingOfferSettlement) Apply(
	ctx context.Context,
	postings []postingReplication,
	accepted []postingOffer,
	retryAfters *postingRetryAfters,
) {
	s.recordAccepted(ctx, accepted)
	s.dropStaleReplicas(ctx, postings)
	s.reschedule(ctx, postings, retryAfters)
}

func (s *postingOfferSettlement) recordAccepted(ctx context.Context, accepted []postingOffer) {
	for _, offer := range accepted {
		if err := s.ledger.RecordAccepted(ctx, offer); err != nil {
			slog.WarnContext(ctx, "replicas not recorded",
				slog.String("peer", offer.Peer.Hash.String()),
				slog.Any("error", err))
		}
	}
}

func (s *postingOfferSettlement) dropStaleReplicas(
	ctx context.Context,
	postings []postingReplication,
) {
	for _, replication := range postings {
		if len(replication.PeerHashesNoLongerResponsible) == 0 {
			continue
		}

		word, url := replication.Posting.WordHash, replication.Posting.URLHash
		dropped, err := s.ledger.RecordDropped(
			ctx, word, url, replication.PeerHashesNoLongerResponsible,
		)
		if err != nil {
			slog.WarnContext(ctx, "stale replicas not dropped",
				slog.String("word", word.String()),
				slog.String("url", url.String()),
				slog.Any("error", err))

			continue
		}
		if dropped > 0 {
			s.observer.ObserveLedgerPrune(dropped)
		}
	}
}

func (s *postingOfferSettlement) reschedule(
	ctx context.Context,
	postings []postingReplication,
	retryAfters *postingRetryAfters,
) {
	now := s.now()
	for _, replication := range postings {
		id := postingIdentity{Word: replication.Posting.WordHash, URL: replication.Posting.URLHash}
		at := s.cadence.NextDue(now, s.redundancyMet(ctx, replication), retryAfters.RetryAfter(id))

		if err := s.schedule.Reschedule(ctx, id.Word, id.URL, at); err != nil {
			slog.WarnContext(ctx, "posting not rescheduled",
				slog.String("word", id.Word.String()),
				slog.String("url", id.URL.String()),
				slog.Any("error", err))
		}
	}
}

func (s *postingOfferSettlement) redundancyMet(
	ctx context.Context,
	replication postingReplication,
) bool {
	if replication.CopiesNeeded == 0 {
		return true
	}

	word, url := replication.Posting.WordHash, replication.Posting.URLHash
	replicas, err := s.ledger.Replicas(ctx, word, url)
	if err != nil {
		slog.WarnContext(ctx, "replica count not read",
			slog.String("word", word.String()),
			slog.String("url", url.String()),
			slog.Any("error", err))

		return false
	}

	return len(replicas) >= s.redundancy
}
