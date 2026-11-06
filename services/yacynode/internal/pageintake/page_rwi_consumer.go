// Package pageintake stores the reverse word index of each page that the pagerwi service builds.
// The node takes an RWI only under the postings root it computes for those postings itself, so
// an RWI built for another DHT partition exponent halts the intake.
package pageintake

import (
	"context"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageadmission"
)

var ErrPostingsRootMismatch = errors.New(
	"rwi postings root differs from the root this node computes",
)

type PageRWIConsumer struct {
	pageRWISource            pullintake.MessageSource
	emptyProof               yacymodel.PostingsProof
	pageReceiver             pageadmission.PageReceiver
	pageIntakeObserver       PageIntakeObserver
	pageRWIIntakeConcurrency int
}

type PageRWIConsumerConfig struct {
	PageRWISource            pullintake.MessageSource
	Partitions               yacymodel.DHTRingPartitions
	PageReceiver             pageadmission.PageReceiver
	PageIntakeObserver       PageIntakeObserver
	PageRWIIntakeConcurrency int
}

func NewPageRWIConsumer(config PageRWIConsumerConfig) *PageRWIConsumer {
	return &PageRWIConsumer{
		pageRWISource:            config.PageRWISource,
		emptyProof:               yacymodel.EmptyPostingsProof(config.Partitions),
		pageReceiver:             config.PageReceiver,
		pageIntakeObserver:       config.PageIntakeObserver,
		pageRWIIntakeConcurrency: config.PageRWIIntakeConcurrency,
	}
}

func (c *PageRWIConsumer) Run(ctx context.Context) error {
	return pullintake.Run(
		ctx,
		pullintake.MessageQueueOf(c.pageRWISource),
		c.pageRWIIntakeConcurrency,
		c.takeIn,
	)
}

func (c *PageRWIConsumer) takeIn(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	pageRWI, err := pagerwicontract.UnmarshalPageRWI(message.Body())
	if err != nil {
		c.pageIntakeObserver.PageRWIInvalid(ctx)

		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	c.pageIntakeObserver.PageRWIReceived(ctx, message.Identity(), pageRWI.PageURL)
	pagePostings, err := c.pagePostingsOf(pageRWI.Postings)
	if err != nil {
		c.pageIntakeObserver.PageRWIInvalid(ctx)

		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	if rootOf(pagePostings) != pageRWI.Metadata.PostingsRoot {
		c.pageIntakeObserver.PostingsRootMismatched(ctx, message.Identity(), pageRWI.PageURL)

		return fmt.Errorf("%s: %w", message.Identity(), ErrPostingsRootMismatch)
	}
	c.store(ctx, message, pageRWI.PageURL, pageRWI.Metadata, pagePostings)

	return nil
}

func (c *PageRWIConsumer) pagePostingsOf(
	postings []yacymodel.RWIPosting,
) (yacymodel.Optional[yacymodel.PagePostings], error) {
	if len(postings) == 0 {
		return yacymodel.None[yacymodel.PagePostings](), nil
	}
	pagePostings, err := yacymodel.ProvenPagePostingsFrom(postings, c.emptyProof)
	if err != nil {
		return yacymodel.None[yacymodel.PagePostings](), fmt.Errorf("prove page postings: %w", err)
	}

	return yacymodel.Some(pagePostings), nil
}

func rootOf(
	pagePostings yacymodel.Optional[yacymodel.PagePostings],
) yacymodel.Optional[yacymodel.PostingsRoot] {
	if pagePostings, present := pagePostings.Get(); present {
		return pagePostings.Root()
	}

	return yacymodel.None[yacymodel.PostingsRoot]()
}

func (c *PageRWIConsumer) store(
	ctx context.Context,
	message pullintake.PendingMessage,
	pageURL canonicalurl.CanonicalURL,
	metadata yacymodel.URLMetadata,
	pagePostings yacymodel.Optional[yacymodel.PagePostings],
) {
	if !c.admitPage(ctx, message, pageURL, metadata, pagePostings) {
		return
	}
	message.Acknowledge(ctx)
	c.pageIntakeObserver.PageIndexed(ctx, message.Identity(), pageURL)
}

func (c *PageRWIConsumer) admitPage(
	ctx context.Context,
	message pullintake.PendingMessage,
	pageURL canonicalurl.CanonicalURL,
	metadata yacymodel.URLMetadata,
	pagePostings yacymodel.Optional[yacymodel.PagePostings],
) bool {
	amountOfPostings := amountOfPostingsIn(pagePostings)
	receipt, err := c.pageReceiver.Receive(ctx, metadata, pagePostings)
	if err != nil {
		c.pageIntakeObserver.PageAdmissionFailed(
			ctx,
			message.Identity(),
			pageURL,
			amountOfPostings,
			err,
		)
		message.Return(ctx)

		return false
	}
	if receipt.Busy {
		c.pageIntakeObserver.PageAdmissionBusy(
			ctx,
			message.Identity(),
			pageURL,
			amountOfPostings,
		)
		message.Return(ctx)

		return false
	}
	c.pageIntakeObserver.PageAdmitted(
		ctx,
		message.Identity(),
		pageURL,
		amountOfPostings,
	)

	return true
}

func amountOfPostingsIn(pagePostings yacymodel.Optional[yacymodel.PagePostings]) int {
	indexedPostings, indexed := pagePostings.Get()
	if !indexed {
		return 0
	}

	return indexedPostings.AmountOfPostings()
}
