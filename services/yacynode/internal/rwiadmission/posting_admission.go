package rwiadmission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

var errEscrowFull = errors.New("escrow holds as many postings as it can")

var _ PostingReceiver = PostingAdmission{}

type PostingAdmission struct {
	vault     *vault.Vault
	urls      urlmeta.URLDirectory
	escrow    Escrow
	judge     postingJudge
	admitter  rwipostings.PostingAdmitter
	stopwords Stopwords
	observer  Observer

	postingCap int
	pause      time.Duration
}

func (a PostingAdmission) Receive(
	ctx context.Context,
	origin rwiorigin.Origin,
	postings []yacymodel.RWIPosting,
) (Receipt, error) {
	if len(postings) > a.postingCap {
		return a.busyReceiptRefusing(OutcomeRefusedTooManyPostings, postings), nil
	}

	atCapacity, err := a.vault.AtCapacity(ctx)
	if err != nil {
		return Receipt{}, fmt.Errorf("check capacity: %w", err)
	}
	if atCapacity {
		return a.busyReceiptRefusing(OutcomeRefusedStorageFull, postings), nil
	}

	escrowFull, err := a.escrow.AtCapacity(ctx)
	if err != nil {
		return Receipt{}, fmt.Errorf("check escrow capacity: %w", err)
	}

	postingsWithoutStopwords := a.postingsWithoutStopwordsOf(postings)
	referenced := referencedURLsOf(postingsWithoutStopwords)

	var unknown []yacymodel.URLHash

	err = a.vault.Update(ctx, func(tx *vault.Txn) error {
		a.observe(tx, OutcomeDroppedStopword, len(postings)-len(postingsWithoutStopwords))
		currentVersions, err := a.urls.CurrentVersionsOf(tx, referenced)
		if err != nil {
			return fmt.Errorf("current versions of urls: %w", err)
		}
		awaitedURLs, err := a.placeEach(ctx, tx, origin, postingsWithoutStopwords, currentVersions)
		if err != nil {
			return err
		}
		if escrowFull && len(awaitedURLs) > 0 {
			return errEscrowFull
		}
		unknown = awaitedURLs

		return nil
	})
	if errors.Is(err, vault.ErrAtCapacity) {
		return a.busyReceiptRefusing(OutcomeRefusedStorageFull, postings), nil
	}
	if errors.Is(err, errEscrowFull) {
		return a.busyReceiptRefusing(OutcomeRefusedEscrowFull, postings), nil
	}
	if err != nil {
		return Receipt{}, fmt.Errorf("receive rwi: %w", err)
	}

	return Receipt{UnknownURL: unknown}, nil
}

func (a PostingAdmission) busyReceiptRefusing(
	outcome Outcome,
	postings []yacymodel.RWIPosting,
) Receipt {
	a.observer.ObserveOutcome(outcome, len(postings))

	return Receipt{Busy: true, Pause: a.pause}
}

func (a PostingAdmission) postingsWithoutStopwordsOf(
	postings []yacymodel.RWIPosting,
) []yacymodel.RWIPosting {
	postingsWithoutStopwords := make([]yacymodel.RWIPosting, 0, len(postings))
	for _, posting := range postings {
		if !a.stopwords.Contains(posting.WordHash, posting.Language) {
			postingsWithoutStopwords = append(postingsWithoutStopwords, posting)
		}
	}

	return postingsWithoutStopwords
}

func referencedURLsOf(postings []yacymodel.RWIPosting) []yacymodel.URLHash {
	referenced := make(map[yacymodel.URLHash]struct{}, len(postings))
	urls := make([]yacymodel.URLHash, 0, len(postings))
	for _, posting := range postings {
		if _, seen := referenced[posting.URLHash]; !seen {
			referenced[posting.URLHash] = struct{}{}
			urls = append(urls, posting.URLHash)
		}
	}

	return urls
}

func (a PostingAdmission) observe(tx *vault.Txn, outcome Outcome, postings int) {
	tx.RunAfterCommit(func() { a.observer.ObserveOutcome(outcome, postings) })
}

func (a PostingAdmission) placeEach(
	ctx context.Context,
	tx *vault.Txn,
	origin rwiorigin.Origin,
	postings []yacymodel.RWIPosting,
	currentVersions map[yacymodel.URLHash]urlmeta.CurrentVersion,
) ([]yacymodel.URLHash, error) {
	var awaitedURLs []yacymodel.URLHash
	awaited := map[yacymodel.URLHash]struct{}{}
	for _, posting := range postings {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context: %w", err)
		}
		wasHeld, err := a.placeReceived(
			tx,
			origin,
			posting,
			currentVersionOf(currentVersions, posting.URLHash),
		)
		if err != nil {
			return nil, err
		}
		if _, listed := awaited[posting.URLHash]; wasHeld && !listed {
			awaited[posting.URLHash] = struct{}{}
			awaitedURLs = append(awaitedURLs, posting.URLHash)
		}
	}

	return awaitedURLs, nil
}

func currentVersionOf(
	currentVersions map[yacymodel.URLHash]urlmeta.CurrentVersion,
	url yacymodel.URLHash,
) yacymodel.Optional[urlmeta.CurrentVersion] {
	version, found := currentVersions[url]
	if !found {
		return yacymodel.None[urlmeta.CurrentVersion]()
	}

	return yacymodel.Some(version)
}

func (a PostingAdmission) placeReceived(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	currentVersion yacymodel.Optional[urlmeta.CurrentVersion],
) (bool, error) {
	outcome, err := a.judge.outcomeOf(tx, origin, posting, currentVersion)
	if err != nil {
		return false, err
	}
	if outcome == OutcomeHeldInEscrow {
		if err := a.escrow.Hold(tx, origin, posting); err != nil {
			return false, fmt.Errorf("hold posting: %w", err)
		}

		return true, nil
	}
	a.observe(tx, outcome, 1)

	return false, a.carryOut(tx, posting, outcome)
}

func (a PostingAdmission) carryOut(
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
	outcome Outcome,
) error {
	if outcome != OutcomeIndexed {
		return nil
	}
	if err := a.admitter.Admit(tx, posting); err != nil {
		return fmt.Errorf("admit posting: %w", err)
	}

	return nil
}

func (a PostingAdmission) URLStored(
	tx *vault.Txn,
	url yacymodel.URLHash,
	currentVersion urlmeta.CurrentVersion,
) error {
	waiting, err := a.escrow.PostingsWaitingFor(tx, url)
	if err != nil {
		return fmt.Errorf("postings waiting for url: %w", err)
	}
	for _, posting := range waiting {
		if err := a.placeWaiting(tx, posting, currentVersion); err != nil {
			return err
		}
	}

	return nil
}

func (a PostingAdmission) placeWaiting(
	tx *vault.Txn,
	waiting rwiescrow.WaitingPosting,
	currentVersion urlmeta.CurrentVersion,
) error {
	outcome, err := a.judge.outcomeOf(
		tx,
		waiting.Origin,
		waiting.Posting,
		yacymodel.Some(currentVersion),
	)
	if err != nil {
		return err
	}
	if outcome == OutcomeHeldInEscrow {
		return nil
	}
	a.observe(tx, outcome, 1)
	if err := a.carryOut(tx, waiting.Posting, outcome); err != nil {
		return err
	}
	if err := a.escrow.Release(tx, waiting.Posting); err != nil {
		return fmt.Errorf("release posting: %w", err)
	}

	return nil
}
