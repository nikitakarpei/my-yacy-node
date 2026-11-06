package pageintake

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageIntakeObserver interface {
	PageRWIInvalid(ctx context.Context)
	PageRWIReceived(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
	)
	PostingsRootMismatched(
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

func (observers PageIntakeObservers) PageRWIInvalid(ctx context.Context) {
	for _, observer := range observers {
		observer.PageRWIInvalid(ctx)
	}
}

func (observers PageIntakeObservers) PageRWIReceived(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.PageRWIReceived(ctx, messageIdentity, pageURL)
	}
}

func (observers PageIntakeObservers) PostingsRootMismatched(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.PostingsRootMismatched(ctx, messageIdentity, pageURL)
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
