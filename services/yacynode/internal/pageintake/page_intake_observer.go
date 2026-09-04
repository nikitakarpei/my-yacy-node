package pageintake

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

//nolint:interfacebloat // One page intake has these distinct observable domain facts.
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
	URLMetadataAdmitted(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
	)
	URLMetadataAdmissionBusy(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
	)
	URLMetadataAdmissionFailed(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	PostingsAdmitted(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
		postings int,
	)
	PostingsAdmissionBusy(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
		postings int,
	)
	PostingsAdmissionFailed(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
		postings int,
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

func (observers PageIntakeObservers) URLMetadataAdmitted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.URLMetadataAdmitted(ctx, messageIdentity, pageURL)
	}
}

func (observers PageIntakeObservers) URLMetadataAdmissionBusy(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.URLMetadataAdmissionBusy(ctx, messageIdentity, pageURL)
	}
}

func (observers PageIntakeObservers) URLMetadataAdmissionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.URLMetadataAdmissionFailed(ctx, messageIdentity, pageURL, cause)
	}
}

func (observers PageIntakeObservers) PostingsAdmitted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	postings int,
) {
	for _, observer := range observers {
		observer.PostingsAdmitted(ctx, messageIdentity, pageURL, postings)
	}
}

func (observers PageIntakeObservers) PostingsAdmissionBusy(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	postings int,
) {
	for _, observer := range observers {
		observer.PostingsAdmissionBusy(ctx, messageIdentity, pageURL, postings)
	}
}

func (observers PageIntakeObservers) PostingsAdmissionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	postings int,
	cause error,
) {
	for _, observer := range observers {
		observer.PostingsAdmissionFailed(ctx, messageIdentity, pageURL, postings, cause)
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
