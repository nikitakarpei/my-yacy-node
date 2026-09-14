package pageintake

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageIntakeObserver interface {
	OfferedPageInvalid(ctx context.Context)
	PageOffered(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
	)
	DocumentExtractionFailed(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	NoIndexDerived(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
	)
	PageAdmitted(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
		amountOfPostings int,
	)
	PageAdmissionBusy(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
		amountOfPostings int,
	)
	PageAdmissionFailed(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
		amountOfPostings int,
		cause error,
	)
	PageIndexed(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
	)
}

type PageIntakeObservers []PageIntakeObserver

func (observers PageIntakeObservers) OfferedPageInvalid(ctx context.Context) {
	for _, observer := range observers {
		observer.OfferedPageInvalid(ctx)
	}
}

func (observers PageIntakeObservers) PageOffered(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.PageOffered(ctx, messageIdentity, pageURL)
	}
}

func (observers PageIntakeObservers) DocumentExtractionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.DocumentExtractionFailed(ctx, messageIdentity, pageURL, cause)
	}
}

func (observers PageIntakeObservers) NoIndexDerived(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.NoIndexDerived(ctx, messageIdentity, pageURL)
	}
}

func (observers PageIntakeObservers) PageAdmitted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	amountOfPostings int,
) {
	for _, observer := range observers {
		observer.PageAdmitted(ctx, messageIdentity, pageURL, amountOfPostings)
	}
}

func (observers PageIntakeObservers) PageAdmissionBusy(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	amountOfPostings int,
) {
	for _, observer := range observers {
		observer.PageAdmissionBusy(ctx, messageIdentity, pageURL, amountOfPostings)
	}
}

func (observers PageIntakeObservers) PageAdmissionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	amountOfPostings int,
	cause error,
) {
	for _, observer := range observers {
		observer.PageAdmissionFailed(ctx, messageIdentity, pageURL, amountOfPostings, cause)
	}
}

func (observers PageIntakeObservers) PageIndexed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.PageIndexed(ctx, messageIdentity, pageURL)
	}
}
