package pageintake

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

//nolint:interfacebloat // One page intake has these distinct observable domain facts.
type IntakeProgress interface {
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
	IntakeReceiptNotSent(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	PageIndexed(
		ctx context.Context,
		messageIdentity string,
		pageURL canonicalurl.CanonicalURL,
	)
}

type IntakeProgressObservers []IntakeProgress

func (observers IntakeProgressObservers) OfferedPageInvalid(ctx context.Context) {
	for _, observer := range observers {
		observer.OfferedPageInvalid(ctx)
	}
}

func (observers IntakeProgressObservers) PageOffered(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.PageOffered(ctx, messageIdentity, pageURL)
	}
}

func (observers IntakeProgressObservers) DocumentExtractionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.DocumentExtractionFailed(ctx, messageIdentity, pageURL, cause)
	}
}

func (observers IntakeProgressObservers) NoIndexDerived(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.NoIndexDerived(ctx, messageIdentity, pageURL)
	}
}

func (observers IntakeProgressObservers) URLMetadataAdmitted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.URLMetadataAdmitted(ctx, messageIdentity, pageURL)
	}
}

func (observers IntakeProgressObservers) URLMetadataAdmissionBusy(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.URLMetadataAdmissionBusy(ctx, messageIdentity, pageURL)
	}
}

func (observers IntakeProgressObservers) URLMetadataAdmissionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.URLMetadataAdmissionFailed(ctx, messageIdentity, pageURL, cause)
	}
}

func (observers IntakeProgressObservers) PostingsAdmitted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	postings int,
) {
	for _, observer := range observers {
		observer.PostingsAdmitted(ctx, messageIdentity, pageURL, postings)
	}
}

func (observers IntakeProgressObservers) PostingsAdmissionBusy(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	postings int,
) {
	for _, observer := range observers {
		observer.PostingsAdmissionBusy(ctx, messageIdentity, pageURL, postings)
	}
}

func (observers IntakeProgressObservers) PostingsAdmissionFailed(
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

func (observers IntakeProgressObservers) IntakeReceiptNotSent(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.IntakeReceiptNotSent(ctx, messageIdentity, pageURL, cause)
	}
}

func (observers IntakeProgressObservers) PageIndexed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.PageIndexed(ctx, messageIdentity, pageURL)
	}
}
