package scraperequestintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

//nolint:interfacebloat // One scrape attempt has these distinct observable domain facts.
type ScrapeProgress interface {
	ScrapeRequestInvalid(ctx context.Context)
	OriginFetchFailed(
		ctx context.Context,
		messageIdentity string,
		fetchURL canonicalurl.CanonicalURL,
		cause error,
	)
	OriginFetchDeferred(
		ctx context.Context,
		messageIdentity string,
		fetchURL canonicalurl.CanonicalURL,
		deferFor time.Duration,
	)
	NothingToScrape(
		ctx context.Context,
		messageIdentity string,
		fetchURL canonicalurl.CanonicalURL,
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
	ScrapeRequestCompleted(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
	)
}

type ScrapeProgressObservers []ScrapeProgress

func (observers ScrapeProgressObservers) ScrapeRequestInvalid(ctx context.Context) {
	for _, observer := range observers {
		observer.ScrapeRequestInvalid(ctx)
	}
}

func (observers ScrapeProgressObservers) OriginFetchFailed(
	ctx context.Context,
	messageIdentity string,
	fetchURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.OriginFetchFailed(ctx, messageIdentity, fetchURL, cause)
	}
}

func (observers ScrapeProgressObservers) OriginFetchDeferred(
	ctx context.Context,
	messageIdentity string,
	fetchURL canonicalurl.CanonicalURL,
	deferFor time.Duration,
) {
	for _, observer := range observers {
		observer.OriginFetchDeferred(ctx, messageIdentity, fetchURL, deferFor)
	}
}

func (observers ScrapeProgressObservers) NothingToScrape(
	ctx context.Context,
	messageIdentity string,
	fetchURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.NothingToScrape(ctx, messageIdentity, fetchURL)
	}
}

func (observers ScrapeProgressObservers) DocumentExtractionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.DocumentExtractionFailed(ctx, messageIdentity, pageURL, cause)
	}
}

func (observers ScrapeProgressObservers) NoIndexDerived(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.NoIndexDerived(ctx, messageIdentity, pageURL)
	}
}

func (observers ScrapeProgressObservers) URLMetadataAdmitted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.URLMetadataAdmitted(ctx, messageIdentity, pageURL)
	}
}

func (observers ScrapeProgressObservers) URLMetadataAdmissionBusy(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.URLMetadataAdmissionBusy(ctx, messageIdentity, pageURL)
	}
}

func (observers ScrapeProgressObservers) URLMetadataAdmissionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.URLMetadataAdmissionFailed(ctx, messageIdentity, pageURL, cause)
	}
}

func (observers ScrapeProgressObservers) PostingsAdmitted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	postings int,
) {
	for _, observer := range observers {
		observer.PostingsAdmitted(ctx, messageIdentity, pageURL, postings)
	}
}

func (observers ScrapeProgressObservers) PostingsAdmissionBusy(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	postings int,
) {
	for _, observer := range observers {
		observer.PostingsAdmissionBusy(ctx, messageIdentity, pageURL, postings)
	}
}

func (observers ScrapeProgressObservers) PostingsAdmissionFailed(
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

func (observers ScrapeProgressObservers) ScrapeRequestCompleted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.ScrapeRequestCompleted(ctx, messageIdentity, pageURL)
	}
}
